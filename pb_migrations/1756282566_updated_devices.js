/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update collection data
  unmarshal({
    "createRule": "",
    "deleteRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "updateRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\""
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_2153001328")

  // update collection data
  unmarshal({
    "createRule": "@request.auth.collectionName = \"_superusers\"",
    "deleteRule": null,
    "updateRule": ""
  }, collection)

  return app.save(collection)
})
