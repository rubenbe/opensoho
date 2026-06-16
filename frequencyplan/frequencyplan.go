// Package frequencyplan contains the pure Wi-Fi channel/band math and the
// channel-bonding overview generation used by the dashboard's "Frequency
// Overview" card. It has no PocketBase (or any other I/O) dependency so it can
// be unit-tested in isolation; the HTTP handler in package main feeds it plain
// structs.
package frequencyplan

import "strconv"

// Channel is a single 20 MHz channel in a band's standard plan.
type Channel struct {
	Number    int
	Frequency int
}

// Bands are the bands we draw an overview for, in display order.
var Bands = []string{"2.4", "5", "6"}

// BandLabels maps a band key to its human label.
var BandLabels = map[string]string{
	"2.4": "2.4 GHz",
	"5":   "5 GHz",
	"6":   "6 GHz",
	"60":  "60 GHz",
}

// BandWidths lists the channel widths (MHz) drawn per band. 2.4 GHz only
// supports 20/40 MHz; 5/6 GHz support the wider bonded widths.
var BandWidths = map[string][]int{
	"2.4": {20, 40},
	"5":   {20, 40, 80, 160},
	"6":   {20, 40, 80, 160},
}

var standardChannels = map[string][]Channel{
	"2.4": channels24(),
	"5":   channels5(),
	"6":   channels6(),
}

func channels24() []Channel {
	list := make([]Channel, 0, 14)
	for ch := 1; ch <= 13; ch++ {
		list = append(list, Channel{Number: ch, Frequency: 2407 + ch*5})
	}
	list = append(list, Channel{Number: 14, Frequency: 2484})
	return list
}

func channels5() []Channel {
	chans := []int{
		36, 40, 44, 48, 52, 56, 60, 64,
		100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144,
		149, 153, 157, 161, 165,
	}
	list := make([]Channel, 0, len(chans))
	for _, ch := range chans {
		list = append(list, Channel{Number: ch, Frequency: 5000 + ch*5})
	}
	return list
}

func channels6() []Channel {
	list := make([]Channel, 0, 59)
	// 6 GHz 20 MHz channels: 1, 5, 9, ... 233 (step 4), freq = 5950 + 5*ch.
	for ch := 1; ch <= 233; ch += 4 {
		list = append(list, Channel{Number: ch, Frequency: 5950 + ch*5})
	}
	return list
}

// StandardChannels returns the standard 20 MHz channel plan for a band in
// column order, or nil for an unknown band.
func StandardChannels(band string) []Channel {
	return standardChannels[band]
}

// bandRanges is the single source of truth for the inclusive MHz bounds of each
// band, in the order FrequencyToBand checks them.
var bandRanges = []struct {
	band     string
	min, max int
}{
	{"2.4", 2400, 2500},
	{"5", 5170, 5835},
	{"6", 5925, 7125},
	{"60", 57000, 71000},
}

// FrequencyToBand maps a frequency in MHz to a band string. Ported verbatim
// from opensoho.go so it remains the single source of truth.
func FrequencyToBand(frequency int) string {
	for _, r := range bandRanges {
		if frequency >= r.min && frequency <= r.max {
			return r.band
		}
	}
	return "unknown"
}

// BandFrequencyRange returns the inclusive MHz range for a band; ok is false for
// an unknown band.
func BandFrequencyRange(band string) (min, max int, ok bool) {
	for _, r := range bandRanges {
		if r.band == band {
			return r.min, r.max, true
		}
	}
	return 0, 0, false
}

