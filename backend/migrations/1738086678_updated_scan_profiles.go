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

		collection, err := dao.FindCollectionByNameOrId("4ae108li7hjsxlz")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\" || @request.auth.group.permissions.read ~ \"scan_profiles\" || @request.auth.group.permissions.read ~ \"*\")")

		collection.ViewRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\" || @request.auth.group.permissions.read ~ \"scan_profiles\" || @request.auth.group.permissions.read ~ \"*\")")

		collection.CreateRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\" || @request.auth.group.permissions.write ~ \"scan_profiles\" || @request.auth.group.permissions.write ~ \"*\")")

		collection.UpdateRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\" || @request.auth.group.permissions.write ~ \"scan_profiles\" || @request.auth.group.permissions.write ~ \"*\")")

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\" || @request.auth.group.permissions.delete ~ \"scan_profiles\" || @request.auth.group.permissions.delete ~ \"*\")")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("4ae108li7hjsxlz")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.group.name = \"admin\" || @request.auth.group.permissions.read ~ \"scan_profiles\" || @request.auth.group.permissions.read ~ \"*\")")

		collection.ViewRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.group.name = \"admin\" || @request.auth.group.permissions.read ~ \"scan_profiles\" || @request.auth.group.permissions.read ~ \"*\")")

		collection.CreateRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.group.name = \"admin\" || @request.auth.group.permissions.write ~ \"scan_profiles\" || @request.auth.group.permissions.write ~ \"*\")")

		collection.UpdateRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.group.name = \"admin\" || @request.auth.group.permissions.write ~ \"scan_profiles\" || @request.auth.group.permissions.write ~ \"*\")")

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.group.name = \"admin\" || @request.auth.group.permissions.delete ~ \"scan_profiles\" || @request.auth.group.permissions.delete ~ \"*\")")

		return dao.SaveCollection(collection)
	})
}
