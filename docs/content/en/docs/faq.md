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
