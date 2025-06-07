/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // update field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "number3762690831",
    "max": null,
    "min": 0,
    "name": "radio",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // update field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "number3762690831",
    "max": null,
    "min": null,
    "name": "radio",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
})
