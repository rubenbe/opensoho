package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/liyue201/goqr"
	"github.com/rubenbe/pocketbase/core"
	"github.com/rubenbe/pocketbase/tests"
	"github.com/rubenbe/pocketbase/tools/router"
	"github.com/rubenbe/pocketbase/tools/types"
	"github.com/stretchr/testify/assert"
)

/*
const testDataDir = "./test_pb_data"

func generateToken(collectionNameOrId string, email string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	record, err := app.FindAuthRecordByEmail(collectionNameOrId, email)
	if err != nil {
		return "", err
	}

	return record.NewAuthToken()
}*/

func TestGetDataDirPath(t *testing.T) {
	path, err := GetDataDirPath()
	assert.Equal(t, nil, err)
	cwdpath, err := os.Getwd()
	assert.Equal(t, nil, err)
	assert.Equal(t, cwdpath, path, "verify the working directory defaults to the current directory")

	// Pretend to run in a container
	os.Setenv("KO_DATA_PATH", "/var/run/ko")
	path, err = GetDataDirPath()
	// Unset it before any asserts
	os.Unsetenv("KO_DATA_PATH")

	assert.Equal(t, nil, err)
	assert.Equal(t, "/ko-app", path, "verify the working directory has changed in the container")

	// Pretend to run in a container
	os.Setenv("KO_DATA_PATH", "/something")
	path, err = GetDataDirPath()
	// Unset it before any asserts
	os.Unsetenv("KO_DATA_PATH")

	assert.Equal(t, nil, err)
	assert.Equal(t, cwdpath, path, "verify the working directory only changes with a specific ko data path")
}

func TestReportStatusEndpoint(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "a3qnbxklglw121g"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("config_status", "healthy")
	d1.Set("key", "aaaabbbbccccddddaaaabbbbccccdddd")
	err := app.Save(d1)
	assert.Equal(t, nil, err)
	{

		// Setup fake report-status event
		event := core.RequestEvent{}
		reqbody := strings.NewReader(url.Values{
			"status":       {"error"},
			"key":          {"aaaabbbbccccddddaaaabbbbccccdddd"},
			"error_reason": {"something"},
		}.Encode())
		event.Request, err = http.NewRequest("POST", "/controller/report-status/somethindevice1/", reqbody)

		event.Request.Header.Set("content-type", "application/x-www-form-urlencoded")
		event.App = app
		rec := httptest.NewRecorder()
		event.Response = rec

		err = handleDeviceStatusUpdate(&event)
		assert.Equal(t, nil, err)

		// Check the response
		httpResponse := rec.Result()

		defer httpResponse.Body.Close()
		body, err := io.ReadAll(httpResponse.Body)
		assert.Equal(t, nil, err)
		assert.Equal(t, 200, httpResponse.StatusCode)
		assert.Equal(t, "report-result: success\ncurrent-status: error\n", string(body))
	}

	// Check the updated record
	record, err := app.FindRecordById("devices", d1.Id)
	assert.Equal(t, nil, err)
	assert.Equal(t, "a3qnbxklglw121g", record.Id)
	assert.Equal(t, "error", record.GetString("config_status"))
	assert.Equal(t, "something", record.GetString("error_reason"))

	// Second step is to reset it to healthy
	{
		event := core.RequestEvent{}
		reqbody := strings.NewReader(url.Values{
			"status": {"healthy"},
			"key":    {"aaaabbbbccccddddaaaabbbbccccdddd"},
		}.Encode())
		event.Request, err = http.NewRequest("POST", "/controller/report-status/somethindevice1/", reqbody)

		event.Request.Header.Set("content-type", "application/x-www-form-urlencoded")
		event.App = app
		rec := httptest.NewRecorder()
		event.Response = rec

		err = handleDeviceStatusUpdate(&event)
		assert.Equal(t, nil, err)

		// Check the response
		httpResponse := rec.Result()

		defer httpResponse.Body.Close()
		body, err := io.ReadAll(httpResponse.Body)
		assert.Equal(t, nil, err)
		assert.Equal(t, 200, httpResponse.StatusCode)
		assert.Equal(t, "report-result: success\ncurrent-status: healthy\n", string(body))
	}

	// Check the updated record
	// Error reason should be reset and status back to healthy
	record, err = app.FindRecordById("devices", d1.Id)
	assert.Equal(t, nil, err)
	assert.Equal(t, "a3qnbxklglw121g", record.Id)
	assert.Equal(t, "healthy", record.GetString("config_status"))
	assert.Equal(t, "", record.GetString("error_reason"))
}

func TestRegisterEndpoint(t *testing.T) {
	// setup the test ApiScenario app instance
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp()
		if err != nil {
			t.Fatal(err)
		}
		// no need to cleanup since scenario.Test() will do that for us
		// defer testApp.Cleanup()

		bindAppHooks(testApp, "testsecret", true)

		return testApp
	}

	scenarios := []tests.ApiScenario{
		{
			Name:            "try with different http method GET",
			Method:          http.MethodGet,
			URL:             "/controller/register/",
			ExpectedStatus:  404,
			ExpectedContent: []string{"\"data\":{}"},
			TestAppFactory:  setupTestApp,
		},
		{
			Name:            "try with http method POST",
			Method:          http.MethodPost,
			URL:             "/controller/register/",
			ExpectedStatus:  403,
			ExpectedContent: []string{"\"data\":{}"},
			TestAppFactory:  setupTestApp,
		},
		{
			Name:           "POST with data",
			Method:         http.MethodPost,
			URL:            "/controller/register/",
			ExpectedStatus: 403,
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: strings.NewReader(url.Values{
				"backend":     {""},
				"key":         {""},
				"secret":      {""},
				"name":        {""},
				"hardware_id": {""},
				"mac_address": {""},
				"model":       {""},
				"os":          {""},
				"system":      {""},
			}.Encode()),
			ExpectedContent: []string{"\"data\":{}"},
			TestAppFactory:  setupTestApp,
		},
		{
			Name:           "POST with valid data",
			Method:         http.MethodPost,
			URL:            "/controller/register/",
			ExpectedStatus: 400, // TODO when DB is properly set up this should be 200
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: strings.NewReader(url.Values{
				"backend":     {"netjsonconfig.OpenWrt"},
				"key":         {""},
				"secret":      {"testsecret"},
				"name":        {""},
				"hardware_id": {""},
				"mac_address": {""},
				"model":       {""},
				"os":          {""},
				"system":      {""},
			}.Encode()),
			ExpectedContent: []string{"\"data\":{}"},
			TestAppFactory:  setupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestHandleBridgeMonitoring(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	interfacescollection := setupInterfacesCollection(t, app)
	bridgescollection := setupBridgesCollection(t, app, devicecollection, interfacescollection, ethernetcollection)

	// Add a device to start with
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("ip_address", "8.8.8.8")
	err := app.Save(d1)
	assert.Equal(t, nil, err)

	statistics := Statistics{
		TxBytes: 100 * 1000 * 1000,
		RxBytes: 200 * 1000 * 1000,
	}

	// Add an interface
	iface := Interface{
		Name:       "br-lan123",
		Speed:      "10000F",
		Statistics: &statistics,
		BridgeMembers: []string{
			"lan2",
			"phy1-ap0",
		},
	}

	err = handleBridgeMonitoring(app, iface, d1, bridgescollection, interfacescollection, ethernetcollection)
	assert.Equal(t, "Unknown bridge member phy1-ap0", err.Error())
	records, err := app.FindAllRecords("bridges")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(records))
	assert.Equal(t, "somethindevice1", records[0].GetString("device"))
	assert.Equal(t, 100*1000*1000, records[0].GetInt("tx_bytes"))
	assert.Equal(t, 200*1000*1000, records[0].GetInt("rx_bytes"))
	assert.Equal(t, []string{}, records[0].GetStringSlice("ethernet"))
	assert.Equal(t, []string{}, records[0].GetStringSlice("wifi"))

	// Now add a Wifi interface only
	w1 := core.NewRecord(interfacescollection)
	w1.Id = "wifiwifidevice1"
	fmt.Println(d1)
	w1.Set("device", d1.Id)
	w1.Set("interface", "phy1-ap0")
	w1.Set("band", "2.4")
	err = app.Save(w1)
	assert.Equal(t, nil, err)

	err = handleBridgeMonitoring(app, iface, d1, bridgescollection, interfacescollection, ethernetcollection)
	assert.Equal(t, "Unknown bridge member lan2", err.Error())

	// Now add an Ethernet interface too
	e1 := core.NewRecord(ethernetcollection)
	e1.Id = "ethernetdevice1"
	e1.Set("device", d1.Id)
	e1.Set("name", "lan2")
	err = app.Save(e1)
	assert.Equal(t, nil, err)

	err = handleBridgeMonitoring(app, iface, d1, bridgescollection, interfacescollection, ethernetcollection)
	assert.Equal(t, nil, err)

	records, err = app.FindAllRecords("bridges")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(records))
	assert.Equal(t, "somethindevice1", records[0].GetString("device"))
	assert.Equal(t, 100*1000*1000, records[0].GetInt("tx_bytes"))
	assert.Equal(t, 200*1000*1000, records[0].GetInt("rx_bytes"))
	// Everything good!
	assert.Equal(t, []string{"ethernetdevice1"}, records[0].GetStringSlice("ethernet"))
	assert.Equal(t, []string{"wifiwifidevice1"}, records[0].GetStringSlice("wifi"))
	// Now make the lookup ambigious by adding a wifi ap to the ethernet collection
	e2 := core.NewRecord(ethernetcollection)
	e2.Id = "ethernetdevice2"
	e2.Set("device", d1.Id)
	e2.Set("name", "phy1-ap0")
	err = app.Save(e2)
	assert.Equal(t, nil, err)

	err = handleBridgeMonitoring(app, iface, d1, bridgescollection, interfacescollection, ethernetcollection)
	assert.Equal(t, "Ambigious bridge member phy1-ap0", err.Error())

	records, err = app.FindAllRecords("bridges")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(records))
	assert.Equal(t, "somethindevice1", records[0].GetString("device"))
	assert.Equal(t, 100*1000*1000, records[0].GetInt("tx_bytes"))
	assert.Equal(t, 200*1000*1000, records[0].GetInt("rx_bytes"))
	// Bridge members should be emptied again
	assert.Equal(t, []string{}, records[0].GetStringSlice("ethernet"))
	assert.Equal(t, []string{}, records[0].GetStringSlice("wifi"))
}

func TestHandleEthernetMonitoring(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, vlancollection)
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("ip_address", "8.8.8.8")
	err := app.Save(d1)
	assert.Equal(t, nil, err)

	// Initial without statistics
	iface := Interface{
		Name:  "blah1",
		Speed: "10000F",
	}

	handleEthernetMonitoring(app, iface, d1, ethernetcollection)
	records, err := app.FindAllRecords("ethernet")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(records))
	assert.Equal(t, "somethindevice1", records[0].GetString("device"))
	assert.Equal(t, "10000F", records[0].GetString("speed"))
	assert.Equal(t, 0, records[0].GetInt("tx_bytes"))
	assert.Equal(t, 0, records[0].GetInt("rx_bytes"))

	// Update with statistics
	statistics := Statistics{
		TxBytes: 100 * 1000 * 1000,
		RxBytes: 200 * 1000 * 1000,
	}
	iface = Interface{
		Name:       "blah1",
		Speed:      "100F",
		Statistics: &statistics,
	}

	handleEthernetMonitoring(app, iface, d1, ethernetcollection)
	records, err = app.FindAllRecords("ethernet")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(records))
	assert.Equal(t, "somethindevice1", records[0].GetString("device"))
	assert.Equal(t, "100F", records[0].GetString("speed"))
	assert.Equal(t, 100*1000*1000, records[0].GetInt("tx_bytes"))
	assert.Equal(t, 200*1000*1000, records[0].GetInt("rx_bytes"))

	// Add a second interface
	iface = Interface{
		Name:       "blah2",
		Speed:      "1000F",
		Statistics: &statistics,
	}
	handleEthernetMonitoring(app, iface, d1, ethernetcollection)
	records, err = app.FindAllRecords("ethernet")
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(records))

	assert.Equal(t, "somethindevice1", records[0].GetString("device"))
	assert.Equal(t, "100F", records[0].GetString("speed"))
	assert.Equal(t, 100*1000*1000, records[0].GetInt("tx_bytes"))
	assert.Equal(t, 200*1000*1000, records[0].GetInt("rx_bytes"))

	assert.Equal(t, "somethindevice1", records[1].GetString("device"))
	assert.Equal(t, "1000F", records[1].GetString("speed"))
	assert.Equal(t, 100*1000*1000, records[1].GetInt("tx_bytes"))
	assert.Equal(t, 200*1000*1000, records[1].GetInt("rx_bytes"))

}
func TestHandleDeviceInfoUpdate(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Equal(t, nil, err)

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	id, err := hexToPocketBaseID("0123456789abcdef0123456789abcdef")
	assert.Equal(t, nil, err)
	assert.Equal(t, "2fapl4n1azs5kkw", id)

	{
		d1 := core.NewRecord(devicecollection)
		d1.Id = id
		d1.Set("name", "the_device1")
		d1.Set("health_status", "healthy")
		d1.Set("ip_address", "8.8.8.8")
		d1.Set("key", "0123456789abcdef0123456789abcdef")
		d1.Set("model", "old router model")
		d1.Set("os", "old openwrt")
		d1.Set("system", "old system")
		d1.Set("uuid", "44ee4fee-1a20-470e-9044-ad9df88a0889")
		err = app.Save(d1)
		assert.Equal(t, nil, err)
	}

	reqbody := strings.NewReader(url.Values{
		"key":    {"0123456789abcdef0123456789abcdef"},
		"model":  {"router-ng"},
		"os":     {"shiny new openwrt"},
		"system": {"system-ng"},
	}.Encode())

	event := core.RequestEvent{}
	event.Request, err = http.NewRequest("POST", "/controller/update-info/44ee4fee-1a20-470e-9044-ad9df88a0889", reqbody)
	event.Request.Header.Set("content-type", "application/x-www-form-urlencoded")
	event.App = app
	rec := httptest.NewRecorder()
	event.Response = rec

	// Test the update
	err = handleDeviceInfoUpdate(&event)
	assert.Equal(t, nil, err)

	// Check the response
	httpResponse := rec.Result()

	defer httpResponse.Body.Close()
	body, err := io.ReadAll(httpResponse.Body)
	assert.Equal(t, nil, err)
	bodystring := string(body)
	assert.Equal(t, "update-info: success\n", bodystring)

	// Verify the updated record
	record, err := app.FindRecordById("devices", id)
	assert.Equal(t, nil, err)
	assert.Equal(t, "router-ng", record.GetString("model"))
	assert.Equal(t, "shiny new openwrt", record.GetString("os"))
	assert.Equal(t, "system-ng", record.GetString("system"))
	assert.Equal(t, "44ee4fee-1a20-470e-9044-ad9df88a0889", record.GetString("uuid"))
}

