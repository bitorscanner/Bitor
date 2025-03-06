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

		// Find the nuclei_results collection
		collection, err := dao.FindCollectionByNameOrId("nuclei_results")
		if err != nil {
			return err
		}

		// Add scan_ids field
		scan_ids := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "scan_ids",
			"name": "scan_ids",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000
			}
		}`), scan_ids); err != nil {
			return err
		}

		// Initialize scan_ids with an array containing the current scan_id
		collection.Schema.AddField(scan_ids)

		// Rename the collection
		collection.Name = "nuclei_findings"

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Find the nuclei_findings collection
		collection, err := dao.FindCollectionByNameOrId("nuclei_findings")
		if err != nil {
			return err
		}

		// Remove scan_ids field
		collection.Schema.RemoveField("scan_ids")

		// Rename back to original name
		collection.Name = "nuclei_results"

		return dao.SaveCollection(collection)
	})
}
