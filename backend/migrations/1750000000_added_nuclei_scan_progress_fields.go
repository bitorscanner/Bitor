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

		collection, err := dao.FindCollectionByNameOrId("zqdmvqo2mym808a")
		if err != nil {
			return err
		}

		// add progress_message field
		progress_message := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "progress_msg",
			"name": "progress_message",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": 500,
				"pattern": ""
			}
		}`), progress_message); err != nil {
			return err
		}
		collection.Schema.AddField(progress_message)

		// add progress_percentage field
		progress_percentage := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "progress_pct",
			"name": "progress_percentage",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": 100,
				"noDecimal": true
			}
		}`), progress_percentage); err != nil {
			return err
		}
		collection.Schema.AddField(progress_percentage)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("zqdmvqo2mym808a")
		if err != nil {
			return err
		}

		// remove fields
		collection.Schema.RemoveField("progress_msg")
		collection.Schema.RemoveField("progress_pct")

		return dao.SaveCollection(collection)
	})
}
