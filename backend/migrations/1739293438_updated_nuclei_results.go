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

		collection, err := dao.FindCollectionByNameOrId("sgc6cuzt2qx3tmo")
		if err != nil {
			return err
		}

		// add
		new_extracted_results := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "dbarvrid",
			"name": "extracted_results",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000000
			}
		}`), new_extracted_results); err != nil {
			return err
		}
		collection.Schema.AddField(new_extracted_results)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("sgc6cuzt2qx3tmo")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("dbarvrid")

		return dao.SaveCollection(collection)
	})
}
