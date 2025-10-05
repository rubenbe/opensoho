# VLAN

# Important information (read first!)

* VLAN configuration is tricky to get right, so OpenSOHO tries to applies its configuration in a way that limits possible connection loss.
* VLAN tagging is only applied when on a device "apply" contains `VLAN`. Removing `vlan` will *NOT* undo the configuration.
* LAN remains untagged on all ports (it is planned feature to override this for individual ports).
  The rationale is that the network should always keep working when VLAN tagging is enabled.
* CIDR settings on the `lan` will be ignored since forcing those would break the network.
  The CIDR settings are applied on the selected gateway.

# Getting started
1. Under `Vlans`, ensure a `Vlan` named `lan` exists.
   Give it a number e.g. 100, this will be your `lan` VLAN id.
2. Under `Devices`, add `vlan` as `apply` value for each device that requires VLAN support.
   Do this *GRADUALLY* as there is no easy way back (using OpenSOHO).
   A good strategy is to slowly work your way up from the least important APs towards my main router.
3. When all (required) devices ave `vlan` applied, your network should be ready for additional `VLAN`s.
