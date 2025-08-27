/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

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
  const collection = app.findCollectionByNameOrId("pbc_3087215819")

  // update collection data
  unmarshal({
    "createRule": null,
    "deleteRule": null,
    "listRule": "@request.auth.collectionName = \"_superusers\"",
    "updateRule": null,
    "viewRule": "@request.auth.collectionName = \"_superusers\""
  }, collection)

  return app.save(collection)
})
