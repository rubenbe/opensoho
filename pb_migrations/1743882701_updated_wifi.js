/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select1539169211",
    "maxSelect": 1,
    "name": "encryption",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "psk2+ccmp",
      "psk-mixed+ccmp"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select1539169211",
    "maxSelect": 1,
    "name": "encryption",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "psk-mixed+ccmp"
    ]
  }))

  return app.save(collection)
})
