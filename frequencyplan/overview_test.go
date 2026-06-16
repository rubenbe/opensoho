package frequencyplan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func findBand(ov []BandOverview, band string) *BandOverview {
	for i := range ov {
		if ov[i].Band == band {
			return &ov[i]
		}
	}
	return nil
}

func findTier(b *BandOverview, width int) *Tier {
	if b == nil {
		return nil
	}
	for i := range b.Tiers {
		if b.Tiers[i].Width == width {
			return &b.Tiers[i]
		}
	}
	return nil
}

func refNames(refs []DeviceRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Name
	}
	return out
}

func blockAt(tier *Tier, startIndex int) *Block {
	if tier == nil {
		return nil
	}
	for i := range tier.Groups {
		if tier.Groups[i].StartIndex == startIndex {
			return &tier.Groups[i]
		}
	}
	return nil
}

func TestBuildOverviewFallbackNoFreqData(t *testing.T) {
	radios := []Radio{
		{Device: "dev1", Frequency: 5180, Htmode: "VHT40"}, // 5 GHz ch36, 40 MHz
		{Device: "dev2", Frequency: 2437, Htmode: "HT20"},  // 2.4 GHz ch6, 20 MHz
	}
	names := map[string]string{"dev1": "AP-1", "dev2": "AP-2"}

	ov := BuildOverview(radios, nil, names)

	b5 := findBand(ov, "5")
	assert.NotNil(t, b5)

	// 40 MHz tier, first group (ch36-40) is in use by AP-1.
	used := blockAt(findTier(b5, 40), 0)
	assert.NotNil(t, used)
	assert.Equal(t, "used", used.State)
	assert.Equal(t, "36–40", used.Label)
	assert.Equal(t, []string{"AP-1"}, refNames(used.Devices))
	assert.Equal(t, "dev1", used.Devices[0].Id)

	// 20 MHz tier, ch36 is valid but unused (no radio_frequencies data -> available).
	avail := blockAt(findTier(b5, 20), 0)
	assert.Equal(t, "available", avail.State)

	// 160 MHz tier, first group (36-64) is complete -> available even without freq data.
	g160 := blockAt(findTier(b5, 160), 0)
	assert.Equal(t, "available", g160.State)

	// 2.4 GHz ch6 in use at 20 MHz.
	b24 := findBand(ov, "2.4")
	ch6 := blockAt(findTier(b24, 20), 5) // index 5 == channel 6
	assert.Equal(t, "used", ch6.State)
	assert.Equal(t, []string{"AP-2"}, refNames(ch6.Devices))
}

func TestBuildOverviewMissingChannelInvalid(t *testing.T) {
	// Advertise only 5 GHz channels 36 and 40.
	freqs := []Frequency{
		{Device: "dev1", Frequency: 5180}, // ch36
		{Device: "dev1", Frequency: 5200}, // ch40
	}
	ov := BuildOverview(nil, freqs, nil)
	b5 := findBand(ov, "5")
	assert.NotNil(t, b5)

	// ch36 advertised -> available; ch44 (index 2) not advertised -> invalid.
	assert.Equal(t, "available", blockAt(findTier(b5, 20), 0).State)
	assert.Equal(t, "invalid", blockAt(findTier(b5, 20), 2).State)

	// 40 MHz: 36-40 advertised+complete -> available; 44-48 has missing member -> invalid.
	assert.Equal(t, "available", blockAt(findTier(b5, 40), 0).State)
	assert.Equal(t, "invalid", blockAt(findTier(b5, 40), 2).State)

	// 80 MHz: 36-48 group has missing members 44/48 -> invalid.
	assert.Equal(t, "invalid", blockAt(findTier(b5, 80), 0).State)
}

func TestBuildOverviewFlagForbidsWidth(t *testing.T) {
	// Advertise a full 80 MHz worth of channels, but flag no_80mhz on the primary.
	freqs := []Frequency{
		{Device: "dev1", Frequency: 5180, Flags: []string{"no_80mhz"}}, // ch36
		{Device: "dev1", Frequency: 5200},                              // ch40
		{Device: "dev1", Frequency: 5220},                              // ch44
		{Device: "dev1", Frequency: 5240},                              // ch48
	}
	ov := BuildOverview(nil, freqs, nil)
	b5 := findBand(ov, "5")

	// 40 MHz over 36-40 is allowed.
	assert.Equal(t, "available", blockAt(findTier(b5, 40), 0).State)
	// 80 MHz over 36-48 is forbidden by the no_80mhz flag -> no supporter.
	g80 := blockAt(findTier(b5, 80), 0)
	assert.Equal(t, "invalid", g80.State)
	assert.Empty(t, g80.SupportedBy)
}

