/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // add field
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "number2526027604",
    "max": 4094,
    "min": null,
    "name": "number",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // remove field
  collection.fields.removeById("number2526027604")

  return app.save(collection)
})
