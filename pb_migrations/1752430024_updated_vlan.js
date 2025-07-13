/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // remove field
  collection.fields.removeById("bool2355255437")

  // remove field
  collection.fields.removeById("number4163962017")

  // remove field
  collection.fields.removeById("bool2618364956")

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // add field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "bool2355255437",
    "name": "no_wan",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "number4163962017",
    "max": 4094,
    "min": 0,
    "name": "vlan",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(4, new Field({
    "hidden": false,
    "id": "bool2618364956",
    "name": "no_lan",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
})
