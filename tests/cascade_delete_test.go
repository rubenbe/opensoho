package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	_ "github.com/pocketbase/pocketbase/migrations" // registers core.SystemMigrations
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	loadJSMigrations sync.Once
	jsMigrations     core.MigrationsList
)

// jsvm registers the JS migrations in the global core.AppMigrations list,
// which tests.NewTestApp also runs. Collect them in a separate list instead.
func pbMigrations() core.MigrationsList {
	loadJSMigrations.Do(func() {
		orig := core.AppMigrations
		core.AppMigrations = core.MigrationsList{}
		jsvm.MustRegister(core.NewBaseApp(core.BaseAppConfig{}), jsvm.Config{MigrationsDir: "../pb_migrations"})
		jsMigrations = core.AppMigrations
		core.AppMigrations = orig
		if len(jsMigrations.Items()) == 0 {
			panic("no JS migrations found in ../pb_migrations")
		}
	})
	return jsMigrations
}

// setupMigratedMemoryApp returns an app backed by in-memory SQLite databases
// with the real pb_migrations applied, so the actual schema is tested.
func setupMigratedMemoryApp(t *testing.T) core.App {
	t.Helper()

	dbPrefix := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir: t.TempDir(),
		DBConnect: func(dbPath string) (*dbx.DB, error) {
			// The memdb vfs shares the database between all connections
			// of the app that use the same name.
			name := dbPrefix + "_" + strings.TrimSuffix(dbPath[strings.LastIndex(dbPath, "/")+1:], ".db")
			return dbx.Open("sqlite", fmt.Sprintf("file:/%s?vfs=memdb&_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)", name))
		},
	})
	t.Cleanup(func() { app.ResetBootstrapState() })

	require.NoError(t, app.Bootstrap())

	migrations := core.MigrationsList{}
	migrations.Copy(core.SystemMigrations)
	migrations.Copy(pbMigrations())
	_, err := core.NewMigrationsRunner(app, migrations).Up()
	require.NoError(t, err)

	return app
}

func TestDeviceDeleteCascades(t *testing.T) {
	app := setupMigratedMemoryApp(t)

	devices, err := app.FindCollectionByNameOrId("devices")
	require.NoError(t, err)

	newDevice := func(name string) *core.Record {
		d := core.NewRecord(devices)
		d.Set("name", name)
		require.NoError(t, app.SaveNoValidate(d))
		return d
	}
	newChild := func(collection string, device *core.Record) *core.Record {
		c, err := app.FindCollectionByNameOrId(collection)
		require.NoError(t, err)
		r := core.NewRecord(c)
		// Unique text values to satisfy the unique indexes (mac_address, name, ...)
		for _, f := range c.Fields {
			if _, ok := f.(*core.TextField); ok && !f.GetSystem() {
				r.Set(f.GetName(), f.GetName()+"_"+core.GenerateDefaultRandomId())
			}
		}
		r.Set("device", device.Id)
		require.NoError(t, app.SaveNoValidate(r))
		return r
	}

	cascaded := []string{
		"bridges",
		"client_resources",
		"clients",
		"device_resources",
		"ethernet",
		"interfaces",
		"lldp",
		"poe",
		"radio_frequencies",
		"radio_ht_modes",
		"radio_tx_powers",
		"radios",
	}

	doomed := newDevice("doomed")
	survivor := newDevice("survivor")

	for _, collection := range cascaded {
		newChild(collection, doomed)
		newChild(collection, survivor)
	}

	// A bridge referencing the device's own ethernet port and wifi interface
	eth := newChild("ethernet", doomed)
	iface := newChild("interfaces", doomed)
	bridge := newChild("bridges", doomed)
	bridge.Set("ethernet", []string{eth.Id})
	bridge.Set("wifi", []string{iface.Id})
	require.NoError(t, app.SaveNoValidate(bridge))

	require.NoError(t, app.Delete(doomed))

	for _, collection := range cascaded {
		remaining, err := app.FindAllRecords(collection, dbx.HashExp{"device": doomed.Id})
		assert.NoError(t, err, collection)
		assert.Empty(t, remaining, collection)

		kept, err := app.FindAllRecords(collection, dbx.HashExp{"device": survivor.Id})
		assert.NoError(t, err, collection)
		assert.Len(t, kept, 1, collection)
	}

	_, err = app.FindRecordById("devices", survivor.Id)
	assert.NoError(t, err)
}

func TestClientDeleteCascadesToItsHistory(t *testing.T) {
	app := setupMigratedMemoryApp(t)

	newRecord := func(collection string, data map[string]any) *core.Record {
		c, err := app.FindCollectionByNameOrId(collection)
		require.NoError(t, err)
		r := core.NewRecord(c)
		for k, v := range data {
			r.Set(k, v)
		}
		require.NoError(t, app.SaveNoValidate(r))
		return r
	}

	device := newRecord("devices", map[string]any{"name": "ap"})
	doomed := newRecord("clients", map[string]any{"mac_address": "11:22:33:44:55:66"})
	survivor := newRecord("clients", map[string]any{"mac_address": "66:55:44:33:22:11"})
	for _, client := range []*core.Record{doomed, survivor} {
		newRecord("client_resources", map[string]any{"client": client.Id, "device": device.Id, "time": "2026-10-25 12:00:00.000Z"})
	}

	require.NoError(t, app.Delete(doomed))

	remaining, err := app.FindAllRecords("client_resources")
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, survivor.Id, remaining[0].GetString("client"))
	assert.Equal(t, device.Id, remaining[0].GetString("device"))
}
