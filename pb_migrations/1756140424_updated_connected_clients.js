/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_1876670203")

  // update collection data
  unmarshal({
    "viewQuery": "  SELECT clients.mac_address, COALESCE(ip_address,\"\") as ip_address, COALESCE(hostname,\"\") as hostname, alias, device, ssid, frequency, channel, band, signal, tx_rate, rx_rate, tx_bytes, rx_bytes, dhcp_leases.updated AS last_seen, dhcp_leases.id FROM clients LEFT JOIN dhcp_leases ON dhcp_leases.mac_address == clients.mac_address WHERE clients.updated >= datetime('now', '-30 seconds')"
  }, collection)

  // remove field
  collection.fields.removeById("_clone_41Af")

  // remove field
  collection.fields.removeById("_clone_3dHj")

  // remove field
  collection.fields.removeById("_clone_THDZ")

  // remove field
  collection.fields.removeById("_clone_HBZy")

  // remove field
  collection.fields.removeById("_clone_OCRP")

  // remove field
  collection.fields.removeById("_clone_WEV2")

  // remove field
  collection.fields.removeById("_clone_0xQJ")

  // remove field
  collection.fields.removeById("_clone_DVyu")

  // remove field
  collection.fields.removeById("_clone_3wkP")

  // add field
  collection.fields.addAt(0, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_tlPv",
    "max": 17,
    "min": 17,
    "name": "mac_address",
    "pattern": "^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_Bcc2",
    "max": 0,
    "min": 0,
    "name": "alias",
    "pattern": "",
    "presentable": true,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(4, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "_clone_z3VX",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "device",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_Sq6u",
    "max": 0,
    "min": 0,
    "name": "ssid",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "_clone_3Yee",
    "max": null,
    "min": null,
    "name": "frequency",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "_clone_lXVP",
    "max": null,
    "min": null,
    "name": "channel",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "_clone_hF2S",
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
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "_clone_NwIo",
    "max": null,
    "min": null,
    "name": "signal",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "_clone_KsTf",
    "max": null,
    "min": null,
    "name": "tx_rate",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(11, new Field({
    "hidden": false,
    "id": "_clone_ovbQ",
    "max": null,
    "min": null,
    "name": "rx_rate",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(12, new Field({
    "hidden": false,
    "id": "_clone_TZqk",
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
  collection.fields.addAt(13, new Field({
    "hidden": false,
    "id": "_clone_vNRY",
    "max": null,
    "min": null,
    "name": "rx_bytes",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(14, new Field({
    "hidden": false,
    "id": "_clone_Ep0N",
    "name": "last_seen",
    "onCreate": true,
    "onUpdate": true,
    "presentable": false,
    "system": false,
    "type": "autodate"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_1876670203")

  // update collection data
  unmarshal({
    "viewQuery": "  SELECT clients.mac_address, COALESCE(ip_address,\"\") as ip_address, COALESCE(hostname,\"\") as hostname, alias, device, ssid, frequency, channel, band, signal, dhcp_leases.updated AS last_seen, dhcp_leases.id FROM clients LEFT JOIN dhcp_leases ON dhcp_leases.mac_address == clients.mac_address WHERE clients.updated >= datetime('now', '-30 seconds')"
  }, collection)

  // add field
  collection.fields.addAt(0, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_41Af",
    "max": 17,
    "min": 17,
    "name": "mac_address",
    "pattern": "^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$",
    "presentable": false,
    "primaryKey": false,
    "required": true,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(3, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_3dHj",
    "max": 0,
    "min": 0,
    "name": "alias",
    "pattern": "",
    "presentable": true,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(4, new Field({
    "cascadeDelete": false,
    "collectionId": "pbc_2153001328",
    "hidden": false,
    "id": "_clone_THDZ",
    "maxSelect": 1,
    "minSelect": 0,
    "name": "device",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "relation"
  }))

  // add field
  collection.fields.addAt(5, new Field({
    "autogeneratePattern": "",
    "hidden": false,
    "id": "_clone_HBZy",
    "max": 0,
    "min": 0,
    "name": "ssid",
    "pattern": "",
    "presentable": false,
    "primaryKey": false,
    "required": false,
    "system": false,
    "type": "text"
  }))

  // add field
  collection.fields.addAt(6, new Field({
    "hidden": false,
    "id": "_clone_OCRP",
    "max": null,
    "min": null,
    "name": "frequency",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "_clone_WEV2",
    "max": null,
    "min": null,
    "name": "channel",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(8, new Field({
    "hidden": false,
    "id": "_clone_0xQJ",
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
  }))

  // add field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "_clone_DVyu",
    "max": null,
    "min": null,
    "name": "signal",
    "onlyInt": false,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  // add field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "_clone_3wkP",
    "name": "last_seen",
    "onCreate": true,
    "onUpdate": true,
    "presentable": false,
    "system": false,
    "type": "autodate"
  }))

  // remove field
  collection.fields.removeById("_clone_tlPv")

  // remove field
  collection.fields.removeById("_clone_Bcc2")

  // remove field
  collection.fields.removeById("_clone_z3VX")

  // remove field
  collection.fields.removeById("_clone_Sq6u")

  // remove field
  collection.fields.removeById("_clone_3Yee")

  // remove field
  collection.fields.removeById("_clone_lXVP")

  // remove field
  collection.fields.removeById("_clone_hF2S")

  // remove field
  collection.fields.removeById("_clone_NwIo")

  // remove field
  collection.fields.removeById("_clone_KsTf")

  // remove field
  collection.fields.removeById("_clone_ovbQ")

  // remove field
  collection.fields.removeById("_clone_TZqk")

  // remove field
  collection.fields.removeById("_clone_vNRY")

  // remove field
  collection.fields.removeById("_clone_Ep0N")

  return app.save(collection)
})
