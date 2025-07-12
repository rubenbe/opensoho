# Home Assistant

The home assistant integration is currently limited, but can be used to notify when an OpenWRT device is no longer checking in with OpenSOHO (the router/AP becomes *unhealthy*)

First add a normal user to OpenSOHO using the admin interface. This is safer than using your admin account.
Pocketbase uses access tokens which are valid for one month. This means we need to update it reguraly.


## Generating an initial token
We need to generate an authentication token using the username and password that Home Assistant can use.
```
curl -X POST http://192.168.1.1:8090/api/collections/users/auth-with-password -H "Content-type:application/json" -d '{"identity":"EMAIL", "password":"PASSWORD"}'
```

This will return a JSON, but we're only interested in the `token` value, which needs to be used in the Home Assistant yaml configuration.

## Storing the token in Home Assistant
In Home assistant go to `Settings > Devices > Helpers`. Create a new input helper named `opensoho_access_token`. Set the length to 250 and set the type to password.
Paste your access token in here.

## Configuring the REST sensors

Next we can configure a REST sensor for each OpenWRT device.
The `DEVICE_ID` is the id found for the record in the `devices` collection.

```
  - platform: rest
    name: "OpenWrt-Router via OpenSOHO"
    resource: "http://192.168.1.1:8090/api/hass/v1/devicestatus/DEVICE_ID"
    headers:
      Authorization: "{{states.input_text.opensoho_access_token.state}}"
    device_class: connectivity
```

Reload the Home Assistant config after each config file modification. The sensor should now show a valid configuration!

To test/debug the token, you can run:
```
curl -H "Authorization: TOKEN" http://192.168.1.1:8090/api/hass/v1/devicestatus/DEVICE_ID
```

## Automatically updating the access token
Since the access token needs to be rotated monthly it's best to set up an automation:


*TODO* add details
