package poe

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sample = `{
    "budget": 55,
    "consumption": 10,
    "ports": {
        "lan1": { "priority": 0, "mode": "PoE", "status": "Searching" },
        "lan2": { "priority": 0, "mode": "PoE", "status": "Delivering power", "consumption": 3.300000 }
    }
}`

func TestPriorityLabel(t *testing.T) {
	cases := map[int]string{
		0:  "low",
		1:  "normal",
		2:  "high",
		3:  "critical",
		-1: "low", // invalid falls back to low
		4:  "low", // invalid falls back to low
	}
	for n, want := range cases {
		assert.Equal(t, want, PriorityLabel(n), "PriorityLabel(%d)", n)
	}
}

func TestNormalizedPorts(t *testing.T) {
	var info Info
	assert.NoError(t, json.Unmarshal([]byte(sample), &info))

	ports := info.NormalizedPorts()
	assert.Len(t, ports, 2)

	// lan1: searching, no consumption reported -> 0. Sorted first.
	assert.Equal(t, "lan1", ports[0].Name)
	assert.Equal(t, "low", ports[0].Priority)
	assert.Equal(t, "Searching", ports[0].Status)
	assert.Equal(t, 0.0, ports[0].Consumption)

	// lan2: delivering power with a consumption value. Sorted second.
	assert.Equal(t, "lan2", ports[1].Name)
	assert.Equal(t, "low", ports[1].Priority)
	assert.Equal(t, "Delivering power", ports[1].Status)
	assert.Equal(t, 3.3, ports[1].Consumption)
}

// Non-numeric port names are no longer dropped: the name is stored verbatim.
func TestNormalizedPortsKeepsNonNumericNames(t *testing.T) {
	info := Info{Ports: map[string]RawPort{
		"lan2":    {Priority: 2, Status: "Delivering power", Consumption: 5},
		"invalid": {Priority: 0, Status: "n/a"},
	}}

	ports := info.NormalizedPorts()
	assert.Len(t, ports, 2)
	// Sorted by name: "invalid" < "lan2".
	assert.Equal(t, "invalid", ports[0].Name)
	assert.Equal(t, "lan2", ports[1].Name)
	assert.Equal(t, "high", ports[1].Priority)
}
