/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_4047009785")

  // update collection data
  unmarshal({
    "createRule": "@request.auth.collectionName = \"_superusers\"",
    "deleteRule": "@request.auth.collectionName = \"_superusers\"",
    "listRule": "@request.auth.collectionName = \"_superusers\"",
    "updateRule": "@request.auth.collectionName = \"_superusers\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\""
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_4047009785")

  // update collection data
  unmarshal({
    "createRule": null,
    "deleteRule": null,
    "listRule": null,
    "updateRule": null,
    "viewRule": null
  }, collection)

  return app.save(collection)
})
