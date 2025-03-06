package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// Try to find the collection under any of its possible names/IDs
		collection, err := dao.FindCollectionByNameOrId("jvg28mp4ucwl8nw")
		if err != nil {
			collection, err = dao.FindCollectionByNameOrId("finding_rollups")
			if err != nil {
				collection, err = dao.FindCollectionByNameOrId("nuclei_findings_rollups")
				if err != nil {
					collection, err = dao.FindCollectionByNameOrId("nuclei_finding_rollups")
					if err != nil {
						// If no collection is found, skip this migration
						return nil
					}
				}
			}
		}

		// update
		edit_scan_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gegttucq",
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
		}`), edit_scan_id); err != nil {
			return err
		}

		// Check if the field already exists
		hasField := false
		for _, field := range collection.Schema.Fields() {
			if field.Id == edit_scan_id.Id || field.Name == edit_scan_id.Name {
				hasField = true
				break
			}
		}

		// Only add the field if it doesn't exist
		if !hasField {
			collection.Schema.AddField(edit_scan_id)
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Try to find the collection under any of its possible names/IDs
		collection, err := dao.FindCollectionByNameOrId("jvg28mp4ucwl8nw")
		if err != nil {
			collection, err = dao.FindCollectionByNameOrId("finding_rollups")
			if err != nil {
				collection, err = dao.FindCollectionByNameOrId("nuclei_findings_rollups")
				if err != nil {
					collection, err = dao.FindCollectionByNameOrId("nuclei_finding_rollups")
					if err != nil {
						// If no collection is found, skip this migration
						return nil
					}
				}
			}
		}

		// update
		edit_scan_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gegttucq",
			"name": "scan_id_",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_scan_id); err != nil {
			return err
		}

		// Check if the field exists before trying to modify it
		hasField := false
		for _, field := range collection.Schema.Fields() {
			if field.Id == edit_scan_id.Id || field.Name == "scan_id_" {
				hasField = true
				break
			}
		}

		// Only try to modify the field if it exists
		if hasField {
			collection.Schema.AddField(edit_scan_id)
		}

		return dao.SaveCollection(collection)
	})
}
