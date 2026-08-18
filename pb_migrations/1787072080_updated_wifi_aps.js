/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1850611088")

  // update field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "select1222615787",
    "maxSelect": 3,
    "name": "band",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "2.4",
      "5",
      "6"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1850611088")

  // update field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "select1222615787",
    "maxSelect": 2,
    "name": "band",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "2.4",
      "5",
      "6"
    ]
  }))

  return app.save(collection)
})
