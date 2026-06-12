/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select4237742649",
    "maxSelect": 1,
    "name": "tx_power_mode",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "auto",
      "dBm",
      "mW"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select4237742649",
    "maxSelect": 1,
    "name": "tx_power_mode",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "auto",
      "dB",
      "mW"
    ]
  }))

  return app.save(collection)
})
