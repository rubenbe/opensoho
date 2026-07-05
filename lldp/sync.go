package lldp

import (
	"github.com/pocketbase/dbx"
	"github.com/rubenbe/pocketbase/core"
)

func Sync(app core.App, device *core.Record, info Info) error {
	coll, err := app.FindCollectionByNameOrId("lldp")
	if err != nil {
		return err
	}

	neighbors := info.Normalized()

	return app.RunInTransaction(func(txApp core.App) error {
		existing, err := txApp.FindAllRecords("lldp", dbx.HashExp{"device": device.Id})
		if err != nil {
			return err
		}
		// A single local port can carry several neighbours, so key by the
		// composite (port, name) rather than port alone.
		byKey := make(map[string]*core.Record, len(existing))
		for _, rec := range existing {
			byKey[rowKey(rec.GetString("port"), rec.GetString("neighbor_name"), rec.GetString("neighbor_mac_address"))] = rec
		}

		reported := make(map[string]bool, len(neighbors))
		for _, n := range neighbors {
			key := rowKey(n.Port, n.Name, n.Mac)
			reported[key] = true
			if _, found := byKey[key]; found {
				// port, name and MAC are the whole key; an existing match has
				// nothing to update, so leave it (and its timestamp) alone.
				continue
			}
			rec := core.NewRecord(coll)
			rec.Set("device", device.Id)
			rec.Set("port", n.Port)
			rec.Set("neighbor_name", n.Name)
			rec.Set("neighbor_mac_address", n.Mac)
			if err := txApp.Save(rec); err != nil {
				return err
			}
		}

		// Delete rows absent from this report, but only when the report actually
		// listed neighbours: an empty report must not wipe the collection.
		if len(neighbors) > 0 {
			for _, rec := range existing {
				if reported[rowKey(rec.GetString("port"), rec.GetString("neighbor_name"), rec.GetString("neighbor_mac_address"))] {
					continue
				}
				if err := txApp.Delete(rec); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func rowKey(port, name, mac string) string {
	return port + "\x00" + name + "\x00" + mac
}
