package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/endobit/oui"
	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/rubenbe/pocketbase"
	"github.com/rubenbe/pocketbase/apis"
	"github.com/rubenbe/pocketbase/core"
	//"github.com/rubenbe/pocketbase/plugins/ghupdate"
	"github.com/reugn/wifiqr"
	"github.com/rubenbe/opensoho/frequencyplan"
	"github.com/rubenbe/opensoho/lldp"
	"github.com/rubenbe/opensoho/mqtt"
	"github.com/rubenbe/opensoho/poe"
	"github.com/rubenbe/opensoho/ui"
	"github.com/rubenbe/pocketbase/plugins/jsvm"
	"github.com/rubenbe/pocketbase/plugins/migratecmd"
	"github.com/rubenbe/pocketbase/tools/filesystem"
	"github.com/rubenbe/pocketbase/tools/hook"
	"github.com/rubenbe/pocketbase/tools/security"
	"github.com/rubenbe/pocketbase/tools/types"
)

// Files that need to be extracted at startup
//
//go:embed pb_migrations/**
var embeddedFiles embed.FS

// Files that can be served directly from the binary
//
//go:embed favicon.png logo.svg
var internalFiles embed.FS

// Hotplug script pushed to the router at /etc/hotplug.d/openwisp/opensoho
//
//go:embed scripts/dump-radios.sh
var dumpRadiosScript string

// Hotplug script pushed to the router at /etc/hotplug.d/openwisp/opensoho-poe
//
//go:embed scripts/dump-poe.sh
var dumpPoeScript string

func copyEmbedDirToDisk(embedFS fs.FS, targetDir string) error {
	return fs.WalkDir(embedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, path)

		if d.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
		}

		// Open embedded file
		srcFile, err := embedFS.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// Create or overwrite file on disk
		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func extractRadioNumber(s string) (int, error) {
	re := regexp.MustCompile(`^(?:phy|wl)(\d+)-`)
	match := re.FindStringSubmatch(s)
	if len(match) < 2 {
		return 0, fmt.Errorf("radio number not found in string: %s", s)
	}
	return strconv.Atoi(match[1])
}

// parseRadioName extracts the radio index from an OpenSoho dump radio name
// (the UCI wifi-device section name, e.g. "radio0" -> 0).
func parseRadioName(name string) (int, error) {
	return strconv.Atoi(strings.TrimPrefix(name, "radio"))
}

func updateDeviceHealth(app core.App, currenttime types.DateTime) {
	oldesttime := currenttime.Add(-60 * time.Second)

	// Collect the devices that are about to transition to unhealthy so we can
	// flip their Home Assistant availability to offline.
	var transitioning []struct {
		Id string `db:"id"`
	}
	err := app.DB().
		NewQuery("select id from devices where health_status != \"unhealthy\" and last_seen <= {:offset}").
		Bind(dbx.Params{"offset": oldesttime.String()}).All(&transitioning)
	if err != nil {
		fmt.Println("Failed to query transitioning devices")
		fmt.Println(err)
	}

	_, err = app.DB().
		NewQuery("update devices set health_status = \"unhealthy\" where last_seen <= {:offset}").
		Bind(dbx.Params{"offset": oldesttime.String()}).Execute()
	if err != nil {
		fmt.Println("Failed to update device health")
		fmt.Println(err)
		return
	}

	for _, d := range transitioning {
		mqtt.PublishDeviceOffline(d.Id)
	}
}
func updateLastSeen(e *core.RequestEvent, record *core.Record) error {
	record.Set("last_seen", time.Now())
	record.Set("health_status", "healthy")
	record.Set("ip_address", e.RealIP())
	return e.App.Save(record)
}

func frequencyToBand(frequency int) string {
	return frequencyplan.FrequencyToBand(frequency)
}

// frequencyToUciBand maps a frequency to the value UCI expects for the
// wifi-device "band" option (e.g. 2412 -> "2g"). Returns "" for unknown bands.
func frequencyToUciBand(frequency int) string {
	switch frequencyToBand(frequency) {
	case "2.4":
		return "2g"
	case "5":
		return "5g"
	case "6":
		return "6g"
	case "60":
		return "60g"
	default:
		return ""
	}
}

func isRandomizedMAC(mac string) bool {
	parts := strings.SplitN(mac, ":", 2)
	if len(parts) == 0 {
		return false
	}
	b, err := strconv.ParseUint(parts[0], 16, 8)
	if err != nil {
		return false
	}
	return b&0x02 != 0
}

