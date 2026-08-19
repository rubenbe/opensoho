package frequencyplan

import (
	"fmt"
	"sort"
)

// Radio is a configured radio (already filtered to the requested scope).
type Radio struct {
	Device    string
	Frequency int
	Htmode    string
}

// Frequency is a hardware-advertised frequency row (already scope-filtered).
type Frequency struct {
	Device    string
	Radio     int // per-device radio index, matches HtModes.Radio
	Frequency int
	Flags     []string
}

// HtModes lists the channel modes one radio advertises (iwinfo's htmodes),
// already scope-filtered. A device with no entry for a radio has unknown
// capabilities for that radio and is not constrained by it.
type HtModes struct {
	Device string
	Radio  int
	Modes  []string
}

// DeviceRef identifies a device for display and for linking to its radios.
type DeviceRef struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// Block is one rendered block in a width tier.
type Block struct {
	StartIndex    int         `json:"startIndex"`
	Span          int         `json:"span"`
	State         string      `json:"state"` // "used" | "available" | "invalid"
	Label         string      `json:"label"`
	Channels      []int       `json:"channels"`
	Frequencies   []int       `json:"frequencies"`
	Devices       []DeviceRef `json:"devices"`       // devices that have this mode configured (in use)
	SupportedBy   []DeviceRef `json:"supportedBy"`   // devices whose capabilities support this mode
	UnsupportedBy []DeviceRef `json:"unsupportedBy"` // known-capability devices that don't support it
}

// Tier is one channel-width row within a band.
type Tier struct {
	Width  int     `json:"width"`
	Groups []Block `json:"groups"`
}

// BandOverview is the rendered model for a single band.
type BandOverview struct {
	Band    string `json:"band"`
	Label   string `json:"label"`
	Cols    int    `json:"cols"`
	FreqMin int    `json:"freqMin"` // inclusive band frequency bounds (MHz), for filtering
	FreqMax int    `json:"freqMax"`
	Tiers   []Tier `json:"tiers"`
}

// BuildOverview turns the in-scope radios and advertised frequencies into the
// per-band channel-bonding model the dashboard renders. It is a pure function:
// scope filtering happens in the caller. Output ordering is deterministic.
func BuildOverview(radios []Radio, freqs []Frequency, htModes []HtModes, deviceNames map[string]string) []BandOverview {
	out := make([]BandOverview, 0, len(Bands))
	for _, band := range Bands {
		if b := buildBand(band, radios, freqs, htModes, deviceNames); b != nil {
			out = append(out, *b)
		}
	}
	return out
}

