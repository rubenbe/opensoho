/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3180122789")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_W0V5biTr4G` ON `radio_frequencies` (\n  `device`,\n  `radio`,\n  `frequency`\n)",
      "CREATE UNIQUE INDEX `idx_bEKRSwWtea` ON `radio_frequencies` (\n  `device`,\n  `radio`,\n  `channel`\n)"
    ]
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3180122789")

  // update collection data
  unmarshal({
    "indexes": [
      "CREATE UNIQUE INDEX `idx_W0V5biTr4G` ON `radio_frequencies` (\n  `device`,\n  `radio`,\n  `frequency`\n)",
      "CREATE INDEX `idx_bEKRSwWtea` ON `radio_frequencies` (\n  `device`,\n  `radio`,\n  `channel`\n)"
    ]
  }, collection)

  return app.save(collection)
})
