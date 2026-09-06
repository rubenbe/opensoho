package frequencyplan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrequencyToBand(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{2412, "2.4"},
		{2472, "2.4"},
		{5160, "5"},
		{5170, "5"},
		{5180, "5"},
		{5825, "5"},
		{5845, "5"},
		{5865, "5"},
		{5885, "5"},
		{5905, "unknown"}, // gap between the 5 GHz and 6 GHz bands
		{5935, "6"},
		{5955, "6"},
		{6975, "6"},
		{58320, "60"},
		{66960, "60"},
		{1000, "unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, FrequencyToBand(tt.input), tt.input)
	}
}

func TestFrequencyToChannel(t *testing.T) {
	tests := []struct {
		freq            int
		expectedChannel int
		expectedOk      bool
	}{
		{2412, 1, true},
		{2437, 6, true},
		{2484, 14, true},
		{5160, 32, true},
		{5170, 34, true}, // arithmetic only: 34 isn't a real allocated channel (legacy Japan numbering, absent from standardChannels)
		{5180, 36, true},
		{5200, 40, true},
		{5500, 100, true},
		{5825, 165, true},
		{5845, 169, true},
		{5865, 173, true},
		{5885, 177, true},
		{5895, 0, false}, // band "5" (5150-5895) but past the last channel center
		{5935, 2, true},  // 6 GHz channel 2, below the 5950+5n grid
		{5945, 0, false}, // band "6" (5925-7125) but in the gap before channel 1
		{5955, 1, true},
		{7115, 233, true},
		{58320, 1, true},
		{69120, 6, true},
		{72000, 0, false},
	}
	for _, tt := range tests {
		ch, ok := FrequencyToChannel(tt.freq)
		assert.Equal(t, tt.expectedOk, ok, tt.freq)
		assert.Equal(t, tt.expectedChannel, ch, tt.freq)
	}
}

func TestBandFrequencyRange(t *testing.T) {
	tests := []struct {
		band     string
		min, max int
		ok       bool
	}{
		{"2.4", 2400, 2500, true},
		{"5", 5150, 5895, true},
		{"6", 5925, 7125, true},
		{"60", 57000, 71000, true},
		{"unknown", 0, 0, false},
	}
	for _, tt := range tests {
		min, max, ok := BandFrequencyRange(tt.band)
		assert.Equal(t, tt.ok, ok, tt.band)
		assert.Equal(t, tt.min, min, tt.band)
		assert.Equal(t, tt.max, max, tt.band)
		// Consistency: the band's own bounds map back to that band.
		if ok {
			assert.Equal(t, tt.band, FrequencyToBand(min), tt.band)
			assert.Equal(t, tt.band, FrequencyToBand(max), tt.band)
		}
	}
}

// TestStandardChannelsRoundTrip pins bandRanges and FrequencyToChannel's
// switch to each other: every channel in the standard plans must map back to
// its own band and its own channel number, so the two hand-maintained tables
// can't drift apart again as they did before (band edges vs. channel centers
// disagreeing on 5170-5180 and 5825-5835, and 6 GHz channel 2 unmapped).
func TestStandardChannelsRoundTrip(t *testing.T) {
	for _, band := range Bands {
		for _, c := range StandardChannels(band) {
			assert.Equal(t, band, FrequencyToBand(c.Frequency), c.Frequency)
			ch, ok := FrequencyToChannel(c.Frequency)
			assert.True(t, ok, c.Frequency)
			assert.Equal(t, c.Number, ch, c.Frequency)
		}
	}
}

func TestHtmodeWidth(t *testing.T) {
	tests := []struct {
		htmode string
		width  int
		ok     bool
	}{
		{"", 20, true},
		{"HT20", 20, true},
		{"HT40", 40, true},
		{"VHT40", 40, true},
		{"VHT80", 80, true},
		{"HE160", 160, true},
		{"AUTO", 0, false},
		{"HT", 0, false},
	}
	for _, tt := range tests {
		w, ok := HtmodeWidth(tt.htmode)
		assert.Equal(t, tt.ok, ok, tt.htmode)
		assert.Equal(t, tt.width, w, tt.htmode)
	}
}

