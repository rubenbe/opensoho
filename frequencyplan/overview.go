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
	Frequency int
	Flags     []string
}

// Block is one rendered block in a width tier.
type Block struct {
	StartIndex  int      `json:"startIndex"`
	Span        int      `json:"span"`
	State       string   `json:"state"` // "used" | "available" | "invalid"
	Label       string   `json:"label"`
	Channels    []int    `json:"channels"`
	Frequencies []int    `json:"frequencies"`
	Flags       []string `json:"flags"`
	Devices     []string `json:"devices"`     // devices that have this mode configured (in use)
	SupportedBy []string `json:"supportedBy"` // devices whose capabilities support this mode
}

// Tier is one channel-width row within a band.
type Tier struct {
	Width  int     `json:"width"`
	Groups []Block `json:"groups"`
}

// BandOverview is the rendered model for a single band.
type BandOverview struct {
	Band  string `json:"band"`
	Label string `json:"label"`
	Cols  int    `json:"cols"`
	Tiers []Tier `json:"tiers"`
}

// BuildOverview turns the in-scope radios and advertised frequencies into the
// per-band channel-bonding model the dashboard renders. It is a pure function:
// scope filtering happens in the caller. Output ordering is deterministic.
func BuildOverview(radios []Radio, freqs []Frequency, deviceNames map[string]string) []BandOverview {
	out := make([]BandOverview, 0, len(Bands))
	for _, band := range Bands {
		if b := buildBand(band, radios, freqs, deviceNames); b != nil {
			out = append(out, *b)
		}
	}
	return out
}

func buildBand(band string, radios []Radio, freqs []Frequency, deviceNames map[string]string) *BandOverview {
	// Per-device hardware capabilities for this band: which frequencies each
	// device advertises and the flags on each. Support is evaluated per device
	// and OR-ed across devices, so an aggregate scope only greys a mode when no
	// in-scope device supports it.
	deviceFreqs := map[string]map[int]bool{}     // device -> set(frequency)
	deviceFlags := map[string]map[int][]string{} // device -> frequency -> flags
	for _, f := range freqs {
		if FrequencyToBand(f.Frequency) != band {
			continue
		}
		if deviceFreqs[f.Device] == nil {
			deviceFreqs[f.Device] = map[int]bool{}
			deviceFlags[f.Device] = map[int][]string{}
		}
		deviceFreqs[f.Device][f.Frequency] = true
		deviceFlags[f.Device][f.Frequency] = f.Flags
	}
	hasFreqData := len(deviceFreqs) > 0

	// All in-scope device ids (those with advertised frequencies and/or a
	// configured radio in this band) — used to OR support across devices.
	scopeDevices := map[string]bool{}
	for d := range deviceFreqs {
		scopeDevices[d] = true
	}

	// Configured radios: widths in use per freq + which devices use them.
	usedWidths := map[int]map[int]bool{}        // freq -> set(width)
	usedDevices := map[string]map[string]bool{} // "freq:width" -> set(device name)
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
		name := deviceNames[r.Device]
		if name == "" {
			name = r.Device
		}
		if name == "" {
			name = "?"
		}
		usedDevices[key][name] = true
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
			supporters := map[string]bool{}
			mergedFlags := map[string]bool{} // union over the scope, for display only
			for d := range scopeDevices {
				if deviceFreqs[d] == nil { // unknown capabilities
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
						mergedFlags[fl] = true
					}
				}
				if advertisesAll && !WidthForbidden(width, sortedKeys(devFlags)) {
					anySupport = true
					name := deviceNames[d]
					if name == "" {
						name = d
					}
					supporters[name] = true
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
				StartIndex:  g.StartIndex,
				Span:        g.Span,
				State:       state,
				Label:       label,
				Channels:    channels,
				Frequencies: g.Frequencies,
				Flags:       sortedKeys(mergedFlags),
				Devices:     sortedKeys(devs),
				SupportedBy: sortedKeys(supporters),
			})
		}
		tiers = append(tiers, Tier{Width: width, Groups: blocks})
	}

	return &BandOverview{
		Band:  band,
		Label: BandLabels[band],
		Cols:  len(StandardChannels(band)),
		Tiers: tiers,
	}
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
