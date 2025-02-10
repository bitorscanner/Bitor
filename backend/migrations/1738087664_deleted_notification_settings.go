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
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("notification_settings")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	}, func(db dbx.Builder) error {
		jsonData := `{
			"id": "notification_settings",
			"created": "2025-01-10 16:03:25.109Z",
			"updated": "2025-01-22 01:23:30.760Z",
			"name": "notification_settings",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "rules",
					"name": "rules",
					"type": "json",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSize": 0
					}
				}
			],
			"indexes": [],
			"listRule": "@request.auth.manage_notifications = true",
			"viewRule": "@request.auth.manage_notifications = true",
			"createRule": "@request.auth.manage_notifications = true",
			"updateRule": "@request.auth.manage_notifications = true",
			"deleteRule": "@request.auth.manage_notifications = true",
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	})
}
