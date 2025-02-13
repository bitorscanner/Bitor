package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `{
			"id": "wt89oezala8ljah",
			"created": "2025-02-13 03:06:54.616Z",
			"updated": "2025-02-13 03:06:54.616Z",
			"name": "user_preferences",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "jtgy4nd9",
					"name": "findings_filters",
					"type": "json",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSize": 2000000
					}
				}
			],
			"indexes": [],
			"listRule": "",
			"viewRule": "",
			"createRule": "",
			"updateRule": "",
			"deleteRule": "",
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("wt89oezala8ljah")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