// FrequencyToChannel maps a frequency in MHz to a channel number. The bool is
// false when the frequency is outside the known ranges. Ported verbatim from
// opensoho.go (incl. 60 GHz handling).
func FrequencyToChannel(freqMHz int) (int, bool) {
	switch {
	// 2.4 GHz band: Channels 1–14
	case freqMHz >= 2412 && freqMHz <= 2484:
		if freqMHz == 2484 {
			return 14, true
		}
		return (freqMHz - 2407) / 5, true

	// 5 GHz band: Channels 36–165
	case freqMHz >= 5180 && freqMHz <= 5825:
		return (freqMHz - 5000) / 5, true

	// 6 GHz band: Channels 1–233 (starting at 5955 MHz, 5 MHz spacing)
	case freqMHz >= 5955 && freqMHz <= 7115:
		return (freqMHz - 5950) / 5, true

	// 60 GHz band (WiGig): Channels 1–6 (center freqs: 58320 + 2160 × (n − 1))
	case freqMHz >= 58320 && freqMHz <= 70200:
		ch := ((freqMHz - 58320) / 2160) + 1
		if ch >= 1 && ch <= 6 {
			return ch, true
		}
		return 0, false

	default:
		return 0, false
	}
}

// HtmodeWidth returns the channel width (MHz) for an htmode such as "HT20",
// "VHT40", "HE160". A blank htmode is treated as 20 MHz, the conservative Wi-Fi
// default. The bool is false for an unrecognised value.
func HtmodeWidth(htmode string) (int, bool) {
	if htmode == "" {
		return 20, true
	}
	i := len(htmode)
	for i > 0 && htmode[i-1] >= '0' && htmode[i-1] <= '9' {
		i--
	}
	if i == len(htmode) {
		return 0, false
	}
	w, err := strconv.Atoi(htmode[i:])
	if err != nil {
		return 0, false
	}
	return w, true
}

// BondingGroup is one block in a width tier: the contiguous channels it bonds,
// the column index of its first channel and its column span.
type BondingGroup struct {
	Frequencies []int
	StartIndex  int
	Span        int
	Complete    bool
}

// BondingGroups returns the channel-bonding groups for a band at a given width,
// in column order. For 5/6 GHz, channels are grouped within frequency-contiguous
// runs (consecutive plan entries 20 MHz apart) into chunks of width/20; this
// prevents bonding across band gaps (e.g. the 5 GHz 64->100 jump and the
// 144->149 boundary). For 2.4 GHz, whose channels overlap at 5 MHz spacing,
// groups are consecutive adjacent-channel chunks (an accepted approximation). A
// trailing chunk shorter than width/20 has Complete == false.
func BondingGroups(band string, width int) []BondingGroup {
	plan := standardChannels[band]
	k := width / 20
	if k <= 1 {
		groups := make([]BondingGroup, 0, len(plan))
		for i, c := range plan {
			groups = append(groups, BondingGroup{
				Frequencies: []int{c.Frequency},
				StartIndex:  i,
				Span:        1,
				Complete:    true,
			})
		}
		return groups
	}

	// Split into contiguous runs of plan indices. 2.4 GHz is one run (overlapping
	// channels); 5/6 GHz break runs where the frequency step != 20.
	var runs [][]int
	var run []int
	for i := range plan {
		contiguous := len(run) == 0 || band == "2.4" || plan[i].Frequency-plan[i-1].Frequency == 20
		if !contiguous {
			runs = append(runs, run)
			run = nil
		}
		run = append(run, i)
	}
	if len(run) > 0 {
		runs = append(runs, run)
	}

	var groups []BondingGroup
	for _, indices := range runs {
		for off := 0; off < len(indices); off += k {
			end := off + k
			if end > len(indices) {
				end = len(indices)
			}
			chunk := indices[off:end]
			freqs := make([]int, len(chunk))
			for j, idx := range chunk {
				freqs[j] = plan[idx].Frequency
			}
			groups = append(groups, BondingGroup{
				Frequencies: freqs,
				StartIndex:  chunk[0],
				Span:        len(chunk),
				Complete:    len(chunk) == k,
			})
		}
	}
	return groups
}

// WidthForbidden reports whether the radio_frequencies flags forbid the given
// channel width. Mirrors validateRadioHtModeFlags in opensoho.go.
func WidthForbidden(width int, flags []string) bool {
	switch width {
	case 40:
		return contains(flags, "no_ht40-") && contains(flags, "no_ht40+")
	case 80:
		return contains(flags, "no_80mhz")
	case 160:
		return contains(flags, "no_160mhz")
	case 20:
		return contains(flags, "no_20mhz")
	default:
		return false
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
