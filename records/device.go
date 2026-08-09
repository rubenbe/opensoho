package records

import (
	"github.com/rubenbe/pocketbase/core"
)

const CollectionNameDevices = "devices"

// Possible values of the config_status field of a device.
const (
	ConfigStatusApplied      = "applied"
	ConfigStatusModified     = "modified"
	ConfigStatusError        = "error"
	ConfigStatusDeactivating = "deactivating"
	ConfigStatusDeactivated  = "deactivated"
)

var (
	_ core.Model       = (*Device)(nil)
	_ core.RecordProxy = (*Device)(nil)
)

// Device is a Record proxy for the devices collection. Only config_status is
// exposed as a typed field, all other fields are still accessed through the
// embedded Record.
type Device struct {
	core.BaseRecordProxy
}

// NewDevice wraps an existing device record into a *Device proxy.
func NewDevice(record *core.Record) *Device {
	d := &Device{}
	d.SetProxyRecord(record)

	return d
}

// ConfigStatus returns the raw config_status of the device.
func (d *Device) ConfigStatus() string {
	return d.GetString("config_status")
}

// IsConfigApplied reports whether the device is running the configuration the
// controller generated for it.
func (d *Device) IsConfigApplied() bool {
	return d.ConfigStatus() == ConfigStatusApplied
}
