/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_P2CNTUhxfP` ON `wifi_ssids` (`ssid`)"
    ],
    "name": "wifi_ssids"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_P2CNTUhxfP` ON `wifi` (`ssid`)"
    ],
    "name": "wifi"
  }, collection)

  return app.save(collection)
})
