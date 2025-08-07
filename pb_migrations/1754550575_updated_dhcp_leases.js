/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2401716254")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "date1058804653",
    "max": "",
    "min": "",
    "name": "expiry",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "date"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2401716254")

  // update field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "date1058804653",
    "max": "",
    "min": "",
    "name": "Expiry",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "date"
  }))

  return app.save(collection)
})
