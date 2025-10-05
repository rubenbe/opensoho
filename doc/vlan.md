# VLAN

VLAN support 

* VLAN configuration is tricky to get right, so OpenSOHO tries to applies its configuration in a way that limits possible connection loss.
* VLAN tagging is only applied when on a device "apply" contains `VLAN`. Removing `vlan` will *NOT* undo the configuration.
* LAN remains untagged on all ports (it is planned feature to override this for individual ports).
  The rationale is that the network should always keep working when VLAN tagging is enabled.
* CIDR settings on the `lan` will be ignored since forcing those would break the network.
  The CIDR settings are applied on the selected gateway.
