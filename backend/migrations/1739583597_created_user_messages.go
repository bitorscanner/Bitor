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
			"id": "xk9p2n4m7eoyhwt",
			"name": "user_messages",
			"type": "base",
			"system": false,
			"schema": [
				{
					"id": "vr5h2k8n3eoyhwp",
					"name": "message",
					"type": "text",
					"system": false,
					"required": true,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"id": "qm2j2p6b9eoyhwc",
					"name": "type",
					"type": "select",
					"system": false,
					"required": true,
					"options": {
						"maxSelect": 1,
						"values": ["info", "success", "warning", "error"]
					}
				},
				{
					"id": "tn7r2d5v8eoyhwx",
					"name": "read",
					"type": "bool",
					"system": false,
					"required": false,
					"options": {}
				},
				{
					"id": "wf4k2m1h6eoyhwb",
					"name": "user",
					"type": "relation",
					"system": false,
					"required": false,
					"options": {
						"collectionId": "27do0wbcuyfmbmx",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				}
			],
			"listRule": "@request.auth.id = user.id || @request.auth.id = admin_id",
			"viewRule": "@request.auth.id = user.id || @request.auth.id = admin_id",
			"createRule": "@request.auth.id != \"\"",
			"updateRule": "@request.auth.id = user.id || @request.auth.id = admin_id",
			"deleteRule": "@request.auth.id = user.id || @request.auth.id = admin_id",
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)
		collection, err := dao.FindCollectionByNameOrId("xk9p2n4m7eoyhwt")
		if err != nil {
			return err
		}
		return dao.DeleteCollection(collection)
	})
}
