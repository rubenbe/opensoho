package lldp

import (
	"sort"
	"strings"

	"github.com/maruel/natural"
)

// Row is a single stored lldp record for the device currently in scope: the local
// port it was seen on, the neighbour's advertised name, and its chassis MAC.
type Row struct {
	Port string
	Name string
	Mac  string
}

// EthernetPort is a physical port opensoho knows about for a device (from the
// ethernet collection), annotated with the bridge it is a member of (from bridges,
// "" when it belongs to no bridge).
type EthernetPort struct {
	Name   string
	Speed  string
	Bridge string
}

// OverviewNeighbor is the render model for one LLDP neighbour seen on a port.
// KnownDeviceId is the id of the opensoho-managed device that owns this MAC, or ""
// when the neighbour is not managed by opensoho (the UI then renders it as plain
// text rather than a link).
type OverviewNeighbor struct {
	Name          string `json:"name"`
	Mac           string `json:"mac"`
	KnownDeviceId string `json:"knownDeviceId"`
}

// Port is the render model for one known port: its name, link speed, the bridge it
// belongs to (or ""), and any LLDP neighbours seen on it.
type Port struct {
	Port      string             `json:"port"`
	Speed     string             `json:"speed"`
	Bridge    string             `json:"bridge"`
	Neighbors []OverviewNeighbor `json:"neighbors"`
}

// BuildPortOverview lists every known port of a device (sourced from the ethernet +
// bridges collections) and overlays the LLDP neighbours seen on each. LLDP rows whose
// local port is not among the known ethernet ports are appended as extra port rows so
// no neighbour is lost. macOwners maps a MAC -> owning device id; matching is
// case-insensitive. The result is sorted by port name.
func BuildPortOverview(ports []EthernetPort, rows []Row, macOwners map[string]string) []Port {
	owners := make(map[string]string, len(macOwners))
	for mac, device := range macOwners {
		owners[strings.ToLower(mac)] = device
	}

	// Group neighbours by the local port they were seen on.
	byPort := map[string][]OverviewNeighbor{}
	for _, r := range rows {
		byPort[r.Port] = append(byPort[r.Port], OverviewNeighbor{
			Name:          r.Name,
			Mac:           r.Mac,
			KnownDeviceId: owners[strings.ToLower(r.Mac)],
		})
	}

	out := make([]Port, 0, len(ports)+len(byPort))
	known := make(map[string]bool, len(ports))
	for _, p := range ports {
		known[p.Name] = true
		out = append(out, Port{
			Port:      p.Name,
			Speed:     p.Speed,
			Bridge:    p.Bridge,
			Neighbors: sortNeighbors(byPort[p.Name]),
		})
	}
	// LLDP neighbours seen on ports we don't otherwise know about.
	for port, ns := range byPort {
		if known[port] {
			continue
		}
		out = append(out, Port{Port: port, Neighbors: sortNeighbors(ns)})
	}
	// Natural order so numeric runs sort by value: lan1 < lan2 < lan10.
	sort.Slice(out, func(a, b int) bool { return natural.Less(out[a].Port, out[b].Port) })
	return out
}

func sortNeighbors(ns []OverviewNeighbor) []OverviewNeighbor {
	sort.Slice(ns, func(a, b int) bool {
		if ns[a].Name != ns[b].Name {
			return ns[a].Name < ns[b].Name
		}
		return ns[a].Mac < ns[b].Mac
	})
	return ns
}
