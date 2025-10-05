/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // update field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text2445427222",
    "max": 0,
    "min": 0,
    "name": "cidr",
    "pattern": "^([0-9]{1,3}.){3}[0-9]{1,3}/[0-9]{1,2}$",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3745276689")

  // update field
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text2445427222",
    "max": 0,
    "min": 0,
    "name": "cidr",
    "pattern": "^([0-9]{1,3}.){3}[0-9]{1,3}($|/([0-9]{1,2}))$",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
})
