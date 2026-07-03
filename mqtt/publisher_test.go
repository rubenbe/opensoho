package mqtt

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopics(t *testing.T) {
	assert.Equal(t, "opensoho/abc/status", deviceStatusTopic("abc"))
	assert.Equal(t, "opensoho/abc/poe/lan4", stateTopic("abc", "lan4"))
	assert.Equal(t, "homeassistant/sensor/opensoho_abc/lan4/config", discoveryTopic("abc", "lan4"))
}

func TestFormatWatts(t *testing.T) {
	cases := map[float64]string{
		0:    "0",
		6.4:  "6.4",
		12:   "12",
		3.25: "3.25",
	}
	for in, want := range cases {
		assert.Equal(t, want, formatWatts(in), "formatWatts(%v)", in)
	}
}

func TestBuildDiscoveryConfig(t *testing.T) {
	payload, err := buildDiscoveryConfig("dev123", "switch-rack", "lan4")
	assert.NoError(t, err)

	var cfg discoveryConfig
	assert.NoError(t, json.Unmarshal(payload, &cfg))

	assert.Equal(t, "PoE lan4", cfg.Name)
	assert.Equal(t, "opensoho_dev123_poe_lan4", cfg.UniqueID)
	assert.Equal(t, "opensoho/dev123/poe/lan4", cfg.StateTopic)
	assert.Equal(t, "W", cfg.UnitOfMeasurement)
	assert.Equal(t, "power", cfg.DeviceClass)
	assert.Equal(t, "measurement", cfg.StateClass)
	assert.Equal(t, "all", cfg.AvailabilityMode)
	assert.Len(t, cfg.Availability, 2)
	assert.Equal(t, serviceStatusTopic, cfg.Availability[0].Topic)
	assert.Equal(t, "opensoho/dev123/status", cfg.Availability[1].Topic)
	assert.Equal(t, "switch-rack", cfg.Device.Name)
	assert.Equal(t, []string{"opensoho_dev123"}, cfg.Device.Identifiers)
}

func TestBuildDiscoveryConfigFallsBackToID(t *testing.T) {
	payload, err := buildDiscoveryConfig("dev123", "", "lan1")
	assert.NoError(t, err)

	var cfg discoveryConfig
	assert.NoError(t, json.Unmarshal(payload, &cfg))
	assert.Equal(t, "dev123", cfg.Device.Name, "Use the device ID as backup device name")
}

// Nil/unconnected publishers must be no-ops, since call sites publish
// unconditionally.
func TestNilPublisherIsNoop(t *testing.T) {
	var p *Publisher
	assert.NotPanics(t, func() {
		p.PublishDeviceOffline("x")
		p.Close()

		// Default singleton is nil until Configure is called.
		PublishDeviceOffline("x")
		Close()
	})
}
