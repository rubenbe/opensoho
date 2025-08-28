/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "number2255476553",
    "max": 65535,
    "min": 1000,
    "name": "ieee80211r_reassoc_deadline",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // remove field
  collection.fields.removeById("number2255476553")

  return app.save(collection)
})
