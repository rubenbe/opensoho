/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // remove field
  collection.fields.removeById("select3174009887")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "select3174009887",
    "maxSelect": 1,
    "name": "apply",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "vlan"
    ]
  }))

  return app.save(collection)
})
