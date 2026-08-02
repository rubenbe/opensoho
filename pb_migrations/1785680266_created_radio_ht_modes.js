/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
    "deleteRule": null,
    "fields": [
      {
        "cascadeDelete": false,
        "collectionId": "pbc_2153001328",
        "hidden": false,
        "id": "relation154121870",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "device",
        "presentable": false,
        "required": true,
        "system": false,
        "type": "relation"
      },
      {
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
      },
      {
        "hidden": false,
        "id": "select527213132",
        "maxSelect": 15,
        "name": "ht_modes",
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
          "HE160",
          "EHT20",
          "EHT40",
          "EHT80",
          "EHT160",
          "EHT320"
        ]
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
    "id": "pbc_3652201430",
    "indexes": [
      "CREATE UNIQUE INDEX `idx_uDVF4PFJIc` ON `radio_ht_modes` (\n  `radio`,\n  `device`\n)"
    ],
    "listRule": null,
    "name": "radio_ht_modes",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3652201430");

  return app.delete(collection);
})
