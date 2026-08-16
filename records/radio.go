package records

import (
	"github.com/rubenbe/opensoho/frequencyplan"
	"github.com/rubenbe/pocketbase/core"
)

const CollectionNameRadios = "radios"

var (
	_ core.Model       = (*Radio)(nil)
	_ core.RecordProxy = (*Radio)(nil)
)

var radioConfigFields = []string{
	"radio", "device", "frequency", "htmode",
	"auto_frequency", "enabled", "tx_power", "tx_power_mode",
}

type Radio struct {
	core.BaseRecordProxy
}

func NewRadio(record *core.Record) *Radio {
	r := &Radio{}
	r.SetProxyRecord(record)

	return r
}

func (r *Radio) DeviceId() string {
	return r.GetString("device")
}

// Band returns the Wi-Fi band ("2.4", "5", "6", "60", or "unknown") the
// radio's configured frequency falls into.
func (r *Radio) Band() string {
	return frequencyplan.FrequencyToBand(r.GetInt("frequency"))
}

func (r *Radio) ConfigChanged() bool {
	original := r.Original()
	for _, field := range radioConfigFields {
		if r.GetString(field) != original.GetString(field) {
			return true
		}
	}
	return false
}

func (r *Radio) MarkDeviceModified(app core.App) error {
	deviceId := r.DeviceId()
	if deviceId == "" {
		return nil
	}

	record, err := app.FindRecordById(CollectionNameDevices, deviceId)
	if err != nil {
		return err
	}

	device := NewDevice(record)
	if !device.MarkConfigModified() {
		return nil
	}
	return app.Save(device)
}