func buildBand(band string, radios []Radio, freqs []Frequency, htModes []HtModes, deviceNames map[string]string) *BandOverview {
	// Per-device hardware capabilities for this band: which frequencies each
	// device advertises and the flags on each. Support is evaluated per device
	// and OR-ed across devices, so an aggregate scope only greys a mode when no
	// in-scope device supports it.
	deviceFreqs := map[string]map[int]bool{}     // device -> set(frequency)
	deviceFlags := map[string]map[int][]string{} // device -> frequency -> flags
	deviceRadios := map[string]map[int]bool{}    // device -> set(radio index) advertising in this band
	for _, f := range freqs {
		if FrequencyToBand(f.Frequency) != band {
			continue
		}
		if deviceFreqs[f.Device] == nil {
			deviceFreqs[f.Device] = map[int]bool{}
			deviceFlags[f.Device] = map[int][]string{}
			deviceRadios[f.Device] = map[int]bool{}
		}
		deviceFreqs[f.Device][f.Frequency] = true
		deviceFlags[f.Device][f.Frequency] = f.Flags
		deviceRadios[f.Device][f.Radio] = true
	}
	hasFreqData := len(deviceFreqs) > 0

	// deviceWidths[device] is the set of widths advertised by that device's
	// radios in this band, derived from their htmodes (e.g. "HE160" -> 160). A
	// device absent from the map has no ht-mode data for any radio in this band
	// yet and is left unconstrained by deviceSupportsWidth, mirroring the
	// permissive fallback in supportedHtModes (opensoho.go).
	htModesByDeviceRadio := map[string]map[int][]string{}
	for _, h := range htModes {
		if htModesByDeviceRadio[h.Device] == nil {
			htModesByDeviceRadio[h.Device] = map[int][]string{}
		}
		htModesByDeviceRadio[h.Device][h.Radio] = h.Modes
	}
	deviceWidths := map[string]map[int]bool{}
	for d, dRadios := range deviceRadios {
		for r := range dRadios {
			modes, ok := htModesByDeviceRadio[d][r]
			if !ok {
				continue
			}
			for _, m := range modes {
				if w, ok := HtmodeWidth(m); ok {
					if deviceWidths[d] == nil {
						deviceWidths[d] = map[int]bool{}
					}
					deviceWidths[d][w] = true
				}
			}
		}
	}

	// All in-scope device ids (those with advertised frequencies and/or a
	// configured radio in this band) — used to OR support across devices.
	scopeDevices := map[string]bool{}
	for d := range deviceFreqs {
		scopeDevices[d] = true
	}

	// Configured radios: widths in use per freq + which devices use them (by id).
	usedWidths := map[int]map[int]bool{}        // freq -> set(width)
	usedDevices := map[string]map[string]bool{} // "freq:width" -> set(device id)
	hasRadios := false
	for _, r := range radios {
		if FrequencyToBand(r.Frequency) != band {
			continue
		}
		hasRadios = true
		scopeDevices[r.Device] = true
		width, ok := HtmodeWidth(r.Htmode)
		if !ok {
			continue
		}
		if usedWidths[r.Frequency] == nil {
			usedWidths[r.Frequency] = map[int]bool{}
		}
		usedWidths[r.Frequency][width] = true

		key := fmt.Sprintf("%d:%d", r.Frequency, width)
		if usedDevices[key] == nil {
			usedDevices[key] = map[string]bool{}
		}
		usedDevices[key][r.Device] = true
	}

	// Skip bands with nothing configured and no advertised frequencies.
	if !hasRadios && !hasFreqData {
		return nil
	}

	tiers := make([]Tier, 0, len(BandWidths[band]))
	for _, width := range BandWidths[band] {
		groups := BondingGroups(band, width)
		blocks := make([]Block, 0, len(groups))
		for _, g := range groups {
			channels := make([]int, len(g.Frequencies))
			for i, f := range g.Frequencies {
				ch, _ := FrequencyToChannel(f)
				channels[i] = ch
			}

			// Evaluate capability support per device, OR-ed across the scope.
			// A device with no advertised frequencies in this band has unknown
			// capabilities and is treated as supporting (mirrors the skip in
			// validateRadioHtModeFlags). Such devices are not listed in
			// SupportedBy but still keep the block from greying out.
			anySupport := false
			supporters := map[string]bool{}    // set(device id) with confirmed support
			notSupporters := map[string]bool{} // set(device id), known-capability but no support
			for d := range scopeDevices {
				if deviceFreqs[d] == nil { // unknown capabilities — claim neither way
					anySupport = true
					continue
				}
				advertisesAll := true
				devFlags := map[string]bool{}
				for _, f := range g.Frequencies {
					if !deviceFreqs[d][f] {
						advertisesAll = false
					}
					for _, fl := range deviceFlags[d][f] {
						devFlags[fl] = true
					}
				}
				if advertisesAll && !WidthForbidden(width, sortedKeys(devFlags)) && deviceSupportsWidth(deviceWidths, d, width) {
					anySupport = true
					supporters[d] = true
				} else {
					notSupporters[d] = true
				}
			}

			used := false
			for _, f := range g.Frequencies {
				if usedWidths[f][width] {
					used = true
					break
				}
			}

			invalid := !g.Complete || !anySupport

			state := "available"
			switch {
			case used:
				state = "used"
			case invalid:
				state = "invalid"
			}

			label := fmt.Sprintf("%d", channels[0])
			if g.Span > 1 {
				label = fmt.Sprintf("%d–%d", channels[0], channels[len(channels)-1])
			}

			devs := map[string]bool{}
			for _, f := range g.Frequencies {
				for d := range usedDevices[fmt.Sprintf("%d:%d", f, width)] {
					devs[d] = true
				}
			}

			blocks = append(blocks, Block{
				StartIndex:    g.StartIndex,
				Span:          g.Span,
				State:         state,
				Label:         label,
				Channels:      channels,
				Frequencies:   g.Frequencies,
				Devices:       deviceRefs(devs, deviceNames),
				SupportedBy:   deviceRefs(supporters, deviceNames),
				UnsupportedBy: deviceRefs(notSupporters, deviceNames),
			})
		}
		tiers = append(tiers, Tier{Width: width, Groups: blocks})
	}

	freqMin, freqMax, _ := BandFrequencyRange(band)
	return &BandOverview{
		Band:    band,
		Label:   BandLabels[band],
		Cols:    len(StandardChannels(band)),
		FreqMin: freqMin,
		FreqMax: freqMax,
		Tiers:   tiers,
	}
}

// deviceRefs turns a set of device ids into DeviceRefs (name falling back to id),
// sorted by name then id for deterministic output.
func deviceRefs(ids map[string]bool, deviceNames map[string]string) []DeviceRef {
	refs := make([]DeviceRef, 0, len(ids))
	for id := range ids {
		name := deviceNames[id]
		if name == "" {
			name = id
		}
		refs = append(refs, DeviceRef{Id: id, Name: name})
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Name != refs[j].Name {
			return refs[i].Name < refs[j].Name
		}
		return refs[i].Id < refs[j].Id
	})
	return refs
}

// deviceSupportsWidth reports whether the device advertises an htmode of this
// width in this band. A device with no ht-mode data has unknown capabilities
// and is not constrained, mirroring supportedHtModes in opensoho.go.
func deviceSupportsWidth(deviceWidths map[string]map[int]bool, device string, width int) bool {
	widths, ok := deviceWidths[device]
	if !ok {
		return true
	}
	return widths[width]
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
