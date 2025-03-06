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
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cxzqhrd7om4n8od")
		if err != nil {
			return err
		}

		// update
		edit_use := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "wuogycyi",
			"name": "use",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 3,
				"values": [
					"dns",
					"compute",
					"terraform_storage",
					"scan_storage",
					"notification",
					"discovery"
				]
			}
		}`), edit_use); err != nil {
			return err
		}
		collection.Schema.AddField(edit_use)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cxzqhrd7om4n8od")
		if err != nil {
			return err
		}

		// update
		edit_use := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "wuogycyi",
			"name": "use",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 3,
				"values": [
					"dns",
					"compute",
					"terraform_storage",
					"scan_storage",
					"notification"
				]
			}
		}`), edit_use); err != nil {
			return err
		}
		collection.Schema.AddField(edit_use)

		return dao.SaveCollection(collection)
	})
}
