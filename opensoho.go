package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"database/sql"
	"embed"
	"encoding/hex"
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
	"time"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/rubenbe/pocketbase"
	"github.com/rubenbe/pocketbase/apis"
	"github.com/rubenbe/pocketbase/core"
	//"github.com/rubenbe/pocketbase/plugins/ghupdate"
	"github.com/reugn/wifiqr"
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
	re := regexp.MustCompile(`phy(\d+)-`)
	match := re.FindStringSubmatch(s)
	if len(match) < 2 {
		return 0, fmt.Errorf("phy number not found in string: %s", s)
	}
	return strconv.Atoi(match[1])
}

func updateDeviceHealth(app core.App, currenttime types.DateTime) {
	oldesttime := currenttime.Add(-60 * time.Second)
	_, err := app.DB().
		NewQuery("update devices set health_status = \"unhealthy\" where last_seen <= {:offset}").
		Bind(dbx.Params{"offset": oldesttime.String()}).Execute()
	if err != nil {
		fmt.Println("Failed to update device health")
		fmt.Println(err)
	}

}
func updateLastSeen(e *core.RequestEvent, record *core.Record) error {
	record.Set("last_seen", time.Now())
	record.Set("health_status", "healthy")
	record.Set("ip_address", e.RealIP())
	return e.App.Save(record)
}

