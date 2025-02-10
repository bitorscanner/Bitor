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

		collection, err := dao.FindCollectionByNameOrId("yw3iuuu4rx2d4ku")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_system = true")

		collection.ViewRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_system = true")

		collection.CreateRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_system = true")

		collection.UpdateRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_system = true")

		collection.DeleteRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_system = true")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("yw3iuuu4rx2d4ku")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\" || @request.auth.group.permissions.read ~ \"scan_profiles\" || @request.auth.group.permissions.read ~ \"*\")")

		collection.ViewRule = types.Pointer("")

		collection.CreateRule = types.Pointer("")

		collection.UpdateRule = types.Pointer("")

		collection.DeleteRule = types.Pointer("")

		return dao.SaveCollection(collection)
	})
}
