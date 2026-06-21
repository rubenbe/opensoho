// Publishes to MQTT primarily for integration within Home Assistant.
// Uses the autodiscovery mechanisms
// Only live data can be published even if OpenWisp can send data retroactively.
// TODO: SSL not suppored yet.

package mqtt

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/rubenbe/opensoho/poe"
	"github.com/rubenbe/pocketbase/core"
)

const (
	// Describes the availability of opensoho itself.
	// It is set using the MQTT Last Will mechanism
	serviceStatusTopic = "opensoho/status"

	payloadOnline  = "online"
	payloadOffline = "offline"

	connectTimeout = 10 * time.Second
)

// Config holds the broker connection parameters, sourced from the settings
// collection.
type Config struct {
	Enabled  bool
	Broker   string // e.g. tcp://host:1883 or ssl://host:8883
	Username string
	Password string
}

// Publisher wraps an MQTT client plus a cache of which (device, port) discovery
// configs have already been published in the current connection.
type Publisher struct {
	client paho.Client

	mu        sync.Mutex
	published map[string]bool // key: "<deviceID>/<port>"
}

// per-device availability topic.
func deviceStatusTopic(deviceID string) string {
	return fmt.Sprintf("opensoho/%s/status", deviceID)
}

// where a port's current consumption (W) will be published.
func stateTopic(deviceID string, port int) string {
	return fmt.Sprintf("opensoho/%s/poe/port%d", deviceID, port)
}

// the retained HA autodiscovery config topic.
func discoveryTopic(deviceID string, port int) string {
	return fmt.Sprintf("homeassistant/sensor/opensoho_%s/port%d/config", deviceID, port)
}

// mirrors Home Assistant's availability list item.
type availabilityEntry struct {
	Topic string `json:"topic"`
}

// groups all of a switch's port sensors under one HA device.
type discoveryDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
}

// the JSON payload of a HA MQTT sensor discovery message.
type discoveryConfig struct {
	Name              string              `json:"name"`
	UniqueID          string              `json:"unique_id"`
	ObjectID          string              `json:"object_id"`
	StateTopic        string              `json:"state_topic"`
	UnitOfMeasurement string              `json:"unit_of_measurement"`
	DeviceClass       string              `json:"device_class"`
	StateClass        string              `json:"state_class"`
	Availability      []availabilityEntry `json:"availability"`
	AvailabilityMode  string              `json:"availability_mode"`
	Device            discoveryDevice     `json:"device"`
}

// buildDiscoveryConfig assembles the autodiscovery payload for one PoE port.
func buildDiscoveryConfig(deviceID, deviceName string, port int) ([]byte, error) {
	if deviceName == "" {
		deviceName = deviceID
	}
	cfg := discoveryConfig{
		Name:              fmt.Sprintf("PoE Port %d", port),
		UniqueID:          fmt.Sprintf("opensoho_%s_poe_port%d", deviceID, port),
		ObjectID:          fmt.Sprintf("opensoho_%s_poe_port%d", deviceName, port),
		StateTopic:        stateTopic(deviceID, port),
		UnitOfMeasurement: "W",
		DeviceClass:       "power",
		StateClass:        "measurement",
		Availability: []availabilityEntry{
			{Topic: serviceStatusTopic},
			{Topic: deviceStatusTopic(deviceID)},
		},
		AvailabilityMode: "all",
		Device: discoveryDevice{
			Identifiers:  []string{"opensoho_" + deviceID},
			Name:         deviceName,
			Manufacturer: "OpenSOHO",
		},
	}
	return json.Marshal(cfg)
}