func lookupVendor(mac string) string {
	if isRandomizedMAC(mac) {
		return "randomized"
	}
	return oui.Vendor(mac)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func frequencyToChannel(freqMHz int) (int, bool) {
	return frequencyplan.FrequencyToChannel(freqMHz)
}

func validateRadioHtModeBandCombo(band string, htmode string) error {
	validHtModes := map[string][]string{
		"2.4": {"HT20", "HT40", "HE20", "HE40", "EHT20", "EHT40"},
		"5":   {"HT20", "HT40", "VHT20", "VHT40", "VHT80", "VHT160", "HE20", "HE40", "HE80", "HE160", "EHT20", "EHT40", "EHT80", "EHT160"},
		"6":   {"HE20", "HE40", "HE80", "HE160", "EHT20", "EHT40", "EHT80", "EHT160", "EHT320"},
	}

	htmodes, ok := validHtModes[band]
	if !ok {
		return validation.NewError("validation_invalid_value", "Invalid band")
	}

	for _, h := range htmodes {
		if h == htmode {
			return nil
		}
	}

	return validation.NewError("validation_invalid_value", "HT mode does not match selected band")
}

// validateRadioFrequency checks the user-set frequency against the frequencies
// the device actually advertised in the radio_frequencies collection. If the
// device hasn't reported a freqlist for this radio yet (no rows), validation is
// skipped so the radio can still be configured.
func validateRadioFrequency(app core.App, device string, radio int, frequency int) error {
	freqs, err := app.FindAllRecords("radio_frequencies",
		dbx.HashExp{"device": device, "radio": radio})
	if err != nil {
		return validation.NewError("validation_invalid_value", "Failed to look up supported frequencies")
	}
	if len(freqs) == 0 {
		return nil
	}
	for _, f := range freqs {
		if f.GetInt("frequency") == frequency {
			return nil
		}
	}
	return validation.NewError("validation_invalid_value", "Frequency is not supported by this radio")
}

// lookupTxPowerDbm returns the highest advertised dBm whose mW value equals mw,
// for the given device+radio. found is false when the device has no matching
// radio_tx_powers row (or none at all).
func lookupTxPowerDbm(app core.App, device string, radio int, mw int) (int, bool, error) {
	rows, err := app.FindAllRecords("radio_tx_powers",
		dbx.HashExp{"device": device, "radio": radio, "mw": mw})
	if err != nil {
		return 0, false, err
	}
	best, found := 0, false
	for _, r := range rows {
		if d := r.GetInt("dbm"); !found || d > best {
			best, found = d, true
		}
	}
	return best, found, nil
}

// nearestTxPower scans radio_tx_powers for device+radio and reports whether
// value exactly matches the field column ("mw" or "dbm"), plus the row whose
// field value is closest to it. closest is nil only when the radio has no rows.
// Distance is compared as the squared difference: tx power values are
// non-negative but v-value is not, and squaring avoids an abs helper while
// giving the same nearest result.
func nearestTxPower(app core.App, device string, radio int, field string, value int) (bool, *core.Record, error) {
	rows, err := app.FindAllRecords("radio_tx_powers",
		dbx.HashExp{"device": device, "radio": radio})
	if err != nil {
		return false, nil, err
	}
	exact := false
	var closest *core.Record
	bestDist := 0
	for _, r := range rows {
		v := r.GetInt(field)
		if v == value {
			exact = true
		}
		if dist := (v - value) * (v - value); closest == nil || dist < bestDist {
			closest, bestDist = r, dist
		}
	}
	return exact, closest, nil
}

// validateRadioTxPower checks a mW- or dBm-mode tx_power against the device's
// advertised radio_tx_powers table. Any value without a matching row is
// rejected, with a hint at the closest supported level (or a note that the
// radio has reported none). auto/empty modes need no lookup.
func validateRadioTxPower(app core.App, device string, radio int, mode string, txpower int) error {
	var field string
	switch mode {
	case "mW":
		field = "mw"
	case "dBm":
		field = "dbm"
	default: // auto / empty — no lookup
		return nil
	}

	exact, closest, err := nearestTxPower(app, device, radio, field, txpower)
	if err != nil {
		return validation.NewError("validation_invalid_value", "Failed to look up supported tx powers")
	}
	if exact {
		return nil
	}
	if closest == nil {
		return validation.NewError("validation_invalid_value", fmt.Sprintf(
			"%d %s is not a supported tx power; this radio has not reported any supported power levels yet",
			txpower, mode))
	}
	return validation.NewError("validation_invalid_value", fmt.Sprintf(
		"%d %s is not a supported tx power for this radio; closest supported value is %d mW (%d dBm)",
		txpower, mode, closest.GetInt("mw"), closest.GetInt("dbm")))
}

// validateRadioHtModeFlags rejects channel widths the device flagged as unusable on the configured channel.
// If the device hasn't a row for this frequency, validation is skipped  As such the radio can still be configured.
func validateRadioHtModeFlags(app core.App, device string, radio int, frequency int, htmode string) error {
	rows, err := app.FindAllRecords("radio_frequencies",
		dbx.HashExp{"device": device, "radio": radio, "frequency": frequency})
	if err != nil {
		return validation.NewError("validation_invalid_value", "Failed to look up supported frequencies")
	}
	if len(rows) == 0 {
		return nil
	}
	flags := rows[0].GetStringSlice("flags")

	switch {
	case strings.HasSuffix(htmode, "40"):
		if slices.Contains(flags, "no_ht40-") && slices.Contains(flags, "no_ht40+") {
			// A 40 MHz width needs an adjacent secondary channel, so it is not allowed when both no_ht40- and no_ht40+ are set
			return validation.NewError("validation_invalid_value", "40 MHz width not allowed on this channel")
		}
	case strings.HasSuffix(htmode, "320"):
		if slices.Contains(flags, "no_320mhz") {
			return validation.NewError("validation_invalid_value", "320 MHz width not allowed on this channel")
		}
	case strings.HasSuffix(htmode, "160"):
		if slices.Contains(flags, "no_160mhz") {
			return validation.NewError("validation_invalid_value", "160 MHz width not allowed on this channel")
		}
	case strings.HasSuffix(htmode, "80"):
		if slices.Contains(flags, "no_80mhz") {
			return validation.NewError("validation_invalid_value", "80 MHz width not allowed on this channel")
		}
	}
	return nil
}

func validateRadio(app core.App, record *core.Record) error {
	errs := validation.Errors{}
	frequency := record.GetInt("frequency")

	err := validateRadioFrequency(app, record.GetString("device"), record.GetInt("radio"), frequency)
	if err != nil {
		errs["frequency"] = err
	}

	band := frequencyToBand(frequency)
	htmode := record.GetString("htmode")

	err = validateRadioHtModeBandCombo(band, htmode)
	if err != nil {
		errs["htmode"] = err
	} else if err = validateRadioHtModeFlags(app, record.GetString("device"), record.GetInt("radio"), frequency, htmode); err != nil {
		errs["htmode"] = err
	}

	if err := validateRadioTxPower(app, record.GetString("device"), record.GetInt("radio"),
		record.GetString("tx_power_mode"), record.GetInt("tx_power")); err != nil {
		errs["tx_power"] = err
	}
	if len(errs) > 0 {
		return apis.NewBadRequestError("Failed to create record.", errs)
	}
	return nil
}

type Client struct {
	MAC    string `json:"mac"`
	Assoc  bool   `json:"assoc"`
	Signal int    `json:"signal"`
	Bytes  struct {
		Rx uint64 `json:"rx"`
		Tx uint64 `json:"tx"`
	} `json:"bytes"`

	Rate struct {
		Rx uint64 `json:"rx"`
		Tx uint64 `json:"tx"`
	} `json:"rate"`
}

type Radio struct {
	Frequency int    `json:"frequency"`
	Channel   int    `json:"channel"`
	HTmode    string `json:"htmode"`
	TxPower   int    `json:"tx_power"`
}

type Wireless struct {
	Clients   []Client `json:"clients"`
	SSID      string   `json:"ssid"`
	Frequency int      `json:"frequency"`
	Channel   int      `json:"channel"`
	HTmode    string   `json:"htmode"`
	TxPower   int      `json:"tx_power"`
}

type DHCPLease struct {
	MACAddress string `json:"mac"`
	ClientID   string `json:"client_id,omitempty"`
	Hostname   string `json:"client_name,omitempty"`
	IPAddress  string `json:"ip"`
	Expiry     int    `json:"expiry"`
}

type Statistics struct {
	RxFrameErrors     uint64 `json:"rx_frame_errors"`
	RxCrcErrors       uint64 `json:"rx_crc_errors"`
	TxHeartbeatErrors uint64 `json:"tx_heartbeat_errors"`
	RxOverErrors      uint64 `json:"rx_over_errors"`
	RxErrors          uint64 `json:"rx_errors"`
	TxPackets         uint64 `json:"tx_packets"`
	TxCarrierErrors   uint64 `json:"tx_carrier_errors"`
	RxPackets         uint64 `json:"rx_packets"`
	RxLengthErrors    uint64 `json:"rx_length_errors"`
	TxErrors          uint64 `json:"tx_errors"`
	TxAbortedErrors   uint64 `json:"tx_aborted_errors"`
	TxWindowErrors    uint64 `json:"tx_window_errors"`
	TxBytes           uint64 `json:"tx_bytes"`
	Collisions        uint64 `json:"collisions"`
	RxBytes           uint64 `json:"rx_bytes"`
	RxFifoErrors      uint64 `json:"rx_fifo_errors"`
	RxDropped         uint64 `json:"rx_dropped"`
	TxFifoErrors      uint64 `json:"tx_fifo_errors"`
	RxCompressed      uint64 `json:"rx_compressed"`
	Multicast         uint64 `json:"multicast"`
	TxCompressed      uint64 `json:"tx_compressed"`
	RxMissedErrors    uint64 `json:"rx_missed_errors"`
	TxDropped         uint64 `json:"tx_dropped"`
}

type Interface struct {
	MAC           string      `json:"mac"`
	Type          string      `json:"type"`
	Name          string      `json:"name"`
	Wireless      *Wireless   `json:"wireless,omitempty"`
	Statistics    *Statistics `json:"statistics,omitempty"`
	Speed         string      `json:"speed,omitempty"`
	BridgeMembers []string    `json:"bridge_members,omitempty"`
}

type Resources struct {
	Load []float32 `json:"load"`
}

type Neighbor struct {
	MAC       string `json:"mac"`
	State     string `json:"state"`
	Interface string `json:"Interface"`
	IP        string `json:"ip"`
}

type GeneralInfo struct {
	LocalTime int `json:"local_time"`
	Uptime    int `json:"uptime"`
}

type MonitoringData struct {
	Type       string      `json:"type"`
	General    GeneralInfo `json:"general"`
	Interfaces []Interface `json:"interfaces"`
	Resources  Resources   `json:"resources"`
	DNSServers []string    `json:"dns_servers"`
	Neighbors  []Neighbor  `json:"neighbors"`
	DHCPLeases []DHCPLease `json:"dhcp_leases,omitempty"`
}

// OpenSoho monitoring payload, produced by scripts/dump-radios.sh.
// Shape: {"type":"OpenSoho","radios":[{"name":"radio0",...},...]} where each
// entry mirrors a UCI wifi-device augmented with iwinfo info / freqlist.

// IwinfoInfo holds the subset of `ubus call iwinfo info` we care about.
type IwinfoInfo struct {
	Channel   int      `json:"channel"`
	Frequency int      `json:"frequency"`
	TxPower   int      `json:"txpower"`
	Country   string   `json:"country"`
	HwModes   []string `json:"hwmodes"`
	HtModes   []string `json:"htmodes"`
}

// IwinfoFreq is a single entry of `ubus call iwinfo freqlist`.
type IwinfoFreq struct {
	Channel    int      `json:"channel"`
	MHz        int      `json:"mhz"`
	Restricted bool     `json:"restricted"`
	Flags      []string `json:"flags"`
}

// IwinfoTxPower is a single entry of `ubus call iwinfo txpowerlist`.
type IwinfoTxPower struct {
	Dbm int `json:"dbm"`
	Mw  int `json:"mw"`
}

// OpenSohoRadio is one wifi-device entry of the OpenSoho payload.
type OpenSohoRadio struct {
	Name     string     `json:"name"`
	Phy      string     `json:"phy"`
	Disabled string     `json:"disabled"`
	Info     IwinfoInfo `json:"info"`
	FreqList struct {
		Results []IwinfoFreq `json:"results"`
	} `json:"freqlist"`
	TxPowerList struct {
		Results []IwinfoTxPower `json:"results"`
	} `json:"txpowerlist"`
}

// OpenSohoData is the decoded OpenSoho payload. The radios dump
// (scripts/dump-radios.sh) carries "radios"; the PoE dump
// (scripts/dump-poe.sh) carries "poe". Both share type "OpenSoho", so one
// struct decodes either: the absent key simply stays nil/empty.
type OpenSohoData struct {
	Type   string          `json:"type"`
	Radios []OpenSohoRadio `json:"radios"`
	Poe    *poe.Info       `json:"poe"`
	Lldp   *lldp.Info      `json:"lldp"`
}

// radioBands returns the distinct Wi-Fi bands a radio supports, derived from
// its advertised frequency list. The result is sorted for deterministic output
// and excludes the "unknown" sentinel from frequencyToBand.
func radioBands(radio OpenSohoRadio) []string {
	seen := map[string]struct{}{}
	for _, freq := range radio.FreqList.Results {
		band := frequencyToBand(freq.MHz)
		if band == "unknown" {
			continue
		}
		seen[band] = struct{}{}
	}
	bands := make([]string, 0, len(seen))
	for band := range seen {
		bands = append(bands, band)
	}
	sort.Strings(bands)
	return bands
}

// handleOpenSohoMonitoring is the entry point for parsed OpenSoho radio dumps.
// It persists each radio's advertised frequency list into the
// radio_frequencies collection, keyed by (device, radio index).
func handleOpenSohoMonitoring(app core.App, device *core.Record, data OpenSohoData, current bool) {
	coll, err := app.FindCollectionByNameOrId("radio_frequencies")
	if err != nil {
		app.Logger().Error("Failed to find radio_frequencies collection", "error", err)
		return
	}
	txColl, err := app.FindCollectionByNameOrId("radio_tx_powers")
	if err != nil {
		app.Logger().Error("Failed to find radio_tx_powers collection", "error", err)
		return
	}

	for _, radio := range data.Radios {
		idx, err := parseRadioName(radio.Name)
		if err != nil {
			app.Logger().Error("Skipping radio with unparseable name",
				"device", device.GetString("id"), "radio", radio.Name, "error", err)
			continue
		}

		if err := syncRadioFrequencies(app, coll, device, idx, radio.FreqList.Results); err != nil {
			app.Logger().Error("Failed to sync radio frequencies",
				"device", device.GetString("id"), "radio", radio.Name, "error", err)
			continue
		}

		if err := syncRadioTxPowers(app, txColl, device, idx, radio.TxPowerList.Results); err != nil {
			app.Logger().Error("Failed to sync radio tx powers",
				"device", device.GetString("id"), "radio", radio.Name, "error", err)
			continue
		}
	}

	if data.Poe != nil {
		if err := poe.Sync(app, device, *data.Poe); err != nil {
			app.Logger().Error("Failed to sync poe ports",
				"device", device.GetString("id"), "error", err)
		}
		if current {
			mqtt.PublishPoE(device, *data.Poe)
		}
	}

	if data.Lldp != nil {
		if err := lldp.Sync(app, device, *data.Lldp); err != nil {
			app.Logger().Error("Failed to sync lldp neighbors",
				"device", device.GetString("id"), "error", err)
		}
	}
}

// syncRadioFrequencies reconciles the radio_frequencies rows for a single
// (device, radio) with the supplied freqlist. Rows are matched by frequency and
// adjusted in place; only newly advertised frequencies are inserted and only
// dropped frequencies are deleted, so the stored set always reflects the latest
// dump without churning unchanged rows.
func syncRadioFrequencies(app core.App, coll *core.Collection, device *core.Record, idx int, freqs []IwinfoFreq) error {
	// The flags field is a select with a fixed set of accepted values; drop any
	// flag the schema doesn't know about so an unexpected one doesn't fail the
	// whole save.
	var allowedFlags []string
	if field, ok := coll.Fields.GetByName("flags").(*core.SelectField); ok {
		allowedFlags = field.Values
	}

	return app.RunInTransaction(func(txApp core.App) error {
		existing, err := txApp.FindAllRecords("radio_frequencies",
			dbx.HashExp{"device": device.Id, "radio": idx})
		if err != nil {
			return err
		}
		// Index the existing rows by frequency. Together with the (device, radio)
		// scope of the query above this is the (device, radio, frequency) key we
		// upsert against, so unchanged frequencies keep their row instead of being
		// deleted and re-created.
		byFreq := make(map[int]*core.Record, len(existing))
		for _, rec := range existing {
			byFreq[rec.GetInt("frequency")] = rec
		}

		for _, f := range freqs {
			rec, ok := byFreq[f.MHz]
			if ok {
				// Adjust the existing row in place and remove it from the map so it
				// isn't treated as stale below.
				delete(byFreq, f.MHz)
			} else {
				rec = core.NewRecord(coll)
				rec.Set("device", device.Id)
				rec.Set("radio", idx)
				rec.Set("frequency", f.MHz)
			}
			rec.Set("channel", f.Channel)
			rec.Set("flags", knownFlags(f.Flags, allowedFlags))
			if err := txApp.Save(rec); err != nil {
				return err
			}
		}

		// Whatever frequencies remain are no longer advertised; drop them.
		for _, rec := range byFreq {
			if err := txApp.Delete(rec); err != nil {
				return err
			}
		}
		return nil
	})
}

func syncRadioTxPowers(app core.App, coll *core.Collection, device *core.Record, idx int, powers []IwinfoTxPower) error {
	return app.RunInTransaction(func(txApp core.App) error {
		existing, err := txApp.FindAllRecords("radio_tx_powers",
			dbx.HashExp{"device": device.Id, "radio": idx})
		if err != nil {
			return err
		}
		byDbm := make(map[int]*core.Record, len(existing))
		for _, rec := range existing {
			byDbm[rec.GetInt("dbm")] = rec
		}

		for _, p := range powers {
			rec, ok := byDbm[p.Dbm]
			if ok {
				// Adjust the existing row in place and remove it from the map so it
				// isn't treated as stale below.
				delete(byDbm, p.Dbm)
			} else {
				rec = core.NewRecord(coll)
				rec.Set("device", device.Id)
				rec.Set("radio", idx)
				rec.Set("dbm", p.Dbm)
			}
			rec.Set("mw", p.Mw)
			if err := txApp.Save(rec); err != nil {
				return err
			}
		}

		// Whatever power levels remain are no longer advertised; drop them.
		for _, rec := range byDbm {
			if err := txApp.Delete(rec); err != nil {
				return err
			}
		}
		return nil
	})
}

// knownFlags returns the subset of flags that appear in allowed, preserving
// order. When allowed is empty no filtering is applied.
func knownFlags(flags, allowed []string) []string {
	if len(allowed) == 0 {
		return flags
	}
	kept := make([]string, 0, len(flags))
	for _, f := range flags {
		if slices.Contains(allowed, f) {
			kept = append(kept, f)
		}
	}
	return kept
}

type WifiRecord struct {
	Record *core.Record
}

func updateRadios(device *core.Record, app core.App, newradios map[int]Radio) {
	// Radio has radio number as index, it is not an index in a list.
	// Function modifies the existing newradios list, important for tests
	oldradios, err := app.FindAllRecords("radios", dbx.HashExp{"device": device.GetString("id")})
	if err != nil {
		fmt.Println(err)
		return
	}
	// Loop over the existing (old) radios for this device, and update if not found.
	// Update the MAC address and set it to enabled
	for _, oldradio := range oldradios {
		fmt.Println("oldradio:", oldradio)
		oldradionum := oldradio.GetInt("radio")
		if newradio, ok := newradios[oldradionum]; ok {
			// Old radio exists within the updated list (newradios)
			fmt.Println("EXISTS", newradio, oldradio)
			dirty := false
			if oldradio.GetBool("enabled") == false {
				oldradio.Set("enabled", true)
				dirty = true
			}
			// tx_power_mode is a required field; rows created before it existed
			// hold an empty value that fails validation on save. Normalise the
			// empty value (which means auto) so the record can be saved again.
			mode := oldradio.GetString("tx_power_mode")
			if mode == "" {
				oldradio.Set("tx_power_mode", "auto")
				mode = "auto"
				dirty = true
			}
			// In auto mode the txpower option is omitted and the driver picks the
			// power; record the value it reports so tx_power reflects what the
			// radio is actually transmitting at. Never overwrite a value the user
			// pinned in dBm/mW mode.
			if mode == "auto" && newradio.TxPower > 0 &&
				oldradio.GetInt("tx_power") != newradio.TxPower {
				oldradio.Set("tx_power", newradio.TxPower)
				dirty = true
			}
			if dirty {
				if err := app.Save(oldradio); err != nil {
					fmt.Println("Failed to update radio:", err)
				}
			}
			delete(newradios, oldradionum)
		} else {
			fmt.Println("Not in list:", oldradio)
			// Old radio does not exist within the updated list
			if oldradio.GetBool("enabled") == true {
				oldradio.Set("enabled", false)
				err := app.Save(oldradio)
				if err != nil {
					fmt.Println("Failed to mark radio as disabled:", err)
				}
			}

		}
	}

	if len(newradios) == 0 {
		return
	}
	radiocollection, err := app.FindCollectionByNameOrId("radios")
	if err != nil {
		fmt.Println("Failed to find radio collection")
	}

	for numradio, radio := range newradios {
		fmt.Println(numradio, radio, device.GetString("id"))
		record := core.NewRecord(radiocollection)
		record.Set("device", device.GetString("id"))
		record.Set("radio", numradio)
		record.Set("channel", radio.Channel)
		record.Set("frequency", radio.Frequency)
		record.Set("enabled", true)
		// New radios default to auto power; store the reported value (in dBm) so
		// tx_power reflects what the driver chose.
		record.Set("tx_power_mode", "auto")
		if radio.TxPower > 0 {
			record.Set("tx_power", radio.TxPower)
		}
		err := app.Save(record)
		if err != nil {
			fmt.Println("Failed to save radio config")
		}
	}
}

func generateLedConfig(led *core.Record) string {
	name := led.GetString("name")
	return fmt.Sprintf(`
config led 'led_%s'
        option name '%s'
        option sysfs '%s'
        option trigger '%s'
`, strings.ToLower(name), name, led.GetString("led_name"), led.GetString("trigger"))
}

func generateRadioConfig(app core.App, radio *core.Record, country_code string) string {

	frequency_txt := "        option channel 'auto'\n"
	band_txt := ""
	if radio.GetBool("auto_frequency") != true {
		frequency := radio.GetInt("frequency")
		if channel, ok := frequencyToChannel(frequency); ok == true {
			frequency_txt = fmt.Sprintf("        option channel '%d'\n", channel)
		}
		// A specific frequency pins the band; emit it so the driver picks
		// the right radio band (e.g. option band '2g').
		if band := frequencyToUciBand(frequency); len(band) > 0 {
			band_txt = fmt.Sprintf("        option band '%[1]s'\n", band)
		}
	}
	htmode_txt := ""
	if htmode := radio.GetString("htmode"); len(htmode) > 0 {
		htmode_txt = fmt.Sprintf("        option htmode '%[1]s'\n", htmode)
	}

	country_txt := ""
	if len(country_code) > 0 {
		country_txt = fmt.Sprintf("        option country '%[1]s'\n", country_code)
	}

	// txpower in UCI is always dBm. mW mode is translated via the device's
	// advertised radio_tx_powers table; anything else omits the option so the
	// driver picks the power ("auto" is not a valid UCI value).
	txpower_txt := ""
	switch radio.GetString("tx_power_mode") {
	case "dBm":
		txpower_txt = fmt.Sprintf("        option txpower '%d'\n", radio.GetInt("tx_power"))
	case "mW":
		if dbm, found, _ := lookupTxPowerDbm(app, radio.GetString("device"),
			radio.GetInt("radio"), radio.GetInt("tx_power")); found {
			txpower_txt = fmt.Sprintf("        option txpower '%d'\n", dbm)
		}
	}

	return fmt.Sprintf(`
config wifi-device 'radio%[1]d'
%[2]s%[6]s%[3]s%[4]s%[5]s`, radio.GetInt("radio"), frequency_txt, country_txt, htmode_txt, txpower_txt, band_txt)
}

func getRadiosForDevice(device *core.Record, app core.App) ([]*core.Record, error) {
	records := []*core.Record{}
	err := app.RecordQuery("radios").AndWhere(dbx.HashExp{"device": device.GetString("id")}).OrderBy("radio ASC").All(&records)
	return records, err
}

func generateRadioConfigs(device *core.Record, app core.App) string {
	countryrecord, err := app.FindFirstRecordByData("settings", "name", "country")
	country := ""
	if err == nil {
		country = countryrecord.GetString("value")
	}
	output := ""
	records, err := getRadiosForDevice(device, app)
	if err != nil {
		fmt.Println("Error finding Radios", err)
		return ""
	}
	for _, record := range records {
		output += generateRadioConfig(app, record, country)

	}
	return output
}

func getVlan(wifi *core.Record, app core.App) string {
	errs := app.ExpandRecord(wifi, []string{"network"}, nil)
	if len(errs) > 0 {
		log.Println(errs)
		return "lan"
	}
	networkentry := wifi.ExpandedOne("network")
	if networkentry == nil {
		return "lan"
	}
	networkname := networkentry.GetString("name")
	if len(networkname) == 0 {
		return "lan"
	}
	return networkname
}

func generateOpenWispConfig() string {
	return fmt.Sprintf(`
config controller 'http'
        option enabled 'monitoring'
        option interval '30'
`)
}

func generateMonitoringConfig() string {
	return fmt.Sprintf(`
config monitoring 'monitoring'
        option interval '15'
`)
}

func JoinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func generateSshKeyConfig(app core.App) string {
	keys, err := app.FindAllRecords("ssh_keys")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	output := []string{}
	for _, key := range keys {
		output = append(output, strings.TrimSpace(key.GetString("key")))
	}
	return JoinLines(output)
}

func getTimeAdvertisementValues(vta string) (int, string) {
	vta_flag := 0
	if len(vta) > 0 && vta != "Disabled" {
		vta_flag = 2
	}
	return vta_flag, GetTzData(vta)
}

// uciQuote escapes a value for inclusion inside a single-quoted UCI option.
func uciQuote(value string) string {
	return strings.ReplaceAll(value, "'", `'\''`)
}

func generateWifiConfig(wifirecord WifiRecord, wifiid int, radio uint, app core.App, device *core.Record) (string, bool) {
	wifi := wifirecord.Record

	ssid := wifi.GetString("ssid")
	key := wifi.GetString("key")
	encryption := wifi.GetString("encryption")
	if len(encryption) == 0 {
		encryption = "psk2+ccmp"
	}

	ifaceName := fmt.Sprintf("wifi_%d_radio%d", wifiid, radio)
	vlanName := getVlan(wifi, app)

	steeringconfig, err := generateMacClientSteeringConfig(app, wifi, device)
	if err != nil {
		fmt.Println("Steering error:", err)
	}

	clientpskconfig := generateHostApdPskForWifi(app, wifi, ifaceName)
	vta_flag, vta_tz := getTimeAdvertisementValues(wifi.GetString("ieee80211v_time_advertisement"))

	rDeadLine := max(1000, wifi.GetInt("ieee80211r_reassoc_deadline"))
	dtim := maxInt(1, wifi.GetInt("dtim_period"))

	disabled := 0
	if !wifi.GetBool("enabled") {
		disabled = 1
	}

	return fmt.Sprintf(`
config wifi-iface '%s'
        option device 'radio%d'
        option network '%s'
        option disabled '%d'
        option mode 'ap'
        option ssid '%s'
        option encryption '%s'
        option key '%s'
        option hidden '%d'
        option isolate '%d'
        option ieee80211k '%d'
        option ieee80211r '%d'
        option reassociation_deadline '%d'
        option time_advertisement '%d'
        option time_zone '%s'
        option wnm_sleep_mode '%d'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '%d'
        option bss_transition '%d'
        option dtim_period '%d'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
%s%s`,
			ifaceName, radio, vlanName, disabled,
			uciQuote(ssid), encryption, uciQuote(key),
			wifi.GetInt("hidden"),
			wifi.GetInt("isolate_clients"),
			wifi.GetInt("ieee80211k"),
			wifi.GetInt("ieee80211r"),
			rDeadLine,
			vta_flag, vta_tz,
			wifi.GetInt("ieee80211v_wnm_sleep_mode"),
			wifi.GetInt("ieee80211v_proxy_arp"),
			wifi.GetInt("ieee80211v_bss_transition"),
			dtim,
			steeringconfig, clientpskconfig),
		len(clientpskconfig) > 0
}

func createConfigTar(files map[string]string) ([]byte, string, error) {
	var buf bytes.Buffer

	gzWriter := gzip.NewWriter(&buf)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	var filenames []string
	for filename := range files {
		filenames = append(filenames, filename)
	}

	sort.Strings(filenames)
	executables := []string{"etc/hotplug.d/openwisp/opensoho", "etc/hotplug.d/openwisp/opensoho-poe"}

	for _, filePath := range filenames {
		var mode int64 = 0644
		// TODO ugly hardcoded but works for now
		if slices.Contains(executables, filePath) {
			mode = 0755
		}
		fileBytes := []byte(files[filePath])

		header := &tar.Header{
			Name: filePath,
			Size: int64(len(fileBytes)),
			Mode: mode,
		}

		// Write header and file content to tar archive
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, "", err
		}
		if _, err := tarWriter.Write(fileBytes); err != nil {
			return nil, "", err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, "", err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, "", err
	}

	// Compress
	tarGzData := buf.Bytes()

	// Compute MD5 checksum
	md5Checksum := md5.Sum(tarGzData)
	md5Hex := hex.EncodeToString(md5Checksum[:])

	return tarGzData, md5Hex, nil
}

func generateHostnameConfig(device *core.Record) string {
	output := fmt.Sprintf(`
config system 'system'
        option hostname '%[1]s'
`, normalizeHostname(device.GetString("name")))

	return output
}

func generateLedConfigs(leds []*core.Record) string {
	output := ""
	for _, led := range leds {
		fmt.Println(led)
		output += generateLedConfig(led)
	}

	return output
}

func isWifiEnabledOnBand(wifi WifiRecord, band string, device *core.Record, app core.App) bool {
	record, err := app.FindFirstRecordByFilter(
		"wifi_aps",
		"device ~ {:device} && wifi ~ {:wifi}",
		dbx.Params{"device": device.Id},
		dbx.Params{"wifi": wifi.Record.Id},
	)
	if err != nil {
		return false
	}
	for _, b := range record.GetStringSlice("band") {
		if b == band {
			return true
		}
	}
	return false
}

func generateWifiConfigs(wifis []WifiRecord, numradios uint, app core.App, device *core.Record) (string, bool) {
	radios, _ := getRadiosForDevice(device, app)
	output := ""
	glob_has_vlan_config := false
	for i, wifi := range wifis {
		for j := range numradios {
			fmt.Println(wifi)

			var radio *core.Record
			for _, r := range radios {
				if r.GetInt("radio") == int(j) {
					radio = r
					break
				}
			}
			if radio != nil && !isWifiEnabledOnBand(wifi, frequencyToBand(radio.GetInt("frequency")), device, app) {
				continue
			}
			config_output, has_vlan_config := generateWifiConfig(wifi, i, j, app, device)
			output += config_output
			glob_has_vlan_config = glob_has_vlan_config || has_vlan_config
		}
	}
	return output, glob_has_vlan_config
}

func hexToPocketBaseID(hexStr string) (string, error) {
	// Convert hex string to bytes
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	// Ensure the byte array is exactly 16 bytes (PocketBase requirement)
	if len(bytes) != 16 {
		return "", fmt.Errorf("invalid length: keys must be 16 bytes")
	}
	bigInt := new(big.Int).SetBytes(bytes)

	// Convert to Base36
	return bigInt.Text(36)[0:15], nil
}
func getDeviceRecord(e *core.RequestEvent, key string) (*core.Record, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("Key not valid")
	}
	pbID, err := hexToPocketBaseID(key)
	if err != nil {
		return nil, fmt.Errorf("Key not hex")
	}
	record, err := e.App.FindRecordById("devices", pbID)
	if err != nil {
		return nil, fmt.Errorf("Device not known")
	}
	if !security.Equal(record.GetString("key"), key) {
		return nil, fmt.Errorf("Key not allowed")
	}
	// Successful authentication means the device is alive and online
	fmt.Println("UpdateLastSeen")
	if updateLastSeen(e, record) != nil {
		fmt.Println("could not update last seen")
		// Let's not make this an error for now
	}

	return record, nil
}