func TestHandleDeviceRegistration(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	_ = setupDeviceCollection(t, app, wificollection)

	extracted_uuid := "dummy"

	// First registration
	{
		pbID, err := hexToPocketBaseID("0123456789abcdef0123456789abcdef")
		reqbody := strings.NewReader(url.Values{
			"backend":     {"netjsonconfig.OpenWrt"},
			"key":         {"0123456789abcdef0123456789abcdef"},
			"secret":      {"testsecret"},
			"name":        {""},
			"hardware_id": {""},
			"mac_address": {""},
			"model":       {""},
			"os":          {""},
			"system":      {""},
		}.Encode())
		event := core.RequestEvent{}
		event.Request, err = http.NewRequest("POST", "/controller/register/", reqbody)
		event.Request.Header.Set("content-type", "application/x-www-form-urlencoded")
		event.App = app
		rec := httptest.NewRecorder()
		event.Response = rec

		err = handleDeviceRegistration(&event, "testsecret", true)
		assert.Equal(t, nil, err)

		// Check the response
		httpResponse := rec.Result()

		defer httpResponse.Body.Close()
		body, err := io.ReadAll(httpResponse.Body)
		assert.Equal(t, nil, err)
		bodystring := string(body)
		re := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
		extracted_uuid = re.FindString(bodystring)
		normalized := re.ReplaceAllString(bodystring, "44ee4fee-1a20-470e-9044-ad9df88a0889")

		assert.Equal(t, `registration-result: success
uuid: 44ee4fee-1a20-470e-9044-ad9df88a0889
key: 0123456789abcdef0123456789abcdef
hostname: 
is-new: 1
`, normalized)
		record, err := app.FindRecordById("devices", pbID)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, record.GetBool("enabled"))
	}

	// Reregistration of the same device, ensure enabled flag is not overwritten
	{
		pbID, err := hexToPocketBaseID("0123456789abcdef0123456789abcdef")
		assert.Equal(t, nil, err)
		reqbody := strings.NewReader(url.Values{
			"backend":     {"netjsonconfig.OpenWrt"},
			"key":         {"0123456789abcdef0123456789abcdef"},
			"secret":      {"testsecret"},
			"name":        {""},
			"hardware_id": {""},
			"mac_address": {""},
			"model":       {""},
			"os":          {""},
			"system":      {""},
		}.Encode())
		event := core.RequestEvent{}
		event.Request, err = http.NewRequest("POST", "/controller/register/", reqbody)
		assert.Equal(t, nil, err)
		event.Request.Header.Set("content-type", "application/x-www-form-urlencoded")
		event.App = app
		rec := httptest.NewRecorder()
		event.Response = rec

		err = handleDeviceRegistration(&event, "testsecret", false)
		assert.Equal(t, nil, err)

		// Check the response
		httpResponse := rec.Result()

		defer httpResponse.Body.Close()
		body, err := io.ReadAll(httpResponse.Body)
		assert.Equal(t, nil, err)
		bodystring := string(body)
		normalized := strings.ReplaceAll(bodystring, extracted_uuid, "44ee4fee-1a20-470e-9044-ad9df88a0889")

		assert.Equal(t, `registration-result: success
uuid: 44ee4fee-1a20-470e-9044-ad9df88a0889
key: 0123456789abcdef0123456789abcdef
hostname: 
is-new: 0
`, normalized)

		record, err := app.FindRecordById("devices", pbID)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, record.GetBool("enabled"))
	}

	// Registration of the another device, set enabled flag to false
	{
		pbID, err := hexToPocketBaseID("ffffffffffffffffffffffffffffffff")
		reqbody := strings.NewReader(url.Values{
			"backend":     {"netjsonconfig.OpenWrt"},
			"key":         {"ffffffffffffffffffffffffffffffff"},
			"secret":      {"testsecret"},
			"name":        {""},
			"hardware_id": {""},
			"mac_address": {""},
			"model":       {""},
			"os":          {""},
			"system":      {""},
		}.Encode())
		event := core.RequestEvent{}
		event.Request, err = http.NewRequest("POST", "/controller/register/", reqbody)
		event.Request.Header.Set("content-type", "application/x-www-form-urlencoded")
		event.App = app
		rec := httptest.NewRecorder()
		event.Response = rec

		err = handleDeviceRegistration(&event, "testsecret", false)
		assert.Equal(t, nil, err)

		// Check the response
		httpResponse := rec.Result()

		defer httpResponse.Body.Close()
		body, err := io.ReadAll(httpResponse.Body)
		assert.Equal(t, nil, err)
		bodystring := string(body)
		// Replace the regex with a fixed one

		re := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
		new_extracted_uuid := re.FindString(bodystring)
		assert.NotEqual(t, new_extracted_uuid, extracted_uuid)
		normalized := strings.ReplaceAll(bodystring, new_extracted_uuid, "44ee4fee-1a20-470e-9044-ad9df88a0889")

		assert.Equal(t, `registration-result: success
uuid: 44ee4fee-1a20-470e-9044-ad9df88a0889
key: ffffffffffffffffffffffffffffffff
hostname: 
is-new: 1
`, normalized)
		record, err := app.FindRecordById("devices", pbID)
		assert.Equal(t, nil, err)
		assert.Equal(t, false, record.GetBool("enabled"))
	}
}

// Test that the default VLAN is present
func TestInterfacesConfigDefaultNoVLAN(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("ip_address", "8.8.8.8")
	err := app.Save(d1)
	assert.Equal(t, nil, err)
	// VLAN config does not always properly revert, so leave it in place even when disabled
	//	assert.Equal(t, `
	//config interface 'lan'
	//        option device 'br-lan'
	//`, generateInterfacesConfig(app, d1))
	assert.Equal(t, ``, generateInterfacesConfig(app, d1))
}

func TestGeneratePortTaggingConfig(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, vlancollection)

	portsconfig := []PortTaggingConfig{}

	//Test empty
	assert.Equal(t, "", generatePortTaggingConfig(app, portsconfig /*, "u*", "unused"*/))

	e1 := core.NewRecord(ethernetcollection)
	e1.Id = "somethindevice1"
	e1.Set("name", "lan1")
	e1.Set("speed", "1000F")
	err := app.Save(e1) // Saving is not really required
	assert.Equal(t, nil, err)
	portsconfig = []PortTaggingConfig{{Port: "lan1", Mode: "u*"}}

	assert.Equal(t, "        list ports 'lan1:u*'\n", generatePortTaggingConfig(app, portsconfig /*, []*core.Record{e1}, "u*", "unused"*/))

	e2 := core.NewRecord(ethernetcollection)
	e2.Id = "somethindevice2"
	e2.Set("name", "lan2")
	e2.Set("speed", "1000F")
	err = app.Save(e2) // Saving is not really required
	assert.Equal(t, nil, err)

	// Wrong order, expect sorting to maintain a clean config
	portsconfig = []PortTaggingConfig{{Port: "lan2", Mode: "t"}, {Port: "lan1", Mode: "t"}}
	assert.Equal(t, "        list ports 'lan1:t'\n        list ports 'lan2:t'\n", generatePortTaggingConfig(app, portsconfig /*, []*core.Record{e2, e1}, "t", "unused"*/))
}

func TestGetPortTagConfigForVlan(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	_ = setupBridgesCollection(t, app, devicecollection, wificollection, ethernetcollection)
	expandVlanCollection(t, app, vlancollection, devicecollection)

	guest_vlan := core.NewRecord(vlancollection)
	guest_vlan.Set("name", "guest")
	guest_vlan.Set("number", "400")
	err = app.Save(guest_vlan)
	assert.Equal(t, nil, err)

	iot_vlan := core.NewRecord(vlancollection)
	iot_vlan.Set("name", "iot")
	iot_vlan.Set("number", "300")
	err = app.Save(iot_vlan)
	assert.Equal(t, nil, err)

	// Add an LAN vlan with device 2 as gateway
	lan_vlan := core.NewRecord(vlancollection)
	lan_vlan.Set("name", "lan")
	lan_vlan.Set("number", "200")
	err = app.Save(lan_vlan)
	assert.Equal(t, nil, err)

	// Configure a default
	defaultSettings := core.NewRecord(porttaggingcollection)
	defaultSettings.Id = "defaultportcfg1"
	defaultSettings.Set("name", "default")
	defaultSettings.Set("untagged", lan_vlan.Id)
	defaultSettings.Set("trunk", true)

	assert.Equal(t, "u*", getPortTagConfigForVlan(lan_vlan.Id, defaultSettings))
	assert.Equal(t, "t", getPortTagConfigForVlan(guest_vlan.Id, defaultSettings))
	assert.Equal(t, "t", getPortTagConfigForVlan(iot_vlan.Id, defaultSettings))

	defaultSettings.Set("untagged", nil)

	assert.Equal(t, "t", getPortTagConfigForVlan(lan_vlan.Id, defaultSettings))
	assert.Equal(t, "t", getPortTagConfigForVlan(guest_vlan.Id, defaultSettings))
	assert.Equal(t, "t", getPortTagConfigForVlan(iot_vlan.Id, defaultSettings))

	defaultSettings.Set("trunk", false)

	assert.Equal(t, "", getPortTagConfigForVlan(lan_vlan.Id, defaultSettings))
	assert.Equal(t, "", getPortTagConfigForVlan(guest_vlan.Id, defaultSettings))
	assert.Equal(t, "", getPortTagConfigForVlan(iot_vlan.Id, defaultSettings))

	defaultSettings.Set("untagged", guest_vlan.Id)

	assert.Equal(t, "", getPortTagConfigForVlan(lan_vlan.Id, defaultSettings))
	assert.Equal(t, "u*", getPortTagConfigForVlan(guest_vlan.Id, defaultSettings))
	assert.Equal(t, "", getPortTagConfigForVlan(iot_vlan.Id, defaultSettings))

	// Configuration prefers untagged

	defaultSettings.Set("tagged", []string{guest_vlan.Id, iot_vlan.Id, lan_vlan.Id})

	assert.Equal(t, "t", getPortTagConfigForVlan(lan_vlan.Id, defaultSettings))
	assert.Equal(t, "u*", getPortTagConfigForVlan(guest_vlan.Id, defaultSettings))
	assert.Equal(t, "t", getPortTagConfigForVlan(iot_vlan.Id, defaultSettings))
}

func TestValidateRadioFrequencyBandCombo(t *testing.T) {
	// Valid combinations
	assert.Nil(t, validateRadioFrequencyBandCombo("2.4", "2412"))
	assert.Nil(t, validateRadioFrequencyBandCombo("5", "5180"))
	assert.Nil(t, validateRadioFrequencyBandCombo("6", "5995"))
	// Invalid combinations
	assert.Error(t, validateRadioFrequencyBandCombo("5", "2180"))
	assert.Error(t, validateRadioFrequencyBandCombo("2.4", "5955"))

	// Invalid input
	assert.Error(t, validateRadioFrequencyBandCombo("7", "5955"))
	assert.Error(t, validateRadioFrequencyBandCombo("5", "1000"))
}

func TestValidateRadioHtModeBandCombo(t *testing.T) {
	// Valid combinations
	assert.Nil(t, validateRadioHtModeBandCombo("2.4", "HT20"))
	assert.Nil(t, validateRadioHtModeBandCombo("2.4", "HT40"))
	assert.Nil(t, validateRadioHtModeBandCombo("5", "HT20"))
	assert.Nil(t, validateRadioHtModeBandCombo("5", "HT40"))
	assert.Nil(t, validateRadioHtModeBandCombo("5", "VHT20"))
	assert.Nil(t, validateRadioHtModeBandCombo("5", "VHT40"))
	assert.Nil(t, validateRadioHtModeBandCombo("5", "VHT80"))
	assert.Nil(t, validateRadioHtModeBandCombo("5", "VHT160"))
	assert.Nil(t, validateRadioHtModeBandCombo("6", "HE20"))
	assert.Nil(t, validateRadioHtModeBandCombo("6", "HE40"))
	assert.Nil(t, validateRadioHtModeBandCombo("6", "HE80"))
	assert.Nil(t, validateRadioHtModeBandCombo("6", "HE160"))
	// Invalid combinations
	assert.Error(t, validateRadioHtModeBandCombo("5", "HE20"))
	assert.Error(t, validateRadioHtModeBandCombo("2.4", "VHT40"))
	assert.Error(t, validateRadioHtModeBandCombo("6", "VHT40"))

	// Invalid input
	assert.Error(t, validateRadioHtModeBandCombo("60", "HE160"))
	assert.Error(t, validateRadioHtModeBandCombo("5", "HT1000"))
}

func TestValidateRadio(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	radiocollection := setupRadioCollection(t, app, devicecollection)

	r := core.NewRecord(radiocollection)
	r.Set("frequency", "2412")
	r.Set("band", "2.4")
	r.Set("ht_mode", "HT40")

	assert.Nil(t, validateRadio(r))

	r.Set("ht_mode", "VHT40")
	assert.Error(t, validateRadio(r))

	r.Set("frequency", "5180")
	assert.Error(t, validateRadio(r))

	r.Set("band", "5")
	assert.Nil(t, validateRadio(r))
}

func TestValidateSetting(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	settingscollection := setupSettingsCollection(t, app)

	s := core.NewRecord(settingscollection)
	s.Set("name", "country")
	s.Set("value", "")
	assert.Nil(t, validateSetting(s))

	s.Set("value", "BE")
	assert.Nil(t, validateSetting(s))

	s.Set("value", "XX")
	assert.Error(t, validateSetting(s))
}

