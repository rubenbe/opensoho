# VLAN cofiguration

## Important information (read first!)
* VLAN configuration is tricky to get right, so OpenSOHO tries to applies its configuration in a way that limits possible connection loss.
* VLAN tagging is only applied when on a device "apply" contains `VLAN`. Removing `vlan` will *NOT* undo the configuration.
* LAN remains untagged on all ports (it is planned feature to override this for individual ports).
  The rationale is that the network should always keep working when VLAN tagging is enabled.
* CIDR settings on the `lan` will be ignored since forcing those would break the network.
  The CIDR settings are applied on the selected gateway only. The interface on APs and switches be configured with `option proto 'none'`.
* VLANs named `lan` and `wan` are extra protected from accidental modification.
  If you rename your `lan` or `wan` to something else, this protection will not be active.
* OpenSOHO takes the ethernet interfaces reported by the OpenWRT device (via OpenWISP monitoring) and will:
  * Add an untagged config (`u*`) towards the `lan` for each ethernet port.
  * Add a tagged config (`t`) for each ethernet port towards all other VLANs.
  * (currently nothing else is supported)

## Getting started
1. Under `Vlans`, ensure a `Vlan` named `lan` exists.
   Give it a number e.g. 100, this will be your `lan` VLAN id. Avoid VLAN IDs 1 and 2.

<p align="center">
  <a href="https://raw.githubusercontent.com/opensoho/assets/16c557f7d38a467ed81d0f5119c5c1e1dedff787/vlan/vlan.png" target="_blank">
    <img src="https://raw.githubusercontent.com/opensoho/assets/16c557f7d38a467ed81d0f5119c5c1e1dedff787/vlan/vlan.png" width="80%" />
  </a>
</p>

2. Under `Devices`, add `vlan` as `apply` value for each device that requires VLAN support.
   Do this *GRADUALLY* as there is no easy way back (using OpenSOHO).
   A good strategy is to slowly work your way up from the least important APs towards my main router.
3. When all (required) devices ave `vlan` applied, your network should be ready for additional VLANs.

## Configuring extra VLANs
1. Add an extra configuration entry under `Vlans`. At the minimum give it a `name` and a `number`. e.g. `guest` and `200`
   OpenSOHO will ensure this VLAN is available tagged on all interfaces on all devices (with the `vlan` setting applied)
2. Configure a gateway, this is probably the same one as your gateway for your `lan`.
3. Add a CIDR to define the subnet e.g. `192.168.1.1/24`. The IP will be used for the subnet if you configure a gateway in the next step.
   The `/24` subnet size is best tested. OpenSOHO will add this IP address on the gateway device only, the other devices will not get an IP address.
4. Wifi interfaces can be added, via the `Wifi` config.
5. OpenSOHO does not configure DHCP or your firewall yet (this is planned). Use Luci on the gateway device for that.
   * A good tutorial can be found on the [OpenWRT wiki](https://openwrt.org/docs/guide-user/network/wifi/guestwifi/configuration_webinterface#firewall).
   * To avoid MTU problems, you also want to enable `MSS clamping` on the firewall zones on the gateway device.