func getWifiRecord(app core.App, ssid string) (*core.Record, error) {
	return app.FindFirstRecordByData("wifi_ssids", "ssid", ssid)
}

// Returns true when:
// * All == true: One device needs to be offline (found in the set)
// * All == false: All devices need to be offline (found in the set)
func isUnHealthyQuorumReached(unhealthyfullset map[string]struct{}, subset []string, all bool) bool {
	quorum := 1
	count := 0
	if all == false {
		quorum = len(subset)
	}
	for _, device := range subset {
		if _, found := unhealthyfullset[device]; found == true {
			count = count + 1
			if count == quorum {
				return true
			}
		}
	}
	return false
}

func CIDRToMask(prefix int) (string, error) {
	if prefix < 0 {
		prefix = 0
	}
	if prefix > 32 {
		prefix = 32
	}

	mask := net.CIDRMask(prefix, 32)
	ip := net.IP(mask)
	return ip.String(), nil
}

func lastOctet(ipStr string) (byte, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, errors.New("invalid IP address")
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return 0, errors.New("not an IPv4 address")
	}

	return ip4[3], nil
}

func replaceLastOctet(ip_range string, device_ip string) (string, error) {
	ip := net.ParseIP(ip_range)
	if ip == nil {
		return "", errors.New("invalid IP address")
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return "", errors.New("not an IPv4 address")
	}

	newOctet, err := lastOctet(device_ip)
	if err != nil {
		return "", err
	}

	ip4[3] = newOctet
	return ip4.String(), nil
}

