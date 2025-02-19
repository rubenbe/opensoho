/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_965791478")

  // update collection data
  unmarshal({
    "name": "devices"
  }, collection)

  // add field
  collection.fields.addAt(4, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "text3514781862",
    "max": 0,
    "min": 0,
    "name": "uuid",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_965791478")

  // update collection data
  unmarshal({
    "name": "Device"
  }, collection)

  // remove field
  collection.fields.removeById("text3514781862")

  return app.save(collection)
})