// Test making a full map with the port tagging config
func TestGenerateFullTaggingMap(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	bridgecollection := setupBridgesCollection(t, app, devicecollection, wificollection, ethernetcollection)
	expandVlanCollection(t, app, vlancollection, devicecollection)

	e1 := core.NewRecord(ethernetcollection)
	e1.Id = "somethindeveth1"
	e1.Set("name", "lan1")
	e1.Set("speed", "1000F")
	err := app.Save(e1) // Saving is not really required
	assert.Equal(t, nil, err)

	e2 := core.NewRecord(ethernetcollection)
	e2.Id = "somethindeveth2"
	e2.Set("name", "lan2")
	e2.Set("speed", "1000F")
	err = app.Save(e2) // Saving is not really required
	assert.Equal(t, nil, err)

	e3 := core.NewRecord(ethernetcollection)
	e3.Id = "somethindeveth3"
	e3.Set("name", "lan3")
	e3.Set("speed", "1000F")
	err = app.Save(e3) // Saving is not really required
	assert.Equal(t, nil, err)

	iot_vlan := core.NewRecord(vlancollection)
	iot_vlan.Id = "zzzzziotvlan300"
	iot_vlan.Set("name", "iot")
	iot_vlan.Set("number", "300")
	err = app.Save(iot_vlan)
	assert.Equal(t, nil, err)

	// Add an LAN vlan with device 2 as gateway
	lan_vlan := core.NewRecord(vlancollection)
	lan_vlan.Id = "zzzzzlanvlan200"
	lan_vlan.Set("name", "lan")
	lan_vlan.Set("number", "200")
	err = app.Save(lan_vlan)
	assert.Equal(t, nil, err)

	b1 := core.NewRecord(bridgecollection)
	err = app.Save(b1) // Saving is not really required
	assert.Equal(t, nil, err)

	{ // No custom config
		tagmap := generateFullTaggingMap(app, []*core.Record{e1, e2, e3}, []*core.Record{iot_vlan, lan_vlan})
		expected := map[string][]PortTaggingConfig{
			"zzzzzlanvlan200": {
				{Port: "lan1", Mode: "u*"},
				{Port: "lan2", Mode: "u*"},
				{Port: "lan3", Mode: "u*"},
			},
			"zzzzziotvlan300": {
				{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "t"},
			},
		}
		assert.Equal(t, expected, tagmap)
	}

	// Set lan3 to iot untagged only
	iot_only_config := core.NewRecord(porttaggingcollection)
	iot_only_config.Id = "somethinconfig1"
	iot_only_config.Set("name", "iot_only")
	iot_only_config.Set("untagged", iot_vlan.Id)
	err = app.Save(iot_only_config)
	assert.Equal(t, nil, err)

	e3.Set("config", iot_only_config.Id)
	err = app.Save(e3)
	assert.Nil(t, err)

	tagmap := generateFullTaggingMap(app, []*core.Record{e1, e2, e3}, []*core.Record{iot_vlan, lan_vlan})
	expected := map[string][]PortTaggingConfig{
		"zzzzzlanvlan200": {
			{Port: "lan1", Mode: "u*"},
			{Port: "lan2", Mode: "u*"},
			{Port: "lan3", Mode: ""},
		},
		"zzzzziotvlan300": {
			{Port: "lan1", Mode: "t"},
			{Port: "lan2", Mode: "t"},
			{Port: "lan3", Mode: "u*"},
		},
	}
	assert.Equal(t, expected, tagmap)

	// Enable trunk
	iot_only_config.Set("trunk", true)
	err = app.Save(iot_only_config)
	assert.Equal(t, nil, err)

	{
		tagmap := generateFullTaggingMap(app, []*core.Record{e1, e2, e3}, []*core.Record{iot_vlan, lan_vlan})
		expected := map[string][]PortTaggingConfig{
			"zzzzzlanvlan200": {
				{Port: "lan1", Mode: "u*"},
				{Port: "lan2", Mode: "u*"},
				{Port: "lan3", Mode: "t"},
			},
			"zzzzziotvlan300": {
				{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "u*"},
			},
		}
		assert.Equal(t, expected, tagmap)
	}

	// Add an extra guest VLAN
	guest_vlan := core.NewRecord(vlancollection)
	guest_vlan.Id = "zzzguestvlan400"
	guest_vlan.Set("name", "guest")
	guest_vlan.Set("number", "400")
	err = app.Save(guest_vlan)
	assert.Equal(t, nil, err)
	{
		tagmap := generateFullTaggingMap(app, []*core.Record{e1, e2, e3}, []*core.Record{iot_vlan, lan_vlan, guest_vlan})
		expected := map[string][]PortTaggingConfig{
			"zzzzzlanvlan200": {
				{Port: "lan1", Mode: "u*"},
				{Port: "lan2", Mode: "u*"},
				{Port: "lan3", Mode: "t"},
			},
			"zzzzziotvlan300": {
				{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "u*"},
			},
			"zzzguestvlan400": {
				{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "t"},
			},
		}
		assert.Equal(t, expected, tagmap)
	}

	// Add disabled lan4 port

	disabled_config := core.NewRecord(porttaggingcollection)
	disabled_config.Id = "somethinconfig2"
	disabled_config.Set("name", "disabled")
	err = app.Save(disabled_config)
	assert.Equal(t, nil, err)

	e4 := core.NewRecord(ethernetcollection)
	e4.Id = "somethindeveth4"
	e4.Set("name", "lan4")
	e4.Set("speed", "1000F")
	e4.Set("config", disabled_config.Id)
	err = app.Save(e4)
	assert.Nil(t, err)

	{
		tagmap := generateFullTaggingMap(app, []*core.Record{e1, e2, e3, e4}, []*core.Record{iot_vlan, lan_vlan, guest_vlan})
		expected := map[string][]PortTaggingConfig{
			"zzzzzlanvlan200": {
				{Port: "lan1", Mode: "u*"},
				{Port: "lan2", Mode: "u*"},
				{Port: "lan3", Mode: "t"},
				{Port: "lan4", Mode: ""},
			},
			"zzzzziotvlan300": {
				{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "u*"},
				{Port: "lan4", Mode: ""},
			},
			"zzzguestvlan400": {
				{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "t"},
				{Port: "lan4", Mode: ""},
			},
		}
		assert.Equal(t, expected, tagmap)
	}

	// Switch lan4 to the iot config
	e4.Set("config", iot_only_config.Id)
	err = app.Save(e4)
	assert.Nil(t, err)

	{
		tagmap := generateFullTaggingMap(app, []*core.Record{ /*e1, e2, e3, */ e4}, []*core.Record{iot_vlan, lan_vlan, guest_vlan})
		expected := map[string][]PortTaggingConfig{
			"zzzzzlanvlan200": {
				/*{Port: "lan1", Mode: "u*"},
				{Port: "lan2", Mode: "u*"},
				{Port: "lan3", Mode: "t"},*/
				{Port: "lan4", Mode: "t"},
			},
			"zzzzziotvlan300": {
				/*{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "u*"},*/
				{Port: "lan4", Mode: "u*"},
			},
			"zzzguestvlan400": {
				/*{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "t"},*/
				{Port: "lan4", Mode: "t"},
			},
		}
		assert.Equal(t, expected, tagmap)
	}

	// Verify that the tagging prioritizes untagged
	guest_untagged_config := core.NewRecord(porttaggingcollection)
	guest_untagged_config.Id = "somethinconfig3"
	guest_untagged_config.Set("name", "untagged_guest")
	guest_untagged_config.Set("untagged", guest_vlan.Id)
	guest_untagged_config.Set("tagged", []string{guest_vlan.Id})
	err = app.Save(guest_untagged_config)
	assert.Equal(t, nil, err)

	// Switch lan4 to the iot config
	e4.Set("config", guest_untagged_config.Id)
	err = app.Save(e4)
	assert.Nil(t, err)

	{
		tagmap := generateFullTaggingMap(app, []*core.Record{ /*e1, e2, e3, */ e4}, []*core.Record{iot_vlan, lan_vlan, guest_vlan})
		expected := map[string][]PortTaggingConfig{
			"zzzzzlanvlan200": {
				/*{Port: "lan1", Mode: "u*"},
				{Port: "lan2", Mode: "u*"},
				{Port: "lan3", Mode: "t"},*/
				{Port: "lan4", Mode: ""},
			},
			"zzzzziotvlan300": {
				/*{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "u*"},*/
				{Port: "lan4", Mode: ""},
			},
			"zzzguestvlan400": {
				/*{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "t"},*/
				{Port: "lan4", Mode: "u*"},
			},
		}
		assert.Equal(t, expected, tagmap)
	}

	guest_untagged_config.Set("+tagged", []string{iot_vlan.Id, lan_vlan.Id})
	err = app.Save(guest_untagged_config)
	assert.Equal(t, nil, err)

	{
		tagmap := generateFullTaggingMap(app, []*core.Record{ /*e1, e2, e3, */ e4}, []*core.Record{iot_vlan, lan_vlan, guest_vlan})
		expected := map[string][]PortTaggingConfig{
			"zzzzzlanvlan200": {
				/*{Port: "lan1", Mode: "u*"},
				{Port: "lan2", Mode: "u*"},
				{Port: "lan3", Mode: "t"},*/
				{Port: "lan4", Mode: "t"},
			},
			"zzzzziotvlan300": {
				/*{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "u*"},*/
				{Port: "lan4", Mode: "t"},
			},
			"zzzguestvlan400": {
				/*{Port: "lan1", Mode: "t"},
				{Port: "lan2", Mode: "t"},
				{Port: "lan3", Mode: "t"},*/
				{Port: "lan4", Mode: "u*"},
			},
		}
		assert.Equal(t, expected, tagmap)
	}

}

func TestGenerateInterfaceVlanConfigInt(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	bridgecollection := setupBridgesCollection(t, app, devicecollection, wificollection, ethernetcollection)

	e1 := core.NewRecord(ethernetcollection)
	e1.Id = "somethindevice1"
	e1.Set("name", "lan1")
	e1.Set("speed", "1000F")
	err := app.Save(e1) // Saving is not really required
	assert.Equal(t, nil, err)

	e2 := core.NewRecord(ethernetcollection)
	e2.Id = "somethindevice2"
	e2.Set("name", "lan2")
	e2.Set("speed", "1000F")
	err = app.Save(e2) // Saving is not really required
	assert.Equal(t, nil, err)

	b1 := core.NewRecord(bridgecollection)
	err = app.Save(b1) // Saving is not really required
	assert.Equal(t, nil, err)

	configMap := make([]PortTaggingConfig, 0)

	// Test empty (maybe don't configure it in this case?)
	assert.Equal(t, `
config interface 'iot'
        option device 'br-lan.123'
        option proto 'none'

config bridge-vlan 'bridge_vlan_123'
        option device 'br-lan'
        option vlan '123'
`, generateInterfaceVlanConfigInt(app, b1, "iot", 123, "", configMap))

	// Common config with two ethernet ports on this bridge
	b1.Set("ethernet", []string{e1.Id, e2.Id})
	err = app.Save(b1)
	assert.Equal(t, nil, err)

	// lan3 should be ignored
	configMap = []PortTaggingConfig{{Port: "lan1", Mode: "t"}, {Port: "lan2", Mode: "t"}, {Port: "lan3", Mode: ""}}

	assert.Equal(t, `
config interface 'iot'
        option device 'br-lan.456'
        option proto 'none'

config bridge-vlan 'bridge_vlan_456'
        option device 'br-lan'
        option vlan '456'
        list ports 'lan1:t'
        list ports 'lan2:t'
`, generateInterfaceVlanConfigInt(app, b1, "iot", 456, "", configMap))

	configMap = []PortTaggingConfig{{Port: "lan1", Mode: "u*"}, {Port: "lan2", Mode: "u*"}}

	// lan should be untagged
	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.4000'

config bridge-vlan 'bridge_vlan_4000'
        option device 'br-lan'
        option vlan '4000'
        list ports 'lan1:u*'
        list ports 'lan2:u*'
`, generateInterfaceVlanConfigInt(app, b1, "lan", 4000, "", configMap))

	// Don't generate configs with invalid vlan ids
	assert.Equal(t, "", generateInterfaceVlanConfigInt(app, b1, "iot", -1, "", configMap))
	assert.Equal(t, "", generateInterfaceVlanConfigInt(app, b1, "iot", 0, "", configMap))
	assert.Equal(t, "", generateInterfaceVlanConfigInt(app, b1, "iot", 4095, "", configMap))
	assert.Equal(t, "", generateInterfaceVlanConfigInt(app, b1, "iot", 100000, "", configMap))

	// Don't generate configs with empty vlan name
	assert.Equal(t, "", generateInterfaceVlanConfigInt(app, b1, "", 100, "", configMap))
}

func TestGenerateInterfaceVlanConfigIntWithCIDR(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	bridgecollection := setupBridgesCollection(t, app, devicecollection, wificollection, ethernetcollection)

	emptyMap := make([]PortTaggingConfig, 0)

	e1 := core.NewRecord(ethernetcollection)
	e1.Id = "somethindevice1"
	e1.Set("name", "lan1")
	e1.Set("speed", "1000F")
	err := app.Save(e1) // Saving is not really required
	assert.Equal(t, nil, err)

	e2 := core.NewRecord(ethernetcollection)
	e2.Id = "somethindevice2"
	e2.Set("name", "lan2")
	e2.Set("speed", "1000F")
	err = app.Save(e2) // Saving is not really required
	assert.Equal(t, nil, err)

	b1 := core.NewRecord(bridgecollection)
	err = app.Save(b1) // Saving is not really required
	assert.Equal(t, nil, err)
	assert.Equal(t, `
config interface 'guest'
        option device 'br-lan.100'
        option proto 'static'
        option ipaddr '192.168.1.1'
        option netmask '255.255.255.0'

config bridge-vlan 'bridge_vlan_100'
        option device 'br-lan'
        option vlan '100'
`, generateInterfaceVlanConfigInt(app, b1, "guest", 100, "192.168.1.1/24", emptyMap))
	assert.Equal(t, `
config interface 'iot'
        option device 'br-lan.100'
        option proto 'static'
        option ipaddr '10.11.12.13'
        option netmask '255.255.128.0'

config bridge-vlan 'bridge_vlan_100'
        option device 'br-lan'
        option vlan '100'
`, generateInterfaceVlanConfigInt(app, b1, "iot", 100, "10.11.12.13/17", emptyMap))

	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.100'

config bridge-vlan 'bridge_vlan_100'
        option device 'br-lan'
        option vlan '100'
`, generateInterfaceVlanConfigInt(app, b1, "lan", 100, "10.11.12.13/17", emptyMap), "lan network should never be reconfigured")
}

func TestGenerateInterfaceVlanConfigForGateway(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	bridgecollection := setupBridgesCollection(t, app, devicecollection, wificollection, ethernetcollection)
	expandVlanCollection(t, app, vlancollection, devicecollection)

	b1 := core.NewRecord(bridgecollection)
	err := app.Save(b1) // Saving is not really required
	assert.Equal(t, nil, err)

	// Add two devices
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("apply", []string{"vlan"})
	d1.Set("ip_address", "8.8.8.8")
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	// Add two devices
	d2 := core.NewRecord(devicecollection)
	d2.Id = "somethindevice2"
	d2.Set("name", "the_device2")
	d2.Set("health_status", "healthy")
	d2.Set("apply", []string{"vlan"})
	d2.Set("ip_address", "8.8.8.9")
	err = app.Save(d2)
	assert.Equal(t, nil, err)

	// Add an IOT vlan with device 2 as gateway
	iot_vlan := core.NewRecord(vlancollection)
	iot_vlan.Set("name", "iot")
	iot_vlan.Set("number", "300")
	iot_vlan.Set("cidr", "172.16.0.1/17")
	iot_vlan.Set("gateway", "somethindevice2")
	err = app.Save(iot_vlan)
	assert.Equal(t, nil, err)

	// Add an LAN vlan with device 2 as gateway
	lan_vlan := core.NewRecord(vlancollection)
	lan_vlan.Set("name", "lan")
	lan_vlan.Set("number", "200")
	lan_vlan.Set("cidr", "192.168.1.1/24")     //Should be ignored
	lan_vlan.Set("gateway", "somethindevice2") // Should be ignored
	err = app.Save(lan_vlan)
	assert.Equal(t, nil, err)

	emptymap := make([]PortTaggingConfig, 0)

	assert.Equal(t, `
config interface 'iot'
        option device 'br-lan.300'
        option proto 'none'

config bridge-vlan 'bridge_vlan_300'
        option device 'br-lan'
        option vlan '300'
`, generateInterfaceVlanConfig(app, d1, b1, iot_vlan, emptymap), "device1 should have proto=none")
	assert.Equal(t, `
config interface 'iot'
        option device 'br-lan.300'
        option proto 'static'
        option ipaddr '172.16.0.1'
        option netmask '255.255.128.0'

config bridge-vlan 'bridge_vlan_300'
        option device 'br-lan'
        option vlan '300'
`, generateInterfaceVlanConfig(app, d2, b1, iot_vlan, emptymap), "device2 should have proto=static")

	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.200'

config bridge-vlan 'bridge_vlan_200'
        option device 'br-lan'
        option vlan '200'
`, generateInterfaceVlanConfig(app, d1, b1, lan_vlan, emptymap), "device1 should have no proto on lan")
	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.200'

config bridge-vlan 'bridge_vlan_200'
        option device 'br-lan'
        option vlan '200'
`, generateInterfaceVlanConfig(app, d2, b1, lan_vlan, emptymap), "device2 should have no proto on lan")

}

