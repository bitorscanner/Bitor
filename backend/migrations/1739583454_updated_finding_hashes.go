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

		collection, err := dao.FindCollectionByNameOrId("az9g2m6y2eoyhwo")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")

		collection.ViewRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")

		collection.CreateRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")

		collection.UpdateRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")

		collection.DeleteRule = types.Pointer("@request.auth.id != '' && @request.auth.role = 'admin'")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("az9g2m6y2eoyhwo")
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
