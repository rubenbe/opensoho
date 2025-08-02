/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_501785886")

  // update field
  collection.fields.addAt(4, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3072911721",
    "max": 17,
    "min": 17,
    "name": "bssid",
    "pattern": "^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_501785886")

  // update field
  collection.fields.addAt(4, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3072911721",
    "max": 17,
    "min": 17,
    "name": "mac_address",
    "pattern": "^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
})
