/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // add field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_502121861",
    "hidden": false,
    "id": "relation2539889227",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "wifi",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // remove field
  collection.fields.removeById("relation2539889227")

  return app.save(collection)
})