func IsFeatureApplied(record *core.Record, target string) bool {
	values := record.GetStringSlice("apply")
	for _, v := range values {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}

func generateDhcpConfigForDeviceVLAN(vlanname string, masksize int) string {
	// Should not occur, but extra safety
	if vlanname == "" || vlanname == "lan" || vlanname == "wan" {
		fmt.Println("Not a configurable vlan")
		return ""
	}
	if masksize > 24 {
		fmt.Println("Subnet too small for the DHCP config")
		return ""
	}
	return fmt.Sprintf(`
config dhcp '%[1]s'
        option interface '%[1]s'
        option start '100'
        option limit '150'
        option leasetime '12h'
`, vlanname)
}

func generateDhcpConfigForDevice(app core.App, device *core.Record, vlan *core.Record) string {
	vlangateway := vlan.GetString("gateway")
	if vlangateway == "" || vlangateway != device.Id {
		fmt.Println("Not a gateway")
		return ""
	}
	vlanname := vlan.GetString("name")
	vlancidr := vlan.GetString("cidr")
	if vlancidr == "" {
		return ""
	}
	_, ipNet, err := net.ParseCIDR(vlancidr)
	if err != nil {
		fmt.Println("Invalid CIDR")
		return ""
	}
	prefixsize, _ := ipNet.Mask.Size()
	return generateDhcpConfigForDeviceVLAN(vlanname, prefixsize)
}

func generateDhcpConfig(app core.App, device *core.Record) string {
	vlans, err := app.FindRecordsByFilter(
		"vlan",                           // collection
		"name != 'wan' && name != 'lan'", // filter
		"created",                        // sorting by creation time is the most stable
		0,                                // limit
		0,                                // offset
	)

	if err != nil {
		fmt.Println(err)
		return ""
	}

	output := ""
	for _, vlan := range vlans {
		output += generateDhcpConfigForDevice(app, device, vlan)
	}
	return output
}

type PortTaggingConfig struct {
	Port string
	Mode string
}

func getPortTagConfigForVlan(vlanId string, portConfig *core.Record) string {
	untagged := portConfig.GetString("untagged")
	if len(untagged) > 0 && untagged == vlanId {
		return "u*"
	}
	if portConfig.GetBool("trunk") == true {
		return "t"
	}
	tagged := portConfig.GetStringSlice("tagged")
	if slices.Contains(tagged, vlanId) {
		return "t"
	}
	return ""
}

// By default lan is untagged, all others are tagged
func generateFullTaggingMap(app core.App, ports []*core.Record, vlans []*core.Record) map[string][]PortTaggingConfig {
	for _, port := range ports {
		errs := app.ExpandRecord(port, []string{"config"}, nil)
		if len(errs) > 0 {
			fmt.Println(errs)
		}
	}
	fullmap := make(map[string][]PortTaggingConfig)
	// First populate the full matrix
	for _, vlan := range vlans {
		vlanname := vlan.GetString("name")
		defaultmode := "t"
		if vlanname == "lan" {
			defaultmode = "u*"
		}
		fullmap[vlan.Id] = generateTaggingMap(app, ports, defaultmode, vlan.Id)
	}
	return fullmap
}

func generateTaggingMap(app core.App, ports []*core.Record, defaultmode string, vlanConfigId string) []PortTaggingConfig {
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].GetString("name") < ports[j].GetString("name")
	})
	config := make([]PortTaggingConfig, len(ports))
	for i, port := range ports {
		mode := defaultmode
		if len(port.GetString("config")) > 0 {
			errs := app.ExpandRecord(port, []string{"config"}, nil)
			if len(errs) > 0 {
				fmt.Println(errs)
			}
			mode = getPortTagConfigForVlan(vlanConfigId, port.ExpandedOne("config"))
		}
		config[i] = PortTaggingConfig{Port: port.GetString("name"), Mode: mode}
	}
	return config
}

// Currently all of them are the same mode
func generatePortTaggingConfig(app core.App, portsconfig []PortTaggingConfig) string {
	// TODO add the full map here
	// Sort the static records
	sort.Slice(portsconfig, func(i, j int) bool {
		return portsconfig[i].Port < portsconfig[j].Port
	})

	portslist := ""
	for _, portconfig := range portsconfig {
		if len(portconfig.Mode) > 0 {
			portslist += fmt.Sprintf("        list ports '%s:%s'\n", portconfig.Port, portconfig.Mode)
		}
	}
	return portslist
}

func generateInterfaceVlanConfig(app core.App, device *core.Record, bridgeConfig *core.Record, vlanConfig *core.Record, taggingConfig []PortTaggingConfig) string {
	vlanname := vlanConfig.GetString("name")
	vlanid := vlanConfig.GetInt("number")
	gatewayid := vlanConfig.GetString("gateway")
	cidr := ""

	if len(gatewayid) > 0 && gatewayid == device.Id {
		cidr = vlanConfig.GetString("cidr")
	}

	return generateInterfaceVlanConfigInt(app, bridgeConfig, vlanname, vlanid, cidr, taggingConfig)
}

func isValidVlanNumber(vlanid int) bool {
	return !(vlanid < 1 || vlanid > 4094)
}

func generateInterfaceVlanConfigInt(app core.App, bridgeConfig *core.Record, vlanname string, vlanid int, cidr string, taggingConfig []PortTaggingConfig) string {
	// TODO we might already have the port config map available around here, since we want to pass it to the portTaggingConfig
	if (!isValidVlanNumber(vlanid)) || len(vlanname) == 0 {
		return ""
	}
	errs := app.ExpandRecord(bridgeConfig, []string{"ethernet"}, nil)
	if len(errs) > 0 {
		fmt.Printf("failed to expand: %v", errs)
		return ""
	}
	bridgename := bridgeConfig.GetString("name")
	if bridgename == "" {
		// TODO Maybe we actually want this to be an error
		bridgename = "br-lan"
	}
	//mode := "t"
	intfmode := "\n        option proto 'none'"

	// Avoid overwriting the default LAN configuration
	if len(cidr) > 0 && vlanname != "lan" {
		// Parse the IP address from the CIDR notation (before the /)
		cidrParts := strings.Split(cidr, "/")
		if len(cidrParts) == 2 {
			ipAddr := net.ParseIP(cidrParts[0])
			_, ipNet, err := net.ParseCIDR(cidr)
			if err == nil && ipAddr != nil {
				prefixsize, _ := ipNet.Mask.Size()
				prefixmask, _ := CIDRToMask(prefixsize)
				intfmode = fmt.Sprintf(`
        option proto 'static'
        option ipaddr '%[1]s'
        option netmask '%[2]s'`, ipAddr, prefixmask)
			} else {
				app.Logger().Warn("Invalid CIDR", "cidr", cidr)
			}
		} else {
			app.Logger().Warn("Invalid CIDR format", "cidr", cidr)
		}
	}

	if vlanname == "lan" {
		//mode = "u*"
		intfmode = ""
	}

	// TODO add ip address?
	return fmt.Sprintf(`
config interface '%[1]s'
        option device '%[5]s.%[2]d'%[4]s

config bridge-vlan 'bridge_vlan_%[2]d'
        option device '%[5]s'
        option vlan '%[2]d'
%[3]s`, vlanname, vlanid, generatePortTaggingConfig(app, taggingConfig), intfmode, bridgename)
}

// findBridgeForDevice picks the bridge OpenSOHO should attach VLAN config to.
// If the device has exactly one bridge, that one is used regardless of its
// name (covers routers without WiFi where the bridge may be called 'switch'
// or similar). Otherwise it falls back to the conventional 'br-lan'.
func findBridgeForDevice(app core.App, device *core.Record) (*core.Record, error) {
	bridges, err := app.FindRecordsByFilter("bridges",
		"device = {:device}", "", 0, 0,
		dbx.Params{"device": device.Id})
	if err != nil {
		return nil, err
	}
	if len(bridges) == 1 {
		return bridges[0], nil
	}
	return app.FindFirstRecordByFilter("bridges",
		"name = 'br-lan' && device = {:device}",
		dbx.Params{"device": device.Id})
}

