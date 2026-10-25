/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
    "deleteRule": null,
    "fields": [
      {
        "cascadeDelete": true,
        "collectionId": "pbc_2442875294",
        "hidden": false,
        "id": "relation3922196188",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "client",
        "presentable": true,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "cascadeDelete": true,
        "collectionId": "pbc_2153001328",
        "hidden": false,
        "id": "relation665601031",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "device",
        "presentable": true,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
        "hidden": false,
        "id": "date1218909614",
        "max": "",
        "min": "",
        "name": "time",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "date"
      },
      {
        "hidden": false,
        "id": "number1732331400",
        "max": null,
        "min": 0,
        "name": "rx_bytes",
        "onlyInt": true,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "number2704166415",
        "max": null,
        "min": 0,
        "name": "tx_bytes",
        "onlyInt": true,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "autodate2990389176",
        "name": "created",
        "onCreate": true,
        "onUpdate": false,
        "presentable": false,
        "system": false,
        "type": "autodate"
      },
      {
        "hidden": false,
        "id": "autodate3332085495",
        "name": "updated",
        "onCreate": true,
        "onUpdate": true,
        "presentable": false,
        "system": false,
        "type": "autodate"
      },
      {
        "autogeneratePattern": "[a-z0-9]{15}",
        "hidden": false,
        "id": "text3208210256",
        "max": 15,
        "min": 15,
        "name": "id",
        "pattern": "^[a-z0-9]+$",
        "presentable": false,
        "primaryKey": true,
        "required": true,
        "system": true,
        "type": "text"
      }
    ],
    "id": "pbc_71711649",
    "indexes": [
      "CREATE INDEX `idx_client_resources_client_time` ON `client_resources` (`client`, `time`)",
      "CREATE INDEX `idx_client_resources_device_time` ON `client_resources` (`device`, `time`)",
      "CREATE INDEX `idx_client_resources_time` ON `client_resources` (`time`)"
    ],
    "listRule": null,
    "name": "client_resources",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_71711649");

  return app.delete(collection);
})
