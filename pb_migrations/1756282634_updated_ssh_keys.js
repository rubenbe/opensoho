/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_287151464")

  // update collection data
  unmarshal({
    "createRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "deleteRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "listRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "updateRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\""
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_287151464")

  // update collection data
  unmarshal({
    "createRule": "@request.auth.collectionName = \"_superusers\"",
    "deleteRule": "@request.auth.collectionName = \"_superusers\"",
    "listRule": "@request.auth.collectionName = \"_superusers\"",
    "updateRule": "@request.auth.collectionName = \"_superusers\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\""
  }, collection)

  return app.save(collection)
})