func generateLldpdConfig(app core.App, device *core.Record) string {
	bridge, err := findBridgeForDevice(app, device)
	if err != nil {
		return ""
	}
	if errs := app.ExpandRecord(bridge, []string{"ethernet"}, nil); len(errs) > 0 {
		return ""
	}
	names := []string{}
	for _, port := range bridge.ExpandedAll("ethernet") {
		if name := port.GetString("name"); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	// Hardcode type 4 since we are openwrt routers
	output := "\nconfig lldpd 'config'\n        option lldp_class '4'\n"
	for _, name := range names {
		output += fmt.Sprintf("        list interface '%s'\n", name)
	}
	return output
}

func generateInterfacesConfig(app core.App, device *core.Record) string {
	if false == IsFeatureApplied(device, "vlan") {
		return ""
		/*return `
		config interface 'lan'
		        option device 'br-lan'
		`*/
	}
	bridgeConfig, err := findBridgeForDevice(app, device)
	if err != nil {
		fmt.Println("INTERFACES ERROR", err)
		return ""
	}
	errs := app.ExpandRecord(bridgeConfig, []string{"ethernet"}, nil)
	if len(errs) > 0 {
		fmt.Println("FAILED TO EXPAND:", errs)
		return ""
	}

	// Select all interfaces that are OpenSOHO maintained
	// TODO make LAN vlan ID configurable
	vlans, err := app.FindRecordsByFilter(
		"vlan",          // collection
		"name != 'wan'", // filter
		"created",       // sorting by creation time is the most stable
		0,               // limit
		0,               // offset
	)

	if err != nil {
		fmt.Println(err)
		return ""
	}

	fullmap := generateFullTaggingMap(app, bridgeConfig.ExpandedAll("ethernet"), vlans)
	fmt.Println(fullmap)

	fmt.Printf("LOOPING %v\n", vlans)
	output := ""
	for _, vlan := range vlans {
		output += generateInterfaceVlanConfig(app, device, bridgeConfig, vlan, fullmap[vlan.Id])
	}

	return output
}

// The SSID client steering is simple, it is basically a roaming SSID
func generateSsidClientSteeringConfig(app core.App, device *core.Record) ([]*core.Record, error) {
	// Get all the SSID client steering configurations
	//expanded_ssid := []string{}
	expanded_ssid := []*core.Record{}
	ssid_client_steering, err := app.FindRecordsByFilter("client_steering",
		`method ~{:method}`,
		"", // TODO add sort
		0, 0,
		map[string]any{
			"method": "ssid",
		})
	if err != nil {
		return expanded_ssid, err
	}

	// No Wifi Client steering configurations found
	if len(ssid_client_steering) == 0 {
		return expanded_ssid, nil
	}
	for _, entry := range ssid_client_steering {
		//
		// Check if we're in the whitelist or the whitelisting is currently disabled
		if slices.Contains(entry.GetStringSlice("whitelist"), device.Id) || clientHasWhiteListingDisabled(app, entry) {
			// Expand the SSID
			errs := app.ExpandRecord(entry, []string{"wifi"}, nil)
			if len(errs) > 0 {
				return []*core.Record{}, fmt.Errorf("failed to expand: %v", errs)
			}
			expanded_ssid = append(expanded_ssid, entry.ExpandedAll("wifi")...)
			//expanded_ssid = append(expanded_ssid, entry.GetStringSlice("wifi")...)
		}
	}
	return expanded_ssid, nil
}

func clientHasWhiteListingDisabled(app core.App, client *core.Record) bool {
	whitelisted_devices := client.GetStringSlice("whitelist")

	// These lines could be cached, but for the abstraction, we don't care
	unhealthy_devices, _ /*err*/ := app.FindRecordsByFilter("devices", `health_status!="healthy"`, "", 0, 0, map[string]any{})
	unhealthy_device_ids := make(map[string]struct{}, len(unhealthy_devices))
	for _, device := range unhealthy_devices {
		unhealthy_device_ids[device.Id] = struct{}{}
	}

	// Check whether sufficient whitelisted devices are online
	disable_whitelisting := false
	whitelisting_mode := client.GetString("enable")
	if whitelisting_mode != "Always" {
		disable_whitelisting = isUnHealthyQuorumReached(unhealthy_device_ids, whitelisted_devices, whitelisting_mode == "If all healthy")
	}
	return disable_whitelisting
}

func generateMacClientSteeringConfig(app core.App, wifi *core.Record, device *core.Record) (string, error) {
	expandedclients, err := generateClientSteeringConfigInt(app, wifi, device, "mac blacklist")
	if err != nil {
		return "", err
	}
	if len(expandedclients) > 0 {
		output := "        option macfilter 'deny'\n"
		for _, client := range expandedclients {
			output += fmt.Sprintf("        list maclist '%[1]s'\n", client.GetString("mac_address"))

		}
		return output, nil
	}
	return "", err
}
func generateClientSteeringConfigInt(app core.App, wifi *core.Record, device *core.Record, method string) ([]*core.Record, error) {
	// Select all with the current wifi
	// Exclude if in the whitelist
	expandedclients := []*core.Record{}
	client_steering_for_wifi, err := app.FindRecordsByFilter("client_steering",
		`wifi={:wifi} && whitelist!~{:device} && method ~{:method}`,
		"", // TODO add sort
		0, 0,
		map[string]any{ // params for safe interpolation
			"device": device.Id,
			"wifi":   wifi.Id,
			"method": method,
		})
	if err != nil {
		return expandedclients, err
	}
	// We're on the devices whitelist, so don't block it
	if len(client_steering_for_wifi) == 0 {
		return expandedclients, nil
	}

	for _, client := range client_steering_for_wifi {
		// Check whether sufficient whitelisted devices are online
		if clientHasWhiteListingDisabled(app, client) {
			fmt.Println("Disabling whitelisting")
			continue
		}

		// We need the MAC address of the steered client
		errs := app.ExpandRecord(client, []string{"", "client"}, nil)
		if len(errs) > 0 {
			return expandedclients, fmt.Errorf("failed to expand: %v", errs)
		}

		expandedclients = append(expandedclients, client.ExpandedOne("client"))
	}
	return expandedclients, nil
}

func generateWifiRecordList(app core.App, device *core.Record) ([]*core.Record, error) {
	// Find all wifi_aps records for this device
	wifiApRecords, err := app.FindRecordsByFilter(
		"wifi_aps",
		"device ~ {:device}",
		"", 0, 0,
		dbx.Params{"device": device.Id},
	)
	if err != nil {
		return []*core.Record{}, err
	}

	// Collect unique wifi SSID IDs across all matching wifi_aps records
	wifiIDSet := map[string]struct{}{}
	for _, apRecord := range wifiApRecords {
		for _, wifiID := range apRecord.GetStringSlice("wifi") {
			wifiIDSet[wifiID] = struct{}{}
		}
	}
	wifiIDs := make([]string, 0, len(wifiIDSet))
	for id := range wifiIDSet {
		wifiIDs = append(wifiIDs, id)
	}

	// Get the static Wifi configurations
	wifirecords, err := app.FindRecordsByIds("wifi_ssids", wifiIDs)
	if err != nil {
		return []*core.Record{}, err
	}

	// Sort the static records
	sort.Slice(wifirecords, func(i, j int) bool {
		return wifirecords[i].GetDateTime("created").Before(wifirecords[j].GetDateTime("created"))
	})

	// Add the steering records
	wifisteeringrecords, err := generateSsidClientSteeringConfig(app, device)

	if err != nil {
		return []*core.Record{}, err
	}

	// Sort the steering separately
	sort.Slice(wifisteeringrecords, func(i, j int) bool {
		return wifisteeringrecords[i].GetDateTime("created").Before(wifisteeringrecords[j].GetDateTime("created"))
	})

	// Add the dynamic wifi configuration after the static configuration
	return append(wifirecords, wifisteeringrecords...), err
}

func generateUsteerConfig(device *core.Record, app core.App) string {
	wifiApRecords, err := app.FindRecordsByFilter(
		"wifi_aps",
		"device ~ {:device}",
		"", 0, 0,
		dbx.Params{"device": device.Id},
	)
	if err != nil {
		return ""
	}

	wifiIDSet := map[string]struct{}{}
	for _, ap := range wifiApRecords {
		for _, id := range ap.GetStringSlice("wifi") {
			wifiIDSet[id] = struct{}{}
		}
	}
	wifiIDs := make([]string, 0, len(wifiIDSet))
	for id := range wifiIDSet {
		wifiIDs = append(wifiIDs, id)
	}

	wifirecords, err := app.FindRecordsByIds("wifi_ssids", wifiIDs)
	if err != nil {
		return ""
	}
	sort.Slice(wifirecords, func(i, j int) bool {
		return wifirecords[i].GetDateTime("created").Before(wifirecords[j].GetDateTime("created"))
	})

	ssidSet := map[string]struct{}{}
	for _, wifi := range wifirecords {
		if !wifi.GetBool("usteer") {
			continue
		}
		if name := wifi.GetString("ssid"); name != "" {
			ssidSet[name] = struct{}{}
		}
	}

	if len(ssidSet) == 0 {
		return ""
	}

	ssids := make([]string, 0, len(ssidSet))
	for name := range ssidSet {
		ssids = append(ssids, name)
	}
	sort.Strings(ssids)

	output := `
config usteer 'usteer'
        option enabled '1'
        option network 'lan'
        option debug_level '2'
        option ipv6 '0'
        option local_mode '0'
        option syslog '1'
        option roam_trigger '-70'
        option min_signal '-78'
        option roam_delta '10'
        option probe_steering '1'
        option deny_assoc '1'
        option band_steering '1'

`
	for _, name := range ssids {
		output += fmt.Sprintf("        list ssid_list '%s'\n", name)
	}
	return output
}

func generateDeviceConfig(app core.App, record *core.Record) ([]byte, string, error) {
	configfiles := map[string]string{}
	leds := record.GetStringSlice("leds")
	fmt.Println(leds)
	ledrecords, err := app.FindRecordsByIds("leds", leds)
	if err != nil {
		return nil, "", err
	}
	systemconfig := generateHostnameConfig(record)
	systemconfig += generateLedConfigs(ledrecords)
	if len(systemconfig) > 0 {
		configfiles["etc/config/system"] = systemconfig
	}
	numradios := uint(record.GetInt("numradios"))
	fmt.Printf("numradios %d\n", numradios)
	{
		wifirecords, err := generateWifiRecordList(app, record)
		if err != nil {
			return nil, "", err
		}
		// Add the steered Wifi configurations
		//wifisteeringrecords, err := generateSsidClientSteeringConfig(app, record)
		//fmt.Println("SSID STEERING", err)
		//fmt.Println(wifisteeringrecords)
		////if err != nil {
		////	return nil, "", err
		////}
		////wifirecords = append(wifirecords, wifisteeringrecords...)

		//sort.Slice(wifirecords, func(i, j int) bool {
		//	return wifirecords[i].GetDateTime("created").Before(wifirecords[j].GetDateTime("created"))
		//})

		wificonfigstructs := []WifiRecord{}
		for _, wifirecord := range wifirecords {
			wificonfigstruct := WifiRecord{wifirecord}
			wificonfigstructs = append(wificonfigstructs, wificonfigstruct)
		}

		// HostApdPskConfig needs to be called before the generateWifiConfigs
		//wificonfigsstructs /*deviceHasPskConfig*/, _ := generateHostApdPskConfigs(app, wifirecords, &configfiles)
		wificonfigs, has_vlan_config := generateWifiConfigs(wificonfigstructs, numradios, app, record)
		fmt.Println(wificonfigs)
		if len(wificonfigs) > 0 {
			configfiles["etc/config/wireless"] = wificonfigs
		}
		if has_vlan_config {
			configfiles["etc/config/wireless"] += generateHostApdVlanConfig(app)
		}
	}

	{
		radioconfigs := generateRadioConfigs(record, app)
		fmt.Println(radioconfigs)
		if len(radioconfigs) > 0 {
			if existingconfig, ok := configfiles["etc/config/wireless"]; ok {
				configfiles["etc/config/wireless"] = existingconfig + radioconfigs
			} else {
				configfiles["etc/config/wireless"] = radioconfigs
			}
		}
	}
	{
		// Currently the monitoring config is static
		configfiles["etc/config/openwisp-monitoring"] = generateMonitoringConfig()
		configfiles["etc/config/openwisp"] = generateOpenWispConfig()
		configfiles["etc/hotplug.d/openwisp/opensoho"] = dumpRadiosScript
		configfiles["etc/hotplug.d/openwisp/opensoho-poe"] = dumpPoeScript
	}
	{
		sshkeyconfigs := generateSshKeyConfig(app)
		fmt.Println(sshkeyconfigs)
		if len(sshkeyconfigs) > 0 {
			configfiles["etc/dropbear/authorized_keys"] = sshkeyconfigs
		}
	}
	{
		interfacesconfigs := generateInterfacesConfig(app, record)
		fmt.Println(interfacesconfigs)
		if len(interfacesconfigs) > 0 {
			configfiles["etc/config/network"] = interfacesconfigs
		}
	}

	{
		dhcpconfigs := generateDhcpConfig(app, record)
		fmt.Println(dhcpconfigs)
		if len(dhcpconfigs) > 0 {
			configfiles["etc/config/dhcp"] = dhcpconfigs
		}
	}
	{
		usteerconfig := generateUsteerConfig(record, app)
		if len(usteerconfig) > 0 {
			configfiles["etc/config/usteer"] = usteerconfig
		}
	}
	{
		lldpdconfig := generateLldpdConfig(app, record)
		if len(lldpdconfig) > 0 {
			configfiles["etc/config/lldpd"] = lldpdconfig
		}
	}

	if keepList := generateKeepList(configfiles); keepList != "" {
		configfiles["lib/upgrade/keep.d/opensoho"] = keepList
	}

	blob, checksum, err := createConfigTar(configfiles)

	if err != nil {
	}
	return blob, checksum, err
}

// generateKeepList builds the /lib/upgrade/keep.d/opensoho file,
// files under etc/config/ already preserved across sysupgrades
func generateKeepList(configfiles map[string]string) string {
	var paths []string
	for path := range configfiles {
		if strings.HasPrefix(path, "etc/config/") {
			continue
		}
		paths = append(paths, "/"+path)
	}
	sort.Strings(paths)

	output := ""
	for _, path := range paths {
		output += path + "\n"
	}
	return output
}

func findFirstOrNewByFilter(app core.App,
	collection *core.Collection,
	filter string,
	params ...dbx.Params,
) *core.Record {
	record, err := app.FindFirstRecordByFilter(collection, filter, params...)
	if err != nil {
		record = core.NewRecord(collection)
	}
	return record
}

func findFirstOrNew(app core.App, collection *core.Collection, column string, value string) *core.Record {
	record, err := app.FindFirstRecordByData(collection, column, value)
	if err != nil {
		record = core.NewRecord(collection)
	}
	return record
}

func handleBridgeMonitoring(app core.App, iface Interface, device *core.Record, bridgescollection *core.Collection, interfacescollection *core.Collection, ethernetcollection *core.Collection) error {
	record := findFirstOrNewByFilter(app, bridgescollection, "device = {:device} && name = {:name}", dbx.Params{"device": device.Id}, dbx.Params{"name": iface.Name})
	record.Set("name", iface.Name)
	record.Set("device", device.Id)
	if iface.Statistics != nil {
		record.Set("tx_bytes", iface.Statistics.TxBytes)
		record.Set("rx_bytes", iface.Statistics.RxBytes)
	}
	// This is a bit tricky, since we want to split the wired and wireless members
	err := (error)(nil)
	ethernetlist := []string{}
	wifilist := []string{}
	for _, membername := range iface.BridgeMembers {
		eth_record, eth_err := app.FindFirstRecordByFilter(ethernetcollection, "device = {:device} && name = {:name}", dbx.Params{"device": device.Id}, dbx.Params{"name": membername})
		wifi_record, wifi_err := app.FindFirstRecordByFilter(interfacescollection, "device = {:device} && interface = {:name}", dbx.Params{"device": device.Id}, dbx.Params{"name": membername})

		if eth_err != nil && !errors.Is(eth_err, sql.ErrNoRows) {
			err = eth_err
			fmt.Println("Ethernet:", eth_err, membername)
		}
		if wifi_err != nil && !errors.Is(wifi_err, sql.ErrNoRows) {
			err = wifi_err
			fmt.Println("Wifi", wifi_err, membername)
		}
		if eth_record != nil && wifi_record != nil {
			err = fmt.Errorf("Ambigious bridge member %s", membername)
			fmt.Println(err)
		}
		if wifi_record == nil && eth_record == nil {
			err_ignored := fmt.Errorf("Unknown bridge member %s", membername)
			fmt.Println(err_ignored)
		}
		if wifi_record != nil {
			fmt.Println("ADDING WIFI", wifi_record)
			wifilist = append(wifilist, wifi_record.Id)
		}
		if eth_record != nil {
			fmt.Println("ADDING ETH", eth_record)
			ethernetlist = append(ethernetlist, eth_record.Id)
		}
	}
	if err != nil {
		// Delete all interface data since it is inconsitent
		ethernetlist = []string{}
		wifilist = []string{}
	}
	fmt.Println("SAVING")
	record.Set("ethernet", ethernetlist)
	record.Set("wifi", wifilist)
	x := app.Save(record)
	fmt.Println("SAVED", x)
	return err
}

func handleEthernetMonitoring(app core.App, iface Interface, device *core.Record, ethernetcollection *core.Collection) {
	record := findFirstOrNewByFilter(app, ethernetcollection, "device = {:device} && name = {:name}", dbx.Params{"device": device.Id}, dbx.Params{"name": iface.Name})
	record.Set("name", iface.Name)
	record.Set("device", device.Id)
	record.Set("speed", iface.Speed)
	if iface.Statistics != nil {
		record.Set("tx_bytes", iface.Statistics.TxBytes)
		record.Set("rx_bytes", iface.Statistics.RxBytes)
	}
	app.Save(record)
}

func updateInterface(app core.App, iface Interface, deviceId string, interfaceCollection *core.Collection) error {
	band := frequencyToBand(iface.Wireless.Frequency)
	if band == "unknown" { // Todo add another frequencyToBand function that returns an error
		err := fmt.Errorf("Uknown Band %d", iface.Wireless.Frequency)
		fmt.Println(err)
		return err
	}

	// Todo verify that only one record exists
	ifacerecords, err := app.FindRecordsByFilter(
		"interfaces",
		"device = {:device} && interface = {:interface}", "", 0, 0,
		dbx.Params{"device": deviceId, "interface": iface.Name})
	if err != nil {
		fmt.Println(err)
		return err
	}
	wifirecord, err := getWifiRecord(app, iface.Wireless.SSID)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if len(ifacerecords) == 0 {
		fmt.Println("NEW IFACE")
		record := core.NewRecord(interfaceCollection)
		record.Set("device", deviceId)
		record.Set("wifi", wifirecord.Id)
		record.Set("band", band)
		record.Set("mac_address", iface.MAC)
		record.Set("interface", iface.Name)
		app.Save(record)
	} else {
		record := ifacerecords[0]
		if wifirecord.Id != record.Get("wifi") ||
			band != record.Get("band") ||
			record.Get("mac_address") != iface.MAC {
			record.Set("device", deviceId)
			record.Set("wifi", wifirecord.Id)
			record.Set("band", band)
			record.Set("mac_address", iface.MAC)
			record.Set("interface", iface.Name)
			app.Save(record)
			fmt.Println(ifacerecords)
		}
	}
	return nil
}

func handleMonitoring(e *core.RequestEvent, app core.App, device *core.Record, collection *core.Collection) (error, map[int]Radio) {
	e.Response.Header().Set("X-Openwisp-Controller", "true")
	time := e.Request.URL.Query().Get("time")
	// The openwisp agent marks the up-to-date info with current=true
	// Only live data is published to Home Assistant does not support backfill over MQTT
	current := e.Request.URL.Query().Get("current") == "true"
	radios := make(map[int]Radio)
	var payload MonitoringData
	if e.Request.Header.Get("Content-Length") == "0" {
		app.Logger().Info("Ignored empty monitoring request 1", "userIP", e.RealIP())
		return e.Blob(200, "text/plain", []byte("")), radios
	}
	// Both DeviceMonitoring and OpenSoho payloads arrive at this endpoint, so
	// read the body once and dispatch on the "type" discriminator.
	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		fmt.Println(err)
		return e.BadRequestError("Failed to read body", err), radios
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		fmt.Println(err)
		return e.BadRequestError("Failed to parse json", err), radios
	}
	if envelope.Type == "OpenSoho" {
		// Radio dump produced by scripts/dump-radios.sh.
		var osd OpenSohoData
		if err := json.Unmarshal(body, &osd); err != nil {
			fmt.Println(err)
			return e.BadRequestError("Failed to parse OpenSoho json", err), radios
		}
		handleOpenSohoMonitoring(app, device, osd, current)
		return e.Blob(200, "text/plain", []byte("")), radios
	}
	if envelope.Type != "DeviceMonitoring" {
		errormsg := fmt.Sprintf(`Invalid type '%s' in JSON`, envelope.Type)
		fmt.Println(errormsg)
		return e.BadRequestError(errormsg, ""), radios
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Println(err)
		return e.BadRequestError("Failed to parse json", err), radios
	}
	interfacecollection, _ := app.FindCollectionByNameOrId("interfaces")
	wificollection, _ := app.FindCollectionByNameOrId("wifi_ssids")

	ethernetcollection, _ := app.FindCollectionByNameOrId("ethernet")
	bridgescollection, _ := app.FindCollectionByNameOrId("bridges")

	for _, iface := range payload.Interfaces {
		if iface.Type == "wireless" && iface.Wireless != nil {
			if interfacecollection != nil && wificollection != nil {
				err := updateInterface(app, iface, device.Id, interfacecollection)
				if err != nil {
					fmt.Println("Failed to add interface")
				}
			}
			radionum, err := extractRadioNumber(iface.Name)
			if err != nil {
				fmt.Printf("Found an unknown phy pattern '%s', please report a github issue\n", iface.Name)
			} else {
				radios[radionum] = Radio{Frequency: iface.Wireless.Frequency, Channel: iface.Wireless.Channel, HTmode: iface.Wireless.HTmode, TxPower: iface.Wireless.TxPower}
			}

			for _, client := range iface.Wireless.Clients {
				if client.Assoc {
					fmt.Printf("Associated client on %s: %s %s\n", iface.Name, client.MAC, device.GetString("id"))
					cliententry, err := app.FindFirstRecordByData(collection, "mac_address", client.MAC)
					if err != nil {
						cliententry = core.NewRecord(collection)
					}
					cliententry.Set("mac_address", client.MAC)
					cliententry.Set("vendor", lookupVendor(client.MAC))
					cliententry.Set("connected_to_hostname", iface.Name)
					cliententry.Set("signal", client.Signal)
					cliententry.Set("ssid", iface.Wireless.SSID)
					cliententry.Set("frequency", iface.Wireless.Frequency)
					cliententry.Set("channel", iface.Wireless.Channel)
					cliententry.Set("band", frequencyToBand(iface.Wireless.Frequency))
					cliententry.Set("tx_rate", client.Rate.Tx)
					cliententry.Set("rx_rate", client.Rate.Rx)
					cliententry.Set("tx_bytes", client.Bytes.Tx)
					cliententry.Set("rx_bytes", client.Bytes.Rx)
					cliententry.Set("device", device.GetString("id"))
					err = app.Save(cliententry)
					if err != nil {
						return e.InternalServerError("Could not store entry", err), radios
					}
				}
			}
		}
		if iface.Type == "ethernet" {
			handleEthernetMonitoring(app, iface, device, ethernetcollection)
		}
	}

	// Add the bridges after the other interfaces to ensure all new data is available
	for _, iface := range payload.Interfaces {
		if iface.Type == "bridge" {
			handleBridgeMonitoring(app, iface, device, bridgescollection, interfacecollection, ethernetcollection)
		}
	}

	storeDHCPLeases(app, payload.DHCPLeases, types.NowDateTime())

	//current := e.Request.URL.Query().Get("current")
	fmt.Println(payload.Type, "@", time)
	return e.Blob(200, "text/plain", []byte("")), radios
}