func TestBuildOverviewAggregateAnyDeviceSupports(t *testing.T) {
	// Device A advertises the lower 80 MHz block (36-48); device B advertises the
	// 149-161 block. Each supports its own block; neither supports 100-112.
	freqs := []Frequency{
		{Device: "A", Frequency: 5180}, {Device: "A", Frequency: 5200},
		{Device: "A", Frequency: 5220}, {Device: "A", Frequency: 5240},
		{Device: "B", Frequency: 5745}, {Device: "B", Frequency: 5765},
		{Device: "B", Frequency: 5785}, {Device: "B", Frequency: 5805},
	}
	names := map[string]string{"A": "AP-A", "B": "AP-B"}
	ov := BuildOverview(nil, freqs, names)
	tier80 := findTier(findBand(ov, "5"), 80)

	gLow := blockAt(tier80, 0) // 36-48
	assert.Equal(t, "available", gLow.State)
	assert.Equal(t, []string{"AP-A"}, refNames(gLow.SupportedBy))
	assert.Equal(t, "A", gLow.SupportedBy[0].Id)

	gHigh := blockAt(tier80, 20) // 149-161
	assert.Equal(t, "available", gHigh.State)
	assert.Equal(t, []string{"AP-B"}, refNames(gHigh.SupportedBy))

	gMid := blockAt(tier80, 8) // 100-112, supported by neither
	assert.Equal(t, "invalid", gMid.State)
	assert.Empty(t, gMid.SupportedBy)
}

func TestBuildOverviewAggregateFlagRescuedByOtherDevice(t *testing.T) {
	// Both advertise 36-48, but device A forbids 80 MHz on the primary.
	freqs := []Frequency{
		{Device: "A", Frequency: 5180, Flags: []string{"no_80mhz"}},
		{Device: "A", Frequency: 5200}, {Device: "A", Frequency: 5220}, {Device: "A", Frequency: 5240},
		{Device: "B", Frequency: 5180}, {Device: "B", Frequency: 5200},
		{Device: "B", Frequency: 5220}, {Device: "B", Frequency: 5240},
	}
	names := map[string]string{"A": "AP-A", "B": "AP-B"}
	ov := BuildOverview(nil, freqs, names)
	b5 := findBand(ov, "5")

	g80 := blockAt(findTier(b5, 80), 0) // 36-48: A forbidden, B supports
	assert.Equal(t, "available", g80.State)
	assert.Equal(t, []string{"AP-B"}, refNames(g80.SupportedBy))

	g40 := blockAt(findTier(b5, 40), 0) // 36-40: 40 MHz allowed for both
	assert.Equal(t, "available", g40.State)
	assert.Equal(t, []string{"AP-A", "AP-B"}, refNames(g40.SupportedBy))
}

func TestBuildOverviewUnknownCapabilityDevicePreventsGreying(t *testing.T) {
	// Device A advertises only 36,40 (so 100-112 is unsupported for A); device B
	// has no advertised frequencies but has a radio in the band -> unknown
	// capabilities -> keeps the block available without being listed.
	freqs := []Frequency{
		{Device: "A", Frequency: 5180}, {Device: "A", Frequency: 5200},
	}
	radios := []Radio{{Device: "B", Frequency: 5500, Htmode: "HT20"}} // ch100, 20 MHz
	names := map[string]string{"A": "AP-A", "B": "AP-B"}
	ov := BuildOverview(radios, freqs, names)

	g80 := blockAt(findTier(findBand(ov, "5"), 80), 8) // 100-112
	assert.Equal(t, "available", g80.State)
	assert.Empty(t, g80.SupportedBy)
}

func TestBuildOverviewSkipsEmptyBand(t *testing.T) {
	// Only a 2.4 GHz radio -> no 5/6 GHz bands in the output.
	ov := BuildOverview([]Radio{{Device: "d", Frequency: 2412, Htmode: "HT20"}}, nil, nil)
	assert.NotNil(t, findBand(ov, "2.4"))
	assert.Nil(t, findBand(ov, "5"))
	assert.Nil(t, findBand(ov, "6"))
}