// formatWatts renders a consumption value without trailing zeros.
func formatWatts(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// newPublisher connects to the broker. It always returns a usable Publisher
// (auto-reconnect keeps trying); the error only signals that the initial
// connection attempt failed within the timeout.
func newPublisher(cfg Config) (*Publisher, error) {
	p := &Publisher{published: map[string]bool{}}

	opts := paho.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID("opensoho")
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(true)
	opts.SetWill(serviceStatusTopic, payloadOffline, 1, true)
	opts.SetOnConnectHandler(func(c paho.Client) {
		c.Publish(serviceStatusTopic, 1, true, payloadOnline)
		// Retained discovery configs may have been lost (e.g. broker restart);
		// drop the cache so they are republished on the next telemetry.
		p.mu.Lock()
		p.published = map[string]bool{}
		p.mu.Unlock()
	})

	p.client = paho.NewClient(opts)
	token := p.client.Connect()
	if !token.WaitTimeout(connectTimeout) {
		return p, fmt.Errorf("mqtt: connection to %s timed out", cfg.Broker)
	}
	if err := token.Error(); err != nil {
		return p, fmt.Errorf("mqtt: connect to %s: %w", cfg.Broker, err)
	}
	return p, nil
}

// PublishPoE publishes a discovery config (once per connection) and the current
// consumption for every port, plus marks the device available. Safe on a nil
// receiver and when the client is not connected.
func (p *Publisher) PublishPoE(device *core.Record, info poe.Info) {
	if p == nil || p.client == nil || !p.client.IsConnected() {
		return
	}
	deviceID := device.Id
	deviceName := device.GetString("name")

	ports, _ := info.NormalizedPorts()
	for _, port := range ports {
		p.ensureDiscovery(deviceID, deviceName, port.Number)
		p.client.Publish(stateTopic(deviceID, port.Number), 0, false, formatWatts(port.Consumption))
	}
	p.client.Publish(deviceStatusTopic(deviceID), 1, true, payloadOnline)
}

// ensureDiscovery publishes the retained discovery config for a port unless it
// has already been published on this connection.
func (p *Publisher) ensureDiscovery(deviceID, deviceName string, port int) {
	key := fmt.Sprintf("%s/%d", deviceID, port)
	p.mu.Lock()
	already := p.published[key]
	p.published[key] = true
	p.mu.Unlock()
	if already {
		return
	}
	payload, err := buildDiscoveryConfig(deviceID, deviceName, port)
	if err != nil {
		// Roll back the cache entry so a later call retries.
		p.mu.Lock()
		delete(p.published, key)
		p.mu.Unlock()
		return
	}
	p.client.Publish(discoveryTopic(deviceID, port), 1, true, payload)
}

// PublishDeviceOffline marks a device unavailable. Safe on a nil receiver.
func (p *Publisher) PublishDeviceOffline(deviceID string) {
	if p == nil || p.client == nil || !p.client.IsConnected() {
		return
	}
	p.client.Publish(deviceStatusTopic(deviceID), 1, true, payloadOffline)
}

// Close publishes a graceful service-offline message and disconnects.
func (p *Publisher) Close() {
	if p == nil || p.client == nil {
		return
	}
	if p.client.IsConnected() {
		p.client.Publish(serviceStatusTopic, 1, true, payloadOffline).WaitTimeout(time.Second)
	}
	p.client.Disconnect(250)
}

// Package-level singleton so the standalone request handlers can publish without
// threading a Publisher through every call.
var (
	mu      sync.Mutex
	current *Publisher
)

// Configure (re)connects the default publisher from cfg. A disabled or empty
// config tears down any existing connection and leaves publishing as a no-op.
// It is safe to call repeatedly (e.g. when settings change).
func Configure(cfg Config) error {
	mu.Lock()
	old := current
	current = nil
	mu.Unlock()
	if old != nil {
		old.Close()
	}

	if !cfg.Enabled || cfg.Broker == "" {
		return nil
	}
	p, err := newPublisher(cfg)
	// Keep the publisher even on a timeout so auto-reconnect can recover.
	mu.Lock()
	current = p
	mu.Unlock()
	return err
}

func get() *Publisher {
	mu.Lock()
	defer mu.Unlock()
	return current
}

// Publishes PoE telemetry via the default publisher.
func PublishPoE(device *core.Record, info poe.Info) { get().PublishPoE(device, info) }

// Mark device as offline
func PublishDeviceOffline(deviceID string) { get().PublishDeviceOffline(deviceID) }

// Close tears down the default publisher.
func Close() {
	mu.Lock()
	p := current
	current = nil
	mu.Unlock()
	p.Close()
}
