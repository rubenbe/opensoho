/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // add field
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3744042332",
    "max": 0,
    "min": 0,
    "name": "openwrt_version",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "bool1358543748",
    "name": "enabled",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select224463326",
    "maxSelect": 1,
    "name": "config_status",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "applied",
      "modified"
    ]
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "select15371673",
    "maxSelect": 1,
    "name": "health_status",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "critical",
      "unknown",
      "healthy"
    ]
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "autodate1542800728",
    "name": "field",
    "onCreate": true,
    "onUpdate": false,
    "presentable": false,
    "system": false,
    "type": "autodate"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // remove field
  collection.fields.removeById("text3744042332")

  // remove field
  collection.fields.removeById("bool1358543748")

  // remove field
  collection.fields.removeById("select224463326")

  // remove field
  collection.fields.removeById("select15371673")

  // remove field
  collection.fields.removeById("autodate1542800728")

  return app.save(collection)
})
