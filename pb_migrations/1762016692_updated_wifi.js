/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_P2CNTUhxfP` ON `wifi` (`ssid`)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update collection data
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
})
