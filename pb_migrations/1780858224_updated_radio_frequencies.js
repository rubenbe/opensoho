/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3180122789")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select184893882",
    "maxSelect": 10,
    "name": "flags",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "indoor_only",
      "no_ht40-",
      "no_ht40+",
      "no_80mhz",
      "no_160mhz",
      "no_320mhz",
      "no_10mhz",
      "no_20mhz",
      "no_he",
      "no_ir"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3180122789")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select184893882",
    "maxSelect": 6,
    "name": "flags",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "indoor_only",
      "no_ht40-",
      "no_ht40+",
      "no_80mhz",
      "no_160mhz",
      "no_320mhz"
    ]
  }))

  return app.save(collection)
})
