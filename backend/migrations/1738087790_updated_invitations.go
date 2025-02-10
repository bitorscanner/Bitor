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

		collection, err := dao.FindCollectionByNameOrId("5a9so80mi0eefyz")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\")")

		collection.ViewRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\")")

		collection.CreateRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\")")

		collection.UpdateRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\")")

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" && (@request.auth.isAdmin = true || @request.auth.group.name = \"admin\")")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("5a9so80mi0eefyz")
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
