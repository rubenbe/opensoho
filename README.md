*Warning this is a very early release*

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
* OpenWISP monitoring cannot be configure through Luci. Shorten its update interval to make the monitoring behave snappier.
```
uci set openwisp-monitoring.monitoring.interval='15'
```

## Configure OpenSOHO

* Wait for the devices to self register using the shared secret.
* Setup a Wifi access point (SSID+KEY)
* Attach it to a device to have it configured.
* Currently each network will be automatically configured on all radios (configured using `numradios`)
* Optionally leds can also be turned on or off (only static config for now).
* Configuring radios is not yet supported.

## Extras
### Reregister a device
When changing the OpenWISP `Server URL` in Luci doesn't seem to properly register with the new controller.
To fix this:
```
uci delete openwisp.http.uuid
uci delete openwisp.http.key
/etc/init.d/openwisp-config restart
```
