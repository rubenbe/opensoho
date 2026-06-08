/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3180122789")

  // add field
  collection.fields.addAt(0, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "relation154121870",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "device",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "number3762690831",
    "max": null,
    "min": 0,
    "name": "radio",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3180122789")

  // remove field
  collection.fields.removeById("relation154121870")

  // remove field
  collection.fields.removeById("number3762690831")

  return app.save(collection)
})