func storeDHCPLeases(app core.App, leaseslist []DHCPLease, expirytime types.DateTime) {
	// TODO needs to run in a transaction?
	collection, _ := app.FindCollectionByNameOrId("dhcp_leases")
	//var leasesmap map[string]DHCPLease
	for _, lease := range leaseslist {
		fmt.Println("DHCPLEASE", lease.MACAddress, lease.ClientID, lease.IPAddress, lease.Hostname, lease.Expiry)
		record := findFirstOrNew(app, collection, "mac_address", lease.MACAddress)
		record.Set("mac_address", lease.MACAddress)
		record.Set("ip_address", lease.IPAddress)
		if lease.Hostname != "*" {
			record.Set("hostname", lease.Hostname)
		} else {
			record.Set("hostname", nil)
		}
		record.Set("expiry", lease.Expiry)
		err := app.Save(record)
		if err != nil {
			fmt.Println("Could not store DHCP LEASE", err)
		}
		//leasesmap[lease.MACAddress] = lease
	}

	_, err := app.DB().Delete("dhcp_leases", dbx.NewExp("expiry < {:expiry}", dbx.Params{"expiry": expirytime.String()})).Execute()
	if err != nil {
		fmt.Println("Failed to clean expired DHCP Leases", err)
	}

	//return leasesmap
}

func handleDeviceInfoUpdate(e *core.RequestEvent) error {
	e.Response.Header().Set("X-Openwisp-Controller", "true")
	data := struct {
		// unexported to prevent binding
		somethingPrivate string

		Key    string `form:"key"`
		Model  string `form:"model"`
		Os     string `form:"os"`
		System string `form:"system"`
	}{}
	if err := e.BindBody(&data); err != nil {
		return e.BadRequestError("Missing fields", err)
	}
	record, err := getDeviceRecord(e, data.Key)
	if err != nil {
		fmt.Println(err)
		return e.ForbiddenError("Not allowed", err)
	}
	record.Set("model", data.Model)
	record.Set("os", data.Os)
	record.Set("system", data.System)
	err = e.App.Save(record)
	if err != nil {
		return e.InternalServerError("Info update failed", err)
	}
	response := fmt.Sprintf("update-info: success\n")

	return e.Blob(200, "text/plain", []byte(response))
}

func handleDeviceStatusUpdate(e *core.RequestEvent) error {
	e.Response.Header().Set("X-Openwisp-Controller", "true")
	data := struct {
		// unexported to prevent binding
		somethingPrivate string

		Status      string `form:"status"`
		Key         string `form:"key"`
		ErrorReason string `form:"error_reason"`
	}{}
	if err := e.BindBody(&data); err != nil {
		return e.BadRequestError("Missing fields", err)
	}
	record, err := getDeviceRecord(e, data.Key)
	if err != nil {
		fmt.Println(err)
		return e.ForbiddenError("Not allowed", err)
	}
	record.Set("config_status", data.Status)
	record.Set("error_reason", data.ErrorReason)
	err = e.App.Save(record)
	if err != nil {
		return e.InternalServerError("Status update failed", err)
	}
	response := fmt.Sprintf("report-result: success\ncurrent-status: %s\n", data.Status)

	return e.Blob(200, "text/plain", []byte(response))
}

func handleDeviceRegistration(e *core.RequestEvent, shared_secret string, enableNewDevices bool) error {
	e.Response.Header().Set("X-Openwisp-Controller", "true")
	data := struct {
		// unexported to prevent binding
		somethingPrivate string

		Backend    string `form:"backend"`
		Key        string `form:"key"`
		Secret     string `form:"secret"`
		Name       string `form:"name"`
		HardwareId string `form:"hardware_id"`
		MacAddress string `form:"mac_address"`
		Tags       string `form:"tags"`
		Model      string `form:"model"`
		Os         string `form:"os"`
		System     string `form:"system"`
	}{}
	if err := e.BindBody(&data); err != nil {
		fmt.Println(err)
		return e.BadRequestError("Missing fields", err)
	}
	if data.Secret != shared_secret {
		return e.ForbiddenError("Registration failed!", "unrecognized secret")
	}
	pbID, err := hexToPocketBaseID(data.Key)
	if err != nil {
		return e.BadRequestError("Bad key", err)
	}
	record, err := e.App.FindRecordById("devices", pbID)

	isNew := 1
	var device_uuid string = uuid.New().String()

	if err == nil {
		isNew = 0
		fmt.Println("Hello back")
		device_uuid = record.GetString("uuid")
	} else {
		// Register new device
		if data.Backend != "netjsonconfig.OpenWrt" {
			return e.BadRequestError("Registration failed!", "wrong backend")
		}
		fmt.Println("Hello")
		collection, err := e.App.FindCollectionByNameOrId("devices")

		if err != nil {
			return e.BadRequestError("Registration failed!", err)
		}

		record := core.NewRecord(collection)
		record.Set("id", pbID)
		record.Set("backend", data.Backend)
		record.Set("key", data.Key)
		record.Set("name", data.Name)
		record.Set("hardware_id", data.HardwareId)
		record.Set("mac_address", data.MacAddress)
		record.Set("uuid", device_uuid)
		record.Set("model", data.Model)
		record.Set("os", data.Os)
		record.Set("system", data.System)
		record.Set("ip_address", e.RealIP())
		record.Set("health_status", "unknown")
		record.Set("config_status", "applied")
		record.Set("enabled", enableNewDevices)
		err = e.App.Save(record)
		if err != nil {
			return e.BadRequestError("Registration failed!", err)
		}
	}
	response := fmt.Sprintf(`registration-result: %s
uuid: %s
key: %s
hostname: %s
is-new: %d
`, "success", device_uuid, data.Key, data.Name, isNew)

	return e.Blob(201, "text/plain", []byte(response))
}

// keyedMutex serializes work per key (here: per device id). Concurrent
// monitoring posts for the same device queue; different devices run in parallel.
// Use a channel, since Mutexes can't have a timeout
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]chan struct{}
}

