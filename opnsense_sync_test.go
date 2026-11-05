package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/rubenbe/opensoho/integrations/opnsense"
	"github.com/stretchr/testify/assert"
)

func addSetting(t *testing.T, app core.App, collection *core.Collection, name, value string) {
	r := core.NewRecord(collection)
	r.Set("name", name)
	r.Set("value", value)
	assert.NoError(t, app.Save(r))
}

func TestValidateOPNsenseSettings(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	s := core.NewRecord(setupSettingsCollection(t, app))
	for _, name := range []string{"opnsense_enabled", "opnsense_verify_tls"} {
		s.Set("name", name)
		for _, value := range []string{"", "true", "false"} {
			s.Set("value", value)
			assert.Nil(t, validateSetting(s), name+"="+value)
		}
		s.Set("value", "yes")
		assert.Error(t, validateSetting(s), name)
	}

	s.Set("name", "opnsense_url")
	for _, value := range []string{"", "https://192.168.1.1", "https://fw.example.org:4443/"} {
		s.Set("value", value)
		assert.Nil(t, validateSetting(s), value)
	}
	for _, value := range []string{"192.168.1.1", "ftp://fw", "https://"} {
		s.Set("value", value)
		assert.Error(t, validateSetting(s), value)
	}

	for _, name := range []string{"opnsense_api_key", "opnsense_api_secret"} {
		s.Set("name", name)
		for _, value := range []string{"", "key123", "w86XNZob/8Oq8aC5r0kbNarNtdpoQU781fyoeaOBQsBwkXUt"} {
			s.Set("value", value)
			assert.Nil(t, validateSetting(s), name+"="+value)
		}
		for _, value := range []string{"has space", "has\tbreak", "has\nnewline"} {
			s.Set("value", value)
			assert.Error(t, validateSetting(s), name+"="+value)
		}
	}
}

func TestLoadOPNsenseConfig(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	settings := setupSettingsCollection(t, app)
	assert.Equal(t, opnsense.Config{VerifyTLS: true}, loadOPNsenseConfig(app))

	addSetting(t, app, settings, "opnsense_enabled", "true")
	addSetting(t, app, settings, "opnsense_url", "https://fw")
	addSetting(t, app, settings, "opnsense_api_key", "key")
	addSetting(t, app, settings, "opnsense_api_secret", "secret")
	addSetting(t, app, settings, "opnsense_verify_tls", "false")
	assert.Equal(t, opnsense.Config{
		Enabled: true, URL: "https://fw", APIKey: "key", APISecret: "secret", VerifyTLS: false,
	}, loadOPNsenseConfig(app))
}

func TestLoadOPNsenseConfig_VerifyTLSDefault(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	settings := setupSettingsCollection(t, app)
	addSetting(t, app, settings, "opnsense_enabled", "true")
	addSetting(t, app, settings, "opnsense_url", "https://fw")
	addSetting(t, app, settings, "opnsense_api_key", "key")
	addSetting(t, app, settings, "opnsense_api_secret", "secret")

	// opnsense_verify_tls is omitted: verify that VerifyTLS defaults to true.
	cfg := loadOPNsenseConfig(app)
	assert.True(t, cfg.VerifyTLS, "VerifyTLS must default to true when opnsense_verify_tls is omitted")
}

func TestOPNsenseToDHCPLeases(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	leases := opnsenseToDHCPLeases([]opnsense.Lease{
		{MACAddress: "aa:bb:cc:dd:ee:01", IPAddress: "10.0.0.1", Hostname: "a", ClientID: "c", Expire: 2_000_000},
		{MACAddress: "aa:bb:cc:dd:ee:02", IPAddress: "10.0.0.2", Hostname: "*"},
	}, now)
	assert.Equal(t, []DHCPLease{
		{MACAddress: "aa:bb:cc:dd:ee:01", ClientID: "c", Hostname: "a", IPAddress: "10.0.0.1", Expiry: 2_000_000},
		{MACAddress: "aa:bb:cc:dd:ee:02", Hostname: "*", IPAddress: "10.0.0.2", Expiry: int(now.Add(opnsenseStaticLeaseTTL).Unix())},
	}, leases)
}

func TestSyncOPNsenseLeases(t *testing.T) {
	app, err := tests.NewTestApp()
	assert.Nil(t, err)
	defer app.Cleanup()

	settings := setupSettingsCollection(t, app)
	setupDhcpLeaseCollection(t, app)

	future := time.Now().Add(time.Hour).Unix()
	past := time.Now().Add(-time.Hour).Unix()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"rows": [
			{"address": "192.168.68.20", "hwaddr": "AA:BB:CC:DD:EE:01", "hostname": "laptop", "client_id": "", "expire": ` + strconv.FormatInt(future, 10) + `},
			{"address": "192.168.68.30", "hwaddr": "aa:bb:cc:dd:ee:03", "hostname": "*", "client_id": "", "expire": 0},
			{"address": "192.168.68.40", "hwaddr": "aa:bb:cc:dd:ee:04", "hostname": "old", "client_id": "", "expire": ` + strconv.FormatInt(past, 10) + `}
		]}`))
	}))
	defer srv.Close()

	// Disabled: no request is made.
	addSetting(t, app, settings, "opnsense_url", srv.URL)
	addSetting(t, app, settings, "opnsense_api_key", "key")
	addSetting(t, app, settings, "opnsense_api_secret", "secret")
	syncOPNsenseLeases(app)
	assert.Equal(t, 0, calls)

	addSetting(t, app, settings, "opnsense_enabled", "true")
	syncOPNsenseLeases(app)
	assert.Equal(t, 1, calls)

	records, err := app.FindAllRecords("dhcp_leases")
	assert.NoError(t, err)
	byMAC := map[string]*core.Record{}
	for _, r := range records {
		byMAC[r.GetString("mac_address")] = r
	}
	assert.Len(t, byMAC, 2)
	assert.Equal(t, "192.168.68.20", byMAC["aa:bb:cc:dd:ee:01"].GetString("ip_address"))
	assert.Equal(t, "laptop", byMAC["aa:bb:cc:dd:ee:01"].GetString("hostname"))
	assert.Equal(t, "192.168.68.30", byMAC["aa:bb:cc:dd:ee:03"].GetString("ip_address"))
	assert.Equal(t, "", byMAC["aa:bb:cc:dd:ee:03"].GetString("hostname"))
	assert.True(t, byMAC["aa:bb:cc:dd:ee:03"].GetDateTime("expiry").Time().After(time.Now()))
}
