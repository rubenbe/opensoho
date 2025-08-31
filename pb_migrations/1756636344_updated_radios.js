/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "bool196833538",
    "name": "auto_frequency",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // remove field
  collection.fields.removeById("bool196833538")

  return app.save(collection)
})
