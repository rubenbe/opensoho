package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/rubenbe/pocketbase"
	"github.com/rubenbe/pocketbase/apis"
	"github.com/rubenbe/pocketbase/core"
	//"github.com/rubenbe/pocketbase/plugins/ghupdate"
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
func updateLastSeen(app core.App, record *core.Record) error {
	record.Set("last_seen", time.Now())
	record.Set("health_status", "healthy")
	return app.Save(record)
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

func validateRadio(record *core.Record) error {
	if record.Collection().Name != "radios" {
		return nil
	}

	band := record.GetString("band")
	freq := record.GetString("frequency")

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
		return errors.New("invalid band")
	}

	for _, f := range freqs {
		if f == freq {
			return nil
		}
	}

	return errors.New("frequency does not match selected band")
}

type Client struct {
	MAC    string `json:"mac"`
	Assoc  bool   `json:"assoc"`
	Signal int    `json:"signal"`
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

type Interface struct {
	MAC      string    `json:"mac"`
	Type     string    `json:"type"`
	Name     string    `json:"name"`
	Wireless *Wireless `json:"wireless,omitempty"`
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
					fmt.Println("Fialed to mark radio as enabled:", err)
				}
			}
			if len(oldradio.GetString("mac_address")) == 0 {
				oldradio.Set("mac_address", newradio.MAC)
				err = app.Save(oldradio)
				if err != nil {
					fmt.Println("Fialed to update radio with mac", err)
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
					fmt.Println("Fialed to mark radio as disabled:", err)
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

func generateRadioConfig(radio *core.Record) string {
	frequency := radio.GetInt("frequency")
	channel, ok := frequencyToChannel(frequency)
	if ok == false {
		fmt.Println("invalid frequency", frequency)
		return ""
	}
	return fmt.Sprintf(`
config wifi-device 'radio%d'
	option channel '%d'
`, radio.GetInt("radio"), channel)
}
func generateRadioConfigs(device *core.Record, app core.App) string {
	output := ""
	records := []*core.Record{}
	err := app.RecordQuery("radios").AndWhere(dbx.HashExp{"device": device.GetString("id")}).OrderBy("radio ASC").All(&records)
	if err != nil {
		fmt.Println("Error finding Radios", err)
		return ""
	}
	for _, record := range records {
		output += generateRadioConfig(record)

	}
	return output
}

func getVlan(wifi *core.Record, app core.App) string {
	errs := app.ExpandRecord(wifi, []string{"network"}, nil)
	log.Println("NETWORK NAME 0")
	if len(errs) > 0 {
		log.Println(errs)
		return "lan"
	}
	networkentry := wifi.ExpandedOne("network")
	log.Println("NETWORK NAME 1")
	if networkentry == nil {
		return "lan"
	}
	log.Println("NETWORK NAME 2")
	networkname := networkentry.GetString("name")
	if len(networkname) == 0 {
		return "lan"
	}
	log.Println("NETWORK NAME 3")
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
	return strings.Join(output, "\n") + "\n"
}

func generateWifiConfig(wifi *core.Record, wifiid int, radio uint, app core.App, device *core.Record) string {
	ssid := wifi.GetString("ssid")
	key := wifi.GetString("key")
	steeringconfig, err := generateMacClientSteeringConfig(app, wifi, device)
	if err != nil {
		fmt.Println(err)
	}
	return fmt.Sprintf(`
config wifi-iface 'wifi_%[6]d_radio%[3]d'
        option device 'radio%[3]d'
        option network '%[8]s'
        option disabled '0'
        option mode 'ap'
        option ssid '%[1]s'
        option encryption '%[5]s'
        option key '%[4]s'
        option ieee80211r '%[7]d'
        option ieee80211v '%[10]d'
        option bss_transition '%[10]d'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
%[9]s`, ssid, wifi.GetString("id"), radio, key, wifi.GetString("encryption"), wifiid, wifi.GetInt("ieee80211r"), getVlan(wifi, app), steeringconfig, wifi.GetInt("ieee80211v"))
}

func createConfigTar(files map[string]string) ([]byte, string, error) {
	var buf bytes.Buffer

	gzWriter := gzip.NewWriter(&buf)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for filePath, content := range files {
		fileBytes := []byte(content)

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

func generateLedConfigs(leds []*core.Record) string {
	output := ""
	for _, led := range leds {
		fmt.Println(led)
		output += generateLedConfig(led)
	}

	return output
}

func generateWifiConfigs(wifis []*core.Record, numradios uint, app core.App, device *core.Record) string {
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
func getDeviceRecord(app core.App, key string) (*core.Record, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("Key not valid")
	}
	pbID, err := hexToPocketBaseID(key)
	if err != nil {
		return nil, fmt.Errorf("Key not hex")
	}
	record, err := app.FindRecordById("devices", pbID)
	if err != nil {
		return nil, fmt.Errorf("Device not known")
	}
	if !security.Equal(record.GetString("key"), key) {
		return nil, fmt.Errorf("Key not allowed")
	}
	// Successful authentication means the device is alive and online
	fmt.Println("UpdateLastSeen")
	if updateLastSeen(app, record) != nil {
		fmt.Println("could not update last seen")
		// Let's not make this an error for now
	}

	return record, nil
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

	unhealthy_devices, err := app.FindRecordsByFilter("devices", `health_status!="healthy"`, "", 0, 0, map[string]any{})
	unhealthy_device_ids := make(map[string]struct{}, len(unhealthy_devices))
	for _, device := range unhealthy_devices {
		unhealthy_device_ids[device.Id] = struct{}{}
	}
	fmt.Println("Full list", unhealthy_device_ids)
	for _, client := range client_steering_for_wifi {

		whitelisted_devices := client.GetStringSlice("whitelist")
		fmt.Println(whitelisted_devices)

		// Check whether sufficient whitelisted devices are online
		disable_whitelisting := false
		whitelisting_mode := client.GetString("enable")
		if whitelisting_mode != "Always" {
			disable_whitelisting = isUnHealthyQuorumReached(unhealthy_device_ids, whitelisted_devices, whitelisting_mode == "If all healthy")
		}
		if disable_whitelisting == true {
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

func generateDeviceConfig(app core.App, record *core.Record) ([]byte, string, error) {
	configfiles := map[string]string{}
	leds := record.Get("leds").([]string)
	fmt.Println(leds)
	ledrecords, err := app.FindRecordsByIds("leds", leds)
	if err != nil {
		return nil, "", err
	}
	ledconfigs := generateLedConfigs(ledrecords)
	if len(ledconfigs) > 0 {
		configfiles["etc/config/system"] = ledconfigs
	}
	fmt.Println("wifis")
	fmt.Println(record.Get("wifis"))
	numradios := uint(record.GetInt("numradios"))
	fmt.Printf("numradios %d\n", numradios)
	if wifis := record.Get("wifis"); wifis != nil {
		wifirecords, err := app.FindRecordsByIds("wifi", wifis.([]string))
		if err != nil {
			return nil, "", err
		}
		sort.Slice(wifirecords, func(i, j int) bool {
			return wifirecords[i].GetDateTime("created").Before(wifirecords[j].GetDateTime("created"))
		})
		wificonfigs := generateWifiConfigs(wifirecords, numradios, app, record)
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

	blob, checksum, err := createConfigTar(configfiles)
	if err != nil {
	}
	return blob, checksum, err
}

func findFirstOrNew(app core.App, collection *core.Collection, column string, value string) *core.Record {
	record, err := app.FindFirstRecordByData(collection, column, value)
	if err != nil {
		record = core.NewRecord(collection)
	}
	return record
}

func handleMonitoring(e *core.RequestEvent, app core.App, device *core.Record, collection *core.Collection) (error, map[int]Radio) {
	e.Response.Header().Set("X-Openwisp-Controller", "true")
	time := e.Request.URL.Query().Get("time")
	radios := make(map[int]Radio)
	var payload MonitoringData
	if err := e.BindBody(&payload); err != nil {
		fmt.Println(err)
		return e.BadRequestError("Failed to parse json", err), radios
	}
	if payload.Type != "DeviceMonitoring" {
		return e.BadRequestError("Invalid type in JSON", ""), radios
	}

	for _, iface := range payload.Interfaces {
		if iface.Type == "wireless" && iface.Wireless != nil {
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
					cliententry.Set("band", frequencyToBand(iface.Wireless.Frequency))
					cliententry.Set("device", device.GetString("id"))
					err = app.Save(cliententry)
					if err != nil {
						return e.InternalServerError("Could not store entry", err), radios
					}
				}
			}
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

func bindAppHooks(app core.App, shared_secret string) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/controller/register/", func(e *core.RequestEvent) error {
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
			fmt.Println(pbID)
			record, err := app.FindRecordById("devices", pbID)

			isNew := 1
			var device_uuid string = uuid.New().String()
			fmt.Println(device_uuid)

			if err == nil {
				isNew = 0
				fmt.Print("Hello back")
				device_uuid = record.GetString("uuid")
			} else {
				// Register new device
				if data.Backend != "netjsonconfig.OpenWrt" {
					return e.BadRequestError("Registration failed!", "wrong backend")
				}
				fmt.Print("Hello")
				collection, err := app.FindCollectionByNameOrId("devices")

				if err != nil {
					return e.BadRequestError("Registration failed!", err)
				}
				fmt.Print(pbID)

				record := core.NewRecord(collection)
				record.Set("id", pbID)
				record.Set("backend", data.Backend)
				record.Set("key", data.Key)
				record.Set("name", data.Name)
				record.Set("hardware_id", data.HardwareId)
				record.Set("mac_address", data.MacAddress)
				record.Set("uuid", device_uuid)
				record.Set("tags", data.Tags)
				record.Set("model", data.Model)
				record.Set("os", data.Os)
				record.Set("system", data.System)
				record.Set("ip_address", e.RealIP())
				record.Set("health_status", "unknown")
				record.Set("config_status", "applied")
				record.Set("enabled", "true")
				err = app.Save(record)
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
		})
		se.Router.POST("/controller/report-status/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			data := struct {
				// unexported to prevent binding
				somethingPrivate string

				Status string `form:"status"`
				Key    string `form:"key"`
			}{}
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Missing fields", err)
			}
			record, err := getDeviceRecord(app, data.Key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}
			record.Set("config_status", data.Status)
			err = app.Save(record)
			if err != nil {
				return e.InternalServerError("Status update failed", err)
			}
			response := fmt.Sprintf("report-result: success\ncurrent-status: %s\n", data.Status)
			fmt.Println(response)

			return e.Blob(200, "text/plain", []byte(response))
		})

		se.Router.GET("/controller/checksum/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			key := e.Request.URL.Query().Get("key")
			record, err := getDeviceRecord(app, key)
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
			record, err := getDeviceRecord(app, key)
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
			device, err := getDeviceRecord(app, key)
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
	if strings.SplitN(record.GetString("config"), "_", 2)[0] != checksum {
		f, err := filesystem.NewFileFromBytes(data, filename)
		if err != nil {
			return err
		}

		fmt.Println(filename)
		record.Set("config", f)
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

func main() {
	shared_secret := os.Getenv("OPENSOHO_SHARED_SECRET")
	if shared_secret == "" {
		fmt.Println("OPENSOHO_SHARED_SECRET environment variable not set!")
		return
	}
	os.Unsetenv("OPENSOHO_SHARED_SECRET")

	app := pocketbase.New()

	bindAppHooks(app, shared_secret)

	// Upstream commands
	// ---------------------------------------------------------------
	// Optional plugin flags:
	// ---------------------------------------------------------------

	var hooksDir string
	app.RootCmd.PersistentFlags().StringVar(
		&hooksDir,
		"hooksDir",
		"",
		"the directory with the JS app hooks",
	)

	var hooksWatch bool
	app.RootCmd.PersistentFlags().BoolVar(
		&hooksWatch,
		"hooksWatch",
		true,
		"auto restart the app on pb_hooks file change; it has no effect on Windows",
	)

	var hooksPool int
	app.RootCmd.PersistentFlags().IntVar(
		&hooksPool,
		"hooksPool",
		15,
		"the total prewarm goja.Runtime instances for the JS app hooks execution",
	)

	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"migrationsDir",
		"",
		"the directory with the user defined migrations",
	)

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(
		&publicDir,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(
		&indexFallback,
		"indexFallback",
		true,
		"fallback the request to index.html on missing static path, e.g. when pretty urls are used with SPA",
	)

	var doFileExtraction bool
	app.RootCmd.PersistentFlags().BoolVar(
		&doFileExtraction,
		"doEmbeddedFileExtraction",
		true,
		"Extracts the embedded migrations and frontend files",
	)

	var developerMode bool
	app.RootCmd.PersistentFlags().BoolVar(
		&developerMode,
		"developerMode",
		strings.HasPrefix(os.Args[0], os.TempDir()),
		"Run in developer Mode (shows extra controls to modify collections)",
	)

	app.RootCmd.ParseFlags(os.Args[1:])

	app.OnSettingsListRequest().BindFunc(func(e *core.SettingsListRequestEvent) error {
		e.Settings.Meta.AppName = "OpenSOHO"
		e.Settings.Meta.HideControls = !developerMode

		return e.Next()
	})

	// ---------------------------------------------------------------
	// Plugins and hooks:
	// ---------------------------------------------------------------

	if doFileExtraction {
		if err := copyEmbedDirToDisk(embeddedFiles, ""); err != nil {
			log.Fatal(err)
		}
	}

	// load jsvm (pb_hooks and pb_migrations)
	jsvm.MustRegister(app, jsvm.Config{
		MigrationsDir: migrationsDir,
		HooksDir:      hooksDir,
		HooksWatch:    hooksWatch,
		HooksPoolSize: hooksPool,
	})

	// migrate command (with js templates)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  automigrate,
		Dir:          migrationsDir,
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
td.col-field-health_status span.data--health_status--healthy {
	background: var(--successAltColor);
}
td.col-field-health_status span.data--health_status--unhealthy {
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

			e.Router.GET("/api/hass/v1/devicestatus/{device_id}", apiGenerateDeviceStatus).Bind(apis.RequireAuth())

			return e.Next()
		},
		Priority: 0,
	})

	app.OnRecordValidate("radios").BindFunc(func(e *core.RecordEvent) error {
		return validateRadio(e.Record)
	})

	app.Cron().MustAdd("updateDeviceHealth", "* * * * *", func() {
		fmt.Println("Update Device health")
		updateDeviceHealth(app, types.NowDateTime())
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// the default pb_public dir location is relative to the executable
func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// most likely ran with go run
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}

func apiGenerateDeviceStatus(e *core.RequestEvent) error {
	device_id := e.Request.PathValue("device_id")
	record, err := e.App.FindRecordById("devices", device_id)
	if err != nil {
		return e.NotFoundError("Device not found", err)
	}
	health_status := record.GetString("health_status")
	sensor_status := "off"
	if health_status == "healthy" {
		sensor_status = "on"
	}
	fmt.Println("HASS health status", device_id, health_status, sensor_status)

	return e.String(200, sensor_status)
}
