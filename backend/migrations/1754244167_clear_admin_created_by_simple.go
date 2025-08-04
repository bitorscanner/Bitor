package migrations

import (
	"log"

	"github.com/pocketbase/dbx"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		// Clear admin IDs from the created_by field by setting them to empty string
		// instead of NULL to avoid constraint issues
		_, err := db.NewQuery(`
			UPDATE nuclei_findings 
			SET created_by = '' 
			WHERE created_by IN (SELECT id FROM _admins)
		`).Execute()

		if err != nil {
			log.Printf("Error clearing admin created_by fields: %v", err)
			return err
		}

		log.Printf("Successfully cleared admin IDs from nuclei_findings created_by field")
		return nil
	}, func(db dbx.Builder) error {
		// This migration is not reversible since we don't know the original admin IDs
		log.Printf("Rollback not supported for clearing admin created_by fields")
		return nil
	})
}
