---
title: "6GHz Wifi"
linkTitle: "6GHz Wifi"
weight: 3
description: >
  Configure 6GHz Wifi6E and Wifi7 APs.
---

# 6 GHz Wifi
## 6 GHz wifi needs to be advertised on 2.4 or 5 GHz
Clients use the "Reduced Neighbor Report" on the lower frequencies to avoid needlessly scanning the 6GHz band.
Therefore OpenSOHO Requires you to select at least a 2.4 and a 5 GHz in the Wifi APs collection when selecting a 6GHz band.

## Recommended settings:
* Ensure you set a valid `country` in `Settings`. This is a hard requirement, since 6GHz is a regulatory hellhole and your radio wants to know in which country it is before enabling at all.
* For the radio:
  * Keep the `tx_power_mode` on `auto`. This translates to Luci: `Maximum transmit power` on `driver default`
  * Frequency `auto` works as expected
  * Set `enabled` to `true`

## OpenWRT

Support for 6GHz on OpenWRT is still rolling out. 
You might want to have a look at [OpenWRT issue 20276](https://github.com/openwrt/openwrt/issues/20276)

## OpenWisp-config
There is an issue the current release of the OpenWRT OpenWisp config daemon, where it does not remove the default SSID on the 6GHz radio:
[openwisp-config issue 272](https://github.com/openwisp/openwisp-config/issues/272)
I've made a PR that has been merged in the meantime, so expect a fix in the next release of `openwisp-config`.
