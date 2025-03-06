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

		collection, err := dao.FindCollectionByNameOrId("xk9p2n4m7eoyhwt")
		if err != nil {
			return err
		}

		// add
		new_admin_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ej858mpe",
			"name": "admin_id",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_admin_id); err != nil {
			return err
		}
		collection.Schema.AddField(new_admin_id)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("xk9p2n4m7eoyhwt")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("ej858mpe")

		return dao.SaveCollection(collection)
	})
}
