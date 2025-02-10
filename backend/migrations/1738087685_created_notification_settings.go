package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `{
			"id": "aambiuf5a1xzmgk",
			"created": "2025-01-28 18:08:05.317Z",
			"updated": "2025-01-28 18:08:05.317Z",
			"name": "notification_settings",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "id3grbxd",
					"name": "data",
					"type": "json",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSize": 2000000
					}
				}
			],
			"indexes": [],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("aambiuf5a1xzmgk")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
