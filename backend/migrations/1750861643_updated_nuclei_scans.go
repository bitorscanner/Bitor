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

		collection, err := dao.FindCollectionByNameOrId("zqdmvqo2mym808a")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("preserve_vm")

		// add
		new_preserve_vm := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "qevcm55y",
			"name": "preserve_vm",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_preserve_vm); err != nil {
			return err
		}
		collection.Schema.AddField(new_preserve_vm)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("zqdmvqo2mym808a")
		if err != nil {
			return err
		}

		// add
		del_preserve_vm := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "preserve_vm",
			"name": "preserve_vm",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), del_preserve_vm); err != nil {
			return err
		}
		collection.Schema.AddField(del_preserve_vm)

		// remove
		collection.Schema.RemoveField("qevcm55y")

		return dao.SaveCollection(collection)
	})
}
