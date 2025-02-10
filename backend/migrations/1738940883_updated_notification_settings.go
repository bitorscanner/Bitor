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

		collection, err := dao.FindCollectionByNameOrId("aambiuf5a1xzmgk")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.manage_notifications = true")

		collection.ViewRule = types.Pointer("@request.auth.manage_notifications = true")

		collection.CreateRule = types.Pointer("@request.auth.manage_notifications = true")

		collection.UpdateRule = types.Pointer("@request.auth.manage_notifications = true")

		collection.DeleteRule = types.Pointer("@request.auth.manage_notifications = true")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("aambiuf5a1xzmgk")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.manage_notifications = true")

		collection.ViewRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.manage_notifications = true")

		collection.CreateRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.manage_notifications = true")

		collection.UpdateRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.manage_notifications = true")

		collection.DeleteRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.manage_notifications = true")

		return dao.SaveCollection(collection)
	})
}
