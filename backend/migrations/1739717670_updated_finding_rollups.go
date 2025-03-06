package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
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

		// Update the collection name to the new name
		collection.Name = "nuclei_finding_rollups"

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Try to find the collection under any of its possible names/IDs
		collection, err := dao.FindCollectionByNameOrId("nuclei_finding_rollups")
		if err != nil {
			collection, err = dao.FindCollectionByNameOrId("nuclei_findings_rollups")
			if err != nil {
				collection, err = dao.FindCollectionByNameOrId("jvg28mp4ucwl8nw")
				if err != nil {
					// If no collection is found, skip this migration
					return nil
				}
			}
		}

		// Revert the name change
		collection.Name = "finding_rollups"

		return dao.SaveCollection(collection)
	})
}
