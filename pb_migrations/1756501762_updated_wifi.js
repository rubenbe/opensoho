/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // add field
  collection.fields.addAt(10, new Field({
    "hidden": false,
    "id": "bool1071757798",
    "name": "ieee80211v_wnm_sleep_mode",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  // update field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "bool3501268938",
    "name": "ieee80211v_bss_transition",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // remove field
  collection.fields.removeById("bool1071757798")

  // update field
  collection.fields.addAt(9, new Field({
    "hidden": false,
    "id": "bool3501268938",
    "name": "ieee80211v",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "bool"
  }))

  return app.save(collection)
})
