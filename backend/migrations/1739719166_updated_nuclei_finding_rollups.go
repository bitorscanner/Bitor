package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		// Try to find the collection under any of its possible names
		collection, err := dao.FindCollectionByNameOrId("finding_rollups")
		if err != nil {
			collection, err = dao.FindCollectionByNameOrId("nuclei_finding_rollups")
			if err != nil {
				// If neither collection exists, skip this migration
				return nil
			}
		}

		// Ensure the collection has the correct final name
		collection.Name = "nuclei_findings_rollups"

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		// Try to find the collection
		collection, err := dao.FindCollectionByNameOrId("nuclei_findings_rollups")
		if err != nil {
			// If the collection doesn't exist, skip this migration
			return nil
		}

		// Revert the name change
		collection.Name = "nuclei_finding_rollups"

		return dao.SaveCollection(collection)
	})
}
