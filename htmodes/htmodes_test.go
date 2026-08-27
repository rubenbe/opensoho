package htmodes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModes(t *testing.T) {
	tests := []struct {
		name string
		caps Capabilities
		band string
		want []string
	}{
		{
			// Capture from WRT3200ACM, cross-checked against
			// `ubus call iwinfo info` at the same moment.
			name: "real capture - openwrt-garage radio0",
			caps: Capabilities{HTCapa: 6255, VHTCapa: 865827190},
			band: "5",
			want: []string{"HT20", "HT40", "VHT20", "VHT40", "VHT80", "VHT160"},
		},
		{
			name: "no capabilities at all",
			caps: Capabilities{},
			band: "2.4",
			want: nil,
		},
		{
			name: "HT20 only - width bit unset",
			caps: Capabilities{HTCapa: 0x0001},
			band: "2.4",
			want: []string{"HT20"},
		},
		{
			name: "HT40 - width bit set",
			caps: Capabilities{HTCapa: 0x0001 | 1<<1},
			band: "2.4",
			want: []string{"HT20", "HT40"},
		},
		{
			name: "VHT present but band isn't 5 - never emitted",
			caps: Capabilities{VHTCapa: 865827190},
			band: "6",
			want: nil,
		},
		{
			name: "VHT80 only - width bits 00 (80 MHz max)",
			caps: Capabilities{VHTCapa: 0x0001},
			band: "5",
			want: []string{"VHT20", "VHT40", "VHT80"},
		},
		{
			name: "VHT160 - width bits 01",
			caps: Capabilities{VHTCapa: 0x0001 | 1<<2},
			band: "5",
			want: []string{"VHT20", "VHT40", "VHT80", "VHT160"},
		},
		{
			name: "VHT160 - width bits 10 (160 + 80+80, still just VHT160 in our vocabulary)",
			caps: Capabilities{VHTCapa: 0x0001 | 2<<2},
			band: "5",
			want: []string{"VHT20", "VHT40", "VHT80", "VHT160"},
		},
		{
			name: "HE20 only - no HE_CAP_PHY width bits set",
			caps: Capabilities{HECapPHY: []byte{0x00}},
			band: "6",
			want: []string{"HE20"},
		},
		{
			name: "HE40 - bit 1",
			caps: Capabilities{HECapPHY: []byte{1 << 1}},
			band: "5",
			want: []string{"HE20", "HE40"},
		},
		{
			name: "HE80 - bit 2 implies HE40 too",
			caps: Capabilities{HECapPHY: []byte{1 << 2}},
			band: "5",
			want: []string{"HE20", "HE40", "HE80"},
		},
		{
			name: "HE160 - bit 3",
			caps: Capabilities{HECapPHY: []byte{1 << 3}},
			band: "6",
			want: []string{"HE20", "HE160"},
		},
		{
			name: "HE160 - bit 4 (160 + 80+80, still just HE160)",
			caps: Capabilities{HECapPHY: []byte{1 << 4}},
			band: "6",
			want: []string{"HE20", "HE160"},
		},
		{
			name: "empty HE_CAP_PHY attribute - absent, not zero-width",
			caps: Capabilities{HECapPHY: []byte{}},
			band: "6",
			want: nil,
		},
		{
			// BT8's real 6 GHz band (issue #59), per the reporter's board.json.
			name: "6 GHz radio with full EHT320 capability",
			caps: Capabilities{
				HECapPHY:  []byte{1<<1 | 1<<2 | 1<<3 | 1<<4},
				EHTCapPHY: []byte{1 << 1},
			},
			band: "6",
			want: []string{"HE20", "HE40", "HE80", "HE160", "EHT20", "EHT40", "EHT80", "EHT160", "EHT320"},
		},
		{
			name: "EHT320 bit set but band isn't 6 - never emitted",
			caps: Capabilities{
				HECapPHY:  []byte{1<<1 | 1<<2},
				EHTCapPHY: []byte{1 << 1},
			},
			band: "5",
			want: []string{"HE20", "HE40", "HE80", "EHT20", "EHT40", "EHT80"},
		},
		{
			// band "5": bit 1 of eht0 is also the EHT320 trigger on "6".
			name: "EHT present but co-located HE_CAP_PHY absent - EHT20 only",
			caps: Capabilities{
				EHTCapPHY: []byte{1 << 1},
			},
			band: "5",
			want: []string{"EHT20"},
		},
		{
			// HT isn't band-gated (matches upstream); VHT still is.
			name: "unrecognised band - HT still emitted, VHT still isn't",
			caps: Capabilities{HTCapa: 6255, VHTCapa: 865827190},
			band: "unknown",
			want: []string{"HT20", "HT40"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, tt.caps.Modes(tt.band), tt.name)
		})
	}
}

func TestModesIsDeterministic(t *testing.T) {
	caps := Capabilities{
		HTCapa:    6255,
		VHTCapa:   865827190,
		HECapPHY:  []byte{1<<1 | 1<<2 | 1<<3},
		EHTCapPHY: []byte{1 << 1},
	}
	first := caps.Modes("6")
	for i := 0; i < 5; i++ {
		assert.Equal(t, first, caps.Modes("6"), "Modes must return the same order every call")
	}
}

// TestCapabilitiesUnmarshalJSON exercises the wire shape scripts/dump-radios.uc's
// radio_caps() actually produces: ucode's nl80211 module renders HE/EHT PHY
// capability elements as a JSON array of numbers, not base64.
func TestCapabilitiesUnmarshalJSON(t *testing.T) {
	t.Run("real capture - openwrt-garage radio0", func(t *testing.T) {
		var c Capabilities
		err := json.Unmarshal([]byte(`{"ht_capa":6255,"vht_capa":865827190}`), &c)
		assert.NoError(t, err)
		assert.Equal(t, Capabilities{HTCapa: 6255, VHTCapa: 865827190}, c)
		assert.ElementsMatch(t, []string{"HT20", "HT40", "VHT20", "VHT40", "VHT80", "VHT160"}, c.Modes("5"))
	})

	t.Run("he_cap_phy is a number array, not base64", func(t *testing.T) {
		var c Capabilities
		err := json.Unmarshal([]byte(`{"he_cap_phy":[2,26,0,8,0,0,0,0,0]}`), &c)
		assert.NoError(t, err)
		assert.Equal(t, CapPHY{2, 26, 0, 8, 0, 0, 0, 0, 0}, c.HECapPHY)
	})

	t.Run("absent HE/EHT attributes decode to nil", func(t *testing.T) {
		var c Capabilities
		err := json.Unmarshal([]byte(`{"ht_capa":4591}`), &c)
		assert.NoError(t, err)
		assert.Nil(t, c.HECapPHY)
		assert.Nil(t, c.EHTCapPHY)
	})

	t.Run("null HE/EHT attributes decode to nil", func(t *testing.T) {
		var c Capabilities
		err := json.Unmarshal([]byte(`{"ht_capa":4591,"he_cap_phy":null,"eht_cap_phy":null}`), &c)
		assert.NoError(t, err)
		assert.Nil(t, c.HECapPHY)
		assert.Nil(t, c.EHTCapPHY)
	})

	t.Run("out-of-range byte value errors", func(t *testing.T) {
		var c Capabilities
		err := json.Unmarshal([]byte(`{"he_cap_phy":[2,300]}`), &c)
		assert.Error(t, err)
	})
}
