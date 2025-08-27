/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3440715838")

  // update collection data
  unmarshal({
    "listRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\""
  }, collection)

  // remove field
  collection.fields.removeById("_clone_ugRn")

  // remove field
  collection.fields.removeById("_clone_kjHG")

  // remove field
  collection.fields.removeById("_clone_PApG")

  // remove field
  collection.fields.removeById("_clone_y7hV")

  // remove field
  collection.fields.removeById("_clone_Vvyq")

  // remove field
  collection.fields.removeById("_clone_1XrX")

  // remove field
  collection.fields.removeById("_clone_8u1R")

  // remove field
  collection.fields.removeById("_clone_yR2b")

  // remove field
  collection.fields.removeById("_clone_stru")

  // remove field
  collection.fields.removeById("_clone_OEb5")

  // remove field
  collection.fields.removeById("_clone_IjFa")

  // add field
  collection.fields.addAt(0, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_ONv8",
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
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_a4DX",
    "max": 0,
    "min": 0,
    "name": "alias",
    "pattern": "",
    "presentable": true,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(4, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "_clone_3RHh",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "device",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_YnJG",
    "max": 0,
    "min": 0,
    "name": "ssid",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "_clone_nUqJ",
    "max": null,
    "min": null,
    "name": "frequency",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "_clone_cSst",
    "max": null,
    "min": null,
    "name": "channel",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "_clone_DRPl",
    "maxSelect": 1,
    "name": "band",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "2.4",
      "5",
      "6",
      "60"
    ]
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "_clone_JXbI",
    "max": null,
    "min": null,
    "name": "signal",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "_clone_Dv9S",
    "max": null,
    "min": null,
    "name": "tx_bytes",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "_clone_PbRW",
    "max": null,
    "min": null,
    "name": "rx_bytes",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "_clone_ffvM",
    "name": "last_seen",
    "onCreate": true,
    "onUpdate": true,
    "presentable": false,
    "system": false,
    "type": "autodate"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3440715838")

  // update collection data
  unmarshal({
    "listRule": "@request.auth.collectionName = \"_superusers\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\""
  }, collection)

  // add field
  collection.fields.addAt(0, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_ugRn",
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
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_kjHG",
    "max": 0,
    "min": 0,
    "name": "alias",
    "pattern": "",
    "presentable": true,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(4, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "_clone_PApG",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "device",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_y7hV",
    "max": 0,
    "min": 0,
    "name": "ssid",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "_clone_Vvyq",
    "max": null,
    "min": null,
    "name": "frequency",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "_clone_1XrX",
    "max": null,
    "min": null,
    "name": "channel",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "_clone_8u1R",
    "maxSelect": 1,
    "name": "band",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "2.4",
      "5",
      "6",
      "60"
    ]
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "_clone_yR2b",
    "max": null,
    "min": null,
    "name": "signal",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "_clone_stru",
    "max": null,
    "min": null,
    "name": "tx_bytes",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "_clone_OEb5",
    "max": null,
    "min": null,
    "name": "rx_bytes",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "_clone_IjFa",
    "name": "last_seen",
    "onCreate": true,
    "onUpdate": true,
    "presentable": false,
    "system": false,
    "type": "autodate"
  }))

  // remove field
  collection.fields.removeById("_clone_ONv8")

  // remove field
  collection.fields.removeById("_clone_a4DX")

  // remove field
  collection.fields.removeById("_clone_3RHh")

  // remove field
  collection.fields.removeById("_clone_YnJG")

  // remove field
  collection.fields.removeById("_clone_nUqJ")

  // remove field
  collection.fields.removeById("_clone_cSst")

  // remove field
  collection.fields.removeById("_clone_DRPl")

  // remove field
  collection.fields.removeById("_clone_JXbI")

  // remove field
  collection.fields.removeById("_clone_Dv9S")

  // remove field
  collection.fields.removeById("_clone_PbRW")

  // remove field
  collection.fields.removeById("_clone_ffvM")

  return app.save(collection)
})
