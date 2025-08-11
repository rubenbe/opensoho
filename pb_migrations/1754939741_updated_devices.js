/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update field
  collection.fields.addAt(15, new Field({
    "hidden": true,
    "id": "file3565825916",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [],
    "name": "config",
    "presentable": false,
    "protected": true,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update field
  collection.fields.addAt(15, new Field({
    "hidden": false,
    "id": "file3565825916",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [],
    "name": "config",
    "presentable": false,
    "protected": true,
    "required": false,
    "system": false,
    "thumbs": [],
    "type": "file"
  }))

  return app.save(collection)
})
