package main

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/rubenbe/opensoho/integrations/opnsense"
)

// Static leases have no expiry on OPNsense. They are stored with this TTL and
// refreshed every sync, so they disappear once removed from the firewall.
const opnsenseStaticLeaseTTL = 10 * time.Minute

// loadOPNsenseConfig reads the opnsense_* rows from the settings collection.
// TLS verification is on unless opnsense_verify_tls is "false".
func loadOPNsenseConfig(app core.App) opnsense.Config {
	cfg := opnsense.Config{VerifyTLS: true}
	records, err := app.FindAllRecords("settings")
	if err != nil {
		app.Logger().Warn("Failed to load OPNsense settings", "error", err)
		return cfg
	}
	for _, rec := range records {
		value := rec.GetString("value")
		switch rec.GetString("name") {
		case "opnsense_enabled":
			cfg.Enabled = value == "true"
		case "opnsense_url":
			cfg.URL = value
		case "opnsense_api_key":
			cfg.APIKey = value
		case "opnsense_api_secret":
			cfg.APISecret = value
		case "opnsense_verify_tls":
			cfg.VerifyTLS = value != "false"
		}
	}
	return cfg
}

// opnsenseToDHCPLeases converts OPNsense leases to the format OpenWrt devices
// report, so both sources share storeDHCPLeases.
func opnsenseToDHCPLeases(leases []opnsense.Lease, now time.Time) []DHCPLease {
	out := make([]DHCPLease, 0, len(leases))
	for _, l := range leases {
		expiry := l.Expire
		if expiry == 0 {
			expiry = now.Add(opnsenseStaticLeaseTTL).Unix()
		}
		out = append(out, DHCPLease{
			MACAddress: l.MACAddress,
			ClientID:   l.ClientID,
			Hostname:   l.Hostname,
			IPAddress:  l.IPAddress,
			Expiry:     int(expiry),
		})
	}
	return out
}

func syncOPNsenseLeases(app core.App) {
	cfg := loadOPNsenseConfig(app)
	if !cfg.Ready() {
		return
	}
	leases, err := opnsense.FetchLeases(context.Background(), cfg)
	if err != nil {
		app.Logger().Warn("Failed to fetch OPNsense DHCP leases", "error", err)
		return
	}
	storeDHCPLeases(app, opnsenseToDHCPLeases(leases, time.Now()), types.NowDateTime())
}
