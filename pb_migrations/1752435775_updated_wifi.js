/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // add field
  collection.fields.addAt(8, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_3745276689",
    "hidden": false,
    "id": "relation1619298236",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "network",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // remove field
  collection.fields.removeById("relation1619298236")

  return app.save(collection)
})
