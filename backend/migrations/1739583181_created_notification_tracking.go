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
			"id": "eic9dy32f8uaq66",
			"created": "2025-02-15 01:33:01.405Z",
			"updated": "2025-02-15 01:33:01.405Z",
			"name": "notification_tracking",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "v2mctdav",
					"name": "scan_id",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "czxbyl3x",
					"name": "event_type",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"scan_started",
							"scan_finished",
							"scan_failed",
							"scan_stopped",
							"finding_summary"
						]
					}
				},
				{
					"system": false,
					"id": "vdko0o9t",
					"name": "status",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"'sent'",
							"'failed'",
							"'pending'"
						]
					}
				},
				{
					"system": false,
					"id": "xo9qijxi",
					"name": "sent_at",
					"type": "date",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				},
				{
					"system": false,
					"id": "wpp3cznr",
					"name": "channels",
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
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("eic9dy32f8uaq66")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
