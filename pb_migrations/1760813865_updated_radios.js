/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // add field
  collection.fields.addAt(5, new Field({
    "hidden": false,
    "id": "select3108721286",
    "maxSelect": 1,
    "name": "htmode",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "HT20",
      "HT40",
      "VHT20",
      "VHT40",
      "VHT80",
      "VHT160",
      "HE20",
      "HE40",
      "HE80",
      "HE160"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // remove field
  collection.fields.removeById("select3108721286")

  return app.save(collection)
})
