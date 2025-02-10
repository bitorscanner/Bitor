package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("xtuprs0au45l8d3")
		if err != nil {
			return err
		}

		collection.Name = "settings_public"

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("xtuprs0au45l8d3")
		if err != nil {
			return err
		}

		collection.Name = "settings"

		return dao.SaveCollection(collection)
	})
}
