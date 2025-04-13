/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2442875294")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_jPqcDjnlPJ` ON `clients` (`mac_address`)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2442875294")

  // update collection data
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
})
