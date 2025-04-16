/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1876670203")

  // update collection data
  unmarshal({
    "viewQuery": "  SELECT id, mac_address, alias, device FROM clients WHERE updated >= datetime('now', '-30 seconds')"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_5fac")

  // remove field
  collection.fields.removeById("_clone_BEts")

  // add field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_LaTg",
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

  // add field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_n6rf",
    "max": 0,
    "min": 0,
    "name": "alias",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "_clone_22w8",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "device",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1876670203")

  // update collection data
  unmarshal({
    "viewQuery": "  SELECT id, alias, device FROM clients WHERE updated >= datetime('now', '-30 seconds')"
  }, collection)

  // add field
  collection.fields.addAt(1, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_5fac",
    "max": 0,
    "min": 0,
    "name": "alias",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(2, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "_clone_BEts",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "device",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // remove field
  collection.fields.removeById("_clone_LaTg")

  // remove field
  collection.fields.removeById("_clone_n6rf")

  // remove field
  collection.fields.removeById("_clone_22w8")

  return app.save(collection)
})
