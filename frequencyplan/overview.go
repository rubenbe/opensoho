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
	Devices     []string `json:"devices"`
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
	// Hardware-advertised frequencies (union across scope) + merged flags per freq.
	validFreqs := map[int]bool{}
	flagsByFreq := map[int]map[string]bool{}
	for _, f := range freqs {
		if FrequencyToBand(f.Frequency) != band {
			continue
		}
		validFreqs[f.Frequency] = true
		set := flagsByFreq[f.Frequency]
		if set == nil {
			set = map[string]bool{}
			flagsByFreq[f.Frequency] = set
		}
		for _, fl := range f.Flags {
			set[fl] = true
		}
	}
	hasFreqData := len(validFreqs) > 0

	// Configured radios: widths in use per freq + which devices use them.
	usedWidths := map[int]map[int]bool{}        // freq -> set(width)
	usedDevices := map[string]map[string]bool{} // "freq:width" -> set(device name)
	hasRadios := false
	for _, r := range radios {
		if FrequencyToBand(r.Frequency) != band {
			continue
		}
		hasRadios = true
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

			mergedFlags := map[string]bool{}
			for _, f := range g.Frequencies {
				for fl := range flagsByFreq[f] {
					mergedFlags[fl] = true
				}
			}

			used := false
			for _, f := range g.Frequencies {
				if usedWidths[f][width] {
					used = true
					break
				}
			}

			missing := false
			if hasFreqData {
				for _, f := range g.Frequencies {
					if !validFreqs[f] {
						missing = true
						break
					}
				}
			}

			invalid := !g.Complete || missing || WidthForbidden(width, sortedKeys(mergedFlags))

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
