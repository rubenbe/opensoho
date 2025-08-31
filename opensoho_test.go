package opensoho

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

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
				"tags":        {""},
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
				"tags":        {""},
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

func TestHandleEthernetMonitoring(t *testing.T) {
	app, _ := tests.NewTestApp()
	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)
	ethernetcollection := setupEthernetCollection(t, app, devicecollection)
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
			"tags":        {""},
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
			"tags":        {""},
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
			"tags":        {""},
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
	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan'
`, generateInterfacesConfig(app, d1))
}

// Test that the default VLAN is present
func TestInterfacesConfigDefaultVLAN(t *testing.T) {
	app, _ := tests.NewTestApp()

	vlancollection := setupVlanCollection(t, app)
	wificollection := setupWifiCollection(t, app, vlancollection)
	devicecollection := setupDeviceCollection(t, app, wificollection)

	// Add a device
	d1 := core.NewRecord(devicecollection)
	d1.Id = "somethindevice1"
	d1.Set("name", "the_device1")
	d1.Set("health_status", "healthy")
	d1.Set("apply", []string{"vlan"})
	d1.Set("ip_address", "8.8.8.8")
	err := app.Save(d1)
	assert.Equal(t, nil, err)
	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.1'

config bridge-vlan 'bridge_vlan_1'
	option device 'br-lan'
	option vlan '1'
        list ports 'eth0:u*'
        list ports 'lan1:u*'
        list ports 'lan2:u*'
        list ports 'lan3:u*'
        list ports 'lan4:u*'
        list ports 'lan5:u*'
        list ports 'lan6:u*'
        list ports 'lan7:u*'
        list ports 'lan8:u*'
        list ports 'lan9:u*'
        list ports 'lan10:u*'
        list ports 'lan11:u*'
        list ports 'lan12:u*'
        list ports 'lan13:u*'
        list ports 'lan14:u*'
        list ports 'lan15:u*'
        list ports 'lan16:u*'
`, generateInterfacesConfig(app, d1))
}

