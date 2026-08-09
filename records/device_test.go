package records

import (
	"testing"

	"github.com/rubenbe/pocketbase/core"
	"github.com/stretchr/testify/assert"
)

func newDeviceRecord() *core.Record {
	collection := core.NewBaseCollection(CollectionNameDevices)
	collection.Fields.Add(&core.SelectField{
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

	return core.NewRecord(collection)
}

func TestDeviceIsConfigApplied(t *testing.T) {
	scenarios := []struct {
		status   string
		expected bool
	}{
		{ConfigStatusApplied, true},
		{ConfigStatusModified, false},
		{ConfigStatusError, false},
		{ConfigStatusDeactivating, false},
		{ConfigStatusDeactivated, false},
	}

	for _, s := range scenarios {
		t.Run(s.status, func(t *testing.T) {
			record := newDeviceRecord()
			record.Set("config_status", s.status)

			device := NewDevice(record)
			assert.Equal(t, s.status, device.ConfigStatus())
			assert.Equal(t, s.expected, device.IsConfigApplied())
		})
	}
}

func TestDeviceConfigStatusUnset(t *testing.T) {
	record := newDeviceRecord()
	device := NewDevice(record)

	assert.Equal(t, "", device.ConfigStatus(), "an unset config_status is empty")
	assert.Equal(t, false, device.IsConfigApplied())

	assert.Equal(t, record, device.ProxyRecord())
}
