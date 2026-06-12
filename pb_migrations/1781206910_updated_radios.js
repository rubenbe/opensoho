/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "bool3303607855",
    "name": "auto_tx_power",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  // update field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "number47157968",
    "max": null,
    "min": null,
    "name": "tx_power_db",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // remove field
  collection.fields.removeById("bool3303607855")

  // update field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "number47157968",
    "max": null,
    "min": null,
    "name": "tx_power",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
})
