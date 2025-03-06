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

		// Check if collection already exists
		existingCollection, err := dao.FindCollectionByNameOrId("nuclei_findings_history")
		if err == nil && existingCollection != nil {
			// Collection already exists, skip creation
			return nil
		}

		// Also check by ID in case it exists with a different name
		existingCollection, err = dao.FindCollectionByNameOrId("az9g2m6y2eoyhwo")
		if err == nil && existingCollection != nil {
			// Collection exists with the ID, just update the name
			existingCollection.Name = "nuclei_findings_history"
			return dao.SaveCollection(existingCollection)
		}

		jsonData := `{
			"id": "az9g2m6y2eoyhwo",
			"name": "nuclei_findings_history",
			"type": "base",
			"system": false,
			"schema": [
				{
					"id": "xj8k2m9p4eoyhwq",
					"name": "hash",
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
					"id": "vn5r2k7t1eoyhwm",
					"name": "client_id",
					"type": "relation",
					"system": false,
					"required": true,
					"options": {
						"collectionId": "2hmr3iu22ww6uih",
						"cascadeDelete": true,
						"maxSelect": 1,
						"minSelect": 1
					}
				},
				{
					"id": "qs3f2h8n6eoyhwx",
					"name": "first_seen",
					"type": "date",
					"system": false,
					"required": true,
					"options": {}
				},
				{
					"id": "lm7j2d4b9eoyhwc",
					"name": "last_seen",
					"type": "date",
					"system": false,
					"required": true,
					"options": {}
				},
				{
					"id": "wp9g2s5v3eoyhwr",
					"name": "scan_ids",
					"type": "json",
					"system": false,
					"required": true,
					"options": {}
				},
				{
					"id": "bt4n2q6m8eoyhwk",
					"name": "severity",
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
					"id": "hf1c2x7l5eoyhwp",
					"name": "title",
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
					"id": "ry6b2w3g7eoyhwj",
					"name": "description",
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
					"id": "um2z2v8f4eoyhwn",
					"name": "target",
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
					"id": "gk5y2t1h3eoyhws",
					"name": "type",
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
					"id": "cd8x2r4j6eoyhwb",
					"name": "tool",
					"type": "text",
					"system": false,
					"required": true,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				}
			],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("az9g2m6y2eoyhwo")
		if err != nil {
			// If collection doesn't exist, nothing to delete
			return nil
		}

		return dao.DeleteCollection(collection)
	})
}
