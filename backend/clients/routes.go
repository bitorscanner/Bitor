package clients

import (
	"log"
	"orbit/clients/handlers"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes registers the client routes
func RegisterRoutes(app *pocketbase.PocketBase, e *core.ServeEvent) {
	log.Printf("Registering client routes...")

	// Create and register the client handler
	clientHandler := handlers.NewClientHandler(app)
	clientHandler.RegisterRoutes(e)

	// Register model hooks
	app.OnModelBeforeCreate().Add(clientHandler.OnModelBeforeCreate)
	app.OnModelBeforeUpdate().Add(clientHandler.OnModelBeforeUpdate)
}
