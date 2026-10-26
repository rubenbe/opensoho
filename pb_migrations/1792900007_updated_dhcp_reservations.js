/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3986314751")

  // rename field: name -> hostname
  collection.fields.getById("text1579384326").name = "hostname"

  // update index on the renamed field
  collection.indexes = collection.indexes.map((index) =>
    index.startsWith("CREATE UNIQUE INDEX `idx_dhcp_reservation_name`")
      ? "CREATE UNIQUE INDEX `idx_dhcp_reservation_hostname` ON `dhcp_reservations` (`hostname`)"
      : index
  )

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3986314751")

  // rename field: hostname -> name
  collection.fields.getById("text1579384326").name = "name"

  // restore index on the original field
  collection.indexes = collection.indexes.map((index) =>
    index.startsWith("CREATE UNIQUE INDEX `idx_dhcp_reservation_hostname`")
      ? "CREATE UNIQUE INDEX `idx_dhcp_reservation_name` ON `dhcp_reservations` (`name`)"
      : index
  )

  return app.save(collection)
})
