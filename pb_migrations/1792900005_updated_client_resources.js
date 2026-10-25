/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_71711649")

  // add field: the band (GHz) the client was connected on, like clients.band
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "select1877749504",
    "maxSelect": 1,
    "name": "band",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "2.4",
      "5",
      "6",
      "60"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_71711649")

  // remove field
  collection.fields.removeById("select1877749504")

  return app.save(collection)
})
