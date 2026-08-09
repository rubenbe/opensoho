package records

import (
	"testing"

	"github.com/rubenbe/pocketbase/core"
	"github.com/rubenbe/pocketbase/tests"
	"github.com/stretchr/testify/assert"
)

func newRadioRecord() *core.Record {
	collection := core.NewBaseCollection(CollectionNameRadios)
	collection.Fields.Add(&core.NumberField{Name: "radio"})
	collection.Fields.Add(&core.TextField{Name: "device"})
	collection.Fields.Add(&core.NumberField{Name: "frequency"})
	collection.Fields.Add(&core.TextField{Name: "htmode"})
	collection.Fields.Add(&core.BoolField{Name: "auto_frequency"})
	collection.Fields.Add(&core.BoolField{Name: "enabled"})
	collection.Fields.Add(&core.NumberField{Name: "tx_power"})
	collection.Fields.Add(&core.TextField{Name: "tx_power_mode"})

	record := core.NewRecord(collection)
	record.Id = "somerandomradio15"
	return record
}

func TestRadioConfigChanged(t *testing.T) {
	record := newRadioRecord()
	record.Set("radio", 0)
	record.Set("device", "somerandomdevice1")
	record.Set("frequency", 2412)
	record.Set("htmode", "HE20")
	record.Set("enabled", true)
	record.Set("tx_power_mode", "auto")
	// PostScan resets the record's "original" state to whatever is currently
	// set, standing in for a fresh load from the db.
	assert.Nil(t, record.PostScan())

	radio := NewRadio(record)
	assert.False(t, radio.ConfigChanged(), "no fields touched since the last save")

	for _, field := range radioConfigFields {
		t.Run(field, func(t *testing.T) {
			record := newRadioRecord()
			record.Set("radio", 0)
			record.Set("device", "somerandomdevice1")
			record.Set("frequency", 2412)
			record.Set("htmode", "HE20")
			record.Set("enabled", true)
			record.Set("tx_power_mode", "auto")
			assert.Nil(t, record.PostScan())

			switch field {
			case "auto_frequency", "enabled":
				record.Set(field, !record.GetBool(field))
			case "radio", "frequency", "tx_power":
				record.Set(field, record.GetInt(field)+1)
			default:
				record.Set(field, record.GetString(field)+"-changed")
			}

			radio := NewRadio(record)
			assert.True(t, radio.ConfigChanged(), "%s changed", field)
		})
	}
}

func TestRadioMarkDeviceModified(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	devicecollection := core.NewBaseCollection(CollectionNameDevices)
	devicecollection.Fields.Add(&core.SelectField{
		Name:      "config_status",
		MaxSelect: 1,
		Values: []string{
			ConfigStatusApplied,
			ConfigStatusModified,
			ConfigStatusError,
			ConfigStatusDeactivating,
			ConfigStatusDeactivated,
		},
	})
	assert.Nil(t, app.Save(devicecollection))

	scenarios := []struct {
		status  string
		flipped bool
	}{
		{ConfigStatusApplied, true},
		{ConfigStatusModified, false},
		{ConfigStatusError, false},
	}

	for _, s := range scenarios {
		t.Run(s.status, func(t *testing.T) {
			device := core.NewRecord(devicecollection)
			device.Set("config_status", s.status)
			assert.Nil(t, app.Save(device))

			radio := NewRadio(newRadioRecord())
			radio.Set("device", device.Id)

			assert.Nil(t, radio.MarkDeviceModified(app))

			reloaded, err := app.FindRecordById(CollectionNameDevices, device.Id)
			assert.Nil(t, err)
			if s.flipped {
				assert.Equal(t, ConfigStatusModified, reloaded.GetString("config_status"))
			} else {
				assert.Equal(t, s.status, reloaded.GetString("config_status"))
			}
		})
	}
}

func TestRadioMarkDeviceModifiedWithoutDevice(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	radio := NewRadio(newRadioRecord())
	assert.Equal(t, "", radio.DeviceId())
	assert.Nil(t, radio.MarkDeviceModified(app))
}
