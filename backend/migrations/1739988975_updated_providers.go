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

		collection, err := dao.FindCollectionByNameOrId("cxzqhrd7om4n8od")
		if err != nil {
			return err
		}

		// update
		edit_provider_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tchgvws3",
			"name": "provider_type",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"email",
					"slack",
					"teams",
					"discord",
					"telegram",
					"jira",
					"aws",
					"digitalocean",
					"s3",
					"alienvault",
					"binaryedge",
					"bufferover",
					"censys",
					"certspotter",
					"chaos",
					"github",
					"intelx",
					"passivetotal",
					"securitytrails",
					"shodan",
					"virustotal",
					"tailscale"
				]
			}
		}`), edit_provider_type); err != nil {
			return err
		}
		collection.Schema.AddField(edit_provider_type)

		// update
		edit_use := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "wuogycyi",
			"name": "use",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 3,
				"values": [
					"dns",
					"compute",
					"terraform_storage",
					"scan_storage",
					"notification",
					"discovery",
					"vpn"
				]
			}
		}`), edit_use); err != nil {
			return err
		}
		collection.Schema.AddField(edit_use)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cxzqhrd7om4n8od")
		if err != nil {
			return err
		}

		// update
		edit_provider_type := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tchgvws3",
			"name": "provider_type",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"email",
					"slack",
					"teams",
					"discord",
					"telegram",
					"jira",
					"aws",
					"digitalocean",
					"s3",
					"alienvault",
					"binaryedge",
					"bufferover",
					"censys",
					"certspotter",
					"chaos",
					"github",
					"intelx",
					"passivetotal",
					"securitytrails",
					"shodan",
					"virustotal"
				]
			}
		}`), edit_provider_type); err != nil {
			return err
		}
		collection.Schema.AddField(edit_provider_type)

		// update
		edit_use := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "wuogycyi",
			"name": "use",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 3,
				"values": [
					"dns",
					"compute",
					"terraform_storage",
					"scan_storage",
					"notification",
					"discovery"
				]
			}
		}`), edit_use); err != nil {
			return err
		}
		collection.Schema.AddField(edit_use)

		return dao.SaveCollection(collection)
	})
}
