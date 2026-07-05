package lldp

import (
	"bytes"
	"encoding/json"
	"sort"
)

// Neighbor is a single LLDP neighbour seen from this device's viewpoint.
type Neighbor struct {
	Port string // local interface the neighbour is seen on, e.g. "eth0"
	Name string // neighbour's advertised system name; may be "" when unknown
	Mac  string // neighbour's chassis MAC address, e.g. "00:11:22:33:44:55"
}

type Info struct {
	Neighbors []Neighbor
}

// reservedChassisKeys are the field names lldpd emits inside a <chassis> element.
// When a neighbour advertises a SysName, lldpd wraps the chassis fields in an
// object keyed by that name ({"sw-core-01":{"id":...}}); when it does not, the
// fields appear unwrapped ({"id":...,"descr":...}). Distinguishing the two comes
// down to whether a key is one of these reserved field names.
var reservedChassisKeys = map[string]bool{
	"id":         true,
	"descr":      true,
	"mgmt-ip":    true,
	"mgmt-iface": true,
	"capability": true,
	"ttl":        true,
	"via":        true,
	"rid":        true,
	"age":        true,
}

// normalized element of an lldpd "flex list": its name (the key
// lldpd used) and the raw JSON body.
type entry struct {
	key string
	raw json.RawMessage
}

// lldpd's `json` format renders a list as an object when it has one element and
// as an array of single-key objects when it has several, so the same field
// (interface, chassis, ...) has no single static Go type.
// NormalizeLLDPEntries cleans up this misery
func NormalizeLLDPEntries(raw json.RawMessage) ([]entry, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	switch raw[0] {
	case '{':
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return sortedEntries(m), nil
	case '[':
		var arr []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, err
		}
		var entries []entry
		for _, m := range arr {
			entries = append(entries, sortedEntries(m)...)
		}
		return entries, nil
	default:
		return nil, nil
	}
}

func sortedEntries(m map[string]json.RawMessage) []entry {
	entries := make([]entry, 0, len(m))
	for k, v := range m {
		entries = append(entries, entry{key: k, raw: v})
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].key < entries[b].key })
	return entries
}

// chassisInfo returns the neighbour's system name and chassis MAC address from a
// <chassis> body. When lldpd wrapped the fields in a SysName key that key is the
// name and its value holds the fields; when the neighbour advertised no SysName
// the fields appear unwrapped and the name is "". The MAC comes from the chassis
// `id` field ({"type":"mac","value":"aa:bb:..."}).
func chassisInfo(raw json.RawMessage) (name, mac string) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if reservedChassisKeys[k] {
			continue
		}
		keys = append(keys, k)
	}
	// fields defaults to the top-level object (unwrapped chassis); when a SysName
	// wraps the fields, descend into its body.
	fields := m
	if len(keys) > 0 {
		sort.Strings(keys)
		name = keys[0]
		var inner map[string]json.RawMessage
		if err := json.Unmarshal(m[name], &inner); err == nil {
			fields = inner
		}
	}
	return name, chassisMac(fields)
}

// chassisMac pulls the MAC address out of a chassis's `id` field. Returns "" when
// the id is absent or advertises a non-MAC identifier type.
func chassisMac(fields map[string]json.RawMessage) string {
	raw, ok := fields["id"]
	if !ok {
		return ""
	}
	var id struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &id); err != nil {
		return ""
	}
	if id.Type != "mac" {
		return ""
	}
	return id.Value
}

// UnmarshalJSON decodes the verbatim lldpcli output
// shaped {"lldp":{"interface":<flex list>}}, into Neighbors. Each interface
// entry's key is the local port; the neighbour name comes from its chassis.
func (i *Info) UnmarshalJSON(data []byte) error {
	var top struct {
		Lldp struct {
			Interface json.RawMessage `json:"interface"`
		} `json:"lldp"`
	}
	if err := json.Unmarshal(data, &top); err != nil {
		return err
	}
	ifaces, err := NormalizeLLDPEntries(top.Lldp.Interface)
	if err != nil {
		return err
	}
	for _, iface := range ifaces {
		var body struct {
			Chassis json.RawMessage `json:"chassis"`
		}
		if err := json.Unmarshal(iface.raw, &body); err != nil {
			return err
		}
		name, mac := chassisInfo(body.Chassis)
		i.Neighbors = append(i.Neighbors, Neighbor{
			Port: iface.key,
			Name: name,
			Mac:  mac,
		})
	}
	return nil
}

func (i Info) Normalized() []Neighbor {
	seen := make(map[string]bool, len(i.Neighbors))
	out := make([]Neighbor, 0, len(i.Neighbors))
	for _, n := range i.Neighbors {
		key := rowKey(n.Port, n.Name, n.Mac)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, n)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Port != out[b].Port {
			return out[a].Port < out[b].Port
		}
		if out[a].Name != out[b].Name {
			return out[a].Name < out[b].Name
		}
		return out[a].Mac < out[b].Mac
	})
	return out
}
