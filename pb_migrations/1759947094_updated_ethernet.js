/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3702000743")

  // add field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_555501864",
    "hidden": false,
    "id": "relation3565825916",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "config",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3702000743")

  // remove field
  collection.fields.removeById("relation3565825916")

  return app.save(collection)
})
