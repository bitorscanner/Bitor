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

		// update
		edit_ansible_logs := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "erldsmuf",
			"name": "ansible_logs",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 200000000
			}
		}`), edit_ansible_logs); err != nil {
			return err
		}
		collection.Schema.AddField(edit_ansible_logs)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("zqdmvqo2mym808a")
		if err != nil {
			return err
		}

		// update
		edit_ansible_logs := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "erldsmuf",
			"name": "ansible_logs",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000000
			}
		}`), edit_ansible_logs); err != nil {
			return err
		}
		collection.Schema.AddField(edit_ansible_logs)

		return dao.SaveCollection(collection)
	})
}
