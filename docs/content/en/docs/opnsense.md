---
title: "OPNsense DHCP Leases"
linkTitle: "OPNsense"
weight: 8
description: >
  Show client IP addresses and hostnames when OPNsense is the DHCP server.
---

## Overview

OpenSOHO displays client IP addresses and hostnames based on DHCP lease data. When your network uses an external OPNsense firewall as the DHCP server rather than a managed OpenWrt router, OpenSOHO can query OPNsense directly to show client IP addresses and hostnames in the dashboard.

With the OPNsense integration enabled, OpenSOHO queries the OPNsense REST API every minute to fetch active DHCP leases across all configured interfaces and VLANs.

## Compatibility

- **DHCP Backend:** OPNsense provides multiple DHCP server implementations (Dnsmasq, Kea DHCP, and legacy ISC DHCP), each exposing distinct REST API endpoints. This integration queries the **Dnsmasq** API endpoint (`/api/dnsmasq/leases/search`). Leases managed by Kea DHCP or ISC DHCP are not read.
- **Supported Versions:** Tested and verified with **OPNsense 25.1+** (where Dnsmasq is the default DHCP service, tested up to 26.7.x).

## Create an OPNsense API Key

1. In OPNsense, go to *System > Access > Users* and add a user, e.g. `opensoho`. It does not need a password or shell.
2. Under **Effective Privileges**, grant **Services: Dnsmasq DNS/DHCP: Settings**. Note that OPNsense does not have a separate read-only privilege for the lease list, so this privilege also allows changing Dnsmasq configuration. Keep the credentials secure.
3. Under *API keys*, click **+** to generate a key pair. OPNsense downloads a file containing the `key` and `secret`.

## Configure OpenSOHO

Add these rows to the `settings` collection:

| name                  | value                                                        |
|-----------------------|--------------------------------------------------------------|
| `opnsense_enabled`    | `true`                                                       |
| `opnsense_url`        | `https://192.168.1.1` (OPNsense web UI base URL)            |
| `opnsense_api_key`    | the API key                                                  |
| `opnsense_api_secret` | the API secret                                               |
| `opnsense_verify_tls` | *(optional)* `true` by default; set to `false` for self-signed certificates |

TLS certificates are verified by default (`true`). Set `opnsense_verify_tls` to `false` only if your OPNsense firewall uses a self-signed TLS certificate.

The leases appear within a minute. Errors or connection failures are logged as `Failed to fetch OPNsense DHCP leases`.

## Lease Handling

- Leases are matched to clients by MAC address; only IPv4 leases are used.
- Expired leases are removed automatically, the same as leases reported by OpenWrt devices.
- Static (reserved) leases have no expiry timestamp on OPNsense. They are kept for 10 minutes and refreshed every minute during synchronization, so they disappear shortly after the reservation is removed.
- Clients that do not send a hostname are shown without one.
