package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{
			Name:   "migration_status",
			Type:   models.CollectionTypeBase,
			System: false,
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "is_processing",
					Type:     schema.FieldTypeBool,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "total_count",
					Type:     schema.FieldTypeNumber,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "processed_count",
					Type:     schema.FieldTypeNumber,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "progress",
					Type:     schema.FieldTypeNumber,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "error",
					Type:     schema.FieldTypeText,
					Required: false,
				},
				&schema.SchemaField{
					Name:     "current_status",
					Type:     schema.FieldTypeText,
					Required: false,
				},
			),
			ListRule:   types.Pointer("@request.auth.role = 'admin'"),
			ViewRule:   types.Pointer("@request.auth.role = 'admin'"),
			CreateRule: types.Pointer("@request.auth.role = 'admin'"),
			UpdateRule: types.Pointer("@request.auth.role = 'admin'"),
			DeleteRule: types.Pointer("@request.auth.role = 'admin'"),
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("migration_status")
		if err != nil {
			return nil
		}

		return dao.DeleteCollection(collection)
	})
}
