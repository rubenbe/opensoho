/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = new Collection({
    "createRule": null,
    "deleteRule": null,
    "fields": [
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text1579384326",
        "max": 63,
        "min": 1,
        "name": "name",
        "pattern": "^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$",
        "presentable": true,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text2377882623",
        "max": 17,
        "min": 17,
        "name": "mac_address",
        "pattern": "^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "autogeneratePattern": "",
        "hidden": false,
        "id": "text1724086763",
        "max": 15,
        "min": 7,
        "name": "ip_address",
        "pattern": "^([0-9]{1,3}\\.){3}[0-9]{1,3}$",
        "presentable": false,
        "primaryKey": false,
        "required": true,
        "system": false,
        "type": "text"
      },
      {
        "cascadeDelete": false,
        "collectionId": "pbc_2153001328",
        "hidden": false,
        "id": "relation154121870",
        "maxSelect": 999,
        "minSelect": 0,
        "name": "dhcp_server",
        "presentable": false,
        "required": false,
        "system": false,
        "type": "relation"
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
    "id": "pbc_3986314751",
    "indexes": [
      "CREATE UNIQUE INDEX `idx_dhcp_reservation_name` ON `dhcp_reservations` (`name`)",
      "CREATE UNIQUE INDEX `idx_dhcp_reservation_mac` ON `dhcp_reservations` (`mac_address`)",
      "CREATE UNIQUE INDEX `idx_dhcp_reservation_ip` ON `dhcp_reservations` (`ip_address`)"
    ],
    "listRule": null,
    "name": "dhcp_reservations",
    "system": false,
    "type": "base",
    "updateRule": null,
    "viewRule": null
  });

  return app.save(collection);
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3986314751");

  return app.delete(collection);
})
