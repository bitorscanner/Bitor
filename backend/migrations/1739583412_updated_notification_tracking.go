package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("eic9dy32f8uaq66")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.manage_notifications = true)")

		collection.ViewRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.manage_notifications = true)")

		collection.CreateRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.manage_notifications = true)")

		collection.UpdateRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.manage_notifications = true)")

		collection.DeleteRule = types.Pointer("@request.auth.id != '' && @request.auth.role = 'admin'")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("eic9dy32f8uaq66")
		if err != nil {
			return err
		}

		collection.ListRule = nil

		collection.ViewRule = nil

		collection.CreateRule = nil

		collection.UpdateRule = nil

		collection.DeleteRule = nil

		return dao.SaveCollection(collection)
	})
}
