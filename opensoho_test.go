// main_test.go
package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
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
	collection := core.NewAuthCollection("devices")
	collection.Fields.Add(&core.DateField{Name: "last_seen"})
	collection.Fields.Add(&core.SelectField{Name: "health_status", MaxSelect: 1, Values: []string{"unknown", "healthy", "critical"}})

	m := core.NewRecord(collection)
	m.Id = "test_id"
	m.Set("health_status", "unknown")
	assert.Equal(t, m.GetDateTime("last_seen"), types.DateTime{})
	updateLastSeen(app, m)
	assert.NotEqual(t, m.GetDateTime("last_seen"), types.DateTime{})
	assert.WithinDuration(t, m.GetDateTime("last_seen").Time(), types.NowDateTime().Time(), 1*time.Second, "Last Seen should be updated")
	assert.Equal(t, m.GetString("health_status"), "healthy")
	// TODO Check that save is called?
}
