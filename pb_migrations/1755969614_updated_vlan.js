/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // remove field
  collection.fields.removeById("number2336831393")

  // remove field
  collection.fields.removeById("text2445427222")

  // remove field
  collection.fields.removeById("text1285893132")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // add field
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "number2336831393",
    "max": 4096,
    "min": 0,
    "name": "vlan_id",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text2445427222",
    "max": 0,
    "min": 0,
    "name": "subnet",
    "pattern": "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text1285893132",
    "max": 0,
    "min": 0,
    "name": "netmask",
    "pattern": "^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
})