// Test that the default VLAN is present
func TestInterfacesConfigDefaultVLAN(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	bridgecollection := setupBridgesCollection(t, app, devicecollection, wificollection, ethernetcollection)
	expandVlanCollection(t, app, vlancollection, devicecollection)

	e1 := core.NewRecord(ethernetcollection)
	e1.Id = "somethindevice1"
	e1.Set("name", "lan1")
	e1.Set("speed", "1000F")
	err := app.Save(e1) // Saving is not really required
	assert.Equal(t, nil, err)

	e2 := core.NewRecord(ethernetcollection)
	e2.Id = "somethindevice2"
	e2.Set("name", "lan2")
	e2.Set("speed", "1000F")
	err = app.Save(e2) // Saving is not really required
	assert.Equal(t, nil, err)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("apply", []string{"vlan"})
	d1.Set("ip_address", "8.8.8.8")
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	lan := core.NewRecord(vlancollection)
	lan.Set("name", "lan")
	lan.Set("number", "100")
	err = app.Save(lan)
	assert.Equal(t, nil, err)

	// Configure a bridge for this device
	b1 := core.NewRecord(bridgecollection)
	b1.Set("device", d1.Id)
	b1.Set("name", "br-lan")
	b1.Set("ethernet", []string{e1.Id, e2.Id})
	err = app.Save(b1) // Saving is not really required
	assert.Equal(t, nil, err)

	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.100'

config bridge-vlan 'bridge_vlan_100'
        option device 'br-lan'
        option vlan '100'
        list ports 'lan1:u*'
        list ports 'lan2:u*'
`, generateInterfacesConfig(app, d1))
}

func TestGenerateDhcpConfigForDevice(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Equal(t, nil, err)
	defer app.Cleanup()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	bridgecollection := setupBridgesCollection(t, app, devicecollection, wificollection, ethernetcollection)
	expandVlanCollection(t, app, vlancollection, devicecollection)

	b1 := core.NewRecord(bridgecollection)
	err = app.Save(b1) // Saving is not really required
	assert.Equal(t, nil, err)

	// Add two devices
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("apply", []string{"vlan"})
	d1.Set("ip_address", "8.8.8.8")
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	// Add two devices
	d2 := core.NewRecord(devicecollection)
	d2.Id = "somethindevice2"
	d2.Set("name", "the_device2")
	d2.Set("health_status", "healthy")
	d2.Set("apply", []string{"vlan"})
	d2.Set("ip_address", "8.8.8.9")
	err = app.Save(d2)
	assert.Equal(t, nil, err)

	// Add an IOT vlan with device 2 as gateway
	iot_vlan := core.NewRecord(vlancollection)
	iot_vlan.Set("name", "guest")
	iot_vlan.Set("number", "300")
	iot_vlan.Set("cidr", "172.16.0.1/17")
	iot_vlan.Set("gateway", "somethindevice2")
	err = app.Save(iot_vlan)
	assert.Equal(t, nil, err)

	// Add an LAN vlan with device 2 as gateway, should be ignored
	lan_vlan := core.NewRecord(vlancollection)
	lan_vlan.Set("name", "lan")
	lan_vlan.Set("number", "200")
	lan_vlan.Set("cidr", "192.168.1.1/24")
	lan_vlan.Set("gateway", "somethindevice2")
	err = app.Save(lan_vlan)
	assert.Equal(t, nil, err)
	assert.Equal(t, ``, generateDhcpConfig(app, d1), "No DHCP config on a non-gateway device")
	assert.Equal(t, `
config dhcp 'guest'
        option interface 'guest'
        option start '100'
        option limit '150'
        option leasetime '12h'
`, generateDhcpConfig(app, d2))
	// Test some corner case with the internal function
	assert.Equal(t, ``, generateDhcpConfigForDeviceVLAN("", 24))
	assert.Equal(t, ``, generateDhcpConfigForDeviceVLAN("lan", 24))
	assert.Equal(t, ``, generateDhcpConfigForDeviceVLAN("wan", 24))
	assert.Equal(t, `
config dhcp 'dmz'
        option interface 'dmz'
        option start '100'
        option limit '150'
        option leasetime '12h'
`, generateDhcpConfigForDeviceVLAN("dmz", 24))
	// Test non-"/24" subnets
	assert.Equal(t, `
config dhcp 'guest'
        option interface 'guest'
        option start '100'
        option limit '150'
        option leasetime '12h'
`, generateDhcpConfigForDeviceVLAN("guest", 23))
	assert.Equal(t, ``, generateDhcpConfigForDeviceVLAN("guest", 25))
}

func TestInterfacesConfig(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	interfacescollection := setupInterfacesCollection(t, app)
	porttaggingcollection := setupPortTaggingCollection(t, app, vlancollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection, porttaggingcollection)
	bridgecollection := setupBridgesCollection(t, app, devicecollection, interfacescollection, ethernetcollection)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("ip_address", "8.8.8.8")
	err := app.Save(d1)
	assert.Equal(t, nil, err)

	wan := core.NewRecord(vlancollection)
	wan.Set("name", "wan")
	err = app.Save(wan)
	assert.Equal(t, nil, err)

	lan := core.NewRecord(vlancollection)
	lan.Set("name", "lan")
	lan.Set("number", "1000")
	err = app.Save(lan)
	assert.Equal(t, nil, err)

	guest := core.NewRecord(vlancollection)
	guest.Set("name", "guest")
	guest.Set("number", "7")
	guest.Set("subnet", "10.11.12.13")
	guest.Set("netmask", "255.255.128.0")
	err = app.Save(guest)
	assert.Equal(t, nil, err)

	iot := core.NewRecord(vlancollection)
	iot.Set("name", "iot")
	iot.Set("number", "123")
	iot.Set("subnet", "192.168.1.1")
	iot.Set("netmask", "255.255.255.00")
	err = app.Save(iot)
	assert.Equal(t, nil, err)

	e1 := core.NewRecord(ethernetcollection)
	e1.Id = "somethindevice1"
	e1.Set("name", "lan1")
	e1.Set("speed", "1000F")
	err = app.Save(e1) // Saving is not really required
	assert.Equal(t, nil, err)

	e2 := core.NewRecord(ethernetcollection)
	e2.Id = "somethindevice2"
	e2.Set("name", "lan2")
	e2.Set("speed", "1000F")
	err = app.Save(e2) // Saving is not really required
	assert.Equal(t, nil, err)

	// Configure a bridge for this device
	b1 := core.NewRecord(bridgecollection)
	b1.Set("device", d1.Id)
	b1.Set("name", "br-lan")
	b1.Set("ethernet", []string{e1.Id, e2.Id})
	err = app.Save(b1) // Saving is not really required
	assert.Equal(t, nil, err)

	// VLANs not enabled
	assert.Equal(t, /*`
		config interface 'lan'
		        option device 'br-lan'
		`*/"", generateInterfacesConfig(app, d1))

	// VLANs enabled
	d1.Set("apply", []string{"vlan"})
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.1000'

config bridge-vlan 'bridge_vlan_1000'
        option device 'br-lan'
        option vlan '1000'
        list ports 'lan1:u*'
        list ports 'lan2:u*'

config interface 'guest'
        option device 'br-lan.7'
        option proto 'none'

config bridge-vlan 'bridge_vlan_7'
        option device 'br-lan'
        option vlan '7'
        list ports 'lan1:t'
        list ports 'lan2:t'

config interface 'iot'
        option device 'br-lan.123'
        option proto 'none'

config bridge-vlan 'bridge_vlan_123'
        option device 'br-lan'
        option vlan '123'
        list ports 'lan1:t'
        list ports 'lan2:t'
`, generateInterfacesConfig(app, d1))
}

func TestUpdateMonitoring(t *testing.T) {
	json := `
{
  "type": "DeviceMonitoring",
  "general": {
    "local_time": 1000000000,
    "uptime": 2000000,
    "hostname": "OpenWRThostname"
  },
  "interfaces": [
    {
      "mac": "aa:bb:cc:dd:ee:ff",
      "type": "wireless",
      "mtu": 1500,
      "txqueuelen": 1000,
      "name": "phy1-ap0",
      "wireless": {
        "noise": -95,
        "ssid": "OpenWRT",
        "country": "US",
        "clients": [
          {
            "wps": false,
            "wds": false,
            "ht": true,
            "vht": false,
            "wmm": true,
            "aid": 1,
            "assoc": true,
            "bytes": {
              "rx": 1691924,
              "tx": 27379187
            },
            "capabilities": {},
            "mac": "11:22:33:44:55:66",
            "signal": -82,
            "rate": {
              "rx": 4330000,
              "tx": 8670000
            },
            "he": false,
            "rrm": [
              114,
              0,
              0,
              0,
              0
            ],
            "packets": {
              "rx": 6190,
              "tx": 20992
            },
            "airtime": {
              "rx": 557004,
              "tx": 4863610
            },
            "authorized": true,
            "extended_capabilities": [
              4,
              0,
              136,
              128,
              1,
              64,
              0,
              192,
              0,
              0
            ],
            "preauth": false,
            "mbo": false,
            "mfp": false,
            "auth": true
          }
        ],
        "signal": -82,
        "bitrate": 86700,
        "quality_max": 70,
        "quality": 28,
        "channel": 11,
        "tx_power": 22,
        "mode": "access_point",
        "htmode": "HT20",
        "frequency": 2462
      }
    }
  ]
}`
	var err error
	app, _ := tests.NewTestApp()
	event := core.RequestEvent{}
	event.Request, err = http.NewRequest("POST", "/api/v1/monitoring/device/", strings.NewReader(json))
	assert.Equal(t, err, nil)
	event.Request.SetPathValue("key", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	event.Request.Header.Set("content-type", "application/json")
	event.App = app
	rec := httptest.NewRecorder()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	clientcollection := setupClientsCollection(t, app)
	devicecollection := setupDeviceCollection(t, app, wificollection)

	// Add a device
	d := core.NewRecord(devicecollection)
	d.Set("name", "the_device1")
	d.Set("health_status", "healthy")
	err = app.Save(d)
	assert.Equal(t, nil, err)

	event.Response = rec

	// Verify the response
	response, radios := handleMonitoring(&event, app, d, clientcollection)
	//var apiresponse *router.ApiError
	assert.Equal(t, response, nil)
	assert.NotEqual(t, radios, nil)
	httpResponse := rec.Result()
	defer httpResponse.Body.Close()
	body, err := io.ReadAll(httpResponse.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, 200, httpResponse.StatusCode)
	assert.Equal(t, "", string(body))

	// Verify the client data
	clients, err := app.FindAllRecords("clients2")
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, clients)
	assert.Equal(t, 1, len(clients))
	client := clients[0]
	assert.Equal(t, "11:22:33:44:55:66", client.GetString("mac_address"))
	assert.Equal(t, "phy1-ap0", client.GetString("connected_to_hostname"))
	assert.Equal(t, -82, client.GetInt("signal"))
	assert.Equal(t, 2462, client.GetInt("frequency"))
	assert.Equal(t, 11, client.GetInt("channel"))
	assert.NotEqual(t, "", client.GetString("device"))

	// Verify the radio data
	assert.NotEqual(t, nil, radios)
	assert.Equal(t, 1, len(radios))
	fmt.Println(radios)
	radio := radios[1]
	assert.NotEqual(t, nil, radio)
	assert.Equal(t, Radio{Frequency: 2462, Channel: 11, HTmode: "HT20", TxPower: 22, MAC: "aa:bb:cc:dd:ee:ff"}, radio)
}

// Verify an empty monitoring request is correctly ignored
func TestUpdateMonitoringEmptyBody(t *testing.T) {
	var err error
	app, _ := tests.NewTestApp()
	event := core.RequestEvent{}
	event.Request, err = http.NewRequest("POST", "/api/v1/monitoring/device/", strings.NewReader(""))
	assert.Equal(t, err, nil)
	event.Request.SetPathValue("key", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	event.Request.Header.Set("content-type", "application/json")
	// Write this in lower case to verify the case insensitivity
	event.Request.Header.Set("content-length", "0")
	event.App = app
	rec := httptest.NewRecorder()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	clientcollection := setupClientsCollection(t, app)
	devicecollection := setupDeviceCollection(t, app, wificollection)

	// Add a device
	d := core.NewRecord(devicecollection)
	d.Set("name", "the_device1")
	d.Set("health_status", "healthy")
	err = app.Save(d)
	assert.Equal(t, nil, err)

	event.Response = rec

	// Verify the valid, but empty response
	response, radios := handleMonitoring(&event, app, d, clientcollection)
	assert.Equal(t, response, nil)
	assert.NotEqual(t, radios, nil)
	httpResponse := rec.Result()
	defer httpResponse.Body.Close()
	body, err := io.ReadAll(httpResponse.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, 200, httpResponse.StatusCode)
	assert.Equal(t, "", string(body))
}

func TestUpdateInterface(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)

	// Add a vlan
	v := core.NewRecord(vlancollection)
	v.Set("name", "wan")
	err := app.Save(v)
	assert.Equal(t, nil, err)

	// Add a wifi
	{
		w := core.NewRecord(wificollection)
		w.Id = "somethingabcdef"
		w.Set("ssid", "OpenWRT1")
		w.Set("key", "the_key")
		w.Set("ieee80211r", true)
		w.Set("encryption", "the_encryption")
		err = app.Save(w)
		assert.Equal(t, nil, err)
	}

	// Add a second wifi
	{
		w := core.NewRecord(wificollection)
		w.Id = "fffffffffffffff"
		w.Set("ssid", "OpenWRT2")
		w.Set("key", "the_key")
		w.Set("ieee80211r", true)
		w.Set("encryption", "the_encryption")
		err = app.Save(w)
		assert.Equal(t, nil, err)
	}

	// Send a first record
	interfacesCollection := setupInterfacesCollection(t, app)
	w2 := Wireless{SSID: "OpenWRT1", Frequency: 2430}
	iface := Interface{MAC: "aa:bb:cc:dd:ee", Type: "Wireless", Name: "phy1-ap0", Wireless: &w2}
	deviceId := "something"
	err = updateInterface(app, iface, deviceId, interfacesCollection)
	assert.Equal(t, nil, err)

	// Verify the initial record
	interfaces, err := app.FindAllRecords("interfaces")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(interfaces))
	assert.Equal(t, "aa:bb:cc:dd:ee", interfaces[0].GetString("mac_address"))
	assert.Equal(t, "somethingabcdef", interfaces[0].GetString("wifi")) // Make this a reference>
	assert.Equal(t, "phy1-ap0", interfaces[0].GetString("interface"))
	assert.Equal(t, "2.4", interfaces[0].GetString("band"))
	created := interfaces[0].GetString("created")
	updated := interfaces[0].GetString("updated")
	assert.Equal(t, created, updated)

	// Send another, identical record
	err = updateInterface(app, iface, deviceId, interfacesCollection)

	interfaces, err = app.FindAllRecords("interfaces")
	assert.Equal(t, nil, err)
	updated = interfaces[0].GetString("updated")
	assert.Equal(t, created, updated)

	time.Sleep(1 * time.Millisecond)
	{
		// Update the frequency, but in the same frequency band
		w3 := Wireless{SSID: "OpenWRT1", Frequency: 2472}
		iface3 := Interface{MAC: "aa:bb:cc:dd:ee", Type: "Wireless", Name: "phy1-ap0", Wireless: &w3}
		err = updateInterface(app, iface3, deviceId, interfacesCollection)

		interfaces, err = app.FindAllRecords("interfaces")
		assert.Equal(t, nil, err)
		updated = interfaces[0].GetString("updated")
		// Record should not be updated
		assert.Equal(t, created, updated)
	}
	time.Sleep(1 * time.Millisecond)
	{
		// Update the mac
		w3 := Wireless{SSID: "OpenWRT1", Frequency: 2472}
		iface3 := Interface{MAC: "00:bb:cc:dd:ee", Type: "Wireless", Name: "phy1-ap0", Wireless: &w3}
		err = updateInterface(app, iface3, deviceId, interfacesCollection)

		interfaces, err = app.FindAllRecords("interfaces")
		assert.Equal(t, nil, err)
		updated = interfaces[0].GetString("updated")
		mac := interfaces[0].GetString("mac_address")
		// Record should be updated
		assert.NotEqual(t, created, updated)
		assert.Equal(t, "00:bb:cc:dd:ee", mac)
	}
	time.Sleep(1 * time.Millisecond)
	{
		// Update the SSID
		w3 := Wireless{SSID: "OpenWRT2", Frequency: 2472}
		iface3 := Interface{MAC: "00:bb:cc:dd:ee", Type: "Wireless", Name: "phy1-ap0", Wireless: &w3}
		err = updateInterface(app, iface3, deviceId, interfacesCollection)

		interfaces, err = app.FindAllRecords("interfaces")
		assert.Equal(t, nil, err)
		updated2 := interfaces[0].GetString("updated")
		mac := interfaces[0].GetString("mac_address")
		ssid := interfaces[0].GetString("wifi")
		// Record should be updated
		assert.NotEqual(t, created, updated2)
		assert.NotEqual(t, updated, updated2)
		assert.Equal(t, "00:bb:cc:dd:ee", mac)
		assert.Equal(t, "fffffffffffffff", ssid)
	}
}

