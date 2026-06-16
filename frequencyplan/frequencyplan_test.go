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
		{5180, "5"},
		{5825, "5"},
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
		{5180, 36, true},
		{5200, 40, true},
		{5500, 100, true},
		{5825, 165, true},
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
	assert.True(t, WidthForbidden(20, []string{"no_20mhz"}))
	assert.False(t, WidthForbidden(80, []string{"no_160mhz"}))
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
	assert.Equal(t, []int{36, 40}, channelsOf(g40[0]))
	assert.True(t, g40[0].Complete)
	assert.Equal(t, 0, g40[0].StartIndex)
	assert.Equal(t, 2, g40[0].Span)
	// 165 is the last channel of its run and cannot pair.
	last := g40[len(g40)-1]
	assert.Equal(t, []int{165}, channelsOf(last))
	assert.False(t, last.Complete)

	g80 := BondingGroups("5", 80)
	assert.Equal(t, []int{36, 40, 44, 48}, channelsOf(g80[0]))
	assert.True(t, g80[0].Complete)

	g160 := BondingGroups("5", 160)
	assert.Equal(t, []int{36, 40, 44, 48, 52, 56, 60, 64}, channelsOf(g160[0]))
	assert.True(t, g160[0].Complete)
	assert.Equal(t, []int{100, 104, 108, 112, 116, 120, 124, 128}, channelsOf(g160[1]))
	assert.True(t, g160[1].Complete)
	// 132–144 (only 4 channels left before the 149 boundary) cannot form 160 MHz.
	assert.Equal(t, []int{132, 136, 140, 144}, channelsOf(g160[2]))
	assert.False(t, g160[2].Complete)
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
