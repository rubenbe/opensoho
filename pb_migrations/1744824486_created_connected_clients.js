/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
    "deleteRule": null,
    "fields": [
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text3208210256",
        "max": 0,
        "min": 0,
        "name": "id",
        "pattern": "^[a-z0-9]+$",
        "presentable": false,
        "primaryKey": true,
        "required": true,
        "system": true,
        "type": "text"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "_clone_5fac",
        "max": 0,
        "min": 0,
        "name": "alias",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": false,
        "system": false,
        "type": "text"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_2153001328",
        "hidden": false,
        "id": "_clone_BEts",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "device",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      }
    ],
    "id": "pbc_1876670203",
    "indexes": [],
    "listRule": null,
    "name": "connected_clients",
    "system": false,
    "type": "view",
    "updateRule": null,
    "viewQuery": "  SELECT id, alias, device FROM clients WHERE updated >= datetime('now', '-30 seconds')",
    "viewRule": null
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1876670203");

  return app.delete(collection);
})
