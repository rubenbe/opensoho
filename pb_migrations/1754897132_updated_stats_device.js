/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3987150219")

  // update collection data
  unmarshal({
    "name": "device_stats"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3987150219")

  // update collection data
  unmarshal({
    "name": "stats_device"
  }, collection)

  return app.save(collection)
})
