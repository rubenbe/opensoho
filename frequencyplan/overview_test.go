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

	ov := BuildOverview(radios, nil, nil, names)

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
	ov := BuildOverview(nil, freqs, nil, nil)
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
	ov := BuildOverview(nil, freqs, nil, nil)
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
	ov := BuildOverview(nil, freqs, nil, names)
	tier80 := findTier(findBand(ov, "5"), 80)

	gLow := blockAt(tier80, 0) // 36-48
	assert.Equal(t, "available", gLow.State)
	assert.Equal(t, []string{"AP-A"}, refNames(gLow.SupportedBy))
	assert.Equal(t, "A", gLow.SupportedBy[0].Id)
	assert.Equal(t, []string{"AP-B"}, refNames(gLow.UnsupportedBy)) // B has rows but not 36-48

	gHigh := blockAt(tier80, 20) // 149-161
	assert.Equal(t, "available", gHigh.State)
	assert.Equal(t, []string{"AP-B"}, refNames(gHigh.SupportedBy))
	assert.Equal(t, []string{"AP-A"}, refNames(gHigh.UnsupportedBy))

	gMid := blockAt(tier80, 8) // 100-112, supported by neither
	assert.Equal(t, "invalid", gMid.State)
	assert.Empty(t, gMid.SupportedBy)
	assert.Equal(t, []string{"AP-A", "AP-B"}, refNames(gMid.UnsupportedBy))
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
	ov := BuildOverview(nil, freqs, nil, names)
	b5 := findBand(ov, "5")

	g80 := blockAt(findTier(b5, 80), 0) // 36-48: A forbidden, B supports
	assert.Equal(t, "available", g80.State)
	assert.Equal(t, []string{"AP-B"}, refNames(g80.SupportedBy))
	assert.Equal(t, []string{"AP-A"}, refNames(g80.UnsupportedBy))

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
	ov := BuildOverview(radios, freqs, nil, names)

	g80 := blockAt(findTier(findBand(ov, "5"), 80), 8) // 100-112
	assert.Equal(t, "available", g80.State)
	assert.Empty(t, g80.SupportedBy)
	assert.Equal(t, []string{"AP-A"}, refNames(g80.UnsupportedBy)) // A known-but-missing; B unknown, not listed
}

func TestBuildOverviewHtModeCapsWidth(t *testing.T) {
	// Device advertises the full 6 GHz 1-61 PSC range (freqs 5955..6255, step
	// 20) with no restrictive flags, but its htmodes cap out at HE160 (no
	// EHT320) -- mirrors a Wi-Fi 6E radio like Predator's MT7916AN. Absence of
	// a no_320mhz flag alone must not be read as 320 MHz support.
	var freqs []Frequency
	for i := 0; i < 16; i++ {
		freqs = append(freqs, Frequency{Device: "predator", Radio: 1, Frequency: 5955 + 20*i})
	}
	htModes := []HtModes{
		{Device: "predator", Radio: 1, Modes: []string{"HE20", "HE40", "HE80", "HE160"}},
	}
	names := map[string]string{"predator": "Predator"}
	ov := BuildOverview(nil, freqs, htModes, names)
	b6 := findBand(ov, "6")
	assert.NotNil(t, b6)

	// 160 MHz: the 33-61 group is complete and within HE160 -> supported.
	g160 := blockAt(findTier(b6, 160), 8)
	assert.Equal(t, "available", g160.State)
	assert.Equal(t, []string{"Predator"}, refNames(g160.SupportedBy))

	// 320 MHz: the 1-61 group is complete and unflagged, but the device's
	// htmodes never reach 320 MHz -> not a supporter, block greys out.
	g320 := blockAt(findTier(b6, 320), 0)
	assert.Equal(t, "invalid", g320.State)
	assert.Empty(t, g320.SupportedBy)
	assert.Equal(t, []string{"Predator"}, refNames(g320.UnsupportedBy))
}

