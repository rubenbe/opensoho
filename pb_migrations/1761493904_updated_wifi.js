/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // add field
  collection.fields.addAt(7, new Field({
    "hidden": false,
    "id": "number1860293906",
    "max": null,
    "min": 1,
    "name": "dtim_period",
    "onlyInt": true,
    "presentable": false,
    "required": false,
    "system": false,
    "type": "number"
  }))

  const retval = app.save(collection)

  const records = app.findAllRecords(collection, $dbx.exp("true"))
  for (const record of records) {
    record.set("dtim_period", 2)
    app.save(record)
  }

  return retval
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_502121861")

  // remove field
  collection.fields.removeById("number1860293906")

  return app.save(collection)
})
