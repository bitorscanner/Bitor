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
            "id": "nuclei_findings_rollups",
            "name": "nuclei_findings_rollups",
            "type": "base",
            "system": false,
            "schema": [
                {
                    "id": "scan_id",
                    "name": "scan_id",
                    "type": "text",
                    "system": false,
                    "required": true,
                    "unique": false,
                    "options": {
                        "min": null,
                        "max": null,
                        "pattern": ""
                    }
                },
                {
                    "id": "critical_count",
                    "name": "critical_count",
                    "type": "number",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {
                        "min": 0,
                        "max": null
                    }
                },
                {
                    "id": "high_count",
                    "name": "high_count",
                    "type": "number",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {
                        "min": 0,
                        "max": null
                    }
                },
                {
                    "id": "medium_count",
                    "name": "medium_count",
                    "type": "number",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {
                        "min": 0,
                        "max": null
                    }
                },
                {
                    "id": "low_count",
                    "name": "low_count",
                    "type": "number",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {
                        "min": 0,
                        "max": null
                    }
                },
                {
                    "id": "info_count",
                    "name": "info_count",
                    "type": "number",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {
                        "min": 0,
                        "max": null
                    }
                },
                {
                    "id": "new_findings_count",
                    "name": "new_findings_count",
                    "type": "number",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {
                        "min": 0,
                        "max": null
                    }
                },
                {
                    "id": "duplicate_findings_count",
                    "name": "duplicate_findings_count",
                    "type": "number",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {
                        "min": 0,
                        "max": null
                    }
                },
                {
                    "id": "notification_sent",
                    "name": "notification_sent",
                    "type": "bool",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {}
                },
                {
                    "id": "last_notification_time",
                    "name": "last_notification_time",
                    "type": "date",
                    "system": false,
                    "required": false,
                    "unique": false,
                    "options": {}
                }
            ],
            "listRule": "",
            "viewRule": "",
            "createRule": "",
            "updateRule": "",
            "deleteRule": "",
            "options": {}
        }`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("nuclei_findings_rollups")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
