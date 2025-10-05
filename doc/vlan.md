# VLAN

## Important information (read first!)

* VLAN configuration is tricky to get right, so OpenSOHO tries to applies its configuration in a way that limits possible connection loss.
* VLAN tagging is only applied when on a device "apply" contains `VLAN`. Removing `vlan` will *NOT* undo the configuration.
* LAN remains untagged on all ports (it is planned feature to override this for individual ports).
  The rationale is that the network should always keep working when VLAN tagging is enabled.
* CIDR settings on the `lan` will be ignored since forcing those would break the network.
  The CIDR settings are applied on the selected gateway.
* VLANs named `lan` and `wan` are extra protected from accidental modification.
  If you rename your `lan` or `wan` to something else, this protection will not be active.
* OpenSOHO takes the ethernet interfaces reported by the OpenWRT device (via OpenWISP monitoring) and will:
  * Add an untagged config (`u*`) towards the `lan` for each ethernet port.
  * Add a tagged config (`t`) for each ethernet port towards all other VLANs.
  * (currently nothing else is supported)

## Getting started
1. Under `Vlans`, ensure a `Vlan` named `lan` exists.
   Give it a number e.g. 100, this will be your `lan` VLAN id.
2. Under `Devices`, add `vlan` as `apply` value for each device that requires VLAN support.
   Do this *GRADUALLY* as there is no easy way back (using OpenSOHO).
   A good strategy is to slowly work your way up from the least important APs towards my main router.
3. When all (required) devices ave `vlan` applied, your network should be ready for additional VLANs.

## Configuring extra VLANs
1. Add an extra configuration entry under `Vlans`, at the minimum give it a `name` and a `number`.
   OpenSOHO will ensure this VLAN is available tagged on all interfaces on all devices (with the `vlan` setting applied)
2. You probably also want to add a CIDR to define the subnet e.g. `192.168.1.1/24`. The IP will be used for the subnet if you configure a gateway in the next step.
   The `/24` subnet size is best tested.
3. Configure a gateway, this is probably the same one as your gateway for your `lan`.
4. Optionally enable `DHCP`. The DHCP config is very simple, but sufficient for most `/24` networks. A 12h lease for addresses from `100` until `249` (the OpenWRT default).
   * Subnets smaller than `/24` will not get a DHCP at this moment, since the `100->249` range is probably not a valid one. Use Luci instead.
   * Subnets bigger than `/24` will get the same DHCP config. This is probably not what you want. Use Luci instead.
5. Wifi interfaces can also be added, via the `Wifi` config.
