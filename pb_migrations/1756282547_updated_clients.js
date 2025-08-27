/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2442875294")

  // update collection data
  unmarshal({
    "deleteRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "listRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "updateRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "viewRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\""
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2442875294")

  // update collection data
  unmarshal({
    "deleteRule": null,
    "listRule": "@request.auth.collectionName = \"_superusers\"",
    "updateRule": null,
    "viewRule": "@request.auth.collectionName = \"_superusers\""
  }, collection)

  return app.save(collection)
})
