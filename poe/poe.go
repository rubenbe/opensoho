package poe

import (
	"sort"
)

var priorityLabels = []string{"low", "normal", "high", "critical"}

type Info struct {
	Budget      float64            `json:"budget"`
	Consumption float64            `json:"consumption"`
	Ports       map[string]RawPort `json:"ports"`
}

type RawPort struct {
	Priority    int     `json:"priority"`
	Mode        string  `json:"mode"`
	Status      string  `json:"status"`
	Consumption float64 `json:"consumption"`
}

type Port struct {
	Name        string // full UCI port key, e.g. "lan4"
	Priority    string // one of low/normal/high/critical
	Status      string
	Consumption float64
}

func PriorityLabel(n int) string {
	if n >= 0 && n < len(priorityLabels) {
		return priorityLabels[n]
	}
	// Map other names on LOW
	return priorityLabels[0]
}

// NormalizedPorts converts the raw ports map into a slice of Port sorted by
// name for deterministic output.
func (i Info) NormalizedPorts() (ports []Port) {
	for name, raw := range i.Ports {
		ports = append(ports, Port{
			Name:        name,
			Priority:    PriorityLabel(raw.Priority),
			Status:      raw.Status,
			Consumption: raw.Consumption,
		})
	}
	sort.Slice(ports, func(a, b int) bool { return ports[a].Name < ports[b].Name })
	return ports
}
