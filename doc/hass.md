# Home Assistant

The Home Assistant integration is currently limited, but can be used to notify when an OpenWRT device is no longer checking in with OpenSOHO (the router/AP becomes *unhealthy*)

First add an API user to OpenSOHO using the admin interface. This is safer than using your admin account.
OpenSOHO uses access tokens which are valid for 3 years.


## Generating an initial token
We need to generate an authentication token using the username and password that Home Assistant can use.
```
curl -X POST http://192.168.1.1:8090/api/collections/api_users/auth-with-password -H "Content-type:application/json" -d '{"identity":"name", "password":"PASSWORD"}'
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
    resource: "http://192.168.1.1:8090/api/v1/devicestatus/DEVICE_MAC"
    headers:
      Authorization: "{{states.input_text.opensoho_access_token.state}}"
    device_class: connectivity
```

Reload the Home Assistant config after each config file modification. The sensor should now show a valid configuration!

To test/debug the token, you can run:
```
curl -H "Authorization: TOKEN" http://192.168.1.1:8090/api/v1/devicestatus/00:11:22:33:44:55
```

## Updating the access token
In case a token is about to expire, you can request a new one via an API call:

```
curl -X POST -H "Authorization: OLDTOKEN" http://192.168.1.1:8090/api/collections/api_users/auth-refresh
```
