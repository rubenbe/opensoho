/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_501785886")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_yjjx6lPZBa` ON `interfaces` (\n  `device`,\n  `interface`\n)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_501785886")

  // update collection data
  unmarshal({
    "indexes": []
  }, collection)

  return app.save(collection)
})