// Lock acquires the per-key lock, blocking until it is free or ctx is done.
// Returns an unlock func and true on success; nil and false if ctx expired
// (or the client disconnected) before the lock was acquired.
func (k *keyedMutex) Lock(ctx context.Context, key string) (func(), bool) {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = map[string]chan struct{}{}
	}
	ch, ok := k.locks[key]
	if !ok {
		ch = make(chan struct{}, 1)
		k.locks[key] = ch
	}
	k.mu.Unlock()

	select {
	case ch <- struct{}{}:
		return func() { <-ch }, true
	case <-ctx.Done():
		return nil, false
	}
}

func bindAppHooks(app core.App, shared_secret string, enableNewDevices bool) {
	var deviceLocks keyedMutex
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/controller/register/", func(e *core.RequestEvent) error {
			return handleDeviceRegistration(e, shared_secret, enableNewDevices)
		})
		se.Router.POST("/controller/report-status/{device_uuid}/", handleDeviceStatusUpdate)
		se.Router.POST("/controller/update-info/{device_uuid}/", handleDeviceInfoUpdate)

		se.Router.GET("/controller/checksum/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			key := e.Request.URL.Query().Get("key")
			record, err := getDeviceRecord(e, key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}

			data, checksum, err := generateDeviceConfig(app, record)
			if err != nil {
				return e.InternalServerError("Internal error", err)
			}
			saveDeviceConfig(e.App, record, data, checksum)

			return e.Blob(200, "text/plain", []byte(checksum))
		})

		se.Router.GET("/controller/download-config/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			key := e.Request.URL.Query().Get("key")
			record, err := getDeviceRecord(e, key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}

			if record.GetBool("enabled") == false {
				fmt.Println("DISABLED", record.GetString("name"))
				return e.NotFoundError("Disabled", nil)
			}

			fmt.Println("OK", record.GetString("name"), record.GetBool("enabled"))
			response, _, err := generateDeviceConfig(app, record)
			if err != nil {
				return e.InternalServerError("Internal error", err)
			}

			return e.Blob(200, "application/octet-stream", []byte(response))
		})

		se.Router.POST("/api/v1/monitoring/device/", func(e *core.RequestEvent) error {

			key := e.Request.URL.Query().Get("key")
			device, err := getDeviceRecord(e, key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}

			ctx, cancel := context.WithTimeout(e.Request.Context(), 10*time.Second)
			defer cancel()
			unlock, ok := deviceLocks.Lock(ctx, device.Id)
			if !ok {
				e.Response.Header().Set("Retry-After", "10")
				return e.TooManyRequestsError("Device monitoring busy, retry later", nil)
			}
			defer unlock()

			collection, err := app.FindCollectionByNameOrId("clients")
			if err != nil {
				return e.InternalServerError("Could not find collection", err)
			}

			err, radios := handleMonitoring(e, app, device, collection)
			updateRadios(device, app, radios)
			return err
		})

		// Regenerate all configs to have a correct "modified" flag
		regenerateAllDeviceConfigs(se.App)

		// Connect to the MQTT broker (if configured) so PoE telemetry can be
		// published to Home Assistant. Failures are logged but non-fatal.
		if err := mqtt.Configure(loadMQTTConfig(se.App)); err != nil {
			app.Logger().Warn("MQTT connect failed", "error", err)
		}

		return se.Next()
	})

	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		mqtt.Close()
		return e.Next()
	})
}

func saveDeviceConfig(app core.App, record *core.Record, data []byte, checksum string) error {
	filename := checksum + ".tar.gz"
	//os.WriteFile(record.GetString("name")+"_"+checksum+".tar.gz", data, 0644)
	if strings.SplitN(record.GetString("config"), "_", 2)[0] != checksum {
		f, err := filesystem.NewFileFromBytes(data, filename)
		if err != nil {
			return err
		}

		fmt.Println(filename)
		record.Set("config", f)
		record.Set("config_status", "modified")
		err = app.Save(record)
		if err != nil {
			return err
		}
		fmt.Println("SAVED NEW CONFIG TO RECORD", record.GetString("config"), filename)
	} else {
		fmt.Println("record config up to date")
	}
	return nil
}

func GetDataDirPath() (string, error) {
	if ko_data_path := os.Getenv("KO_DATA_PATH"); ko_data_path == "/var/run/ko" {
		return "/ko-app", nil
	}
	cwdPath, err := os.Getwd()
	if err != nil {
		return cwdPath, err
	}
	return cwdPath, nil
}

