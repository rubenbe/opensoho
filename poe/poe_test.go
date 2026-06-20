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

func TestPortNumber(t *testing.T) {
	cases := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{"lan1", 1, false},
		{"lan4", 4, false},
		{"lan10", 10, false},
		{"port0", 0, false},
		{"lan", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := PortNumber(c.name)
		if c.wantErr {
			assert.Error(t, err, "PortNumber(%q)", c.name)
			continue
		}
		assert.NoError(t, err, "PortNumber(%q)", c.name)
		assert.Equal(t, c.want, got, "PortNumber(%q)", c.name)
	}
}

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

	ports, skipped := info.NormalizedPorts()
	assert.Empty(t, skipped, "unexpected skipped ports")
	assert.Len(t, ports, 2)

	// lan1: searching, no consumption reported -> 0. Sorted first.
	assert.Equal(t, 1, ports[0].Number)
	assert.Equal(t, "low", ports[0].Priority)
	assert.Equal(t, "Searching", ports[0].Status)
	assert.Equal(t, 0.0, ports[0].Consumption)

	// lan2: delivering power with a consumption value. Sorted second.
	assert.Equal(t, 2, ports[1].Number)
	assert.Equal(t, "low", ports[1].Priority)
	assert.Equal(t, "Delivering power", ports[1].Status)
	assert.Equal(t, 3.3, ports[1].Consumption)
}

func TestNormalizedPortsSkipsUnparseable(t *testing.T) {
	info := Info{Ports: map[string]RawPort{
		"lan2":    {Priority: 2, Status: "Delivering power", Consumption: 5},
		"invalid": {Priority: 0, Status: "n/a"},
	}}

	ports, skipped := info.NormalizedPorts()
	assert.Len(t, ports, 1)
	assert.Equal(t, 2, ports[0].Number)
	assert.Equal(t, "high", ports[0].Priority)
	assert.Equal(t, []string{"invalid"}, skipped)
}
