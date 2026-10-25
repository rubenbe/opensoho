/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_71711649")

  // add field: the signal (RSSI, dBm) of the client, like clients.signal
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "number1519347580",
    "max": null,
    "min": null,
    "name": "signal",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_71711649")

  // remove field
  collection.fields.removeById("number1519347580")

  return app.save(collection)
})