func frequencyToBand(frequency int) string {
	switch {
	case frequency >= 2400 && frequency <= 2500:
		return "2.4"
	case frequency >= 5170 && frequency <= 5835:
		return "5"
	case frequency >= 5925 && frequency <= 7125:
		return "6"
	case frequency >= 57000 && frequency <= 71000:
		return "60"
	default:
		return "unknown"
	}
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func frequencyToChannel(freqMHz int) (int, bool) {
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

func validateRadioHtModeBandCombo(band string, htmode string) error {
	validHtModes := map[string][]string{
		"2.4": {"HT20", "HT40"},
		"5":   {"HT20", "HT40", "VHT20", "VHT40", "VHT80", "VHT160"},
		"6":   {"HE20", "HE40", "HE80", "HE160"},
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

func validateRadioFrequencyBandCombo(band string, frequency string) error {

	validFrequencies := map[string][]string{
		"2.4": {"2412", "2417", "2422", "2427", "2432", "2437", "2442", "2447", "2452", "2457", "2462", "2467", "2472"},
		"5": {
			"5180", "5200", "5220", "5240", "5260", "5280", "5300", "5320",
			"5500", "5520", "5540", "5560", "5580", "5600", "5620", "5640", "5660", "5680", "5700",
			"5720", "5745", "5765", "5785", "5805", "5825",
		},
		"6": {
			"5955", "5975", "5995", "6015", "6035", "6055", "6075", "6095", "6115", "6135",
			"6155", "6175", "6195", "6215", "6235", "6255", "6275", "6295", "6315", "6335",
			"6355", "6375", "6395", "6415", "6435", "6455", "6475", "6495", "6515", "6535",
			"6555", "6575", "6595", "6615", "6635", "6655", "6675", "6695", "6715", "6735",
			"6755", "6775", "6795", "6815", "6835", "6855", "6875", "6895", "6915", "6935",
			"6955", "6975",
		},
		"60": {"58320", "60480", "62640", "64800", "66960"},
	}

	freqs, ok := validFrequencies[band]
	if !ok {
		return validation.NewError("validation_invalid_value", "Invalid band")
	}

	for _, f := range freqs {
		if f == frequency {
			return nil
		}
	}

	return validation.NewError("validation_invalid_value", "Frequency does not match selected band")
}

func validateRadio(record *core.Record) error {
	errs := validation.Errors{}
	band := record.GetString("band")
	frequency := record.GetString("frequency")

	err := validateRadioFrequencyBandCombo(band, frequency)
	if err != nil {
		errs["frequency"] = err
	}

	htmode := record.GetString("ht_mode")

	err = validateRadioHtModeBandCombo(band, htmode)

	if err != nil {
		errs["ht_mode"] = err
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
	Frequency int
	Channel   int
	HTmode    string
	TxPower   int
	MAC       string
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

type WifiRecord struct {
	Record             *core.Record
	HostApdPskFilename string
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
			fmt.Println("EXISTS", newradio, oldradio, oldradio.GetString("mac_address"))
			if oldradio.GetBool("enabled") == false {
				oldradio.Set("enabled", true)
				err := app.Save(oldradio)
				if err != nil {
					fmt.Println("Failed to mark radio as enabled:", err)
				}
			}
			if len(oldradio.GetString("mac_address")) == 0 {
				oldradio.Set("mac_address", newradio.MAC)
				err = app.Save(oldradio)
				if err != nil {
					fmt.Println("Failed to update radio with mac", err)
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
		record.Set("mac_address", radio.MAC)
		record.Set("radio", numradio)
		record.Set("channel", radio.Channel)
		record.Set("band", frequencyToBand(radio.Frequency))
		record.Set("frequency", radio.Frequency)
		record.Set("enabled", true)
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

func generateRadioConfig(radio *core.Record, country_code string) string {
	frequency := radio.GetInt("frequency")
	channel, ok := frequencyToChannel(frequency)
	if ok == false {
		fmt.Println("invalid frequency", frequency)
		return ""
	}

	frequency_txt := "auto"
	if radio.GetBool("auto_frequency") != true {
		frequency_txt = fmt.Sprintf("%d", channel)
	}
	ht_mode_txt := ""
	if ht_mode := radio.GetString("ht_mode"); len(ht_mode) > 0 {
		ht_mode_txt = fmt.Sprintf("        option htmode '%[1]s'\n", ht_mode)
	}

	country_txt := ""
	if len(country_code) > 0 {
		country_txt = fmt.Sprintf("        option country '%[1]s'\n", country_code)
	}

	return fmt.Sprintf(`
config wifi-device 'radio%[1]d'
        option channel '%[2]s'
%[3]s%[4]s`, radio.GetInt("radio"), frequency_txt, country_txt, ht_mode_txt)
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
		output += generateRadioConfig(record, country)

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

func generateWifiConfig(wifirecord WifiRecord, wifiid int, radio uint, app core.App, device *core.Record) string {
	wifi := wifirecord.Record
	ssid := wifi.GetString("ssid")
	key := wifi.GetString("key")
	steeringconfig, err := generateMacClientSteeringConfig(app, wifi, device)
	if err != nil {
		fmt.Println(err)
	}
	encryption := wifi.GetString("encryption")
	if len(encryption) == 0 {
		encryption = "psk2+ccmp"
	}

	clientpskconfig := wifirecord.HostApdPskFilename
	if len(clientpskconfig) > 0 {
		clientpskconfig = fmt.Sprintf("        option wpa_psk_file '%s'\n", clientpskconfig)
	}

	vta_flag, vta_tz := getTimeAdvertisementValues(wifi.GetString("ieee80211v_time_advertisement"))
	fmt.Println(vta_tz)
	return fmt.Sprintf(`
config wifi-iface 'wifi_%[6]d_radio%[3]d'
        option device 'radio%[3]d'
        option network '%[8]s'
        option disabled '0'
        option mode 'ap'
        option ssid '%[1]s'
        option encryption '%[5]s'
        option key '%[4]s'
        option ieee80211k '%[12]d'
        option ieee80211r '%[7]d'
        option reassociation_deadline '%[11]d'
        option time_advertisement '%[14]d'
        option time_zone '%[15]s'
        option wnm_sleep_mode '%[13]d'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '%[16]d'
        option bss_transition '%[10]d'
        option dtim_period '%[17]d'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
%[18]s%[9]s`,
		ssid, wifi.GetString("id"), radio, key, encryption,
		wifiid, wifi.GetInt("ieee80211r"), getVlan(wifi, app), steeringconfig, wifi.GetInt("ieee80211v_bss_transition"),
		max(1000, wifi.GetInt("ieee80211r_reassoc_deadline")), wifi.GetInt("ieee80211k"), wifi.GetInt("ieee80211v_wnm_sleep_mode"), vta_flag, vta_tz,
		wifi.GetInt("ieee80211v_proxy_arp"), maxInt(1, wifi.GetInt("dtim_period")), clientpskconfig)
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

	for _, filePath := range filenames {
		fileBytes := []byte(files[filePath])

		header := &tar.Header{
			Name: filePath,
			Size: int64(len(fileBytes)),
			Mode: 0644,
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
`, device.GetString("name"))

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

func generateWifiConfigs(wifis []WifiRecord, numradios uint, app core.App, device *core.Record) string {
	output := ""
	for i, wifi := range wifis {
		for j := range numradios {
			fmt.Println(wifi)
			output += generateWifiConfig(wifi, i, j, app, device)
		}
	}

	return output
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
	return app.FindFirstRecordByData("wifi", "ssid", ssid)
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
	//mode := "t"
	intfmode := "\n        option proto 'none'"

	// Avoid overwriting the default LAN configuration
	if len(cidr) > 0 && vlanname != "lan" {
		ipAddr, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			prefixsize, _ := ipNet.Mask.Size()
			prefixmask, _ := CIDRToMask(prefixsize)
			intfmode = fmt.Sprintf(`
        option proto 'static'
        option ipaddr '%[1]s'
        option netmask '%[2]s'`, ipAddr, prefixmask)
		} else {
			app.Logger().Warn("Invalid CIDR", "cidr", cidr)
		}
	}

	if vlanname == "lan" {
		//mode = "u*"
		intfmode = ""
	}

	// TODO add ip address?
	return fmt.Sprintf(`
config interface '%[1]s'
        option device 'br-lan.%[2]d'%[4]s

config bridge-vlan 'bridge_vlan_%[2]d'
        option device 'br-lan'
        option vlan '%[2]d'
%[3]s`, vlanname, vlanid, generatePortTaggingConfig(app, taggingConfig), intfmode)
}
func generateInterfacesConfig(app core.App, device *core.Record) string {
	if false == IsFeatureApplied(device, "vlan") {
		return ""
		/*return `
		config interface 'lan'
		        option device 'br-lan'
		`*/
	}
	bridgeConfig, err := app.FindFirstRecordByFilter("bridges",
		"name = 'br-lan' && device = {:device}", dbx.Params{"device": device.Id})
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
	wifis := device.GetStringSlice("wifis")
	// Get the static Wifi configurations
	wifirecords, err := app.FindRecordsByIds("wifi", wifis)
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

func generateDeviceConfig(app core.App, record *core.Record) ([]byte, string, error) {
	configfiles := map[string]string{}
	leds := record.Get("leds").([]string)
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
	fmt.Println("wifis")
	fmt.Println(record.Get("wifis"))
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

		// HostApdPskConfig needs to be called before the generateWifiConfigs
		wificonfigsstructs := generateHostApdPskConfigs(app, wifirecords, &configfiles)
		wificonfigs := generateWifiConfigs(wificonfigsstructs, numradios, app, record)
		fmt.Println(wificonfigs)
		if len(wificonfigs) > 0 {
			configfiles["etc/config/wireless"] = wificonfigs
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

	blob, checksum, err := createConfigTar(configfiles)

	if err != nil {
	}
	return blob, checksum, err
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
			err = fmt.Errorf("Unknown bridge member %s", membername)
			fmt.Println(err)
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
	radios := make(map[int]Radio)
	var payload MonitoringData
	if e.Request.Header.Get("Content-Length") == "0" {
		app.Logger().Info("Ignored empty monitoring request 1", "userIP", e.RealIP())
		return e.Blob(200, "text/plain", []byte("")), radios
	}
	if err := e.BindBody(&payload); err != nil {
		fmt.Println(err)
		return e.BadRequestError("Failed to parse json", err), radios
	}
	if payload.Type != "DeviceMonitoring" {
		errormsg := fmt.Sprintf(`Invalid type '%s' in JSON`, payload.Type)
		fmt.Println(errormsg)
		return e.BadRequestError(errormsg, ""), radios
	}
	interfacecollection, _ := app.FindCollectionByNameOrId("interfaces")
	wificollection, _ := app.FindCollectionByNameOrId("wifi")

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
				radios[radionum] = Radio{Frequency: iface.Wireless.Frequency, Channel: iface.Wireless.Channel, HTmode: iface.Wireless.HTmode, TxPower: iface.Wireless.TxPower, MAC: iface.MAC}
			}

			for _, client := range iface.Wireless.Clients {
				if client.Assoc {
					fmt.Printf("Associated client on %s: %s %s\n", iface.Name, client.MAC, device.GetString("id"))
					cliententry, err := app.FindFirstRecordByData(collection, "mac_address", client.MAC)
					if err != nil {
						cliententry = core.NewRecord(collection)
					}
					cliententry.Set("mac_address", client.MAC)
					// TODO expand model
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

func bindAppHooks(app core.App, shared_secret string, enableNewDevices bool) {
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

			collection, err := app.FindCollectionByNameOrId("clients")
			if err != nil {
				return e.InternalServerError("Could not find collection", err)
			}

			err, radios := handleMonitoring(e, app, device, collection)
			updateRadios(device, app, radios)
			return err
		})
		return se.Next()
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

			return e.Next()
		},
		Priority: 0,
	})

	app.OnRecordUpdateRequest("radios").BindFunc(func(e *core.RecordRequestEvent) error {
		err := validateRadio(e.Record)
		if err != nil {
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

	app.OnRecordCreateExecute("device").BindFunc(func(e *core.RecordEvent) error {
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

func validateSetting(record *core.Record) error {
	name := record.GetString("name")
	value := record.GetString("value")

	errs := validation.Errors{}
	switch name {
	case "country":
		if IsValidCountryCode(value) == false {
			errs["value"] = validation.NewError("validation_invalid_value", "Value must be a 2 letter country code (e.g. 'BE'). '00' for global. Leave empty for the driver default.")
		}
	}
	if len(errs) > 0 {
		return apis.NewBadRequestError("Failed to create record.", errs)
	}
	return nil
}

func updateAndStoreDeviceConfig(app core.App, record *core.Record) error {
	data, checksum, err := generateDeviceConfig(app, record)
	if err != nil {
		return err
	}
	saveDeviceConfig(app, record, data, checksum)
	return nil
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

func generateHostApdPskConfigFilename(wifirecord *core.Record) string {
	return fmt.Sprintf("etc/hostapd/%s.psk", wifirecord.GetString("ssid"))
}

func generateHostApdPskConfigs(app core.App, wifirecords []*core.Record, configmap *map[string]string) []WifiRecord {
	wificonfigs := []WifiRecord{}
	for _, wifirecord := range wifirecords {
		wificonfig := WifiRecord{wifirecord, ""}
		config := generateHostApdPskForWifi(app, wifirecord)
		if len(config) > 0 {
			filename := generateHostApdPskConfigFilename(wifirecord)
			(*configmap)[filename] = config
			wificonfig.HostApdPskFilename = filename
		}
		wificonfigs = append(wificonfigs, wificonfig)
	}
	return wificonfigs
}

// https://git.w1.fi/cgit/hostap/tree/hostapd/hostapd.wpa_psk
func generateHostApdPskForWifi(app core.App, wifi *core.Record) string {
	records, err := app.FindAllRecords("wifi_client_psk",
		dbx.NewExp("wifi = {:wifi}", dbx.Params{"wifi": wifi.Id}))
	if err != nil {
		fmt.Println("Failed to fetch client psks for wifi", wifi)
		return ""
	}
	return generateHostApdPsk(app, records)
}
func generateHostApdPsk(app core.App, client_psks []*core.Record) string {
	output := ""
	for _, client_psk := range client_psks {
		errs := app.ExpandRecord(client_psk, []string{"clients", "vlan"}, nil)
		if len(errs) > 0 {
			log.Println(errs)
			continue
		}
		vlanconfig := ""
		//if vlan := ExpandedOne("vlan") ;vlan != nil {
		//	vlanconfig = fmt.Sprintf("vlanid=%d ", vlan.GetInt("number"))
		//}
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
		for _, client := range clients {
			output += fmt.Sprintf("%[1]s%[2]s %[3]s\n", vlanconfig, client, password)
		}
	}
	return output
}

func generateHostApdVlanMap(vlans []*core.Record, interfacename string) string {
	sort.Slice(vlans, func(i, j int) bool {
		return vlans[i].GetInt("number") < vlans[j].GetInt("number")
	})
	output := ""
	for _, vlan := range vlans {
		vlanNumber := vlan.GetInt("number")
		if !isValidVlanNumber(vlanNumber) {
			continue
		}
		output += fmt.Sprintf("%[1]d %[2]s.%[1]d\n", vlanNumber, interfacename)
	}
	return output
}
