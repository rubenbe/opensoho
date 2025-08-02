/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_501785886")

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "select1222615787",
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
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_501785886")

  // remove field
  collection.fields.removeById("select1222615787")

  return app.save(collection)
})
