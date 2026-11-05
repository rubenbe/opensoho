// Package opnsense reads the DHCP leases of an OPNsense firewall through its
// REST API, so OpenSOHO can show client IP addresses and hostnames when the
// DHCP server is an external OPNsense firewall.
//
// Currently, the Dnsmasq DHCP service (OPNsense 25.1 and later) is supported.
package opnsense

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	dnsmasqLeasesPath = "/api/dnsmasq/leases/search"
	requestTimeout    = 15 * time.Second
	maxResponseBytes  = 1 << 20
)

// Config holds the API connection parameters, sourced from the settings
// collection.
type Config struct {
	Enabled   bool
	URL       string // base URL, e.g. https://firewall.example.org
	APIKey    string
	APISecret string
	VerifyTLS bool
}

// Ready reports whether the config is enabled and complete.
func (c Config) Ready() bool {
	return c.Enabled && c.URL != "" && c.APIKey != "" && c.APISecret != ""
}

// Lease is an IPv4 DHCP lease as reported by OPNsense.
type Lease struct {
	MACAddress string // lowercase, colon separated
	IPAddress  string
	Hostname   string // "*" or empty when the client sent none
	ClientID   string
	Expire     int64 // unix seconds; 0 means the lease never expires
}

type dnsmasqLease struct {
	Address  string          `json:"address"`
	HWAddr   string          `json:"hwaddr"`
	Hostname string          `json:"hostname"`
	ClientID string          `json:"client_id"`
	Expire   json.RawMessage `json:"expire"`
}

type dnsmasqSearchResponse struct {
	Rows []dnsmasqLease `json:"rows"`
}

// ValidateURL checks that raw is an absolute http(s) URL with a host.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("scheme must be https or http")
	}
	if u.Host == "" {
		return errors.New("missing host")
	}
	return nil
}

// ValidateCredential checks that an API key or secret contains only valid printable characters without whitespace.
func ValidateCredential(s string) error {
	if s == "" {
		return errors.New("credential cannot be empty")
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return errors.New("credential must not contain whitespace")
	}
	for _, r := range s {
		if r < 33 || r > 126 {
			return errors.New("credential must contain only printable ASCII characters")
		}
	}
	return nil
}

// FetchLeases returns the IPv4 Dnsmasq leases of the OPNsense instance in cfg.
func FetchLeases(ctx context.Context, cfg Config) ([]Lease, error) {
	if err := ValidateURL(cfg.URL); err != nil {
		return nil, fmt.Errorf("invalid OPNsense URL: %w", err)
	}

	body, _ := json.Marshal(map[string]string{"selected_protocol": "ipv4"})
	endpoint := strings.TrimRight(cfg.URL, "/") + dnsmasqLeasesPath

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cfg.APIKey, cfg.APISecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !cfg.VerifyTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in for self-signed firewall certificates
	}
	client := &http.Client{Transport: transport, Timeout: requestTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPNsense returned %s", resp.Status)
	}
	return parseLeases(data)
}

func parseLeases(data []byte) ([]Lease, error) {
	var parsed dnsmasqSearchResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("unexpected OPNsense response: %w", err)
	}

	leases := make([]Lease, 0, len(parsed.Rows))
	for _, row := range parsed.Rows {
		ip := net.ParseIP(row.Address)
		if ip == nil || ip.To4() == nil {
			continue
		}
		mac, err := net.ParseMAC(row.HWAddr)
		if err != nil || len(mac) != 6 {
			continue
		}
		leases = append(leases, Lease{
			MACAddress: mac.String(),
			IPAddress:  ip.String(),
			Hostname:   row.Hostname,
			ClientID:   row.ClientID,
			Expire:     parseExpire(row.Expire),
		})
	}
	return leases, nil
}

// parseExpire accepts the expiry as a JSON number or a numeric string.
func parseExpire(raw json.RawMessage) int64 {
	var n json.Number
	if err := json.Unmarshal(raw, &n); err != nil {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return 0
		}
		n = json.Number(s)
	}
	v, err := n.Int64()
	if err != nil || v < 0 {
		return 0
	}
	return v
}