func TestInterfacesConfig(t *testing.T) {
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

	wan := core.NewRecord(vlancollection)
	wan.Set("name", "wan")
	err = app.Save(wan)
	assert.Equal(t, nil, err)

	lan := core.NewRecord(vlancollection)
	lan.Set("name", "lan")
	err = app.Save(lan)
	assert.Equal(t, nil, err)

	guest := core.NewRecord(vlancollection)
	guest.Set("name", "guest")
	guest.Set("vlan_id", "7")
	guest.Set("subnet", "10.11.12.13")
	guest.Set("netmask", "255.255.128.0")
	err = app.Save(guest)
	assert.Equal(t, nil, err)

	iot := core.NewRecord(vlancollection)
	iot.Set("name", "iot")
	iot.Set("vlan_id", "123")
	iot.Set("subnet", "192.168.1.1")
	iot.Set("netmask", "255.255.255.00")
	err = app.Save(iot)
	assert.Equal(t, nil, err)

	// VLANs not enabled
	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan'
`, generateInterfacesConfig(app, d1))

	// VLANs enabled
	d1.Set("apply", []string{"vlan"})
	err = app.Save(d1)
	assert.Equal(t, nil, err)

	assert.Equal(t, `
config interface 'lan'
        option device 'br-lan.1'

config bridge-vlan 'bridge_vlan_1'
	option device 'br-lan'
	option vlan '1'
        list ports 'eth0:u*'
        list ports 'lan1:u*'
        list ports 'lan2:u*'
        list ports 'lan3:u*'
        list ports 'lan4:u*'
        list ports 'lan5:u*'
        list ports 'lan6:u*'
        list ports 'lan7:u*'
        list ports 'lan8:u*'
        list ports 'lan9:u*'
        list ports 'lan10:u*'
        list ports 'lan11:u*'
        list ports 'lan12:u*'
        list ports 'lan13:u*'
        list ports 'lan14:u*'
        list ports 'lan15:u*'
        list ports 'lan16:u*'

config bridge-vlan 'bridge_vlan_7'
        option device 'br-lan'
        option vlan '7'
        list ports 'eth0:t'
        list ports 'lan1:t'
        list ports 'lan2:t'
        list ports 'lan3:t'
        list ports 'lan4:t'
        list ports 'lan5:t'
        list ports 'lan6:t'
        list ports 'lan7:t'
        list ports 'lan8:t'
        list ports 'lan9:t'
        list ports 'lan10:t'
        list ports 'lan11:t'
        list ports 'lan12:t'
        list ports 'lan13:t'
        list ports 'lan14:t'
        list ports 'lan15:t'
        list ports 'lan16:t'

config interface 'guest'
        option device 'br-lan.7'
        option proto 'static'
        option ipaddr '10.11.12.8'
        option netmask '255.255.128.0'

config bridge-vlan 'bridge_vlan_123'
        option device 'br-lan'
        option vlan '123'
        list ports 'eth0:t'
        list ports 'lan1:t'
        list ports 'lan2:t'
        list ports 'lan3:t'
        list ports 'lan4:t'
        list ports 'lan5:t'
        list ports 'lan6:t'
        list ports 'lan7:t'
        list ports 'lan8:t'
        list ports 'lan9:t'
        list ports 'lan10:t'
        list ports 'lan11:t'
        list ports 'lan12:t'
        list ports 'lan13:t'
        list ports 'lan14:t'
        list ports 'lan15:t'
        list ports 'lan16:t'

config interface 'iot'
        option device 'br-lan.123'
        option proto 'static'
        option ipaddr '192.168.1.8'
        option netmask '255.255.255.00'
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

func TestUpdateLastSeen(t *testing.T) {
	app, _ := tests.NewTestApp()
	collection := core.NewBaseCollection("devices")
	collection.Fields.Add(&core.DateField{Name: "last_seen"})
	collection.Fields.Add(&core.SelectField{Name: "health_status", MaxSelect: 1, Values: []string{"unknown", "healthy", "critical"}})
	err := app.Save(collection)
	assert.Equal(t, err, nil)

	m := core.NewRecord(collection)
	m.Id = "testaidalongera"
	m.Set("health_status", "unknown")
	assert.Equal(t, m.GetDateTime("last_seen"), types.DateTime{})
	err = app.Save(m)
	assert.Equal(t, err, nil)

	updateLastSeen(app, m)
	assert.NotEqual(t, m.GetDateTime("last_seen"), types.DateTime{})
	assert.WithinDuration(t, m.GetDateTime("last_seen").Time(), types.NowDateTime().Time(), 1*time.Second, "Last Seen should be updated")
	record, err := app.FindRecordById("devices", "testaidalongera")
	assert.Equal(t, "healthy", m.GetString("health_status"))

	// The record is newer than 60 seconds, so should remain healthy
	now := types.NowDateTime()
	now = now.Add(59 * time.Second)
	updateDeviceHealth(app, now)
	record, err = app.FindRecordById("devices", "testaidalongera")
	assert.Equal(t, err, nil)
	assert.Equal(t, "healthy", record.GetString("health_status"))

	// The record should have been updated to unhealthy
	now = now.Add(2 * time.Second)
	updateDeviceHealth(app, now)
	record, err = app.FindRecordById("devices", "testaidalongera")
	assert.Equal(t, err, nil)
	assert.Equal(t, "unhealthy", record.GetString("health_status"))

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
	assert.Equal(t, generateRadioConfig(record), `
config wifi-device 'radio3'
	option channel '40'
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
	err := app.Save(devicecollection)
	assert.Equal(t, err, nil)
	return devicecollection
}

func setupEthernetCollection(t *testing.T, app core.App, devicecollection *core.Collection) *core.Collection {
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
		Name:     "ieee80211k",
		Required: false,
	})
	wificollection.Fields.Add(&core.RelationField{
		Name:         "network",
		MaxSelect:    1,
		Required:     false,
		CollectionId: vlancollection.Id,
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

func setupVlanCollection(t *testing.T, app core.App) *core.Collection {
	vlancollection := core.NewBaseCollection("vlan")
	vlancollection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
	})
	x := 1.0
	y := 4096.0
	vlancollection.Fields.Add(&core.NumberField{
		Name:     "vlan_id",
		Required: false,
		Min:      &x,
		Max:      &y,
		OnlyInt:  true,
	})
	vlancollection.Fields.Add(&core.TextField{
		Name:     "subnet",
		Pattern:  "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
		Required: false,
	})
	vlancollection.Fields.Add(&core.TextField{
		Name:     "netmask",
		Pattern:  "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
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
		Values:    []string{"mac blacklist", "bss request (ieee80211v)"},
	})
	err := app.Save(cscollection)
	assert.Equal(t, err, nil)
	return cscollection

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
        option bss_transition '1'
        option ft_over_ds '0'
        option ft_psk_generate_local '1'
        option macfilter 'deny'
        list maclist '11:22:33:44:55:66'
`)

	// Test the timezone
	w.Set("ieee80211v_time_advertisement", "Europe/Brussels")

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

func TestClientSteering(t *testing.T) {
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
