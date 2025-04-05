*Warning this is a very early release*

# OpenSOHO

OpenSOHO is built to manage a small number OpenWRT based network devices.

* From 2 to 20 devices
* No multitenancy
* Compatible with openwisp-config and openwisp-monitoring.

It is inspired by OpenWisp, but aims for networks which are too small to be maintained with OpenWISP.
As OpenWisp mentioned:
> However, OpenWISP may not be the best fit for very small networks (fewer than 20 devices), organizations lacking IT expertise, or enterprises seeking open-source alternatives solely for cost-saving purposes.

## Start OpenSOHO

* Install the dependencies

```
go mod tidy
```

* Start OpenSOHO

```
go run . serve --http ipaddress:8090
```

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
* OpenWISP monitoring cannot be configure through Luci.

## Configure OpenSOHO

* Wait for the devices to self register using the shared secret.
* Setup a Wifi access point (SSID+KEY)
* Attach it to a device to have it configured.
* Currently only one network per device is supported, but it will be automatically configured on all radio devices (configured using `numradios`)
* Optionally leds can also be turned on or off (only static config for now).
