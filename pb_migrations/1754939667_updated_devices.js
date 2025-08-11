/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select15371673",
    "maxSelect": 1,
    "name": "health_status",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "unknown",
      "healthy",
      "critical",
      "unhealthy"
    ]
  }))

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
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select15371673",
    "maxSelect": 1,
    "name": "health_status",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "unknown",
      "healthy",
      "critical"
    ]
  }))

  // update field
  collection.fields.addAt(15, new Field({
    "hidden": false,
    "id": "file3565825916",
    "maxSelect": 1,
    "maxSize": 0,
    "mimeTypes": [
      "application/x-tar"
    ],
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
