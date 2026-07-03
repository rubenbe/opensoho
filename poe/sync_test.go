package poe

import (
	"encoding/json"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/rubenbe/pocketbase/core"
	"github.com/rubenbe/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

func setupPoeCollection(t *testing.T, app core.App, devicecollection *core.Collection) *core.Collection {
	col := core.NewBaseCollection("poe")
	col.Fields.Add(&core.RelationField{
		Name:         "device",
		Required:     false,
		MaxSelect:    1,
		CollectionId: devicecollection.Id,
	})
	col.Fields.Add(&core.TextField{
		Name:     "port",
		Required: true,
	})
	col.Fields.Add(&core.SelectField{
		Name:      "priority",
		Required:  true,
		MaxSelect: 1,
		Values:    []string{"low", "normal", "high", "critical"},
	})
	col.Fields.Add(&core.TextField{
		Name: "status",
	})
	col.Fields.Add(&core.NumberField{
		Name: "consumption",
	})
	col.Fields.Add(&core.AutodateField{
		Name:     "updated",
		OnUpdate: true,
	})
	err := app.Save(col)
	assert.Equal(t, nil, err)
	return col
}

func TestSync(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	devicecollection := core.NewBaseCollection("devices")
	assert.Nil(t, app.Save(devicecollection))
	setupPoeCollection(t, app, devicecollection)

	d := core.NewRecord(devicecollection)
	assert.Nil(t, app.Save(d))

	var info Info
	assert.Nil(t, json.Unmarshal([]byte(sample), &info))

	// First sync: every reported port becomes a row.
	assert.Nil(t, Sync(app, d, info))

	recs, err := app.FindAllRecords("poe", dbx.HashExp{"device": d.Id})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recs))

	// lan1 is searching: priority seeded to low, status verbatim, no consumption.
	lan1, err := app.FindFirstRecordByFilter("poe", "port = 'lan1'")
	assert.Nil(t, err)
	assert.Equal(t, d.Id, lan1.GetString("device"))
	assert.Equal(t, "low", lan1.GetString("priority"))
	assert.Equal(t, "Searching", lan1.GetString("status"))
	assert.Equal(t, 0.0, lan1.GetFloat("consumption"))
	lan1Id := lan1.Id

	// lan2 is delivering power: its consumption is recorded.
	lan2, err := app.FindFirstRecordByFilter("poe", "port = 'lan2'")
	assert.Nil(t, err)
	assert.Equal(t, "Delivering power", lan2.GetString("status"))
	assert.Equal(t, 3.3, lan2.GetFloat("consumption"))

	// Set lan1 priority to critical.
	lan1.Set("priority", "critical")
	assert.Nil(t, app.Save(lan1))

	info.Ports["lan1"] = RawPort{Priority: 0, Mode: "PoE", Status: "Delivering power", Consumption: 6.0}
	delete(info.Ports, "lan2")
	assert.Nil(t, Sync(app, d, info))

	lan1, err = app.FindFirstRecordByFilter("poe", "port = 'lan1'")
	assert.Nil(t, err)
	assert.Equal(t, lan1Id, lan1.Id, "lan1 row must be updated in place, not re-created")
	// priority should be preserved
	assert.Equal(t, "critical", lan1.GetString("priority"), "admin priority must be preserved")
	assert.Equal(t, "Delivering power", lan1.GetString("status"))
	assert.Equal(t, 6.0, lan1.GetFloat("consumption"))

	// lan2 was absent from the (non-empty) report, so it is deleted.
	_, err = app.FindFirstRecordByFilter("poe", "port = 'lan2'")
	assert.Error(t, err, "unreported port should be deleted")

	recs, err = app.FindAllRecords("poe", dbx.HashExp{"device": d.Id})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recs))
}

// An empty report must not delete existing records.
func TestSyncEmptyReportKeepsRecords(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	devicecollection := core.NewBaseCollection("devices")
	assert.Nil(t, app.Save(devicecollection))
	setupPoeCollection(t, app, devicecollection)

	d := core.NewRecord(devicecollection)
	assert.Nil(t, app.Save(d))

	var info Info
	assert.Nil(t, json.Unmarshal([]byte(sample), &info))
	assert.Nil(t, Sync(app, d, info))

	// A report with no ports leaves the existing rows untouched.
	assert.Nil(t, Sync(app, d, Info{}))

	recs, err := app.FindAllRecords("poe", dbx.HashExp{"device": d.Id})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recs), "empty report must not delete records")
}

// TestSyncSkipsUnchanged verifies that re-syncing identical telemetry does not
// touch the row: its updated timestamp must stay the same.
func TestSyncSkipsUnchanged(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	devicecollection := core.NewBaseCollection("devices")
	assert.Nil(t, app.Save(devicecollection))
	setupPoeCollection(t, app, devicecollection)

	d := core.NewRecord(devicecollection)
	assert.Nil(t, app.Save(d))

	var info Info
	assert.Nil(t, json.Unmarshal([]byte(sample), &info))

	assert.Nil(t, Sync(app, d, info))

	lan1, err := app.FindFirstRecordByFilter("poe", "port = 'lan1'")
	assert.Nil(t, err)
	updatedBefore := lan1.GetString("updated")

	// Re-sync the exact same telemetry: nothing changed, so no write.
	assert.Nil(t, Sync(app, d, info))

	lan1, err = app.FindFirstRecordByFilter("poe", "port = 'lan1'")
	assert.Nil(t, err)
	assert.Equal(t, updatedBefore, lan1.GetString("updated"),
		"unchanged port must not be re-saved")

	// A changed status does write, bumping the timestamp.
	info.Ports["lan1"] = RawPort{Priority: 0, Mode: "PoE", Status: "Delivering power", Consumption: 6.0}
	assert.Nil(t, Sync(app, d, info))

	lan1, err = app.FindFirstRecordByFilter("poe", "port = 'lan1'")
	assert.Nil(t, err)
	assert.NotEqual(t, updatedBefore, lan1.GetString("updated"),
		"changed port must be re-saved")
}
