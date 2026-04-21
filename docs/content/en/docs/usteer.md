---
title: "Usteer (Band Steering)"
linkTitle: "Usteer"
weight: 6
description: >
  Automatic client roaming using the OpenWRT usteer daemon.
---

## Overview

[Usteer](https://openwrt.org/docs/guide-user/network/wifi/usteer) is an OpenWRT daemon that steers Wi-Fi clients toward the best available access point. When enabled, OpenSOHO generates `/etc/config/usteer` on the device with sensible defaults for roaming behaviour.

## Prerequisites

Usteer requires the full `wpad` package (not `wpad-basic`) and the `usteer` package:

```sh
# OpenWRT 24.10
opkg remove wpad-basic-mbedtls && opkg install wpad-mbedtls usteer && service wpad restart

# OpenWRT 25.12+
apk del wpad-basic-mbedtls && apk add wpad-mbedtls usteer && service wpad restart
```

## Enabling usteer for an SSID

Open the **Wifi SSIDs** configuration and enable the following fields on the SSID(s) you want usteer:

| Field | Required | Purpose |
|-------|----------|---------|
| `ieee80211v_bss_transition` | Yes | BSS Transition frames — the mechanism usteer uses to steer clients |
| `ieee80211k` | Yes | Neighbor reports — lets clients discover better APs |
| `usteer` | Yes | Tells OpenSOHO to enable usteer on this SSID |
| `ieee80211r` | Recommended | Allows faster roaming between APs |

## Generated configuration

OpenSOHO creates a hardcoded config `/etc/config/usteer` with these settings:

```
config usteer 'usteer'
        option enabled '1'
        option network 'lan'
        option roam_trigger '-70'
        option min_signal '-78'
        option probe_steering '1'
        option deny_assoc '1'
        option band_steering '1'
```
The [OpenWRT wiki](https://openwrt.org/docs/guide-user/network/wifi/usteer) explains these values

The thresholds below are also visualised on the dashboard.

| Option | Value | Meaning |
|--------|-------|---------|
| `roam_trigger` | −70 dBm | Signal below this triggers a steering attempt |
| `min_signal` | −78 dBm | Signal below this may be denied new associations |

## Difference from the `client_steering` collection

Usteer and client_steering are two independent mechanisms and can be used together.
You should use `usteer` for your client steering.
