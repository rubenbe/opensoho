/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "select1539169211",
    "maxSelect": 1,
    "name": "encryption",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "sae",
      "sae-mixed",
      "psk2+tkip+ccmp",
      "psk2+tkip+aes",
      "psk2+tkip",
      "psk2+ccmp",
      "psk2+aes",
      "psk2",
      "psk-mixed+tkip+ccmp",
      "psk-mixed+tkip+aes",
      "psk-mixed+tkip",
      "psk-mixed+ccmp",
      "psk-mixed+aes",
      "psk-mixed",
      "psk+tkip+ccmp",
      "psk+tkip+aes",
      "psk+tkip",
      "psk+ccmp",
      "psk+aes",
      "psk",
      "owe",
      "none"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // update field
  collection.fields.addAt(3, new Field({
    "hidden": false,
    "id": "select1539169211",
    "maxSelect": 1,
    "name": "encryption",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "psk2+ccmp",
      "psk-mixed+ccmp"
    ]
  }))

  return app.save(collection)
})
