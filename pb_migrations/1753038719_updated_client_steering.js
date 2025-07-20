/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select1582905952",
    "maxSelect": 1,
    "name": "method",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "mac blacklist",
      "bss request (ieee80211v)"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select1582905952",
    "maxSelect": 1,
    "name": "method",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "mac_blacklist",
      "bss_request (ieee80211v)"
    ]
  }))

  return app.save(collection)
})
