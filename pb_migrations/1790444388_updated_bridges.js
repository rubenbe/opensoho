/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_624525223")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_lWs0xFn6m9` ON `bridges` (\n  `device`,\n  `name`\n)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_624525223")

  // update collection data
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
})