func main() {
	shared_secret := os.Getenv("OPENSOHO_SHARED_SECRET")
	if shared_secret == "" {
		fmt.Println("OPENSOHO_SHARED_SECRET environment variable not set!")
		return
	}
	os.Unsetenv("OPENSOHO_SHARED_SECRET")
	var c pocketbase.Config
	cwdPath, err := GetDataDirPath()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	c.DefaultDataDir = filepath.Join(cwdPath, "pb_data")

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: filepath.Join(cwdPath, "pb_data")})

	enableNewDevices := false
	bindAppHooks(app, shared_secret, enableNewDevices)

	// Upstream commands
	// ---------------------------------------------------------------
	// Optional plugin flags:
	// ---------------------------------------------------------------

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var doFileExtraction bool
	app.RootCmd.PersistentFlags().BoolVar(
		&doFileExtraction,
		"doEmbeddedFileExtraction",
		true,
		"Extracts the embedded migrations and frontend files",
	)

	app.RootCmd.PersistentFlags().BoolVar(
		&enableNewDevices,
		"enableNewDevices",
		true,
		"Enable newly discovered devices, set to false for \"monitoring mode\"",
	)

	app.RootCmd.ParseFlags(os.Args[1:])

	app.OnSettingsListRequest().BindFunc(func(e *core.SettingsListRequestEvent) error {
		e.Settings.Meta.AppName = "OpenSOHO"
		e.Settings.Meta.HideControls = !app.IsDeveloperMode()

		return e.Next()
	})

	// ---------------------------------------------------------------
	// Plugins and hooks:
	// ---------------------------------------------------------------

	if doFileExtraction {
		if err := copyEmbedDirToDisk(embeddedFiles, cwdPath); err != nil {
			log.Fatal(err)
		}
	}

	// load jsvm (pb_hooks and pb_migrations)
	jsvm.MustRegister(app, jsvm.Config{
		MigrationsDir: filepath.Join(cwdPath, "pb_migrations"),
		HooksDir:      "",
		HooksWatch:    true,
		HooksPoolSize: 15,
	})

	// migrate command (with js templates)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  automigrate,
		Dir:          filepath.Join(cwdPath, "pb_migrations"),
	})

	// GitHub selfupdate
	//ghupdate.MustRegister(app, app.RootCmd, ghupdate.Config{})

	// static route to serves files from the provided public dir
	// (if publicDir exists and the route path is not already defined)
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", func(e *core.RequestEvent) error {
					e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
					return e.Redirect(307, "/_/")
				})
			}

			return e.Next()
		},
		Priority: 999, // execute as latest as possible to allow users to provide their own route
	})
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			e.Router.GET("/_/{path...}", apis.Static(ui.DistDirFS, false)).
				BindFunc(func(e *core.RequestEvent) error {
					// ignore root path
					if e.Request.PathValue(apis.StaticWildcardParam) != "" {
						e.Response.Header().Set("Cache-Control", "max-age=1209600, stale-while-revalidate=86400")
					}

					// add a default CSP
					if e.Response.Header().Get("Content-Security-Policy") == "" {
						e.Response.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' http://127.0.0.1:* https://tile.openstreetmap.org data: blob:; connect-src 'self' http://127.0.0.1:* https://nominatim.openstreetmap.org; script-src 'self' 'sha256-GRUzBA7PzKYug7pqxv5rJaec5bwDCw1Vo6/IXwvD3Tc='")
					}

					return e.Next()
				}).
				Bind(apis.Gzip())
			e.Router.GET("/_/images/favicon/apple-touch-icon.png", func(e *core.RequestEvent) error {

				bytes, _ := internalFiles.ReadFile("favicon.png")
				return e.Blob(200, "image/png", bytes)
			})
			e.Router.GET("/_/images/favicon/favicon.png", func(e *core.RequestEvent) error {

				bytes, _ := internalFiles.ReadFile("favicon.png")
				return e.Blob(200, "image/png", bytes)
			})
			e.Router.GET("/_/images/logo.svg", func(e *core.RequestEvent) error {

				bytes, _ := internalFiles.ReadFile("logo.svg")
				return e.Blob(200, "image/svg+xml", bytes)
			})
			e.Router.GET("/_/user.css", func(e *core.RequestEvent) error {
				e.Response.Header().Set("Content-Type", "text/css; charset=utf-8")
				return e.String(200, `
td.col-field-health_status span.data--health_status--healthy,
td.col-field-config_status span.data--config_status--applied {
	background: var(--successAltColor);
}
td.col-field-health_status span.data--health_status--unhealthy,
td.col-field-config_status span.data--config_status--error{

	background: var(--dangerAltColor);
}
/* Hide the API Preview button */
.page-content > .page-header > .btns-group > button:nth-child(1){
	display: none;
}
/* Hide PocketBase references in bottom right footer */
.page-footer > a:nth-child(2),
.page-footer > span:nth-child(3),
	display: none;
}
/* Hide column type icons */
.col-header-content > i {
	display: none;
}
header.page-header > nav.breadcrumbs > div.breadcrumb-item,
div.sidebar-content > a.sidebar-list-item > span.txt,
table.table > thead > tr > th > div.col-header-content > span.txt
{
  text-transform: capitalize;
}
				`)
			})
			e.Router.GET("/_/user.js", func(e *core.RequestEvent) error {
				e.Response.Header().Set("Content-Type", "text/javascript; charset=utf-8")

				return e.String(200, ``)
			})

			e.Router.GET("/api/v1/devicestatus/{mac_address}", apiGenerateDeviceStatus).Bind(apis.RequireAuth())
			e.Router.GET("/api/v1/frequency-overview", apiFrequencyOverview).Bind(apis.RequireAuth())
			e.Router.GET("/api/v1/network-overview", apiNetworkOverview).Bind(apis.RequireAuth())

			return e.Next()
		},
		Priority: 0,
	})

	app.OnRecordUpdateRequest("radios").BindFunc(func(e *core.RecordRequestEvent) error {
		err := validateRadio(e.App, e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordCreateRequest("devices").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := validateDevice(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordUpdateRequest("devices").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := validateDevice(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordCreateRequest("wifi_ssids").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := validateWifiUsteer(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordUpdateRequest("wifi_ssids").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := validateWifiUsteer(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordUpdateRequest("settings").BindFunc(func(e *core.RecordRequestEvent) error {
		err := validateSetting(e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	})

	// Reconnect the MQTT publisher whenever an mqtt_* setting changes.
	reconnectMQTT := func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if strings.HasPrefix(e.Record.GetString("name"), "mqtt_") {
			if err := mqtt.Configure(loadMQTTConfig(e.App)); err != nil {
				e.App.Logger().Warn("MQTT reconnect failed", "error", err)
			}
		}
		return nil
	}
	app.OnRecordAfterUpdateSuccess("settings").BindFunc(reconnectMQTT)
	app.OnRecordAfterCreateSuccess("settings").BindFunc(reconnectMQTT)

	app.OnRecordCreateExecute("devices").BindFunc(func(e *core.RecordEvent) error {
		fmt.Println()
		if err := updateAndStoreDeviceConfig(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.Cron().MustAdd("updateDeviceHealth", "* * * * *", func() {
		fmt.Println("Update Device health")
		updateDeviceHealth(app, types.NowDateTime())
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func validateWifiUsteer(record *core.Record) error {
	errs := validation.Errors{}
	if record.GetBool("usteer") {
		if !record.GetBool("ieee80211v_bss_transition") {
			errs["ieee80211v_bss_transition"] = validation.NewError(
				"validation_required_for_usteer",
				"Must be enabled when usteer is enabled.",
			)
		}
		if !record.GetBool("ieee80211k") {
			errs["ieee80211k"] = validation.NewError(
				"validation_required_for_usteer",
				"Must be enabled when usteer is enabled.",
			)
		}
	}
	if len(errs) > 0 {
		return apis.NewBadRequestError("Failed to save record.", errs)
	}
	return nil
}

func validateDevice(record *core.Record) error {
	errs := validation.Errors{}
	if !isValidHostname(record.GetString("name")) {
		errs["name"] = validation.NewError("validation_invalid_hostname", "Name cannot be converted into a valid hostname.")
	}
	if len(errs) > 0 {
		return apis.NewBadRequestError("Failed to save record.", errs)
	}
	return nil
}

func validateSetting(record *core.Record) error {
	name := record.GetString("name")
	value := record.GetString("value")

	errs := validation.Errors{}
	switch name {
	case "country":
		if IsValidCountryCode(value) == false {
			errs["value"] = validation.NewError("validation_invalid_value", "Value must be a 2 letter country code (e.g. 'BE'). '00' for global. Leave empty for the driver default.")
		}
	case "mqtt_enabled":
		if value != "" && value != "true" && value != "false" {
			errs["value"] = validation.NewError("validation_invalid_value", "Value must be 'true' or 'false'.")
		}
	case "mqtt_broker":
		if value != "" && !strings.HasPrefix(value, "tcp://") && !strings.HasPrefix(value, "ssl://") && !strings.HasPrefix(value, "ws://") && !strings.HasPrefix(value, "wss://") {
			errs["value"] = validation.NewError("validation_invalid_value", "Must be a broker URL, e.g. 'tcp://host:1883', 'ssl://host:8883', 'ws://host:8083' or 'wss://host:8084'.")
		}
	}
	if len(errs) > 0 {
		return apis.NewBadRequestError("Failed to create record.", errs)
	}
	return nil
}

// loadMQTTConfig reads the mqtt_* rows from the settings collection into an
// mqtt.Config. Missing rows leave their fields empty/disabled.
func loadMQTTConfig(app core.App) mqtt.Config {
	cfg := mqtt.Config{}
	records, err := app.FindAllRecords("settings")
	if err != nil {
		app.Logger().Warn("Failed to load MQTT settings", "error", err)
		return cfg
	}
	for _, rec := range records {
		value := rec.GetString("value")
		switch rec.GetString("name") {
		case "mqtt_enabled":
			cfg.Enabled = value == "true"
		case "mqtt_broker":
			cfg.Broker = value
		case "mqtt_username":
			cfg.Username = value
		case "mqtt_password":
			cfg.Password = value
		}
	}
	return cfg
}

func updateAndStoreDeviceConfig(app core.App, record *core.Record) error {
	data, checksum, err := generateDeviceConfig(app, record)
	if err != nil {
		return err
	}
	saveDeviceConfig(app, record, data, checksum)
	return nil
}

func regenerateAllDeviceConfigs(app core.App) {
	records, err := app.FindAllRecords("devices")
	if err != nil {
		fmt.Println("regenerateAllDeviceConfigs: failed to fetch devices", err)
		return
	}
	for _, record := range records {
		if err := updateAndStoreDeviceConfig(app, record); err != nil {
			fmt.Println("regenerateAllDeviceConfigs: failed for", record.GetString("name"), err)
		}
	}
}

func apiGenerateDeviceStatus(e *core.RequestEvent) error {
	mac_address := strings.ToUpper(e.Request.PathValue("mac_address"))
	record, err := e.App.FindFirstRecordByData("devices", "mac_address", mac_address)
	if err != nil {
		fmt.Println("HASS health status NOT FOUND", mac_address)
		return e.NotFoundError("Device not found", err)
	}
	health_status := record.GetString("health_status")
	sensor_status := "off"
	if health_status == "healthy" {
		sensor_status = "on"
	}
	fmt.Println("HASS health status", mac_address, health_status, sensor_status)

	return e.String(200, sensor_status)
}

// apiFrequencyOverview builds the dashboard's per-band channel-bonding overview.
// The scope query param selects which devices are aggregated:
//
//	"healthy" (default) -> only devices with health_status == "healthy"
//	"all"               -> every device
//	<device id>         -> that single device
func apiFrequencyOverview(e *core.RequestEvent) error {
	scope := e.Request.URL.Query().Get("scope")
	if scope == "" {
		scope = "healthy"
	}

	deviceRecords, err := e.App.FindAllRecords("devices")
	if err != nil {
		return e.InternalServerError("Failed to load devices", err)
	}

	deviceNames := map[string]string{}
	allowed := map[string]bool{}
	allowAll := scope == "all"
	for _, d := range deviceRecords {
		deviceNames[d.Id] = d.GetString("name")
		switch {
		case scope == "healthy":
			if d.GetString("health_status") == "healthy" {
				allowed[d.Id] = true
			}
		case !allowAll: // a specific device id
			if d.Id == scope {
				allowed[d.Id] = true
			}
		}
	}
	inScope := func(device string) bool { return allowAll || allowed[device] }

	radioRecords, err := e.App.FindAllRecords("radios")
	if err != nil {
		return e.InternalServerError("Failed to load radios", err)
	}
	freqRecords, err := e.App.FindAllRecords("radio_frequencies")
	if err != nil {
		return e.InternalServerError("Failed to load radio frequencies", err)
	}

	var radios []frequencyplan.Radio
	for _, r := range radioRecords {
		if !inScope(r.GetString("device")) {
			continue
		}
		radios = append(radios, frequencyplan.Radio{
			Device:    r.GetString("device"),
			Frequency: r.GetInt("frequency"),
			Htmode:    r.GetString("htmode"),
		})
	}
	var freqs []frequencyplan.Frequency
	for _, f := range freqRecords {
		if !inScope(f.GetString("device")) {
			continue
		}
		freqs = append(freqs, frequencyplan.Frequency{
			Device:    f.GetString("device"),
			Frequency: f.GetInt("frequency"),
			Flags:     f.GetStringSlice("flags"),
		})
	}

	// Device list for the selector, sorted by name.
	type deviceOption struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
	devices := make([]deviceOption, 0, len(deviceRecords))
	for _, d := range deviceRecords {
		devices = append(devices, deviceOption{Id: d.Id, Name: d.GetString("name")})
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Name < devices[j].Name })

	return e.JSON(200, map[string]any{
		"scope":   scope,
		"devices": devices,
		"bands":   frequencyplan.BuildOverview(radios, freqs, deviceNames),
	})
}

func apiNetworkOverview(e *core.RequestEvent) error {
	deviceRecords, err := e.App.FindAllRecords("devices")
	if err != nil {
		return e.InternalServerError("Failed to load devices", err)
	}

	// Device list for the selector, sorted by name. Ip is the device's last-known
	// address, used by the UI to link to its LuCI package manager. Health mirrors the
	// devices.health_status select so the UI can badge healthy/unhealthy.
	type deviceOption struct {
		Id     string `json:"id"`
		Name   string `json:"name"`
		Ip     string `json:"ip"`
		Health string `json:"health"`
	}
	devices := make([]deviceOption, 0, len(deviceRecords))
	for _, d := range deviceRecords {
		devices = append(devices, deviceOption{
			Id:     d.Id,
			Name:   d.GetString("name"),
			Ip:     d.GetString("ip_address"),
			Health: d.GetString("health_status"),
		})
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Name < devices[j].Name })

	scope := e.Request.URL.Query().Get("device")
	if scope == "" && len(devices) > 0 {
		scope = devices[0].Id
	}

	// macOwners maps a known MAC to the opensoho device that owns it. The devices
	// table holds each device's primary MAC; interfaces add per-interface MACs as a
	// fallback without overwriting a devices-table hit.
	macOwners := map[string]string{}
	for _, d := range deviceRecords {
		if mac := d.GetString("mac_address"); mac != "" {
			macOwners[mac] = d.Id
		}
	}
	interfaceRecords, err := e.App.FindAllRecords("interfaces")
	if err != nil {
		return e.InternalServerError("Failed to load interfaces", err)
	}
	for _, i := range interfaceRecords {
		mac := i.GetString("mac_address")
		if mac == "" {
			continue
		}
		if _, ok := macOwners[mac]; !ok {
			macOwners[mac] = i.GetString("device")
		}
	}

	lldpRecords, err := e.App.FindAllRecords("lldp")
	if err != nil {
		return e.InternalServerError("Failed to load lldp neighbours", err)
	}
	var rows []lldp.Row
	for _, r := range lldpRecords {
		if r.GetString("device") != scope {
			continue
		}
		rows = append(rows, lldp.Row{
			Port: r.GetString("port"),
			Name: r.GetString("neighbor_name"),
			Mac:  r.GetString("neighbor_mac_address"),
		})
	}

	// Known ports come from the ethernet collection, annotated with the bridge each
	// belongs to (bridges.ethernet lists the member ethernet-record ids).
	bridgeRecords, err := e.App.FindAllRecords("bridges")
	if err != nil {
		return e.InternalServerError("Failed to load bridges", err)
	}
	bridgeOfEth := map[string]string{} // ethernet record id -> bridge name
	for _, b := range bridgeRecords {
		if b.GetString("device") != scope {
			continue
		}
		for _, ethId := range b.GetStringSlice("ethernet") {
			bridgeOfEth[ethId] = b.GetString("name")
		}
	}
	ethernetRecords, err := e.App.FindAllRecords("ethernet")
	if err != nil {
		return e.InternalServerError("Failed to load ethernet ports", err)
	}
	var ports []lldp.EthernetPort
	for _, et := range ethernetRecords {
		if et.GetString("device") != scope {
			continue
		}
		ports = append(ports, lldp.EthernetPort{
			Name:   et.GetString("name"),
			Speed:  et.GetString("speed"),
			Bridge: bridgeOfEth[et.Id],
		})
	}

	// PoE draw per port (watts). A port present here is PoE-capable, so the UI
	// renders the PoE column; the whole column is hidden when no port has PoE.
	poeRecords, err := e.App.FindAllRecords("poe")
	if err != nil {
		return e.InternalServerError("Failed to load poe ports", err)
	}
	poeByPort := map[string]float64{}
	for _, pr := range poeRecords {
		if pr.GetString("device") != scope {
			continue
		}
		poeByPort[pr.GetString("port")] = pr.GetFloat("consumption")
	}

	return e.JSON(200, map[string]any{
		"scope":   scope,
		"devices": devices,
		"ports":   lldp.BuildPortOverview(ports, rows, macOwners, poeByPort),
	})
}

func generateWifiQr(wifi *core.Record) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	encProto, err := wifiqr.NewEncryptionProtocol("wpa") // Let's hardcode to WPA for now
	if err != nil {
		fmt.Println("Invalid encryption:", err)
		return buf, err
	}
	qrconfig := wifiqr.NewConfig(wifi.GetString("ssid"), wifi.GetString("key"), encProto, false /*hidden*/)
	qr, err := wifiqr.InitCode(qrconfig)
	if err != nil {
		fmt.Println("Invalid encryption:", err)
		return buf, err
	}
	err = png.Encode(buf, qr.Image(256))
	if err != nil {
		log.Fatalf("Failed to encode PNG: %v", err)
	}
	return buf, nil
}

func generateHostApdPskForWifi(app core.App, wifi *core.Record, wifiname string) string {
	records, err := app.FindAllRecords("wifi_client_psk",
		dbx.NewExp("wifi = {:wifi}", dbx.Params{"wifi": wifi.Id}))
	if err != nil {
		fmt.Println("Failed to fetch client psks for wifi", wifi)
		return ""
	}
	return generateHostApdPsk(app, records, wifiname)
}
func generateHostApdPsk(app core.App, client_psks []*core.Record, wifiname string) string {
	output := ""
	for _, client_psk := range client_psks {
		errs := app.ExpandRecord(client_psk, []string{"clients", "vlan"}, nil)
		if len(errs) > 0 {
			log.Println(errs)
			continue
		}
		vlanconfig := ""
		if vlan := client_psk.ExpandedOne("vlan"); vlan != nil {
			vlanconfig = fmt.Sprintf("        option vid '%d'\n", vlan.GetInt("number"))
		}
		clients := []string{"00:00:00:00:00:00"}
		if clientrecords := client_psk.ExpandedAll("clients"); len(clientrecords) > 0 {
			clients = []string{}
			for _, clientrecord := range clientrecords {
				clients = append(clients, clientrecord.GetString("mac_address"))
			}
			// Sorts macs for a stable config
			sort.Slice(clients, func(i, j int) bool {
				return clients[i] < clients[j]
			})
		}
		password := client_psk.GetString("password")
		for num, client := range clients {
			output += fmt.Sprintf(`
config wifi-station 'psk_%[5]s_%[6]d'
        option iface '%[3]s'
        option key '%[1]s'
        option mac '%[2]s'
%[4]s`, uciQuote(password), client, wifiname, vlanconfig, client_psk.Id, num)
		}
	}
	return output
}

func generateHostApdVlanConfig(app core.App) string {
	vlans, err := app.FindAllRecords("vlan", dbx.NewExp("true"))
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return generateHostApdVlanMap(vlans)
}

func generateHostApdVlanMap(vlans []*core.Record) string {
	sort.Slice(vlans, func(i, j int) bool {
		return vlans[i].GetInt("number") < vlans[j].GetInt("number")
	})
	output := ""
	for _, vlan := range vlans {
		vlanNumber := vlan.GetInt("number")
		if !isValidVlanNumber(vlanNumber) {
			continue
		}
		output += fmt.Sprintf(`
config wifi-vlan 'wifi_vlan_%[1]d'
        option name 'vl%[1]d'
        option network '%[2]s'
        option vid '%[1]d'
`, vlanNumber, vlan.GetString("name"))
	}
	return output
}
