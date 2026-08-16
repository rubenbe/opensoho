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
	zero := 0.0
	collection.Fields.Add(&core.NumberField{
		Name:    "numradios",
		Min:     &zero,
		OnlyInt: true,
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

func TestDeviceMarkConfigModified(t *testing.T) {
	scenarios := []struct {
		status  string
		flipped bool
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
			assert.Equal(t, s.flipped, device.MarkConfigModified())

			if s.flipped {
				assert.Equal(t, ConfigStatusModified, device.ConfigStatus())
			} else {
				assert.Equal(t, s.status, device.ConfigStatus())
			}
		})
	}
}

func TestDeviceEnsureNumRadios(t *testing.T) {
	scenarios := []struct {
		name     string
		current  int
		detected int
		changed  bool
		expected int
	}{
		{"raises below detected", 1, 2, true, 2},
		{"leaves equal alone", 2, 2, false, 2},
		{"never lowers", 2, 1, false, 2},
		{"leaves zero detection alone", 2, 0, false, 2},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			record := newDeviceRecord()
			record.Set("numradios", s.current)

			device := NewDevice(record)
			assert.Equal(t, s.changed, device.EnsureNumRadios(s.detected))
			assert.Equal(t, s.expected, device.NumRadios())
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
