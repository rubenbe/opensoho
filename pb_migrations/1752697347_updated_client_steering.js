/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // add field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "select4087400498",
    "maxSelect": 1,
    "name": "enable",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "Always",
      "If all healthy",
      "If any healthy"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // remove field
  collection.fields.removeById("select4087400498")

  return app.save(collection)
})