func TestBuildOverviewHtModeAllowsWifi7Width(t *testing.T) {
	// A genuine Wi-Fi 7 device (EHT320 in its advertised htmodes) sharing the
	// same 6 GHz 1-61 range as a Wifi 6E device (HE160-only):
	// the 320 MHz block should be counted
	// as actively supported by the Wi-Fi 7 device via deviceWidths, not just
	// left available by the absence-of-flag/no-data fallbacks, and the two
	// devices' capabilities must be judged independently within the same
	// aggregate scope.
	var freqs []Frequency
	for i := 0; i < 16; i++ {
		freqs = append(freqs,
			Frequency{Device: "wifi6eap", Radio: 1, Frequency: 5955 + 20*i},
			Frequency{Device: "wifi7ap", Radio: 0, Frequency: 5955 + 20*i},
		)
	}
	htModes := []HtModes{
		{Device: "wifi6eap", Radio: 1, Modes: []string{"HE20", "HE40", "HE80", "HE160"}},
		{Device: "wifi7ap", Radio: 0, Modes: []string{"HE20", "HE40", "HE80", "HE160", "EHT20", "EHT40", "EHT80", "EHT160", "EHT320"}},
	}
	names := map[string]string{"wifi6eap": "Wifi6E-AP", "wifi7ap": "Wifi7-AP"}
	ov := BuildOverview(nil, freqs, htModes, names)

	g320 := blockAt(findTier(findBand(ov, "6"), 320), 0)
	assert.Equal(t, "available", g320.State)
	assert.Equal(t, []string{"Wifi7-AP"}, refNames(g320.SupportedBy))
	assert.Equal(t, []string{"Wifi6E-AP"}, refNames(g320.UnsupportedBy))
}

func TestBuildOverviewNoHtModeDataUnconstrained(t *testing.T) {
	// Same advertised channels as TestBuildOverviewHtModeCapsWidth, but no
	// radio_ht_modes row yet (e.g. the device hasn't re-dumped since that
	// collection was added) -- support falls back to the existing flags-only
	// heuristic rather than greying out.
	var freqs []Frequency
	for i := 0; i < 16; i++ {
		freqs = append(freqs, Frequency{Device: "predator", Radio: 1, Frequency: 5955 + 20*i})
	}
	names := map[string]string{"predator": "Predator"}
	ov := BuildOverview(nil, freqs, nil, names)

	g320 := blockAt(findTier(findBand(ov, "6"), 320), 0)
	assert.Equal(t, "available", g320.State)
	assert.Equal(t, []string{"Predator"}, refNames(g320.SupportedBy))
}

func TestBuildOverviewHtModeCapsWidth5GHz(t *testing.T) {
	// Device advertises the full 36-64 range but its htmodes cap out at
	// VHT80 (no VHT160/HE160), so 160 MHz support should not be claimed even
	// though no radio_frequencies flag forbids it.
	var freqs []Frequency
	for _, ch := range []int{36, 40, 44, 48, 52, 56, 60, 64} {
		freqs = append(freqs, Frequency{Device: "ap", Radio: 0, Frequency: 5000 + ch*5})
	}
	htModes := []HtModes{
		{Device: "ap", Radio: 0, Modes: []string{"HT20", "HT40", "VHT20", "VHT40", "VHT80"}},
	}
	names := map[string]string{"ap": "AP"}
	ov := BuildOverview(nil, freqs, htModes, names)
	b5 := findBand(ov, "5")

	// 80 MHz: the 36-48 group is within VHT80 -> still supported.
	g80 := blockAt(findTier(b5, 80), 0)
	assert.Equal(t, "available", g80.State)
	assert.Equal(t, []string{"AP"}, refNames(g80.SupportedBy))

	// 160 MHz: the 36-64 group is complete but VHT80 caps the device out.
	g160 := blockAt(findTier(b5, 160), 0)
	assert.Equal(t, "invalid", g160.State)
	assert.Empty(t, g160.SupportedBy)
	assert.Equal(t, []string{"AP"}, refNames(g160.UnsupportedBy))
}

func TestBuildOverviewSkipsEmptyBand(t *testing.T) {
	// Only a 2.4 GHz radio -> no 5/6 GHz bands in the output.
	ov := BuildOverview([]Radio{{Device: "d", Frequency: 2412, Htmode: "HT20"}}, nil, nil, nil)
	assert.NotNil(t, findBand(ov, "2.4"))
	assert.Nil(t, findBand(ov, "5"))
	assert.Nil(t, findBand(ov, "6"))
}
