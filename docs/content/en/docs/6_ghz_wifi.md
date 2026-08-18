# 6 GHz Wifi
## 6 GHz wifi needs to be advertised on 2.4 or 5 GHz
Clients use the "Reduced Neighbor Report" on the lower frequencies to avoid needlessly scanning the 6GHz band.
Therefore OpenSOHO Requires you to select at least a 2.4 and a 5 GHz in the Wifi APs collection when selecting a 6GHz band.
## OpenWRT

6GHz Wifi on OpenWRT is still a bit tricky, I first had to set these values via Luci, since they were not set and this prevented the radio from being enabled.
If the radio does not show the supported channels, OpenSOHO will not be able to configure it. Fix this first. This is not a OpenSOHO issue.


```
uci set wireless.radio1.band='6g'
uci set wireless.radio1.channel='37'
uci set wireless.radio1.country='BE'
uci commit
reboot
```

Also have a look at [OpenWRT issue 20276](https://github.com/openwrt/openwrt/issues/20276)

## OpenWisp-config
There is an issue the current release of the OpenWRT OpenWisp config daemon, where it does not remove the default SSID on the 6GHz radio:
[openwisp-config issue 272](https://github.com/openwisp/openwisp-config/issues/272)
I've made a PR that has been merged in the meantime, so expect a fix in the next release of `openwisp-config`.
