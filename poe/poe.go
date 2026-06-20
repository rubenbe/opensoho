package poe

import (
	"fmt"
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
	Number      int
	Priority    string // one of low/normal/high/critical
	Status      string
	Consumption float64
}

// extract the port number from the UCI port key ("lan4" -> 4).
// error when the key has no trailing digits.
func PortNumber(name string) (int, error) {
	i := len(name)
	for i > 0 && name[i-1] >= '0' && name[i-1] <= '9' {
		i--
	}
	digits := name[i:]
	if digits == "" {
		return 0, fmt.Errorf("port name %q has no trailing number", name)
	}
	n := 0
	for _, c := range digits {
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func PriorityLabel(n int) string {
	if n >= 0 && n < len(priorityLabels) {
		return priorityLabels[n]
	}
	// Map other names on LOW
	return priorityLabels[0]
}

// NormalizedPorts converts the raw ports map into a slice of Port sorted by
// number for deterministic output. Keys whose name carries no parseable port
// number are skipped and returned (sorted) so the caller can log them.
func (i Info) NormalizedPorts() (ports []Port, skipped []string) {
	for name, raw := range i.Ports {
		num, err := PortNumber(name)
		if err != nil {
			skipped = append(skipped, name)
			continue
		}
		ports = append(ports, Port{
			Number:      num,
			Priority:    PriorityLabel(raw.Priority),
			Status:      raw.Status,
			Consumption: raw.Consumption,
		})
	}
	sort.Slice(ports, func(a, b int) bool { return ports[a].Number < ports[b].Number })
	sort.Strings(skipped)
	return ports, skipped
}
