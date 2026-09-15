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

## Encryption
  6GHz Wifi has stricter security regulations (wpa3 minimum).
  
  OpenSOHO will autoupgrade your encryption settings to comply with these 6GHz wifi regulations.
  e.g. when you choose `psk2+ccmp` as encryption on your SSID, OpenSOHO will auto upgrade this to `sae` on a 6GHz radio.
  This allows to have a backwards compatible encryption on the classic 2.4 and 5 GHz frequencies.
  
  The resulting configuration looks like this:
  ```
  ssh root@asusbt8 cat /etc/config/wireless  | grep "encryption\|wifi-iface\|wifi-device\|band\|option device" 
  config wifi-device 'radio0'
  	option band '2g'
  config wifi-device 'radio1'
  	option band '5g'
  config wifi-device 'radio2'
	  option band '6g'
  config wifi-iface 'wifi_0_radio0'
	  option device 'radio0'
	  option encryption 'psk2+ccmp'
  config wifi-iface 'wifi_0_radio1'
	  option device 'radio1'
	  option encryption 'psk2+ccmp'
  config wifi-iface 'wifi_0_radio2'
	  option device 'radio2'
	  option encryption 'sae'
  ```

## OpenWRT

Support for 6GHz on OpenWRT is still rolling out. 
You might want to have a look at [OpenWRT issue 20276](https://github.com/openwrt/openwrt/issues/20276)

## OpenWisp-config
There is an issue the current release of the OpenWRT OpenWisp config daemon, where it does not remove the default SSID on the 6GHz radio:
[openwisp-config issue 272](https://github.com/openwisp/openwisp-config/issues/272)
I've made a PR that has been merged in the meantime, so expect a fix in the next release of `openwisp-config`.
