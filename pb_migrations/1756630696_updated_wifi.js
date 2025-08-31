/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // add field
  collection.fields.addAt(13, new Field({
    "hidden": false,
    "id": "bool3775054730",
    "name": "ieee80211v_proxy_arp",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // remove field
  collection.fields.removeById("bool3775054730")

  return app.save(collection)
})
