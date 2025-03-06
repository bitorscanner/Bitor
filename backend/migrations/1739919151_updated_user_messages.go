package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("xk9p2n4m7eoyhwt")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id = user.id || @request.auth.isAdmin = true")

		collection.ViewRule = types.Pointer("@request.auth.id = user.id || @request.auth.isAdmin = true")

		collection.UpdateRule = types.Pointer("@request.auth.id = user.id || @request.auth.isAdmin = true")

		collection.DeleteRule = types.Pointer("@request.auth.id = user.id || @request.auth.isAdmin = true")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("xk9p2n4m7eoyhwt")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id = user.id || @request.auth.id = admin.id")

		collection.ViewRule = types.Pointer("@request.auth.id = user.id || @request.auth.id = admin.id")

		collection.UpdateRule = types.Pointer("@request.auth.id = user.id || @request.auth.id = admin.id")

		collection.DeleteRule = types.Pointer("@request.auth.id = user.id || @request.auth.id = admin.id")

		return dao.SaveCollection(collection)
	})
}
