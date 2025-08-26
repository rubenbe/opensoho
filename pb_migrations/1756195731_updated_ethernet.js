/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3702000743")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE INDEX `idx_E8lz3SFGI0` ON `ethernet` (\n  `device`,\n  `name`\n)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3702000743")

  // update collection data
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
})
