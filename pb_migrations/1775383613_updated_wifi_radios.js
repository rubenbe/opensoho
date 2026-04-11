/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1850611088")

  // update collection data
  unmarshal({
    "name": "wifi_aps"
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1850611088")

  // update collection data
  unmarshal({
    "name": "wifi_radios"
  }, collection)

  return app.save(collection)
})
