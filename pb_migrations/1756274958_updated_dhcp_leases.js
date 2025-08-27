/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2401716254")

  // update collection data
  unmarshal({
    "deleteRule": "@request.auth.collectionName = \"_superusers\"",
    "listRule": "@request.auth.collectionName = \"_superusers\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\""
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2401716254")

  // update collection data
  unmarshal({
    "deleteRule": null,
    "listRule": null,
    "viewRule": null
  }, collection)

  return app.save(collection)
})
