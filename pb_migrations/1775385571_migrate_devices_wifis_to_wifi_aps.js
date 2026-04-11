/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const wifiApsCol = app.findCollectionByNameOrId("wifi_aps")
  const devices = app.findAllRecords("devices")

  for (const device of devices) {
    const wifis = device.get("wifis") || []

    for (const wifiId of wifis) {
      const ap = new Record(wifiApsCol)
      ap.set("device", device.id)
      ap.set("wifi", wifiId)
      ap.set("band", ["2.4", "5"])
      app.save(ap)
    }
  }
}, (app) => {
  const devices = app.findAllRecords("devices")

  for (const device of devices) {
    const aps = app.findRecordsByFilter("wifi_aps", "device = {:deviceId}", "", 0, 0, {"deviceId": device.id})
    const wifiIds = [...new Set(aps.map(ap => ap.get("wifi")))]
    device.set("wifis", wifiIds)
    app.save(device)
  }

  const allAps = app.findAllRecords("wifi_aps")
  for (const ap of allAps) {
    app.delete(ap)
  }
})
