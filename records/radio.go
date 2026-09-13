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
	"radio", "device", "frequency", "htmode", "band",
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

// Band returns the band the radio operates on: the one its pinned frequency
// falls in, else the band the user picked or the device reported.
func (r *Radio) Band() string {
	if !r.GetBool("auto_frequency") {
		if band := frequencyplan.FrequencyToBand(r.GetInt("frequency")); band != "unknown" {
			return band
		}
	}
	return r.GetString("band")
}

// IsBand2Ghz reports whether the radio's configured frequency is in the
// 2.4 GHz band.
func (r *Radio) IsBand2Ghz() bool {
	return r.Band() == "2.4"
}

// IsBand5GHz reports whether the radio's configured frequency is in the
// 5 GHz band.
func (r *Radio) IsBand5GHz() bool {
	return r.Band() == "5"
}

// IsBand6GHz reports whether the radio's configured frequency is in the
// 6 GHz band.
func (r *Radio) IsBand6GHz() bool {
	return r.Band() == "6"
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