func TestUpdateLastSeen(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	collection := core.NewBaseCollection("devices")
	collection.Fields.Add(&core.DateField{Name: "last_seen"})
	collection.Fields.Add(&core.SelectField{Name: "health_status", MaxSelect: 1, Values: []string{"unknown", "healthy", "critical"}})
	collection.Fields.Add(&core.TextField{Name: "ip_address"})
	err = app.Save(collection)
	assert.Nil(t, err)

	m := core.NewRecord(collection)
	m.Id = "testaidalongera"
	m.Set("health_status", "unknown")
	m.Set("ip_address", "0.0.0.0")
	assert.Equal(t, m.GetDateTime("last_seen"), types.DateTime{})
	err = app.Save(m)
	assert.Equal(t, err, nil)
	event := core.RequestEvent{}
	event.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	event.Request.RemoteAddr = "200.200.100.100:12345"
	event.App = app
	assert.Equal(t, "200.200.100.100", event.RealIP())

	updateLastSeen(&event, m)
	assert.NotEqual(t, m.GetDateTime("last_seen"), types.DateTime{})
	assert.WithinDuration(t, m.GetDateTime("last_seen").Time(), types.NowDateTime().Time(), 1*time.Second, "Last Seen should be updated")
	record, err := app.FindRecordById("devices", "testaidalongera")
	assert.Equal(t, "healthy", m.GetString("health_status"))
	assert.Equal(t, "200.200.100.100", m.GetString("ip_address"))

	// The record is newer than 60 seconds, so should remain healthy
	now := types.NowDateTime()
	now = now.Add(59 * time.Second)
	updateDeviceHealth(app, now)
	record, err = app.FindRecordById("devices", "testaidalongera")
	assert.Equal(t, err, nil)
	assert.Equal(t, "healthy", record.GetString("health_status"))
	assert.Equal(t, "200.200.100.100", m.GetString("ip_address"), "should not be updated")

	// The record should have been updated to unhealthy
	now = now.Add(2 * time.Second)
	updateDeviceHealth(app, now)
	record, err = app.FindRecordById("devices", "testaidalongera")
	assert.Equal(t, err, nil)
	assert.Equal(t, "unhealthy", record.GetString("health_status"))
	assert.Equal(t, "200.200.100.100", m.GetString("ip_address"), "should not be updated")
}

func TestExtractRadioNumber(t *testing.T) {
	tests := []struct {
		input       string
		expected    int
		expectError bool
	}{
		{"phy1-ap0", 1, false},
		{"phy0-ap0", 0, false},
		{"phy2-ap3", 2, false},
		{"invalid-string", 0, true}, // error case
	}

	for _, tt := range tests {
		result, err := extractRadioNumber(tt.input)
		assert.Equal(t, err != nil, tt.expectError)
		assert.Equal(t, result, tt.expected)
	}
}

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
		result := frequencyToBand(tt.input)
		assert.Equal(t, result, tt.expected)
	}
}

func TestUpdateRadios(t *testing.T) {
	radios := make(map[int]Radio)
	radios[0] = Radio{Frequency: 2412, Channel: 1, HTmode: "HT20", TxPower: 23}
	radios[1] = Radio{Frequency: 5200, Channel: 40, HTmode: "HT40", TxPower: 19}
	app, _ := tests.NewTestApp()

	// Create devices collection
	devicecollection := core.NewBaseCollection("devices")
	err := app.Save(devicecollection)
	assert.Equal(t, err, nil)

	setupRadioCollection(t, app, devicecollection)

	// Add a dummy radio
	d := core.NewRecord(devicecollection)
	app.Save(d)

	updateRadios(d, app, radios)
	radiocount, err := app.CountRecords("radios")
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(2), radiocount, "Both radios should have been added")
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "0")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 0)
		assert.Equal(t, r.GetInt("frequency"), 2412)
		assert.Equal(t, r.GetInt("channel"), 1)
		assert.Equal(t, r.GetString("band"), "2.4")
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("device"), d.GetString("id"))
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "1")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 1)
		assert.Equal(t, r.GetInt("frequency"), 5200)
		assert.Equal(t, r.GetInt("channel"), 40)
		assert.Equal(t, r.GetString("band"), "5")
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("device"), d.GetString("id"))
	}
	// Now do this again but only a partial update should happen
	radios[0] = Radio{Frequency: 2417, Channel: 1, HTmode: "HT20", TxPower: 23}
	radios[1] = Radio{Frequency: 5180, Channel: 40, HTmode: "HT40", TxPower: 19}
	radios[2] = Radio{Frequency: 5955, Channel: 100, HTmode: "HT40", TxPower: 19}
	updateRadios(d, app, radios)
	radiocount, err = app.CountRecords("radios")
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(3), radiocount, "Only the new radio should be added")
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "0")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 0)
		assert.Equal(t, r.GetInt("frequency"), 2412)
		assert.Equal(t, r.GetInt("channel"), 1)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "2.4")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "1")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 1)
		assert.Equal(t, r.GetInt("frequency"), 5200)
		assert.Equal(t, r.GetInt("channel"), 40)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "5")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "2")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 2)
		assert.Equal(t, r.GetInt("frequency"), 5955)
		assert.Equal(t, r.GetInt("channel"), 100)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "6")
	}

	radios2 := make(map[int]Radio)
	radios2[0] = Radio{Frequency: 2417, Channel: 1, HTmode: "HT20", TxPower: 23}
	radios2[2] = Radio{Frequency: 5955, Channel: 100, HTmode: "HT40", TxPower: 19}
	fmt.Println("---------------")
	// If a radio is not in the list, it is disabled. Mark it so in the DB
	updateRadios(d, app, radios2)
	radiocount, err = app.CountRecords("radios")
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(3), radiocount, "Disabled radio should not be removed")
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "0")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 0)
		assert.Equal(t, r.GetInt("frequency"), 2412)
		assert.Equal(t, r.GetInt("channel"), 1)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "2.4")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "1")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 1)
		assert.Equal(t, r.GetInt("frequency"), 5200)
		assert.Equal(t, r.GetInt("channel"), 40)
		assert.Equal(t, r.GetBool("enabled"), false)
		assert.Equal(t, r.GetString("band"), "5")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "2")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 2)
		assert.Equal(t, r.GetInt("frequency"), 5955)
		assert.Equal(t, r.GetInt("channel"), 100)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "6")
	}

	fmt.Println("---------------")
	// Re-add the disabled radio and verify it is marked as enabled again
	radios2[0] = Radio{Frequency: 2417, Channel: 1, HTmode: "HT20", TxPower: 23}
	radios2[1] = Radio{Frequency: 5180, Channel: 40, HTmode: "HT40", TxPower: 19}
	radios2[2] = Radio{Frequency: 5955, Channel: 100, HTmode: "HT40", TxPower: 19}
	updateRadios(d, app, radios2)
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "0")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 0)
		assert.Equal(t, r.GetInt("frequency"), 2412)
		assert.Equal(t, r.GetInt("channel"), 1)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "2.4")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "1")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 1)
		assert.Equal(t, r.GetInt("frequency"), 5200)
		assert.Equal(t, r.GetInt("channel"), 40)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "5")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "2")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 2)
		assert.Equal(t, r.GetInt("frequency"), 5955)
		assert.Equal(t, r.GetInt("channel"), 100)
		assert.Equal(t, r.GetBool("enabled"), true)
		assert.Equal(t, r.GetString("band"), "6")
	}
}

func TestFrequencyToChannel(t *testing.T) {
	tests := []struct {
		freq            int
		expectedChannel int
		expectedOk      bool
	}{
		{2412, 1, true}, // 2.4 GHz
		{2437, 6, true},
		{2484, 14, true},
		{5180, 36, true}, // 5 GHz
		{5200, 40, true},
		{5500, 100, true},
		{5825, 165, true},
		{5955, 1, true}, // 6 GHz
		{6110, 32, true},
		{7115, 233, true},
		{58320, 1, true}, // 60 GHz
		{60480, 2, true},
		{62640, 3, true},
		{64800, 4, true},
		{66960, 5, true},
		{69120, 6, true},
		{72000, 0, false}, // Invalid
	}

	for _, tt := range tests {
		result, ok := frequencyToChannel(tt.freq)
		assert.Equal(t, ok, tt.expectedOk, tt.freq)
		assert.Equal(t, result, tt.expectedChannel, tt.freq)
	}
}

func TestGenerateHostnameConfig(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "a3qnbxklglw121g"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("config_status", "healthy")
	d1.Set("key", "aaaabbbbccccddddaaaabbbbccccdddd")
	err := app.Save(d1)
	assert.Equal(t, nil, err)
	assert.Equal(t, `
config system 'system'
        option hostname 'the_device1'
`, generateHostnameConfig(d1))
}

func TestGenerateOpenWispConfig(t *testing.T) {
	assert.Equal(t, generateOpenWispConfig(), `
config controller 'http'
        option enabled 'monitoring'
        option interval '30'
`)
}

func TestGenerateMonitoringConfig(t *testing.T) {
	assert.Equal(t, generateMonitoringConfig(), `
config monitoring 'monitoring'
        option interval '15'
`)
}

func TestSshKeyConfig1(t *testing.T) {
	app, _ := tests.NewTestApp()
	collection := setupSshKeyCollection(t, app)

	key1 := core.NewRecord(collection)
	key1.Set("key", "ssh-key aaaaaa\r\n")
	err := app.Save(key1)
	assert.Equal(t, nil, err)

	key2 := core.NewRecord(collection)
	key2.Set("key", "ssh-key bbbbbb \n\r")
	err = app.Save(key2)
	assert.Equal(t, nil, err)

	key3 := core.NewRecord(collection)
	key3.Set("key", "ssh-key cccccc ")
	err = app.Save(key3)
	assert.Equal(t, nil, err)
	assert.Equal(t, "ssh-key aaaaaa\nssh-key bbbbbb\nssh-key cccccc\n", generateSshKeyConfig(app))
}

func TestSshKeyConfig2Empty(t *testing.T) {
	app, _ := tests.NewTestApp()
	setupSshKeyCollection(t, app)
	assert.Equal(t, "", generateSshKeyConfig(app))
}

func TestGenerateRadioConfig(t *testing.T) {
	app, _ := tests.NewTestApp()
	radiocollection := core.NewBaseCollection("radios")
	err := app.Save(radiocollection)
	assert.Equal(t, err, nil)
	record := core.NewRecord(radiocollection)
	record.Set("device", "something")
	record.Set("radio", "3")
	record.Set("channel", "5")
	record.Set("frequency", "5200")
	assert.Equal(t, generateRadioConfig(record, ""), `
config wifi-device 'radio3'
        option channel '40'
`)

	record.Set("auto_frequency", true)
	assert.Equal(t, generateRadioConfig(record, ""), `
config wifi-device 'radio3'
        option channel 'auto'
`)
	record.Set("ht_mode", "VHT20")
	assert.Equal(t, generateRadioConfig(record, ""), `
config wifi-device 'radio3'
        option channel 'auto'
        option htmode 'VHT20'
`)
	assert.Equal(t, generateRadioConfig(record, "FR"), `
config wifi-device 'radio3'
        option channel 'auto'
        option country 'FR'
        option htmode 'VHT20'
`)
}
func TestGenerateRadioConfigs(t *testing.T) {
	radios := make(map[int]Radio)
	radios[0] = Radio{Frequency: 2412, Channel: 1, HTmode: "HT20", TxPower: 23}
	radios[1] = Radio{Frequency: 5200, Channel: 40, HTmode: "HT40", TxPower: 19}
	app, _ := tests.NewTestApp()

	// Create devices collection
	devicecollection := core.NewBaseCollection("devices")
	err := app.Save(devicecollection)
	assert.Equal(t, err, nil)
	settingscollection := setupSettingsCollection(t, app)

	// Create radios collection
	radiocollection := core.NewBaseCollection("radios")
	x := 0.0
	radiocollection.Fields.Add(&core.NumberField{
		Name:     "radio",
		Required: false,
		Min:      &x,
		OnlyInt:  true,
	})
	radiocollection.Fields.Add(&core.RelationField{
		Name:          "device",
		Required:      false,
		MaxSelect:     1,
		CascadeDelete: false,
		CollectionId:  devicecollection.Id,
	})
	radiocollection.Fields.Add(&core.NumberField{
		Name:     "channel",
		Required: false,
		Min:      &x,
		OnlyInt:  true,
	})
	radiocollection.Fields.Add(&core.NumberField{
		Name:     "frequency",
		Required: false,
		Min:      &x,
		OnlyInt:  true,
	})
	radiocollection.Fields.Add(&core.TextField{
		Name:     "band",
		Required: false,
	})
	err = app.Save(radiocollection)
	assert.Equal(t, err, nil)

	// Add a dummy device
	d := core.NewRecord(devicecollection)
	app.Save(d)

	updateRadios(d, app, radios)
	radiocount, err := app.CountRecords("radios")
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(2), radiocount, "Both radios should have been added")

	assert.Equal(t, `
config wifi-device 'radio0'
        option channel '1'

config wifi-device 'radio1'
        option channel '40'
`, generateRadioConfigs(d, app))

	country := core.NewRecord(settingscollection)
	country.Set("name", "country")
	country.Set("value", "DE")
	app.Save(country)

	assert.Equal(t, `
config wifi-device 'radio0'
        option channel '1'
        option country 'DE'

config wifi-device 'radio1'
        option channel '40'
        option country 'DE'
`, generateRadioConfigs(d, app))

}

func setupSshKeyCollection(t *testing.T, app core.App) *core.Collection {
	sshkeycollection := core.NewBaseCollection("ssh_keys")
	sshkeycollection.Fields.Add(&core.TextField{
		Name:     "key",
		Required: true,
		Min:      10,
	})
	err := app.Save(sshkeycollection)
	assert.Equal(t, err, nil)
	return sshkeycollection
}

