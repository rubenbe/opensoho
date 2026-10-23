/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2769025244")

  // extend the "name" select with the resource history retention setting
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "select1579384326",
    "maxSelect": 1,
    "name": "name",
    "presentable": true,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "country",
      "mqtt_enabled",
      "mqtt_broker",
      "mqtt_username",
      "mqtt_password",
      "history_retention_days"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2769025244")

  // revert the "name" select to the MQTT/Home Assistant settings keys
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "select1579384326",
    "maxSelect": 1,
    "name": "name",
    "presentable": true,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "country",
      "mqtt_enabled",
      "mqtt_broker",
      "mqtt_username",
      "mqtt_password"
    ]
  }))

  return app.save(collection)
})
