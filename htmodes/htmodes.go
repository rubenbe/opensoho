// Package htmodes decodes raw nl80211 HT/VHT/HE/EHT capability fields into
// OpenSOHO's htmode vocabulary ("HT20", "VHT80", "HE160", "EHT320", ...).
//
// Go port of iwinfo's nl80211_eval_modelist, so this can be run per radio
// instead of per (possibly shared) wiphy - see OpenSOHO issue #59
// https://github.com/openwrt/iwinfo/blob/66bdd1a071895d91babc9b9228bb84626bbce226/iwinfo_nl80211.c#L3341-L3418
package htmodes

import (
	"encoding/json"
	"fmt"
	"sort"
)

// CapPHY is a raw PHY capability element (NL80211_BAND_IFTYPE_ATTR_HE_CAP_PHY
// / EHT_CAP_PHY). ucode's nl80211 module renders these DT_U8|DF_ARRAY
// attributes as a JSON array of numbers, not the base64 string
// encoding/json uses for a plain []byte.
type CapPHY []byte

func (c *CapPHY) UnmarshalJSON(data []byte) error {
	if string(data) == "null" { // matches encoding/json's usual no-op for a null slice.
		return nil
	}
	var nums []uint16
	if err := json.Unmarshal(data, &nums); err != nil {
		return err
	}
	b := make([]byte, len(nums))
	for i, n := range nums {
		if n > 0xff {
			return fmt.Errorf("htmodes: byte %d out of range: %d", i, n)
		}
		b[i] = byte(n)
	}
	*c = b
	return nil
}

// Capabilities holds the raw per-band nl80211 capability fields for a
// single radio, as reported by scripts/dump-radios.uc's radio_caps().
type Capabilities struct {
	HTCapa    uint16 `json:"ht_capa"`     // NL80211_BAND_ATTR_HT_CAPA verbatim; 0 = absent.
	VHTCapa   uint32 `json:"vht_capa"`    // NL80211_BAND_ATTR_VHT_CAPA verbatim; 0 = absent.
	HECapPHY  CapPHY `json:"he_cap_phy"`  // only byte 0 is used.
	EHTCapPHY CapPHY `json:"eht_cap_phy"` // only byte 0 is used.
}

// Modes returns the htmode values this radio supports on the given band
// ("2.4", "5", "6", or "60", matching frequencyplan's vocabulary), sorted
// and de-duplicated. Only VHT (5 GHz) and EHT320 (6 GHz) are band-gated,
// matching upstream; HT/HE/EHT<320 are not. 80+80 MHz isn't in this
// project's htmode vocabulary (see htModesForBand in opensoho.go), so its
// bits are only consulted as a *160 trigger, never their own string.
func (c Capabilities) Modes(band string) []string {
	seen := map[string]struct{}{}
	add := func(modes ...string) {
		for _, m := range modes {
			seen[m] = struct{}{}
		}
	}

	if c.HTCapa > 0 {
		add("HT20")
		if c.HTCapa&(1<<1) != 0 { // HT Capabilities Info bit 1: 40 MHz.
			add("HT40")
		}
	}

	if band == "5" && c.VHTCapa > 0 {
		add("VHT20", "VHT40", "VHT80")
		if (c.VHTCapa>>2)&3 != 0 { // bits 2-3: Supported Channel Width Set.
			add("VHT160") // covers 160 MHz and 80+80 MHz.
		}
	}

	// HE/EHT width bits below are BIT(9..12) in upstream's nl80211_modes
	// struct, i.e. bits 1-4 of the raw attribute's first byte - upstream
	// left-pads the attribute into a uint16_t[] for alignment.
	if he0, ok := firstByte(c.HECapPHY); ok {
		add("HE20")
		if he0&(1<<1) != 0 {
			add("HE40")
		}
		if he0&(1<<2) != 0 {
			add("HE40", "HE80")
		}
		if he0&(1<<3)|he0&(1<<4) != 0 {
			add("HE160") // covers 160 MHz and 80+80 MHz.
		}
	}

	if eht0, ok := firstByte(c.EHTCapPHY); ok {
		add("EHT20")
		// Upstream's EHT40/80/160 checks test the HE width bits again
		// (he0), not the EHT element - kept as-is, not "fixed", since
		// this isn't verifiable against the 802.11be spec text here.
		if he0, ok := firstByte(c.HECapPHY); ok {
			if he0&(1<<1) != 0 {
				add("EHT40")
			}
			if he0&(1<<2) != 0 {
				add("EHT40", "EHT80")
			}
			if he0&(1<<3)|he0&(1<<4) != 0 {
				add("EHT160")
			}
		}
		if band == "6" && eht0&(1<<1) != 0 { // EHT element bit 1: 320 MHz.
			add("EHT320")
		}
	}

	if len(seen) == 0 {
		return nil
	}
	modes := make([]string, 0, len(seen))
	for m := range seen {
		modes = append(modes, m)
	}
	sort.Strings(modes)
	return modes
}

// firstByte returns b[0] and true, or (0, false) if b is empty.
func firstByte(b []byte) (byte, bool) {
	if len(b) == 0 {
		return 0, false
	}
	return b[0], true
}
