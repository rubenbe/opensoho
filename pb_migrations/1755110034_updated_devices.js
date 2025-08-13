/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3414933350",
    "max": 45,
    "min": 7,
    "name": "ip_address",
    "pattern": "^((\\d{1,3}\\.){3}\\d{1,3}|([a-fA-F0-9:]+:+)+[a-fA-F0-9]+)$",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3414933350",
    "max": 45,
    "min": 7,
    "name": "last_ip_address",
    "pattern": "^((\\d{1,3}\\.){3}\\d{1,3}|([a-fA-F0-9:]+:+)+[a-fA-F0-9]+)$",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
})
