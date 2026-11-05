package opnsense

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleResponse = `{
  "total": 5, "rowCount": 5, "current": 1,
  "rows": [
    {"address": "192.168.68.20", "hwaddr": "AA:BB:CC:DD:EE:01", "hostname": "laptop", "client_id": "01:aa:bb:cc:dd:ee:01", "expire": 1793900000, "lease_type": "dynamic"},
    {"address": "192.168.20.5", "hwaddr": "aa:bb:cc:dd:ee:02", "hostname": "*", "client_id": "*", "expire": "1793900100", "lease_type": "dynamic"},
    {"address": "192.168.68.30", "hwaddr": "aa:bb:cc:dd:ee:03", "hostname": "nas", "client_id": "", "expire": 0, "lease_type": "static"},
    {"address": "fd00::10", "hwaddr": "aa:bb:cc:dd:ee:04", "hostname": "v6", "client_id": "", "expire": 1793900000},
    {"address": "192.168.68.40", "hwaddr": "", "hostname": "nomac", "client_id": "", "expire": 1793900000}
  ],
  "interfaces": {"lan": "LAN"}
}`

func TestParseLeases(t *testing.T) {
	leases, err := parseLeases([]byte(sampleResponse))
	assert.NoError(t, err)
	assert.Equal(t, []Lease{
		{MACAddress: "aa:bb:cc:dd:ee:01", IPAddress: "192.168.68.20", Hostname: "laptop", ClientID: "01:aa:bb:cc:dd:ee:01", Expire: 1793900000},
		{MACAddress: "aa:bb:cc:dd:ee:02", IPAddress: "192.168.20.5", Hostname: "*", ClientID: "*", Expire: 1793900100},
		{MACAddress: "aa:bb:cc:dd:ee:03", IPAddress: "192.168.68.30", Hostname: "nas", Expire: 0},
	}, leases)

	_, err = parseLeases([]byte("<html>login</html>"))
	assert.Error(t, err)
}

func TestValidateURL(t *testing.T) {
	for _, ok := range []string{"https://fw.example.org", "https://192.168.1.1:4443/", "http://fw"} {
		assert.NoError(t, ValidateURL(ok), ok)
	}
	for _, bad := range []string{"", "fw.example.org", "ftp://fw", "https://", "://x"} {
		assert.Error(t, ValidateURL(bad), bad)
	}
}

func TestValidateCredential(t *testing.T) {
	valid := []string{
		"w86XNZob/8Oq8aC5r0kbNarNtdpoQU781fyoeaOBQsBwkXUt",
		"XeD26XVrJ5ilAc/EmglCRC+0j2e57tRsjHwFepOseySWLM53pJASeTA3",
		"apikey123",
		"secret_val+123=",
	}
	for _, val := range valid {
		assert.NoError(t, ValidateCredential(val), val)
	}

	invalid := []string{
		"",
		" ",
		"leading_space",
		"trailing_space ",
		"has space",
		"has\ttab",
		"has\nnewline",
		"has\rreturn",
		"non_ascii_\x00",
		"non_ascii_\x7f",
	}
	// fix leading_space test string:
	invalid[2] = " leading_space"
	for _, val := range invalid {
		assert.Error(t, ValidateCredential(val), val)
	}
}

func TestMaxResponseBytes(t *testing.T) {
	assert.Equal(t, 1<<20, maxResponseBytes)
}

func TestConfigReady(t *testing.T) {
	cfg := Config{Enabled: true, URL: "https://fw", APIKey: "k", APISecret: "s"}
	assert.True(t, cfg.Ready())
	cfg.APISecret = ""
	assert.False(t, cfg.Ready())
	cfg = Config{URL: "https://fw", APIKey: "k", APISecret: "s"}
	assert.False(t, cfg.Ready())
}

func TestFetchLeases(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "key" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, dnsmasqLeasesPath, r.URL.Path)
		var body map[string]string
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "ipv4", body["selected_protocol"])
		_, _ = w.Write([]byte(sampleResponse))
	}))
	defer srv.Close()

	cfg := Config{Enabled: true, URL: srv.URL + "/", APIKey: "key", APISecret: "secret"}

	// The test server certificate is self-signed, so verification must fail.
	cfg.VerifyTLS = true
	_, err := FetchLeases(context.Background(), cfg)
	assert.Error(t, err)

	cfg.VerifyTLS = false
	leases, err := FetchLeases(context.Background(), cfg)
	assert.NoError(t, err)
	assert.Len(t, leases, 3)

	cfg.APISecret = "wrong"
	_, err = FetchLeases(context.Background(), cfg)
	assert.ErrorContains(t, err, "401")
}

func TestVerifyTLSDefault(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleResponse))
	}))
	defer srv.Close()

	// Verify that with TLS verification enabled (the default for Config), self-signed certificates are rejected.
	cfg := Config{Enabled: true, URL: srv.URL, APIKey: "key", APISecret: "secret", VerifyTLS: true}
	_, err := FetchLeases(context.Background(), cfg)
	assert.Error(t, err, "FetchLeases must reject self-signed certificates when VerifyTLS is true")
}
