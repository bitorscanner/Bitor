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

		collection, err := dao.FindCollectionByNameOrId("wt89oezala8ljah")
		if err != nil {
			return err
		}

		// add
		new_users_relation := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fafmy8od",
			"name": "users_relation",
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
		}`), new_users_relation); err != nil {
			return err
		}
		collection.Schema.AddField(new_users_relation)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("wt89oezala8ljah")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("fafmy8od")

		return dao.SaveCollection(collection)
	})
}
