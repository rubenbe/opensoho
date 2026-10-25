/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1270418693")

  // add field: seconds since the device booted
  collection.fields.addAt(13, new Field({
    "hidden": false,
    "id": "number2304993359",
    "max": null,
    "min": 0,
    "name": "uptime",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1270418693")

  // remove field
  collection.fields.removeById("number2304993359")

  return app.save(collection)
})
