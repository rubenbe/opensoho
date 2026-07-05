package lldp

import (
	"encoding/json"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/rubenbe/pocketbase/core"
	"github.com/rubenbe/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

func setupLldpCollection(t *testing.T, app core.App, devicecollection *core.Collection) *core.Collection {
	col := core.NewBaseCollection("lldp")
	col.Fields.Add(&core.RelationField{
		Name:         "device",
		Required:     true,
		MaxSelect:    1,
		CollectionId: devicecollection.Id,
	})
	col.Fields.Add(&core.TextField{
		Name:     "port",
		Required: true,
	})
	col.Fields.Add(&core.TextField{
		Name: "name",
	})
	col.Fields.Add(&core.AutodateField{
		Name:     "updated",
		OnUpdate: true,
	})
	err := app.Save(col)
	assert.Equal(t, nil, err)
	return col
}

func newTestDevice(t *testing.T) (*tests.TestApp, *core.Record) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)

	devicecollection := core.NewBaseCollection("devices")
	assert.Nil(t, app.Save(devicecollection))
	setupLldpCollection(t, app, devicecollection)

	d := core.NewRecord(devicecollection)
	assert.Nil(t, app.Save(d))
	return app, d
}

func TestSync(t *testing.T) {
	app, d := newTestDevice(t)
	defer app.Cleanup()

	var info Info
	assert.Nil(t, json.Unmarshal([]byte(twoInterfaces), &info))

	// First sync: every reported neighbour becomes a row.
	assert.Nil(t, Sync(app, d, info))

	recs, err := app.FindAllRecords("lldp", dbx.HashExp{"device": d.Id})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recs))

	eth0, err := app.FindFirstRecordByFilter("lldp", "port = 'eth0'")
	assert.Nil(t, err)
	assert.Equal(t, d.Id, eth0.GetString("device"))
	assert.Equal(t, "sw-core-01", eth0.GetString("name"))
	eth0Id := eth0.Id

	// A later report drops eth1 and keeps eth0 unchanged.
	assert.Nil(t, Sync(app, d, Info{Neighbors: []Neighbor{{Port: "eth0", Name: "sw-core-01"}}}))

	eth0, err = app.FindFirstRecordByFilter("lldp", "port = 'eth0'")
	assert.Nil(t, err)
	assert.Equal(t, eth0Id, eth0.Id, "unchanged neighbour must be kept in place, not re-created")

	_, err = app.FindFirstRecordByFilter("lldp", "port = 'eth1'")
	assert.Error(t, err, "unreported neighbour should be deleted")

	recs, err = app.FindAllRecords("lldp", dbx.HashExp{"device": d.Id})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recs))
}

// Multiple neighbours on the same local port each get their own row.
func TestSyncMultipleNeighborsPerPort(t *testing.T) {
	app, d := newTestDevice(t)
	defer app.Cleanup()

	info := Info{Neighbors: []Neighbor{
		{Port: "eth0", Name: "phone-01"},
		{Port: "eth0", Name: "pc-01"},
	}}
	assert.Nil(t, Sync(app, d, info))

	recs, err := app.FindAllRecords("lldp", dbx.HashExp{"device": d.Id})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recs))
}

// An empty report must not delete existing records.
func TestSyncEmptyReportKeepsRecords(t *testing.T) {
	app, d := newTestDevice(t)
	defer app.Cleanup()

	var info Info
	assert.Nil(t, json.Unmarshal([]byte(twoInterfaces), &info))
	assert.Nil(t, Sync(app, d, info))

	// A report with no neighbours leaves the existing rows untouched.
	assert.Nil(t, Sync(app, d, Info{}))

	recs, err := app.FindAllRecords("lldp", dbx.HashExp{"device": d.Id})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recs), "empty report must not delete records")
}

// Re-syncing identical neighbours must not re-save the row (updated stays put).
func TestSyncSkipsUnchanged(t *testing.T) {
	app, d := newTestDevice(t)
	defer app.Cleanup()

	var info Info
	assert.Nil(t, json.Unmarshal([]byte(oneInterface), &info))
	assert.Nil(t, Sync(app, d, info))

	eth0, err := app.FindFirstRecordByFilter("lldp", "port = 'eth0'")
	assert.Nil(t, err)
	updatedBefore := eth0.GetString("updated")

	assert.Nil(t, Sync(app, d, info))

	eth0, err = app.FindFirstRecordByFilter("lldp", "port = 'eth0'")
	assert.Nil(t, err)
	assert.Equal(t, updatedBefore, eth0.GetString("updated"),
		"unchanged neighbour must not be re-saved")
}
