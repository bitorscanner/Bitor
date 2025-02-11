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
		new_severity_override := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ag0kh6js",
			"name": "severity_override",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"critical",
					"high",
					"medium",
					"low",
					"info"
				]
			}
		}`), new_severity_override); err != nil {
			return err
		}
		collection.Schema.AddField(new_severity_override)

		// add
		new_severity_override_order := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "inrsweak",
			"name": "severity_override_order",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_severity_override_order); err != nil {
			return err
		}
		collection.Schema.AddField(new_severity_override_order)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("sgc6cuzt2qx3tmo")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("ag0kh6js")

		// remove
		collection.Schema.RemoveField("inrsweak")

		return dao.SaveCollection(collection)
	})
}
