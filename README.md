*Warning this is an early release*

# OpenSOHO

OpenSOHO is built to manage a small number OpenWRT based network devices. Hence the name SOHO from Small Office Home Office (SOHO) Networks.
Only OpenWRT 24.10.x is tested.

* From 2 to 20 devices
* No multitenancy
* Compatible with openwisp-config and openwisp-monitoring.
* Simple to deploy

It is inspired by OpenWisp, but aims for networks which are too small to be maintained with OpenWISP.
As OpenWisp mentioned:
> However, OpenWISP may not be the best fit for very small networks (fewer than 20 devices), organizations lacking IT expertise, or enterprises seeking open-source alternatives solely for cost-saving purposes.

<p align="center">
  <a href="https://raw.githubusercontent.com/opensoho/assets/1adec68a03db1de7d373707600c3a5807b764f90/devices.png" target="_blank">
    <img src="https://raw.githubusercontent.com/opensoho/assets/1adec68a03db1de7d373707600c3a5807b764f90/devices.png" width="80%" />
  </a>
</p>

## Start OpenSOHO

* Install the dependencies

```
go mod tidy
```

* Start OpenSOHO

The shared secret will be used by openwisp-config to register. Choose a long random string for optimal security.

```
OPENSOHO_SHARED_SECRET=randompassphrase go run . serve --http ipaddress:8090
```
OpenSoho can now be accessed via http://ipaddress:8090/_/

## Configure the OpenWRT devices

Install the OpenWisp packages

```
openwisp-config
openwisp-monitoring
luci-app-openwisp
```

Configure openwisp in Luci:

* Set the `Server URL` and the `Shared secret` only.
* Optionally lower the `Update Interval` to 30 seconds for faster updates.
* OpenWISP monitoring cannot be configure through Luci. Shorten its update interval to make the monitoring behave correctly.
```
uci set openwisp-monitoring.monitoring.interval='15'
uci commit
/etc/init.d/openwisp-monitoring restart
```

It is highly recommended to enable monitoring, since OpenSOHO will deduce a lot of the current OpenWRT settings and fill them in for easy configuration.

## Configure OpenSOHO

* Wait for the devices to self register using the shared secret.
* Set the `numradios` to the correct value. For example for a 2.4 + 5GHz device, this value would be 2.
* Setup a Wifi access point (SSID+KEY)
* Attach the configured Wifi access point to a device to have it configured.
* Currently each network will be automatically configured on all radios (configured using `numradios`).
* Optionally leds can also be turned on or off (only static config for now).
* Configuring radio frequencies is supported now. OpenSOHO will read the current radio config once and make it available under the radios config. It can take a minute or two before radio config appears (the configuration and the monitoring steps need some time to complete)

## Configure
OpenSoho can now be accessed via http://ipaddress:8090/_/

There are several configuration collections:
### Clients
These are the clients connected to wifi. This table should be considered read-only, except for the alias.
It can be used to give devices a human-readable name. This only works properly when the client does not randomize its mac-address.
### Devices
These are the connected devices.
* Use enable/disable to temporarily disable configuration updates. This is useful to avoid updating all devices at once.
* Health status is read-only field. Healty means the device has sent/requested data during the last minute. If it didn't, the health status becomes critical and there might be something wrong with the device or its connection.
* Leds allows to choose led configs
* Numradios allows to set the number of radios on this device. This is not initially sent by OpenWisp, so this needs to be set by the user.
* Wifis allows to select a SSIDs to apply on this device.
### Leds
* Basic LED configuration (more of a POC at this moment)
### Radios
* Allows to set the frequency each radio.
* The band should not be modified, as this allows OpenSOHO to verify the frequency config. This is needed due to limitations within PocketBase.
### Wifi
* Allows to set up the different SSIDs. The selection of encryption modes is currently limited. WEP or Open is not supported by design.

## Monitoring
### Device monitoring
Verify whether the device health is "healthy"
### Connected clients
This view shows all wifi clients that were connected in the last 30 seconds.
### Home Assistant integration
OpenSOHO monitoring can be integrated with Home Assistant. [(More info)](doc/hass.md)
This REST API is designed for use with Home Assistant, but can also be integrated with other tools.

## Troubleshooting
### Reregister a device
When changing the OpenWISP `Server URL` in Luci doesn't seem to properly register with the new controller.
To fix this:
```
uci delete openwisp.http.uuid
uci delete openwisp.http.key
/etc/init.d/openwisp-config restart
```

### First SSID does not get enabled
There seems to be an issue in OpenWRT where the first configured SSID on the a Wifi device is not auto-enabled.
This can be worked around with a click on the enable button in Luci.

# Developers

Running the unit tests:
```
go test -cover -v
```

## Scope
### Features
Some OpenWISP features are (currently) consided out of scope for OpenSOHO
* Firmware updates: OpenWRT doesn't need frequent updating, it should be possible to use the Attended sysupgrade feature on a number of devices manually.
* Multi-tenancy
* Templates: OpenWISP has a powerful and versatile template mechanism, this is replaced by a more opinionated subset of features to provide an easier configuration experience.

### Development
The goal is to leverage the pocketbase to its fullest. Small modifications can be made since the pocketbase backend currently does not contain a plugin system. But it is not the goal to create a fork of pocketbase, so merging upstream changed should be remain straightforward.