func setupRadioCollection(t *testing.T, app core.App, devicecollection *core.Collection) *core.Collection {
	radiocollection := core.NewBaseCollection("radios")
	x := 0.0
	radiocollection.Fields.Add(&core.NumberField{
		Name:     "radio",
		Required: false,
		Min:      &x,
		OnlyInt:  true,
	})
	radiocollection.Fields.Add(&core.RelationField{
		Name:          "device",
		Required:      false,
		MaxSelect:     1,
		CascadeDelete: false,
		CollectionId:  devicecollection.Id,
	})
	radiocollection.Fields.Add(&core.NumberField{
		Name:     "channel",
		Required: false,
		Min:      &x,
		OnlyInt:  true,
	})
	radiocollection.Fields.Add(&core.NumberField{
		Name:     "frequency",
		Required: false,
		Min:      &x,
		OnlyInt:  true,
	})
	radiocollection.Fields.Add(&core.TextField{
		Name:     "band",
		Required: false,
	})
	radiocollection.Fields.Add(&core.TextField{
		Name:     "ht_mode",
		Required: false,
	})
	radiocollection.Fields.Add(&core.BoolField{
		Name:     "auto_frequency",
		Required: false,
	})
	radiocollection.Fields.Add(&core.BoolField{
		Name:     "enabled",
		Required: false,
	})
	err := app.Save(radiocollection)
	assert.Equal(t, err, nil)
	return radiocollection

}
func setupDeviceCollection(t *testing.T, app core.App, wificollection *core.Collection) *core.Collection {
	devicecollection := core.NewBaseCollection("devices")
	devicecollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: false,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "health_status",
		Required: true,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "ip_address",
		Required: false, // True in the real collection
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "mac_address",
		Required: false, // True in the real collection
	})
	devicecollection.Fields.Add(&core.SelectField{
		Name:      "apply",
		MaxSelect: 1,
		Required:  false,
		Values:    []string{"vlan"},
	})
	devicecollection.Fields.Add(&core.RelationField{
		Name:         "wifis",
		MaxSelect:    99,
		Required:     false,
		CollectionId: wificollection.Id,
	})
	devicecollection.Fields.Add(&core.FileField{
		Name:      "config",
		Required:  false,
		Protected: true,
		MimeTypes: []string{"application/x-tar"},
		Hidden:    true,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "config_status",
		Required: false,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "error_reason",
		Required: false,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "key",
		Required: false,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "uuid",
		Required: false,
	})
	devicecollection.Fields.Add(&core.BoolField{
		Name:     "enabled",
		Required: false,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "os",
		Required: false,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "model",
		Required: false,
	})
	devicecollection.Fields.Add(&core.TextField{
		Name:     "system",
		Required: false,
	})
	err := app.Save(devicecollection)
	assert.Equal(t, err, nil)
	return devicecollection
}

func setupBridgesCollection(t *testing.T, app core.App, devicecollection *core.Collection, wificollection *core.Collection, ethernetcollection *core.Collection) *core.Collection {
	bridgescollection := core.NewBaseCollection("bridges")
	bridgescollection.Fields.Add(&core.RelationField{
		Name:          "device",
		Required:      false,
		MaxSelect:     1,
		CascadeDelete: false,
		CollectionId:  devicecollection.Id,
	})
	bridgescollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: false,
	})
	bridgescollection.Fields.Add(&core.NumberField{
		Name:     "tx_bytes",
		Required: false,
	})
	bridgescollection.Fields.Add(&core.NumberField{
		Name:     "rx_bytes",
		Required: false,
	})
	bridgescollection.Fields.Add(&core.RelationField{
		Name:          "ethernet",
		Required:      false,
		MaxSelect:     99,
		CascadeDelete: false,
		CollectionId:  ethernetcollection.Id,
	})
	bridgescollection.Fields.Add(&core.RelationField{
		Name:          "wifi",
		Required:      false,
		MaxSelect:     99,
		CascadeDelete: false,
		CollectionId:  wificollection.Id,
	})
	err := app.Save(bridgescollection)
	assert.Equal(t, err, nil)
	return bridgescollection
}

func setupEthernetCollection(t *testing.T, app core.App, devicecollection *core.Collection, porttaggingCollection *core.Collection) *core.Collection {
	ethernetcollection := core.NewBaseCollection("ethernet")
	ethernetcollection.Fields.Add(&core.RelationField{
		Name:          "device",
		Required:      false,
		MaxSelect:     1,
		CascadeDelete: false,
		CollectionId:  devicecollection.Id,
	})
	ethernetcollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: false,
	})
	ethernetcollection.Fields.Add(&core.RelationField{
		Name:          "config",
		Required:      false,
		MaxSelect:     1,
		CascadeDelete: false,
		CollectionId:  porttaggingCollection.Id,
	})
	ethernetcollection.Fields.Add(&core.TextField{
		Name:     "speed",
		Required: false,
	})
	ethernetcollection.Fields.Add(&core.NumberField{
		Name:     "tx_bytes",
		Required: false,
	})
	ethernetcollection.Fields.Add(&core.NumberField{
		Name:     "rx_bytes",
		Required: false,
	})
	err := app.Save(ethernetcollection)
	assert.Equal(t, err, nil)
	return ethernetcollection
}

func setupWifiCollection(t *testing.T, app core.App, vlancollection *core.Collection) *core.Collection {
	wificollection := core.NewBaseCollection("wifi")
	wificollection.Fields.Add(&core.TextField{
		Name:     "ssid",
		Required: true,
	})
	wificollection.Fields.Add(&core.TextField{
		Name:     "key",
		Required: true,
	})
	wificollection.Fields.Add(&core.TextField{
		Name:     "encryption",
		Required: true,
	})
	wificollection.Fields.Add(&core.BoolField{
		Name:     "ieee80211r",
		Required: true,
	})
	wificollection.Fields.Add(&core.NumberField{
		Name:     "ieee80211r_reassoc_deadline",
		Required: false,
	})
	wificollection.Fields.Add(&core.BoolField{
		Name:     "ieee80211v_bss_transition",
		Required: false,
	})
	wificollection.Fields.Add(&core.TextField{
		Name:     "ieee80211v_time_advertisement",
		Required: false,
	})
	wificollection.Fields.Add(&core.BoolField{
		Name:     "ieee80211v_proxy_arp",
		Required: false,
	})
	wificollection.Fields.Add(&core.BoolField{
		Name:     "ieee80211k",
		Required: false,
	})
	wificollection.Fields.Add(&core.RelationField{
		Name:         "network",
		MaxSelect:    1,
		Required:     false,
		CollectionId: vlancollection.Id,
	})
	wificollection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})

	err := app.Save(wificollection)
	assert.Equal(t, err, nil)
	return wificollection

}

func setupClientsCollection(t *testing.T, app core.App) *core.Collection {
	clientcollection := core.NewBaseCollection("clients2") // TODO figure out why we have a clash here
	clientcollection.Fields.Add(&core.TextField{
		Name:     "mac_address",
		Required: true,
	})
	clientcollection.Fields.Add(&core.TextField{
		Name:     "connected_to_hostname",
		Required: false,
	})
	clientcollection.Fields.Add(&core.NumberField{
		Name:     "signal",
		Required: false,
	})
	clientcollection.Fields.Add(&core.TextField{
		Name:     "ssid",
		Required: false,
	})
	clientcollection.Fields.Add(&core.NumberField{
		Name:     "frequency",
		Required: false,
	})
	clientcollection.Fields.Add(&core.NumberField{
		Name:     "channel",
		Required: false,
	})
	clientcollection.Fields.Add(&core.TextField{
		Name:     "device",
		Required: false,
	})
	err := app.Save(clientcollection)
	assert.Equal(t, err, nil)
	return clientcollection

}

func setupInterfacesCollection(t *testing.T, app core.App) *core.Collection {
	ifacecollection := core.NewBaseCollection("interfaces")
	ifacecollection.Fields.Add(&core.TextField{
		Name:     "device",
		Required: true,
	})
	ifacecollection.Fields.Add(&core.TextField{
		Name:     "wifi",
		Required: false,
	})
	ifacecollection.Fields.Add(&core.SelectField{
		Name:      "band",
		MaxSelect: 1,
		Required:  true,
		Values:    []string{"2.4", "5", "6", "60"},
	})
	ifacecollection.Fields.Add(&core.TextField{
		Name:     "mac_address",
		Required: false,
	})
	ifacecollection.Fields.Add(&core.TextField{
		Name:     "interface",
		Required: true,
	})
	ifacecollection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})
	ifacecollection.Fields.Add(&core.AutodateField{
		Name:     "updated",
		OnCreate: true,
		OnUpdate: true,
	})
	err := app.Save(ifacecollection)
	assert.Equal(t, err, nil)
	return ifacecollection
}

func setupSettingsCollection(t *testing.T, app core.App) *core.Collection {
	settingscollection := core.NewBaseCollection("settings")
	settingscollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
	})
	settingscollection.Fields.Add(&core.TextField{
		Name:     "value",
		Required: true,
	})
	err := app.Save(settingscollection)
	assert.Nil(t, err)
	return settingscollection
}

func setupVlanCollection(t *testing.T, app core.App) *core.Collection {
	vlancollection := core.NewBaseCollection("vlan")
	vlancollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
	})
	x := 1.0
	y := 4096.0
	vlancollection.Fields.Add(&core.NumberField{
		Name:     "number",
		Required: false,
		Min:      &x,
		Max:      &y,
		OnlyInt:  true,
	})
	vlancollection.Fields.Add(&core.TextField{
		Name:     "cidr",
		Pattern:  "^([0-9]{1,3}.){3}[0-9]{1,3}/[0-9]{1,2}$",
		Required: false,
	})
	vlancollection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})
	err := app.Save(vlancollection)
	assert.Equal(t, err, nil)
	return vlancollection
}

func expandVlanCollection(t *testing.T, app core.App, vlancollection *core.Collection, devicecollection *core.Collection) {
	vlancollection.Fields.Add(&core.RelationField{
		Name:         "gateway",
		MaxSelect:    1,
		Required:     false,
		CollectionId: devicecollection.Id,
	})
	err := app.Save(vlancollection)
	assert.Equal(t, err, nil)
}

func setupDhcpLeaseCollection(t *testing.T, app core.App) *core.Collection {
	dhcpcollection := core.NewBaseCollection("dhcp_leases")
	dhcpcollection.Fields.Add(&core.TextField{
		Name:     "mac_address",
		Required: true,
	})
	dhcpcollection.Fields.Add(&core.TextField{
		Name:     "ip_address",
		Required: true,
	})
	dhcpcollection.Fields.Add(&core.TextField{
		Name:     "hostname",
		Required: false,
	})
	dhcpcollection.Fields.Add(&core.DateField{
		Name:     "expiry",
		Required: false,
	})
	err := app.Save(dhcpcollection)
	assert.Equal(t, err, nil)
	return dhcpcollection
}

func setupClientSteeringCollection(t *testing.T, app core.App, clientcollection *core.Collection, devicecollection *core.Collection, wificollection *core.Collection) *core.Collection {
	cscollection := core.NewBaseCollection("client_steering")
	cscollection.Fields.Add(&core.RelationField{
		Name:         "client",
		MaxSelect:    1,
		Required:     true,
		CollectionId: clientcollection.Id,
	})
	cscollection.Fields.Add(&core.RelationField{
		Name:         "wifi",
		MaxSelect:    1,
		Required:     true,
		CollectionId: wificollection.Id,
	})
	cscollection.Fields.Add(&core.RelationField{
		Name:         "whitelist",
		MaxSelect:    99,
		Required:     true,
		CollectionId: devicecollection.Id,
	})
	cscollection.Fields.Add(&core.SelectField{
		Name:      "enable",
		MaxSelect: 1,
		Required:  true,
		Values:    []string{"Always", "If all healthy", "If any healthy"},
	})
	cscollection.Fields.Add(&core.SelectField{
		Name:      "method",
		MaxSelect: 1,
		Required:  true,
		Values:    []string{"mac blacklist", "bss request (ieee80211v)", "ssid"},
	})
	cscollection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})
	err := app.Save(cscollection)
	assert.Equal(t, err, nil)
	return cscollection
}

func setupPortTaggingCollection(t *testing.T, app core.App, vlancollection *core.Collection) *core.Collection {
	ptcollection := core.NewBaseCollection("port_tagging")
	ptcollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
	})
	ptcollection.Fields.Add(&core.RelationField{
		Name:         "untagged",
		MaxSelect:    1,
		Required:     false,
		CollectionId: vlancollection.Id,
	})
	ptcollection.Fields.Add(&core.BoolField{
		Name:     "trunk",
		Required: false,
	})
	ptcollection.Fields.Add(&core.RelationField{
		Name:         "tagged",
		MaxSelect:    99,
		Required:     false,
		CollectionId: vlancollection.Id,
	})
	err := app.Save(ptcollection)
	assert.Equal(t, err, nil)
	return ptcollection
}

func TestGetTimeAdvertisementValue(t *testing.T) {
	flag, tzdata := getTimeAdvertisementValues("")
	assert.Equal(t, 0, flag)
	assert.Equal(t, "", tzdata)
	flag, tzdata = getTimeAdvertisementValues("non-existing")
	assert.Equal(t, 2, flag)
	assert.Equal(t, "non-existing", tzdata, "non-existing tz data should be returned without modification")
	flag, tzdata = getTimeAdvertisementValues("Disabled")
	assert.Equal(t, 0, flag)
	assert.Equal(t, "", tzdata)
	flag, tzdata = getTimeAdvertisementValues("UTC")
	assert.Equal(t, 2, flag)
	assert.Equal(t, "", tzdata)
	flag, tzdata = getTimeAdvertisementValues("Europe/Brussels")
	assert.Equal(t, 2, flag)
	assert.Equal(t, "CET-1CEST,M3.5.0,M10.5.0/3", tzdata)
	flag, tzdata = getTimeAdvertisementValues("America/New York")
	assert.Equal(t, 2, flag)
	assert.Equal(t, "EST5EDT,M3.2.0,M11.1.0", tzdata)
}

// Test for both steered and static SSID configs
func TestGenerateWifiRecordList(t *testing.T) {
	app, err := tests.NewTestApp()
	defer app.Cleanup()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	clientcollection := setupClientsCollection(t, app)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	clientsteeringcollection := setupClientSteeringCollection(t, app, clientcollection, devicecollection, wificollection)

	// Add a vlan
	v := core.NewRecord(vlancollection)
	v.Set("name", "wan")
	err = app.Save(v)
	assert.Equal(t, nil, err)

	// Add a wifi record
	w1 := core.NewRecord(wificollection)
	w1.Id = "somethingabcdef"
	w1.Set("ssid", "the_ssid")
	w1.Set("key", "the_key")
	w1.Set("ieee80211k", false)
	w1.Set("ieee80211r", true)
	w1.Set("ieee80211r_reassoc_deadline", 5000)
	w1.Set("encryption", "the_encryption")
	err = app.Save(w1)
	assert.Equal(t, nil, err)

	// Add a dirty delay to ensure stable sorting
	time.Sleep(1 * time.Millisecond)

	// Ensure this wifi record is alphabetically before the w1
	w2 := core.NewRecord(wificollection)
	w2.Id = "bbbbthingabcdef"
	w2.Set("ssid", "bbb_the_ssid")
	w2.Set("key", "bbb_the_key")
	w2.Set("ieee80211k", false)
	w2.Set("ieee80211r", true)
	w2.Set("ieee80211r_reassoc_deadline", 5000)
	w2.Set("encryption", "the_encryption")
	err = app.Save(w2)
	assert.Equal(t, nil, err)

	// Add a dirty delay to ensure stable sorting
	time.Sleep(1 * time.Millisecond)

	// Ensure this wifi record is alphabetically before the w1 and w2
	w3 := core.NewRecord(wificollection)
	w3.Id = "aaaathingabcdef"
	w3.Set("ssid", "aaa_the_ssid")
	w3.Set("key", "aaa_the_key")
	w3.Set("ieee80211k", false)
	w3.Set("ieee80211r", true)
	w3.Set("ieee80211r_reassoc_deadline", 5000)
	w3.Set("encryption", "the_encryption")
	err = app.Save(w3)
	assert.Equal(t, nil, err)

	// Add a client
	c := core.NewRecord(clientcollection)
	c.Set("mac_address", "11:22:33:44:55:66")
	err = app.Save(c)
	assert.Equal(t, nil, err)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	// Configure the wifis in the "wrong" order too
	d1.Set("wifis", []string{w2.Id, w1.Id})
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	// Add a device
	d2 := core.NewRecord(devicecollection)
	d2.Set("name", "the_device2")
	d2.Set("health_status", "healthy")
	// Only add wifi 2 here, to test w1 is added after w2 for client steering
	d2.Set("wifis", []string{w2.Id})
	err = app.Save(d2)
	assert.Equal(t, nil, err)

	wifilist1, err := generateWifiRecordList(app, d1)
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(wifilist1))
	assert.Equal(t, "somethingabcdef", wifilist1[0].Id)
	assert.Equal(t, "bbbbthingabcdef", wifilist1[1].Id)

	wifilist2, err := generateWifiRecordList(app, d2)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(wifilist2))
	assert.Equal(t, "bbbbthingabcdef", wifilist2[0].Id)

	// On device 2 add a steering config
	cs1 := core.NewRecord(clientsteeringcollection)
	cs1.Set("client", c.Id)
	cs1.Set("whitelist", d2.Id)
	cs1.Set("wifi", w1.Id)
	cs1.Set("enable", "Always")
	cs1.Set("method", "ssid")
	err = app.Save(cs1)
	assert.Equal(t, nil, err)

	// Add a dirty delay to ensure stable sorting
	time.Sleep(1 * time.Millisecond)

	// On device 2 add a steering config
	cs2 := core.NewRecord(clientsteeringcollection)
	cs2.Set("client", c.Id)
	cs2.Set("whitelist", d2.Id)
	cs2.Set("wifi", w3.Id)
	cs2.Set("enable", "Always")
	cs2.Set("method", "ssid")
	err = app.Save(cs2)
	assert.Equal(t, nil, err)

	wifilist1, err = generateWifiRecordList(app, d1)
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(wifilist1))
	assert.Equal(t, "somethingabcdef", wifilist1[0].Id)
	assert.Equal(t, "bbbbthingabcdef", wifilist1[1].Id)

	wifilist2, err = generateWifiRecordList(app, d2)
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(wifilist2))
	assert.Equal(t, "bbbbthingabcdef", wifilist2[0].Id)
	assert.Equal(t, "somethingabcdef", wifilist2[1].Id)
	assert.Equal(t, "aaaathingabcdef", wifilist2[2].Id)
}

