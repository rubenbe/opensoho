---
title: "FAQ"
linkTitle: "FAQ"
weight: 10
description: >
  Frequently asked questions about OpenSOHO.
---

## How do I avoid breaking my entire setup?

Since OpenSOHO can push configuration to multiple devices at once, deploy changes gradually — especially for VLANs.

1. Set all `Devices` to Disabled (`Enabled = false`).
2. Modify one device or one setting (e.g. add `VLAN` to the `Apply` list of a `Device`).
3. Enable one device.
4. Wait for the config to deploy.
5. Check connectivity, then decide your next step.

## How do I show or hide columns?

On the right-hand side of the column headers there are 3 dots, which open a menu to toggle visible columns.

## I forgot my admin password — how do I reset it?

```sh
OPENSOHO_SHARED_SECRET=x ./opensoho superuser update <email> <new password>
```

## Can I reorder collections?

No — this is a limitation of PocketBase. Collections can be **pinned** by clicking the pushpin icon next to the collection name in the collection list.


## My device IPs are not correct when running behind a reverse proxy

In `Settings` > `Application` you need to enable `User IP proxy headers`.
You might also need to configure your proxy to forward the necessary headers.
More information can be found in [the pocketbase documentation](https://pocketbase.io/docs/going-to-production/#using-reverse-proxy).

## When registering a device I get error "curl exit code X"

This exit code often gives a good indication of the root cause of the problem. For example:

```
daemon.err: openwisp: Failed to connect to controller during registration: curl exit code 6
```

Search online these "curl exit codes", or open the manpage of curl on your machine.

e.g. exit code 6 means 

```
6      Could not resolve host. The given remote host could not be resolved.
```

Check your AP's DNS settings in that case. This can differ from the what your main router is distributing via DHCP.

## When I enable option `X` my radios get disabled.

Certain wifi options require `wpad` instead of `wpad-basic`. E.g usteer is one of these options. Details can be found on the [usteer page](/docs/usteer/).

Also see [this post](https://forum.openwrt.org/t/wifi-disabled-when-i-use-80211k-option/211100) on the OpenWrt forum.

You'll see this in `logread` too:


```sh
# logread | grep unknown
Tue Aug 18 18:40:52 2026 daemon.err hostapd: Line 77: unknown configuration item 'proxy_arp'
Tue Aug 18 18:40:52 2026 daemon.err hostapd: Line 78: unknown configuration item 'bss_transition'
Tue Aug 18 18:40:52 2026 daemon.err hostapd: Line 79: unknown configuration item 'wnm_sleep_mode'
Tue Aug 18 18:40:52 2026 daemon.err hostapd: Line 80: unknown configuration item 'wnm_sleep_mode_no_keys'

```
