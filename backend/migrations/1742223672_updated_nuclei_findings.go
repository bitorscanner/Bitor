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

		collection, err := dao.FindCollectionByNameOrId("sgc6cuzt2qx3tmo")
		if err != nil {
			return err
		}

		// add
		new_created_by := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "z5qch0kk",
			"name": "created_by",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "27do0wbcuyfmbmx",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_created_by); err != nil {
			return err
		}
		collection.Schema.AddField(new_created_by)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("sgc6cuzt2qx3tmo")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("z5qch0kk")

		return dao.SaveCollection(collection)
	})
}
