package main

import (
	"errors"
	//"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rubenbe/pocketbase/core"
	"github.com/rubenbe/pocketbase/tests"
	"github.com/rubenbe/pocketbase/tools/router"
	"github.com/rubenbe/pocketbase/tools/types"
	"github.com/stretchr/testify/assert"
)

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
}

func TestRegisterEndpoint(t *testing.T) {
	// setup the test ApiScenario app instance
	setupTestApp := func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}
		// no need to cleanup since scenario.Test() will do that for us
		// defer testApp.Cleanup()

		bindAppHooks(testApp, "testsecret")

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
		assert.Equal(t, r.GetString("device"), d.GetString("id"))
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "1")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 1)
		assert.Equal(t, r.GetInt("frequency"), 5200)
		assert.Equal(t, r.GetInt("channel"), 40)
		assert.Equal(t, r.GetString("band"), "5")
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
		assert.Equal(t, r.GetString("band"), "2.4")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "1")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 1)
		assert.Equal(t, r.GetInt("frequency"), 5200)
		assert.Equal(t, r.GetInt("channel"), 40)
		assert.Equal(t, r.GetString("band"), "5")
	}
	{
		r, err := app.FindFirstRecordByData("radios", "radio", "2")
		assert.Equal(t, err, nil)
		assert.Equal(t, r.GetInt("radio"), 2)
		assert.Equal(t, r.GetInt("frequency"), 5955)
		assert.Equal(t, r.GetInt("channel"), 100)
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

func TestApiGenerateDeviceStatus(t *testing.T) {
	app, err := tests.NewTestApp()
	defer app.Cleanup()

	// Create a devices collection, with one device
	devicecollection := core.NewBaseCollection("devices")
	devicecollection.Fields.Add(&core.TextField{
		Name:     "health_status",
		Required: true,
	})
	err = app.Save(devicecollection)
	assert.Equal(t, err, nil)

	event := core.RequestEvent{}
	event.Request, err = http.NewRequest("GET", "/help", strings.NewReader(""))
	event.Request.SetPathValue("device_id", "somethingabcdef")
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
	m.Set("health_status", "healthy")
	err = app.Save(m)
	assert.Equal(t, nil, err)

	// Add a device record
	m = core.NewRecord(devicecollection)
	m.Id = "somethingabcddd"
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
		event.Request.SetPathValue("device_id", "somethingabcddd")
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
