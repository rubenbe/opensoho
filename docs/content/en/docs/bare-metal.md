---
title: "Bare Metal Deployments"
linkTitle: "Bare Metal"
weight: 3
description: >
  Run OpenSOHO directly on a server as a standalone binary.
---

## Overview

OpenSOHO ships as a single self-contained golang binary with no external
dependencies. Running it directly on a host is the simplest deployment.

## Download

Download the latest release for your platform from
[GitHub Releases](https://github.com/rubenbe/opensoho/releases) and make it
executable:

```sh
chmod +x opensoho
```

## Run

Choose a shared secret that lets OpenWRT devices register with OpenSOHO. Use a
long random string — it must match what you configure in LuCI.

```sh
OPENSOHO_SHARED_SECRET=LoNgExAmPleStrInGoF32cHarActeRs5 ./opensoho serve --http 0.0.0.0:8090
```

OpenSOHO stores all configuration and history in a `pb_data` directory next to
the binary.

## Example systemd service file

To keep OpenSOHO running across reboots, create a systemd unit e.g.
`/etc/systemd/system/opensoho.service`:

```ini
[Unit]
Description=OpenSOHO
After=network.target

[Service]
ExecStart=/opt/opensoho/opensoho serve --http 0.0.0.0:8090
WorkingDirectory=/opt/opensoho
Environment=OPENSOHO_SHARED_SECRET=LoNgExAmPleStrInGoF32cHarActeRs5
Restart=on-failure
User=opensoho

[Install]
WantedBy=multi-user.target
```

Then enable and start it:

```sh
systemctl daemon-reload
systemctl enable --now opensoho
```
