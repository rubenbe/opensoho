/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
    "deleteRule": null,
    "fields": [
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "_clone_NAJU",
        "max": 17,
        "min": 17,
        "name": "mac_address",
        "pattern": "^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "hidden": false,
        "id": "json587191692",
        "maxSize": 1,
        "name": "ip_address",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "json"
      },
      {
        "hidden": false,
        "id": "json3847340049",
        "maxSize": 1,
        "name": "hostname",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "json"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "_clone_AZpA",
        "max": 0,
        "min": 0,
        "name": "alias",
        "pattern": "",
        "presentable": true,
        "primaryKey": false,
        "required": false,
        "system": false,
        "type": "text"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_2153001328",
        "hidden": false,
        "id": "_clone_jYBk",
        "maxSelect": 1,
        "minSelect": 0,
        "name": "device",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "_clone_ldpZ",
        "max": 0,
        "min": 0,
        "name": "ssid",
        "pattern": "",
        "presentable": false,
        "primaryKey": false,
        "required": false,
        "system": false,
        "type": "text"
      },
      {
        "hidden": false,
        "id": "_clone_7zn8",
        "max": null,
        "min": null,
        "name": "frequency",
        "onlyInt": true,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "_clone_hO1Z",
        "maxSelect": 1,
        "name": "band",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "select",
        "values": [
          "2.4",
          "5",
          "6",
          "60"
        ]
      },
      {
        "hidden": false,
        "id": "_clone_7LFW",
        "max": null,
        "min": null,
        "name": "signal",
        "onlyInt": false,
        "presentable": false,
        "required": false,
        "system": false,
        "type": "number"
      },
      {
        "hidden": false,
        "id": "_clone_U7RT",
        "name": "last_seen",
        "onCreate": true,
        "onUpdate": true,
        "presentable": false,
        "system": false,
        "type": "autodate"
      },
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
      }
    ],
    "id": "pbc_3440715838",
    "indexes": [],
    "listRule": null,
    "name": "disconnected_clients",
    "system": false,
    "type": "view",
    "updateRule": null,
    "viewQuery": "  SELECT clients.mac_address, COALESCE(ip_address,\"\") as ip_address, COALESCE(hostname,\"\") as hostname, alias, device, ssid, frequency, band, signal, dhcp_leases.updated AS last_seen, dhcp_leases.id FROM clients LEFT JOIN dhcp_leases ON dhcp_leases.mac_address == clients.mac_address WHERE clients.updated > datetime('now', '-30 seconds')",
    "viewRule": null
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3440715838");

  return app.delete(collection);
})
