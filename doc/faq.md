# FAQ
## How do I not destroy my entire setup?
OK, since with OpenSOHO you can control multiple devices all at once, you might not want to deploy changes to your entire network at once.
This is particularly useful when fiddling with VLANs!
* First set all `Devices` to Disabled (so `Enabled` equals `False`)
* Modify one device or one setting. (e.g. turn the VLAN feature on by setting adding `VLAN` to the `Apply` list of a `Device`)
* Enable one device.
* Wait for the config to be deployed.
* Check the connectivity and decide your next step.

## How can I show/hide columns
On the right hand side of the column names there are 3 dots, which open a menu to toggle the visible columns.

## I forgot my admin password, how can I reset it?

Run this command to update your password
```sh
OPENSOHO_SHARED_SECRET=x ./opensoho superuser update <email> <new password>
```

## Can I reorder the collections?

No that's currently not possible (this is a limitations of pocketbase).
But collections can be *pinned* by clicking the little pushpin next to the collection name in the collection list.
