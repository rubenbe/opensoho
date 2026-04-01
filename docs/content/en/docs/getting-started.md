---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  How to install and configure OpenSOHO.
---

## Overview

Setting up OpenSOHO requires a few simple steps:

1. Install OpenSOHO on your server.
2. Install the OpenWisp packages on your OpenWRT devices.
3. Configure OpenWisp on each OpenWRT device to connect to OpenSOHO.
4. Start configuring in OpenSOHO!

## Install OpenSOHO

Download the latest OpenSOHO release (or use a Docker container) from
[GitHub Releases](https://github.com/rubenbe/opensoho/releases).

Choose a shared secret that allows OpenWRT devices to register with OpenSOHO.
Use a long random string — it must match what you configure in LuCI.

```sh
OPENSOHO_SHARED_SECRET=randompassphrase ./opensoho serve --http 0.0.0.0:8090
```

OpenSOHO prints a URL to create the admin account. Open it in your browser, or use the CLI:

```sh
./opensoho superuser upsert EMAIL PASS
```

## Configure OpenWRT Devices

### Install the OpenWISP packages

On OpenWRT 24.10:

```sh
opkg install openwisp-config openwisp-monitoring luci-app-openwisp
```

On OpenWRT 25.12+:

```sh
apk add openwisp-config openwisp-monitoring luci-app-openwisp
```

If you want 802.11v client steering, replace the basic wpad:

On OpenWRT 24.10:

```sh
opkg remove wpad-basic-mbedtls && opkg install wpad-mbedtls && service wpad restart
```

On OpenWRT 25.12+:

```sh
apk del wpad-basic-mbedtls && apk add wpad-mbedtls && service wpad restart
```

### Configure OpenWISP in LuCI

* Set the `Server URL` and `Shared secret` only.
  * Do **not** append a slash to the `Server URL`.
  * The shared secret must match `OPENSOHO_SHARED_SECRET`.
* Optionally lower the `Update Interval` to 30 seconds for faster updates.

Enable monitoring — OpenSOHO deduces OpenWRT settings from monitoring data.

## Configure OpenSOHO

1. Wait for OpenWRT devices to self-register using the shared secret.
2. Set the device `Enabled` flag to `true`.
3. Set `numradios` to the correct value (e.g., `2` for 2.4 + 5 GHz).
4. Set up a Wifi access point (SSID + KEY).
5. Attach the Wifi access point to a device.

OpenSOHO is accessible at `http://ipaddress:8090/_/`

## Configuration Collections

### Clients

Connected Wifi clients — read-only except for the alias field.
Use aliases to give devices a human-readable name (only works when the client does not randomize its MAC address).

### Devices

Connected OpenWRT devices.

* **Enabled/Disabled** — disables configuration updates while keeping monitoring active.
* **Health status** — `healthy` means the device checked in within the last minute.
* **Numradios** — number of radios on the device; must be set manually.
* **Wifis** — select SSIDs to apply on this device.

### Radios

Set the frequency for each radio. Do not modify the band field.

### Wifi

Configure SSIDs. WEP and Open encryption are not supported by design.

## Troubleshooting

### Reregister a Device

If changing the OpenWISP `Server URL` in LuCI doesn't register with the new controller:

```sh
uci delete openwisp.http.uuid
uci delete openwisp.http.key
/etc/init.d/openwisp-config restart
```

### First SSID Not Enabled

There is a known OpenWRT issue where the first configured SSID is not auto-enabled.
Work-around: click the enable button in LuCI.
