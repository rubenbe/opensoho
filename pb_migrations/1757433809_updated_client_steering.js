/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select1582905952",
    "maxSelect": 1,
    "name": "method",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "mac blacklist",
      "ssid"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select1582905952",
    "maxSelect": 1,
    "name": "method",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "mac blacklist"
    ]
  }))

  return app.save(collection)
})
