package poe

import (
	"github.com/pocketbase/dbx"
	"github.com/rubenbe/pocketbase/core"
)

func Sync(app core.App, device *core.Record, info Info) error {
	coll, err := app.FindCollectionByNameOrId("poe")
	if err != nil {
		return err
	}

	ports := info.NormalizedPorts()

	return app.RunInTransaction(func(txApp core.App) error {
		existing, err := txApp.FindAllRecords("poe", dbx.HashExp{"device": device.Id})
		if err != nil {
			return err
		}
		byPort := make(map[string]*core.Record, len(existing))
		for _, rec := range existing {
			byPort[rec.GetString("port")] = rec
		}

		for _, p := range ports {
			rec, found := byPort[p.Name]
			if found {
				// Don't update existing, unchanged poe entry
				if rec.GetString("status") == p.Status &&
					rec.GetFloat("consumption") == p.Consumption {
					continue
				}
			} else {
				rec = core.NewRecord(coll)
				rec.Set("device", device.Id)
				rec.Set("port", p.Name)
				// Only write the priority the first time to read it in.
				// Afterwards it is considered to be modified only by the user
				rec.Set("priority", p.Priority)
			}
			rec.Set("status", p.Status)
			rec.Set("consumption", p.Consumption)
			if err := txApp.Save(rec); err != nil {
				return err
			}
		}
		return nil
	})
}
