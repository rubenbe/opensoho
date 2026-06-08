/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // remove field
  collection.fields.removeById("select645904403")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // add field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "select645904403",
    "maxSelect": 1,
    "name": "band",
    "presentable": false,
    "required": true,
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
})
