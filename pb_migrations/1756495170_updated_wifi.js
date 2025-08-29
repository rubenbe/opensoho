/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "bool3015144211",
    "name": "ieee80211k",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // remove field
  collection.fields.removeById("bool3015144211")

  return app.save(collection)
})
