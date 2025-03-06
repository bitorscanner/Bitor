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

		// Try to find the collection under any of its possible names/IDs
		collection, err := dao.FindCollectionByNameOrId("jvg28mp4ucwl8nw")
		if err != nil {
			collection, err = dao.FindCollectionByNameOrId("finding_rollups")
			if err != nil {
				collection, err = dao.FindCollectionByNameOrId("nuclei_findings_rollups")
				if err != nil {
					collection, err = dao.FindCollectionByNameOrId("nuclei_finding_rollups")
					if err != nil {
						// If no collection is found, skip this migration
						return nil
					}
				}
			}
		}

		// Update the rules
		collection.ListRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")
		collection.ViewRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")
		collection.CreateRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")
		collection.UpdateRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")
		collection.DeleteRule = types.Pointer("@request.auth.id != '' && (@request.auth.role = 'admin' || @request.auth.view_findings = true)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Try to find the collection under any of its possible names/IDs
		collection, err := dao.FindCollectionByNameOrId("jvg28mp4ucwl8nw")
		if err != nil {
			collection, err = dao.FindCollectionByNameOrId("finding_rollups")
			if err != nil {
				collection, err = dao.FindCollectionByNameOrId("nuclei_findings_rollups")
				if err != nil {
					collection, err = dao.FindCollectionByNameOrId("nuclei_finding_rollups")
					if err != nil {
						// If no collection is found, skip this migration
						return nil
					}
				}
			}
		}

		collection.ListRule = nil
		collection.ViewRule = nil
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		return dao.SaveCollection(collection)
	})
}
