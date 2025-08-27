/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1876670203")

  // update collection data
  unmarshal({
    "listRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\""
  }, collection)

  // remove field
  collection.fields.removeById("_clone_GmHt")

  // remove field
  collection.fields.removeById("_clone_lSV9")

  // remove field
  collection.fields.removeById("_clone_mKdh")

  // remove field
  collection.fields.removeById("_clone_hQWW")

  // remove field
  collection.fields.removeById("_clone_EoSY")

  // remove field
  collection.fields.removeById("_clone_x0fX")

  // remove field
  collection.fields.removeById("_clone_rlhn")

  // remove field
  collection.fields.removeById("_clone_O5rL")

  // remove field
  collection.fields.removeById("_clone_jFDE")

  // remove field
  collection.fields.removeById("_clone_yaoP")

  // remove field
  collection.fields.removeById("_clone_SPi5")

  // add field
  collection.fields.addAt(0, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_H3M4",
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
    "id": "_clone_yDcX",
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
    "id": "_clone_mJRX",
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
    "id": "_clone_g48f",
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
    "id": "_clone_vOvI",
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
    "id": "_clone_pYmi",
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
    "id": "_clone_FiAu",
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
    "id": "_clone_ToSX",
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
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "_clone_nMty",
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
  collection.fields.addAt(13, new Field({
    "hidden": false,
    "id": "_clone_goYL",
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
  collection.fields.addAt(14, new Field({
    "hidden": false,
    "id": "_clone_OGoH",
    "name": "last_seen",
    "onCreate": true,
    "onUpdate": true,
    "presentable": false,
    "system": false,
    "type": "autodate"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1876670203")

  // update collection data
  unmarshal({
    "listRule": "@request.auth.collectionName = \"_superusers\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\""
  }, collection)

  // add field
  collection.fields.addAt(0, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_GmHt",
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
    "id": "_clone_lSV9",
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
    "id": "_clone_mKdh",
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
    "id": "_clone_hQWW",
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
    "id": "_clone_EoSY",
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
    "id": "_clone_x0fX",
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
    "id": "_clone_rlhn",
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
    "id": "_clone_O5rL",
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
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "_clone_jFDE",
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
  collection.fields.addAt(13, new Field({
    "hidden": false,
    "id": "_clone_yaoP",
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
  collection.fields.addAt(14, new Field({
    "hidden": false,
    "id": "_clone_SPi5",
    "name": "last_seen",
    "onCreate": true,
    "onUpdate": true,
    "presentable": false,
    "system": false,
    "type": "autodate"
  }))

  // remove field
  collection.fields.removeById("_clone_H3M4")

  // remove field
  collection.fields.removeById("_clone_yDcX")

  // remove field
  collection.fields.removeById("_clone_mJRX")

  // remove field
  collection.fields.removeById("_clone_g48f")

  // remove field
  collection.fields.removeById("_clone_vOvI")

  // remove field
  collection.fields.removeById("_clone_pYmi")

  // remove field
  collection.fields.removeById("_clone_FiAu")

  // remove field
  collection.fields.removeById("_clone_ToSX")

  // remove field
  collection.fields.removeById("_clone_nMty")

  // remove field
  collection.fields.removeById("_clone_goYL")

  // remove field
  collection.fields.removeById("_clone_OGoH")

  return app.save(collection)
})
