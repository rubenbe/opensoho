/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("_pb_users_auth_")

  // update collection data
  unmarshal({
    "createRule": "@request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "deleteRule": "id = @request.auth.id || @request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "listRule": "id = @request.auth.id || @request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "updateRule": "id = @request.auth.id || @request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\"",
    "viewRule": "id = @request.auth.id || @request.auth.collectionName = \"_superusers\" && @request.auth.id != \"\""
  }, collection)

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("_pb_users_auth_")

  // update collection data
  unmarshal({
    "createRule": "",
    "deleteRule": "id = @request.auth.id",
    "listRule": "id = @request.auth.id",
    "updateRule": "id = @request.auth.id",
    "viewRule": "id = @request.auth.id"
  }, collection)

  return app.save(collection)
})