// Test the different Wifi configs
func TestGenerateWifiConfig(t *testing.T) {
	app, err := tests.NewTestApp()
	defer app.Cleanup()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	clientcollection := setupClientsCollection(t, app)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	clientsteeringcollection := setupClientSteeringCollection(t, app, clientcollection, devicecollection, wificollection)

	// Add a vlan
	v := core.NewRecord(vlancollection)
	v.Set("name", "wan")
	err = app.Save(v)
	assert.Equal(t, nil, err)

	// Add a wifi record
	w := core.NewRecord(wificollection)
	w.Id = "somethingabcdef"
	w.Set("ssid", "the_ssid")
	w.Set("key", "the_key")
	w.Set("ieee80211k", false)
	w.Set("ieee80211r", true)
	w.Set("ieee80211r_reassoc_deadline", 5000)
	w.Set("encryption", "the_encryption")
	err = app.Save(w)
	assert.Equal(t, nil, err)

	// Add a client
	c := core.NewRecord(clientcollection)
	c.Set("mac_address", "11:22:33:44:55:66")
	err = app.Save(c)
	assert.Equal(t, nil, err)

	// Add a device
	d := core.NewRecord(devicecollection)
	d.Set("name", "the_device1")
	d.Set("health_status", "healthy")
	d.Set("wifis", w.Id)
	err = app.Save(d)
	assert.Equal(t, nil, err)

	// Add a device
	d2 := core.NewRecord(devicecollection)
	d2.Set("name", "the_device2")
	d2.Set("health_status", "healthy")
	d2.Set("wifis", w.Id)
	err = app.Save(d2)
	assert.Equal(t, nil, err)

	// Generate a config
	wificonfig := generateWifiConfig(w, 3, 4, app, d)
	assert.Equal(t, wificonfig, `
config wifi-iface 'wifi_3_radio4'
        option device 'radio4'
        option network 'lan'
        option disabled '0'
        option mode 'ap'
        option ssid 'the_ssid'
        option encryption 'the_encryption'
        option key 'the_key'
        option ieee80211k '0'
        option ieee80211r '1'
        option reassociation_deadline '5000'
        option time_advertisement '0'
        option time_zone ''
        option wnm_sleep_mode '0'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '0'
        option bss_transition '0'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
`)

	// Generate a config with 80211k enabled
	w.Set("ieee80211k", true)
	// Generate a config with 80211r disabled
	w.Set("ieee80211r", false)
	w.Set("ieee80211r_reassoc_deadline", 0)
	// and with 80211v enabled
	w.Set("ieee80211v_bss_transition", true)
	err = app.Save(w)
	// Verify the encryption defaults to WPA2
	w.Set("encryption", "")

	// Generate a config
	wificonfig = generateWifiConfig(w, 3, 4, app, d)
	assert.Equal(t, wificonfig, `
config wifi-iface 'wifi_3_radio4'
        option device 'radio4'
        option network 'lan'
        option disabled '0'
        option mode 'ap'
        option ssid 'the_ssid'
        option encryption 'psk2+ccmp'
        option key 'the_key'
        option ieee80211k '1'
        option ieee80211r '0'
        option reassociation_deadline '1000'
        option time_advertisement '0'
        option time_zone ''
        option wnm_sleep_mode '0'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '0'
        option bss_transition '1'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
`)

	// Generate a config with network set to wan
	w.Set("network", v.Id)
	err = app.Save(w)

	// Generate a config
	wificonfig = generateWifiConfig(w, 3, 4, app, d)
	assert.Equal(t, wificonfig, `
config wifi-iface 'wifi_3_radio4'
        option device 'radio4'
        option network 'wan'
        option disabled '0'
        option mode 'ap'
        option ssid 'the_ssid'
        option encryption 'psk2+ccmp'
        option key 'the_key'
        option ieee80211k '1'
        option ieee80211r '0'
        option reassociation_deadline '1000'
        option time_advertisement '0'
        option time_zone ''
        option wnm_sleep_mode '0'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '0'
        option bss_transition '1'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
`)
	// Test the clientsteering, client should be steered away from this AP
	cs := core.NewRecord(clientsteeringcollection)
	cs.Set("client", c.Id)
	cs.Set("whitelist", []string{d2.Id})
	cs.Set("wifi", w.Id)
	cs.Set("enable", "Always")
	cs.Set("method", "mac blacklist")
	err = app.Save(cs)
	assert.Equal(t, nil, err)
	assert.Equal(t, []string{d2.Id}, cs.GetStringSlice("whitelist"))
	// Generate a config
	wificonfig = generateWifiConfig(w, 3, 4, app, d)
	assert.Equal(t, wificonfig, `
config wifi-iface 'wifi_3_radio4'
        option device 'radio4'
        option network 'wan'
        option disabled '0'
        option mode 'ap'
        option ssid 'the_ssid'
        option encryption 'psk2+ccmp'
        option key 'the_key'
        option ieee80211k '1'
        option ieee80211r '0'
        option reassociation_deadline '1000'
        option time_advertisement '0'
        option time_zone ''
        option wnm_sleep_mode '0'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '0'
        option bss_transition '1'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
        option macfilter 'deny'
        list maclist '11:22:33:44:55:66'
`)
	// More checks of different settings since we don't want to check too many at once
	w.Set("ieee80211v_wnm_sleep_mode", true)
	// Explicitely set the time advertisement to disabled
	w.Set("ieee80211v_time_advertisement", "Disabled")
	// Generate a config
	wificonfig = generateWifiConfig(w, 3, 4, app, d)
	assert.Equal(t, wificonfig, `
config wifi-iface 'wifi_3_radio4'
        option device 'radio4'
        option network 'wan'
        option disabled '0'
        option mode 'ap'
        option ssid 'the_ssid'
        option encryption 'psk2+ccmp'
        option key 'the_key'
        option ieee80211k '1'
        option ieee80211r '0'
        option reassociation_deadline '1000'
        option time_advertisement '0'
        option time_zone ''
        option wnm_sleep_mode '1'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '0'
        option bss_transition '1'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
        option macfilter 'deny'
        list maclist '11:22:33:44:55:66'
`)

	// Test the timezone
	w.Set("ieee80211v_time_advertisement", "Europe/Brussels")
	// Test proxy ARP
	w.Set("ieee80211v_proxy_arp", true)

	// Generate a config
	wificonfig = generateWifiConfig(w, 3, 4, app, d)
	assert.Equal(t, wificonfig, `
config wifi-iface 'wifi_3_radio4'
        option device 'radio4'
        option network 'wan'
        option disabled '0'
        option mode 'ap'
        option ssid 'the_ssid'
        option encryption 'psk2+ccmp'
        option key 'the_key'
        option ieee80211k '1'
        option ieee80211r '0'
        option reassociation_deadline '1000'
        option time_advertisement '2'
        option time_zone 'CET-1CEST,M3.5.0,M10.5.0/3'
        option wnm_sleep_mode '1'
        option wnm_sleep_mode_no_keys '0'
        option proxy_arp '1'
        option bss_transition '1'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
        option macfilter 'deny'
        list maclist '11:22:33:44:55:66'
`)
}

func TestIsUnHealthyQuorumReached(t *testing.T) {
	allofflineset := make(map[string]struct{})
	allofflineset["device1"] = struct{}{}
	allofflineset["device2"] = struct{}{}
	allofflineset["device3"] = struct{}{}

	{
		whitelist := []string{}
		assert.False(t, isUnHealthyQuorumReached(allofflineset, whitelist, true))
		assert.False(t, isUnHealthyQuorumReached(allofflineset, whitelist, false))
	}

	{ // List contains one device, which is offline
		whitelist := []string{"device1"}
		assert.True(t, isUnHealthyQuorumReached(allofflineset, whitelist, true))
		assert.True(t, isUnHealthyQuorumReached(allofflineset, whitelist, false))
	}
	{ // List contains two devices, which are offline
		whitelist := []string{"device1", "device2"}
		assert.True(t, isUnHealthyQuorumReached(allofflineset, whitelist, true))
		assert.True(t, isUnHealthyQuorumReached(allofflineset, whitelist, false))
	}
	{ // List contains two devices, one is offline
		whitelist := []string{"device1", "device4"}
		assert.True(t, isUnHealthyQuorumReached(allofflineset, whitelist, true))
		assert.False(t, isUnHealthyQuorumReached(allofflineset, whitelist, false)) // Not all offline
	}
	{ // List contains two devices, none are offline
		whitelist := []string{"device4", "device5"}
		assert.False(t, isUnHealthyQuorumReached(allofflineset, whitelist, true))
		assert.False(t, isUnHealthyQuorumReached(allofflineset, whitelist, false))
	}
}

func TestCIDRToMask(t *testing.T) {
	// Test the classics
	mask, err := CIDRToMask(32)
	assert.Equal(t, nil, err)
	assert.Equal(t, "255.255.255.255", mask)
	mask, err = CIDRToMask(24)
	assert.Equal(t, nil, err)
	assert.Equal(t, "255.255.255.0", mask)
	mask, err = CIDRToMask(16)
	assert.Equal(t, nil, err)
	assert.Equal(t, "255.255.0.0", mask)
	mask, err = CIDRToMask(8)
	assert.Equal(t, nil, err)
	assert.Equal(t, "255.0.0.0", mask)
	mask, err = CIDRToMask(0)
	assert.Equal(t, nil, err)
	assert.Equal(t, "0.0.0.0", mask)
	// Test a funky one too
	mask, err = CIDRToMask(17)
	assert.Equal(t, nil, err)
	assert.Equal(t, "255.255.128.0", mask)
	// Test the outliers
	mask, err = CIDRToMask(33)
	assert.Equal(t, nil, err)
	assert.Equal(t, "255.255.255.255", mask)
	// Test the outliers
	mask, err = CIDRToMask(-1)
	assert.Equal(t, nil, err)
	assert.Equal(t, "0.0.0.0", mask)
}

func TestDhcpClientUpdate(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Equal(t, nil, err)
	defer app.Cleanup()
	leaseslist := []DHCPLease{
		{
			MACAddress: "00:1A:2B:3C:4D:5E",
			ClientID:   "01:00:1A:2B:3C:4D:5E",
			IPAddress:  "192.168.1.10",
			Hostname:   "device1",
			Expiry:     1234,
		},
		{
			MACAddress: "00:1A:2B:3C:4D:5F",
			ClientID:   "01:00:1A:2B:3C:4D:5F",
			IPAddress:  "192.168.1.11",
			Hostname:   "device2",
			Expiry:     5678,
		},
	}
	expiryTime := types.DateTime{}
	setupDhcpLeaseCollection(t, app)
	storeDHCPLeases(app, leaseslist, expiryTime)
	records, err := app.FindAllRecords("dhcp_leases")
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(records))

	assert.Equal(t, "00:1A:2B:3C:4D:5E", records[0].GetString("mac_address"))
	assert.Equal(t, "192.168.1.10", records[0].GetString("ip_address"))
	assert.Equal(t, "device1", records[0].GetString("hostname"))
	assert.Equal(t, "1970-01-01 00:20:34.000Z", records[0].GetString("expiry"))

	assert.Equal(t, "00:1A:2B:3C:4D:5F", records[1].GetString("mac_address"))
	assert.Equal(t, "192.168.1.11", records[1].GetString("ip_address"))
	assert.Equal(t, "device2", records[1].GetString("hostname"))
	assert.Equal(t, "1970-01-01 01:34:38.000Z", records[1].GetString("expiry"))

	// Test another case, hostname should be reset to NULL when a * is received
	leaseslist = []DHCPLease{
		{
			MACAddress: "00:1A:2B:3C:4D:5E",
			ClientID:   "01:00:1A:2B:3C:4D:5E",
			IPAddress:  "192.168.1.10",
			Hostname:   "device3",
			Expiry:     12340,
		},
		{
			MACAddress: "00:1A:2B:3C:4D:5F",
			ClientID:   "01:00:1A:2B:3C:4D:5F",
			IPAddress:  "192.168.1.11",
			Hostname:   "*",
			Expiry:     56780,
		},
		{
			MACAddress: "00:1A:2B:3C:4D:60",
			ClientID:   "01:00:1A:2B:3C:4D:60",
			IPAddress:  "192.168.1.12",
			Hostname:   "*",
			Expiry:     56780,
		},
	}
	storeDHCPLeases(app, leaseslist, expiryTime)
	records, err = app.FindAllRecords("dhcp_leases")
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(records))

	assert.Equal(t, "00:1A:2B:3C:4D:5E", records[0].GetString("mac_address"))
	assert.Equal(t, "192.168.1.10", records[0].GetString("ip_address"))
	assert.Equal(t, "device3", records[0].GetString("hostname"))
	assert.Equal(t, "1970-01-01 03:25:40.000Z", records[0].GetString("expiry"))

	assert.Equal(t, "00:1A:2B:3C:4D:5F", records[1].GetString("mac_address"))
	assert.Equal(t, "192.168.1.11", records[1].GetString("ip_address"))
	assert.Equal(t, "", records[1].GetString("hostname"))
	assert.Equal(t, "1970-01-01 15:46:20.000Z", records[1].GetString("expiry"))

	assert.Equal(t, "00:1A:2B:3C:4D:60", records[2].GetString("mac_address"))
	assert.Equal(t, "192.168.1.12", records[2].GetString("ip_address"))
	assert.Equal(t, "", records[2].GetString("hostname"))
	assert.Equal(t, "1970-01-01 15:46:20.000Z", records[2].GetString("expiry"))

	leaseslist = []DHCPLease{}
	expiryTime, err = types.ParseDateTime("1970-01-01 15:46:20.000Z")
	assert.Equal(t, nil, err)
	storeDHCPLeases(app, leaseslist, expiryTime)
	records, err = app.FindAllRecords("dhcp_leases")
	assert.Equal(t, 2, len(records))
}