func TestWidthForbidden(t *testing.T) {
	assert.False(t, WidthForbidden(40, []string{"no_ht40-"}))
	assert.False(t, WidthForbidden(40, []string{"no_ht40+"}))
	assert.True(t, WidthForbidden(40, []string{"no_ht40-", "no_ht40+"}))
	assert.True(t, WidthForbidden(80, []string{"no_80mhz"}))
	assert.True(t, WidthForbidden(160, []string{"no_160mhz"}))
	assert.True(t, WidthForbidden(320, []string{"no_320mhz"}))
	assert.True(t, WidthForbidden(20, []string{"no_20mhz"}))
	assert.False(t, WidthForbidden(80, []string{"no_160mhz"}))
	assert.False(t, WidthForbidden(320, []string{"no_160mhz"}))
	assert.False(t, WidthForbidden(40, nil))
}

// channelsOf renders a bonding group as its channel numbers for easy assertions.
func channelsOf(g BondingGroup) []int {
	out := make([]int, len(g.Frequencies))
	for i, f := range g.Frequencies {
		out[i], _ = FrequencyToChannel(f)
	}
	return out
}

func TestBondingGroups5GHz(t *testing.T) {
	g40 := BondingGroups("5", 40)
	// Channel 32 is nonBondable: it sits alone as an incomplete leading block
	// even though it's exactly 20 MHz below 36.
	assert.Equal(t, []int{32}, channelsOf(g40[0]))
	assert.False(t, g40[0].Complete)
	assert.Equal(t, 0, g40[0].StartIndex)
	assert.Equal(t, 1, g40[0].Span)

	assert.Equal(t, []int{36, 40}, channelsOf(g40[1]))
	assert.True(t, g40[1].Complete)
	assert.Equal(t, 1, g40[1].StartIndex)
	assert.Equal(t, 2, g40[1].Span)

	// U-NII-4 (169/173/177) is frequency-contiguous with U-NII-3 (up to 165),
	// so the run now pairs cleanly all the way through - no more trailing
	// orphan at 165.
	last := g40[len(g40)-1]
	assert.Equal(t, []int{173, 177}, channelsOf(last))
	assert.True(t, last.Complete)

	g80 := BondingGroups("5", 80)
	assert.Equal(t, []int{32}, channelsOf(g80[0]))
	assert.False(t, g80[0].Complete)
	assert.Equal(t, []int{36, 40, 44, 48}, channelsOf(g80[1]))
	assert.True(t, g80[1].Complete)
	// 165-177 is a real 80 MHz block (center channel 171).
	last80 := g80[len(g80)-1]
	assert.Equal(t, []int{165, 169, 173, 177}, channelsOf(last80))
	assert.True(t, last80.Complete)

	g160 := BondingGroups("5", 160)
	assert.Equal(t, []int{32}, channelsOf(g160[0]))
	assert.False(t, g160[0].Complete)
	assert.Equal(t, []int{36, 40, 44, 48, 52, 56, 60, 64}, channelsOf(g160[1]))
	assert.True(t, g160[1].Complete)
	assert.Equal(t, []int{100, 104, 108, 112, 116, 120, 124, 128}, channelsOf(g160[2]))
	assert.True(t, g160[2].Complete)
	// 132–144 (only 4 channels left before the 149 boundary) cannot form 160 MHz.
	assert.Equal(t, []int{132, 136, 140, 144}, channelsOf(g160[3]))
	assert.False(t, g160[3].Complete)
	// 149-177 (channel 163) is a real 160 MHz block, confirmed via the
	// standard channelization table (U-NII-3 running straight into U-NII-4).
	assert.Equal(t, []int{149, 153, 157, 161, 165, 169, 173, 177}, channelsOf(g160[4]))
	assert.True(t, g160[4].Complete)
}

func TestBondingGroups24GHz(t *testing.T) {
	g40 := BondingGroups("2.4", 40)
	assert.Equal(t, []int{1, 2}, channelsOf(g40[0]))
	assert.Equal(t, []int{3, 4}, channelsOf(g40[1]))
	// 14 channels -> 7 complete pairs.
	assert.Equal(t, 7, len(g40))
	for _, g := range g40 {
		assert.True(t, g.Complete)
	}
}

func TestBondingGroups6GHzContiguous(t *testing.T) {
	// 6 GHz is one contiguous run; 59 channels -> 29 complete 40 MHz pairs + 1 leftover.
	g40 := BondingGroups("6", 40)
	complete := 0
	for _, g := range g40 {
		if g.Complete {
			complete++
		}
	}
	assert.Equal(t, 29, complete)
	assert.False(t, g40[len(g40)-1].Complete)
}
