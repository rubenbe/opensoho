---
title: "VLAN Configuration"
linkTitle: "VLANs"
weight: 3
description: >
  Configure VLANs across your OpenWRT devices with OpenSOHO.
---

## Important (read first!)

* VLAN configuration is tricky to get right, so OpenSOHO applies its configuration in a way that limits possible connection loss.
* VLAN tagging is only applied when a device's `apply` list contains `VLAN`. Removing `vlan` does **not** undo the configuration.
* LAN remains untagged on all ports (configuring this per-port is a planned feature).
  The rationale is to keep the network functional when VLAN tagging is enabled.
* CIDR settings on `lan` are ignored — forcing those would break the network.
  CIDR settings are applied on the selected gateway only. Interfaces on APs and switches should be configured with `option proto 'none'`.
* VLANs named `lan` and `wan` are extra protected from accidental modification.
  If you rename your `lan` or `wan`, this protection is not active.
* OpenSOHO takes the Ethernet interfaces reported by the OpenWRT device (via OpenWISP monitoring) and will:
  * Add an untagged config (`u*`) towards the `lan` for each Ethernet port.
  * Add a tagged config (`t`) for each Ethernet port towards all other VLANs.

## Getting Started

1. Under `Vlans`, ensure a `Vlan` named `lan` exists.
   Give it a number, e.g. `100` — this is your `lan` VLAN ID. Avoid VLAN IDs 1 and 2.

<p align="center">
  <a href="https://raw.githubusercontent.com/opensoho/assets/16c557f7d38a467ed81d0f5119c5c1e1dedff787/vlan/vlan.png" target="_blank">
    <img src="https://raw.githubusercontent.com/opensoho/assets/16c557f7d38a467ed81d0f5119c5c1e1dedff787/vlan/vlan.png" width="80%" />
  </a>
</p>

2. Under `Devices`, add `vlan` as an `apply` value for each device that requires VLAN support.
   Do this **gradually** — there is no easy way back via OpenSOHO.
   Work from the least important APs towards the main router.
3. When all required devices have `vlan` applied, your network is ready for additional VLANs.

## Configuring Extra VLANs

1. Add an extra entry under `Vlans`. At minimum, give it a `name` and a `number`, e.g. `guest` and `200`.
   OpenSOHO ensures this VLAN is available, tagged on all interfaces, on all devices with `vlan` applied.
2. Configure a gateway (usually the same as the `lan` gateway).
3. Add a CIDR to define the subnet, e.g. `192.168.1.1/24`.
   The `/24` subnet size is well tested. OpenSOHO adds this IP address on the gateway only.
4. Wifi interfaces can be added via the `Wifi` config.
5. OpenSOHO does not configure DHCP or the firewall yet (planned). Use LuCI on the gateway for that.
   * See the [OpenWRT wiki guest wifi guide](https://openwrt.org/docs/guide-user/network/wifi/guestwifi/configuration_webinterface#firewall).
   * To avoid MTU problems, enable `MSS clamping` on the firewall zones on the gateway device.
