---
title: "Home Assistant Integration"
linkTitle: "Home Assistant"
weight: 4
description: >
  Monitor OpenWRT device health in Home Assistant using the OpenSOHO REST API.
---

The Home Assistant integration can notify you when an OpenWRT device stops checking in with OpenSOHO (the router/AP becomes `unhealthy`).

## Create an API User

Add an API user in the OpenSOHO admin interface — safer than using your admin account.
OpenSOHO access tokens are valid for 3 years.

## Generate an Initial Token

```sh
curl -X POST http://192.168.1.1:8090/api/collections/api_users/auth-with-password \
  -H "Content-type: application/json" \
  -d '{"identity":"name", "password":"PASSWORD"}'
```

This returns a JSON object. Copy the `token` value for use in Home Assistant.

## Store the Token in Home Assistant

Go to `Settings > Devices > Helpers`. Create a new input helper:

* Name: `opensoho_access_token`
* Type: password
* Max length: 250

Paste your access token into this helper.

## Configure REST Sensors

Add a REST sensor for each OpenWRT device.
The `DEVICE_MAC` is the MAC address found in the `devices` collection.

```yaml
- platform: rest
  name: "OpenWrt-Router via OpenSOHO"
  resource: "http://192.168.1.1:8090/api/v1/devicestatus/DEVICE_MAC"
  headers:
    Authorization: "{{states.input_text.opensoho_access_token.state}}"
  device_class: connectivity
```

Reload the Home Assistant config after each change. The sensor should now show a valid state.

To test the token:

```sh
curl -H "Authorization: TOKEN" http://192.168.1.1:8090/api/v1/devicestatus/00:11:22:33:44:55
```

## Updating the Access Token

Request a new token before the old one expires:

```sh
curl -X POST -H "Authorization: OLDTOKEN" \
  http://192.168.1.1:8090/api/collections/api_users/auth-refresh
```
