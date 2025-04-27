/// <reference path="../pb_data/types.d.ts" />
migrate((app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // add field
  collection.fields.addAt(2, new Field({
    "hidden": false,
    "id": "select6459044032",
    "maxSelect": 1,
    "name": "frequency",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "2412",
      "2417",
      "2422",
      "2427",
      "2432",
      "2437",
      "2442",
      "2447",
      "2452",
      "2457",
      "2462",
      "2467",
      "2472",
      "5180",
      "5200",
      "5220",
      "5240",
      "5260",
      "5280",
      "5300",
      "5320",
      "5500",
      "5520",
      "5540",
      "5560",
      "5580",
      "5600",
      "5620",
      "5640",
      "5660",
      "5680",
      "5700",
      "5720",
      "5745",
      "5765",
      "5785",
      "5805",
      "5825",
      "5955",
      "5975",
      "5995",
      "6015",
      "6035",
      "6055",
      "6075",
      "6095",
      "6115",
      "6135",
      "6155",
      "6175",
      "6195",
      "6215",
      "6235",
      "6255",
      "6275",
      "6295",
      "6315",
      "6335",
      "6355",
      "6375",
      "6395",
      "6415",
      "6435",
      "6455",
      "6475",
      "6495",
      "6515",
      "6535",
      "6555",
      "6575",
      "6595",
      "6615",
      "6635",
      "6655",
      "6675",
      "6695",
      "6715",
      "6735",
      "6755",
      "6775",
      "6795",
      "6815",
      "6835",
      "6855",
      "6875",
      "6895",
      "6915",
      "6935",
      "6955",
      "6975",
      "58320",
      "60480",
      "62640",
      "64800",
      "66960"
    ]
  }))

  // update field
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "select645904403",
    "maxSelect": 1,
    "name": "band",
    "presentable": false,
    "required": true,
    "system": false,
    "type": "select",
    "values": [
      "2.4",
      "5",
      "6",
      "60"
    ]
  }))

  return app.save(collection)
}, (app) => {
  const collection = app.findCollectionByNameOrId("pbc_3158501839")

  // remove field
  collection.fields.removeById("select6459044032")

  // update field
  collection.fields.addAt(1, new Field({
    "hidden": false,
    "id": "select645904403",
    "maxSelect": 1,
    "name": "frequency",
    "presentable": false,
    "required": false,
    "system": false,
    "type": "select",
    "values": [
      "2.4",
      "5"
    ]
  }))

  return app.save(collection)
})
