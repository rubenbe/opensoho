# Home Assistant

The home assistant integration is currently limited, but can be used to notify when an OpenWRT device is no longer checking in (becomes *unhealthy*)

First add a normal user to OpenSOHO using the admin interface.

Now we need to generate an authentication token using the username and password  that Home Assistant can use.

```
curl -X POST http://192.168.1.1:8090/api/collections/users/auth-with-password -H "Content-type:application/json" -d '{"identity":"EMAIL", "password":"PASSWORD"}'
```

This will return a JSON, but we're only interested in the `token` value, which needs to be used in the Home Assistant yaml configuration.

The device ID is the id found for the record in the `devices` collection.

```
  - platform: rest
    name: "OpenWrt-Router via OpenSOHO"
    resource: "http://192.168.1.1:8090/api/hass/v1/devicestatus/DEVICE_ID"
    headers:
      Authorization: "AUTHTOKEN"
    device_class: connectivity
```

Reload the Home Assistant config after each config file modification.
