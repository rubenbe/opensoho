/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_71711649")

  // add field: the SSID the client was connected to, as named at the time
  collection.fields.addAt(2, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3618307619",
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

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_71711649")

  // remove field
  collection.fields.removeById("text3618307619")

  return app.save(collection)
})