func TestSsidClientSteering(t *testing.T) {
	app, err := tests.NewTestApp()
	defer app.Cleanup()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	clientcollection := setupClientsCollection(t, app)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	clientsteeringcollection := setupClientSteeringCollection(t, app, clientcollection, devicecollection, wificollection)

	// Add a wifi record
	w1 := core.NewRecord(wificollection)
	w1.Id = "somethingabcdef"
	w1.Set("ssid", "the_ssid")
	w1.Set("key", "the_key")
	w1.Set("ieee80211r", true)
	w1.Set("encryption", "the_encryption")
	err = app.Save(w1)
	assert.Equal(t, nil, err)

	// Add a wifi record
	w2 := core.NewRecord(wificollection)
	w2.Id = "somethingabctwo"
	w2.Set("ssid", "the_ssid")
	w2.Set("key", "the_key")
	w2.Set("ieee80211r", true)
	w2.Set("encryption", "the_encryption")
	err = app.Save(w2)
	assert.Equal(t, nil, err)

	// Add a client
	c := core.NewRecord(clientcollection)
	c.Id = "somethingclient"
	c.Set("mac_address", "00:11:22:33:44:55")
	err = app.Save(c)
	assert.Equal(t, nil, err)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("wifis", w1.Id)
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	// Add a device (unhealthy)
	d2 := core.NewRecord(devicecollection)
	d2.Id = "somethindevice2"
	d2.Set("name", "the_device2")
	d2.Set("health_status", "unhealthy")
	d2.Set("wifis", w1.Id)
	err = app.Save(d2)
	assert.Equal(t, nil, err)

	// Add a device (with 2 wifi)
	d3 := core.NewRecord(devicecollection)
	d3.Id = "somethindevice3"
	d3.Set("name", "the_device3")
	d3.Set("health_status", "healthy")
	d3.Set("wifis", []string{w1.Id, w2.Id})
	err = app.Save(d3)
	assert.Equal(t, nil, err)

	// Whitelist client on wifi 1 @ device 1
	cs := core.NewRecord(clientsteeringcollection)
	cs.Id = "clientsteering1"
	cs.Set("client", c.Id)
	cs.Set("whitelist", []string{d1.Id})
	cs.Set("wifi", w1.Id)
	cs.Set("enable", "Always")
	cs.Set("method", "ssid")
	err = app.Save(cs)
	assert.Equal(t, nil, err)
	assert.Equal(t, []string{d1.Id}, cs.GetStringSlice("whitelist"))

	// Whitelist client on wifi 2
	cs2 := core.NewRecord(clientsteeringcollection)
	cs2.Id = "clientsteering2"
	cs2.Set("client", c.Id)
	cs2.Set("whitelist", []string{d2.Id, d3.Id})
	cs2.Set("wifi", w2.Id)
	cs2.Set("enable", "Always")
	cs2.Set("method", "ssid")
	err = app.Save(cs2)
	assert.Equal(t, nil, err)

	{
		// D1 is whitelisted, add SSID1
		csconfig, err := generateSsidClientSteeringConfig(app, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)

		// D2 is whitelisted, add SSID2
		csconfig, err = generateSsidClientSteeringConfig(app, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabctwo", csconfig[0].Id)

		// D3 is whitelisted, add SSID2
		csconfig, err = generateSsidClientSteeringConfig(app, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabctwo", csconfig[0].Id)
	}

	// Whitelist a second device
	{
		cs.Set("whitelist", []string{d1.Id, d2.Id})
		err = app.Save(cs)
		assert.Equal(t, nil, err)
		assert.Equal(t, []string{d1.Id, d2.Id}, cs.GetStringSlice("whitelist"))

		// D1 is whitelisted, add SSID1
		csconfig, err := generateSsidClientSteeringConfig(app, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)

		// D2 is whitelisted, add SSID2 and SSID1
		csconfig, err = generateSsidClientSteeringConfig(app, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, 2, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)
		assert.Equal(t, "somethingabctwo", csconfig[1].Id)

		// D3 is whitelisted, add SSID2
		csconfig, err = generateSsidClientSteeringConfig(app, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabctwo", csconfig[0].Id)
	}

	// Device 2 is unhealthy
	// Steering should be lifted for wifi1, apply on all APs
	{
		cs.Set("enable", "If all healthy")
		err = app.Save(cs)
		assert.Equal(t, nil, err)
		assert.Equal(t, []string{d1.Id, d2.Id}, cs.GetStringSlice("whitelist"))

		// Add only the unhealthy SSID
		csconfig, err := generateSsidClientSteeringConfig(app, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)

		// Add both SSID
		csconfig, err = generateSsidClientSteeringConfig(app, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, 2, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)
		assert.Equal(t, "somethingabctwo", csconfig[1].Id)

		// Add both SSID
		csconfig, err = generateSsidClientSteeringConfig(app, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, 2, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)
		assert.Equal(t, "somethingabctwo", csconfig[1].Id)
	}

	// Device 2 is unhealthy, device 1 is healthy, steering should be reinstated
	{
		cs.Set("enable", "If any healthy")
		err = app.Save(cs)
		assert.Equal(t, nil, err)
		assert.Equal(t, []string{d1.Id, d2.Id}, cs.GetStringSlice("whitelist"))

		// Whitelisted, don't block
		csconfig, err := generateSsidClientSteeringConfig(app, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)

		// Whitelisted, don't block
		csconfig, err = generateSsidClientSteeringConfig(app, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, 2, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)
		assert.Equal(t, "somethingabctwo", csconfig[1].Id)

		// Add only the healthy SSID
		csconfig, err = generateSsidClientSteeringConfig(app, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabctwo", csconfig[0].Id)
	}
	// Device 2 is unhealthy, device 1 is healthy, steering should remain enabled
	{
		// Whitelisted, don't block
		csconfig, err := generateSsidClientSteeringConfig(app, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)

		// Whitelisted, don't block
		csconfig, err = generateSsidClientSteeringConfig(app, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, 2, len(csconfig))
		assert.Equal(t, "somethingabcdef", csconfig[0].Id)
		assert.Equal(t, "somethingabctwo", csconfig[1].Id)

		// Add only the healthy SSID
		csconfig, err = generateSsidClientSteeringConfig(app, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, 1, len(csconfig))
		assert.Equal(t, "somethingabctwo", csconfig[0].Id)
	}
}

func TestMacClientSteering(t *testing.T) {
	app, err := tests.NewTestApp()
	defer app.Cleanup()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	clientcollection := setupClientsCollection(t, app)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	clientsteeringcollection := setupClientSteeringCollection(t, app, clientcollection, devicecollection, wificollection)

	// Add a wifi record
	w1 := core.NewRecord(wificollection)
	w1.Id = "somethingabcdef"
	w1.Set("ssid", "the_ssid")
	w1.Set("key", "the_key")
	w1.Set("ieee80211r", true)
	w1.Set("encryption", "the_encryption")
	err = app.Save(w1)
	assert.Equal(t, nil, err)

	// Add a wifi record
	w2 := core.NewRecord(wificollection)
	w2.Id = "somethingabctwo"
	w2.Set("ssid", "the_ssid")
	w2.Set("key", "the_key")
	w2.Set("ieee80211r", true)
	w2.Set("encryption", "the_encryption")
	err = app.Save(w2)
	assert.Equal(t, nil, err)

	// Add a client
	c := core.NewRecord(clientcollection)
	c.Id = "somethingclient"
	c.Set("mac_address", "00:11:22:33:44:55")
	err = app.Save(c)
	assert.Equal(t, nil, err)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("wifis", w1.Id)
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	// Add a device
	d2 := core.NewRecord(devicecollection)
	d2.Id = "somethindevice2"
	d2.Set("name", "the_device2")
	d2.Set("health_status", "unhealthy")
	d2.Set("wifis", w1.Id)
	err = app.Save(d2)
	assert.Equal(t, nil, err)

	// Add a device (with 2 wifi)
	d3 := core.NewRecord(devicecollection)
	d3.Id = "somethindevice3"
	d3.Set("name", "the_device3")
	d3.Set("health_status", "healthy")
	d3.Set("wifis", []string{w1.Id, w2.Id})
	err = app.Save(d3)
	assert.Equal(t, nil, err)

	// Whitelist client on wifi 1
	cs := core.NewRecord(clientsteeringcollection)
	cs.Set("client", c.Id)
	cs.Set("whitelist", []string{d1.Id})
	cs.Set("wifi", w1.Id)
	cs.Set("enable", "Always")
	cs.Set("method", "mac blacklist")
	err = app.Save(cs)
	assert.Equal(t, nil, err)
	assert.Equal(t, []string{d1.Id}, cs.GetStringSlice("whitelist"))

	// Whitelist client on wifi 2
	cs2 := core.NewRecord(clientsteeringcollection)
	cs2.Set("client", c.Id)
	cs2.Set("whitelist", []string{d2.Id, d3.Id})
	cs2.Set("wifi", w2.Id)
	cs2.Set("enable", "Always")
	cs2.Set("method", "mac blacklist")
	err = app.Save(cs2)
	assert.Equal(t, nil, err)

	{
		// Whitelisted, don't block
		csconfig, err := generateMacClientSteeringConfig(app, w1, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Not whitelisted, block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, "        option macfilter 'deny'\n        list maclist '00:11:22:33:44:55'\n", csconfig)

		// Not whitelisted, block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, "        option macfilter 'deny'\n        list maclist '00:11:22:33:44:55'\n", csconfig)
	}

	// Whitelist a second device
	{
		cs.Set("whitelist", []string{d1.Id, d2.Id})
		err = app.Save(cs)
		assert.Equal(t, nil, err)
		assert.Equal(t, []string{d1.Id, d2.Id}, cs.GetStringSlice("whitelist"))

		// Whitelisted, don't block
		csconfig, err := generateMacClientSteeringConfig(app, w1, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Whitelisted, don't block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Not whitelisted, block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, "        option macfilter 'deny'\n        list maclist '00:11:22:33:44:55'\n", csconfig)
	}

	// Test the second wifi
	{
		// Not whitelisted, block
		csconfig, err := generateMacClientSteeringConfig(app, w2, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, "        option macfilter 'deny'\n        list maclist '00:11:22:33:44:55'\n", csconfig)

		// Whitelisted, don't block
		csconfig, err = generateMacClientSteeringConfig(app, w2, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Not whitelisted, block
		csconfig, err = generateMacClientSteeringConfig(app, w2, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)
	}

	// Device 2 is unhealthy, steering should be lifted
	{
		cs.Set("enable", "If all healthy")
		err = app.Save(cs)
		assert.Equal(t, nil, err)
		assert.Equal(t, []string{d1.Id, d2.Id}, cs.GetStringSlice("whitelist"))

		// Whitelisted, don't block
		csconfig, err := generateMacClientSteeringConfig(app, w1, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Whitelisted, don't block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Not whitelisted, block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)
	}

	// Device 2 is unhealthy, device 1 is healthy, steering should be reinstated
	{
		cs.Set("enable", "If any healthy")
		err = app.Save(cs)
		assert.Equal(t, nil, err)
		assert.Equal(t, []string{d1.Id, d2.Id}, cs.GetStringSlice("whitelist"))

		// Whitelisted, don't block
		csconfig, err := generateMacClientSteeringConfig(app, w1, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Whitelisted, don't block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Not whitelisted, block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, "        option macfilter 'deny'\n        list maclist '00:11:22:33:44:55'\n", csconfig)
	}

	// Device 2 is unhealthy, device 1 is healthy, steering should remain enabled
	{
		cs.Set("enable", "Always")
		err = app.Save(cs)
		assert.Equal(t, nil, err)
		assert.Equal(t, []string{d1.Id, d2.Id}, cs.GetStringSlice("whitelist"))

		// Whitelisted, don't block
		csconfig, err := generateMacClientSteeringConfig(app, w1, d1)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Whitelisted, don't block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d2)
		assert.Equal(t, err, nil)
		assert.Equal(t, "", csconfig)

		// Not whitelisted, block
		csconfig, err = generateMacClientSteeringConfig(app, w1, d3)
		assert.Equal(t, err, nil)
		assert.Equal(t, "        option macfilter 'deny'\n        list maclist '00:11:22:33:44:55'\n", csconfig)
	}
}

func TestApiGenerateDeviceStatus(t *testing.T) {
	app, err := tests.NewTestApp()
	defer app.Cleanup()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)

	event := core.RequestEvent{}
	event.Request, err = http.NewRequest("GET", "/help", strings.NewReader(""))
	event.Request.SetPathValue("mac_address", "AA:Bb:CC:ee:ff:00")
	event.App = app
	rec := httptest.NewRecorder()
	event.Response = rec
	{
		response := apiGenerateDeviceStatus(&event)
		var apiresponse *router.ApiError
		errors.As(response, &apiresponse)
		assert.Equal(t, 404, apiresponse.Status)
		assert.Equal(t, "Device not found.", apiresponse.Message)
	}

	// Add a device record
	m := core.NewRecord(devicecollection)
	m.Id = "somethingabcdef"
	m.Set("mac_address", "AA:BB:CC:EE:FF:00")
	m.Set("health_status", "healthy")
	err = app.Save(m)
	assert.Equal(t, nil, err)

	// Add a device record
	m = core.NewRecord(devicecollection)
	m.Id = "somethingabcddd"
	m.Set("mac_address", "AA:BB:CC:EE:FF:11")
	m.Set("health_status", "unhealthy")
	err = app.Save(m)
	assert.Equal(t, nil, err)

	{
		response := apiGenerateDeviceStatus(&event)
		assert.Equal(t, response, nil)
		httpResponse := rec.Result()
		defer httpResponse.Body.Close()
		body, err := io.ReadAll(httpResponse.Body)
		assert.Equal(t, nil, err)
		assert.Equal(t, 200, httpResponse.StatusCode)
		assert.Equal(t, "on", string(body))
	}
	{

		event := core.RequestEvent{}
		event.Request, err = http.NewRequest("GET", "/help", strings.NewReader(""))
		event.Request.SetPathValue("mac_address", "AA:BB:CC:EE:FF:11")
		event.App = app
		rec := httptest.NewRecorder()
		event.Response = rec
		{
			response := apiGenerateDeviceStatus(&event)
			assert.Equal(t, response, nil)
			httpResponse := rec.Result()
			defer httpResponse.Body.Close()
			body, err := io.ReadAll(httpResponse.Body)
			assert.Equal(t, nil, err)
			assert.Equal(t, 200, httpResponse.StatusCode)
			assert.Equal(t, "off", string(body))
		}
	}

}

func TestGenerateWifiQr(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	w := core.NewRecord(wificollection)
	w.Id = "somethingabcdef"
	w.Set("ssid", "OpenWRT1")
	w.Set("key", "the_key")
	w.Set("ieee80211r", true)
	w.Set("encryption", "the_encryption")
	err = app.Save(w)
	assert.Equal(t, nil, err)

	rawpngbuffer, err := generateWifiQr(w)
	assert.Nil(t, err)
	// Test the QR
	img, err := png.Decode(bytes.NewReader(rawpngbuffer.Bytes()))
	assert.Nil(t, err, "failed to decode png")
	symbols, err := goqr.Recognize(img)
	assert.Nil(t, err, "failed to parse QR")
	assert.Equal(t, len(symbols), 1, "Expect one sybol in the QR")
	output := fmt.Sprintf("%s", symbols[0].Payload)
	assert.Equal(t, "WIFI:S:OpenWRT1;T:WPA;P:the_key;H:false;", output)
}
