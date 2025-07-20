/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // update field
  collection.fields.addAt(1, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2442875294",
    "hidden": false,
    "id": "relation154121870",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "client",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "relation"
  }))

  // update field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_502121861",
    "hidden": false,
    "id": "relation2539889227",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "wifi",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "relation"
  }))

  // update field
  collection.fields.addAt(3, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "relation991384453",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "whitelist",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "relation"
  }))

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select1582905952",
    "maxSelect": 1,
    "name": "method",
    "presentable": false,
    "required": true,
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
  collection.fields.addAt(1, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2442875294",
    "hidden": false,
    "id": "relation154121870",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "client",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // update field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_502121861",
    "hidden": false,
    "id": "relation2539889227",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "wifi",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // update field
  collection.fields.addAt(3, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "relation991384453",
    "maxSelect": 999,
    "minSelect": 0,
    "name": "whitelist",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

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
})
