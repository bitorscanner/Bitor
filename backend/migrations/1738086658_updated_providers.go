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

		collection, err := dao.FindCollectionByNameOrId("cxzqhrd7om4n8od")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_providers = true")

		collection.ViewRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_providers = true")

		collection.CreateRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_providers = true")

		collection.UpdateRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_providers = true")

		collection.DeleteRule = types.Pointer("@request.auth.isAdmin = true || @request.auth.group.permissions.manage_providers = true")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("cxzqhrd7om4n8od")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.group.permissions.manage_providers = true")

		collection.ViewRule = types.Pointer("@request.auth.group.permissions.manage_providers = true")

		collection.CreateRule = types.Pointer("@request.auth.group.permissions.manage_providers = true")

		collection.UpdateRule = types.Pointer("@request.auth.group.permissions.manage_providers = true")

		collection.DeleteRule = types.Pointer("@request.auth.group.permissions.manage_providers = true")

		return dao.SaveCollection(collection)
	})
}
