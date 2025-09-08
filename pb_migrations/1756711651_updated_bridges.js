/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_624525223")

  // add field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "number417124542",
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
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "number3736328505",
    "max": null,
    "min": null,
    "name": "rx_bytes",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_624525223")

  // remove field
  collection.fields.removeById("number417124542")

  // remove field
  collection.fields.removeById("number3736328505")

  return app.save(collection)
})
