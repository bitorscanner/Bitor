package handlers

import (
	"fmt"
	"log"
	"net/http"
	"bitor/services"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type ClientHandler struct {
	app            *pocketbase.PocketBase
	faviconFetcher *services.FaviconFetcher
}

func NewClientHandler(app *pocketbase.PocketBase) *ClientHandler {
	return &ClientHandler{
		app:            app,
		faviconFetcher: services.NewFaviconFetcher(),
	}
}

func (h *ClientHandler) RegisterRoutes(e *core.ServeEvent) {
	e.Router.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/api/clients/:id/fetch-favicon",
		Handler: func(c echo.Context) error {
			return h.handleFetchFavicon(c)
		},
		Middlewares: []echo.MiddlewareFunc{
			apis.RequireRecordAuth(),
		},
	})
}

func (h *ClientHandler) handleFetchFavicon(c echo.Context) error {
	clientID := c.PathParam("id")
	if clientID == "" {
		return apis.NewBadRequestError("missing client id", nil)
	}

	// Get the client record
	client, err := h.app.Dao().FindRecordById("clients", clientID)
	if err != nil {
		return apis.NewNotFoundError("client not found", err)
	}

	// Get the homepage URL
	homepage := client.GetString("homepage")
	if homepage == "" {
		return apis.NewBadRequestError("client has no homepage URL", nil)
	}

	// Try to fetch the favicon
	faviconData, contentType, err := h.faviconFetcher.FetchFavicon(homepage)
	if err != nil {
		return apis.NewBadRequestError(fmt.Sprintf("failed to fetch favicon: %v", err), nil)
	}

	// Save the favicon data to the record
	dataURI := h.faviconFetcher.GetDataURI(faviconData, contentType)
	client.Set("favicon", dataURI)
	if err := h.app.Dao().SaveRecord(client); err != nil {
		return apis.NewBadRequestError("failed to save client record", err)
	}

	// Return success response with the favicon data
	return c.JSON(http.StatusOK, map[string]string{
		"favicon": dataURI,
	})
}

// Hook into the client creation/update process
func (h *ClientHandler) OnModelBeforeCreate(e *core.ModelEvent) error {
	record, ok := e.Model.(*models.Record)
	if !ok || record.Collection().Name != "clients" {
		return nil
	}

	// Try to fetch favicon if homepage is provided
	homepage := record.GetString("homepage")
	if homepage != "" {
		if faviconData, contentType, err := h.faviconFetcher.FetchFavicon(homepage); err == nil {
			// Convert to data URI
			dataURI := h.faviconFetcher.GetDataURI(faviconData, contentType)
			record.Set("favicon", dataURI)
		} else {
			log.Printf("Failed to fetch favicon for %s: %v", homepage, err)
		}
	}

	return nil
}

// Hook into the client update process
func (h *ClientHandler) OnModelBeforeUpdate(e *core.ModelEvent) error {
	record, ok := e.Model.(*models.Record)
	if !ok || record.Collection().Name != "clients" {
		return nil
	}

	// Check if homepage field was changed
	homepage := record.GetString("homepage")
	if homepage != "" {
		if faviconData, contentType, err := h.faviconFetcher.FetchFavicon(homepage); err == nil {
			// Convert to data URI
			dataURI := h.faviconFetcher.GetDataURI(faviconData, contentType)
			record.Set("favicon", dataURI)
		} else {
			log.Printf("Failed to fetch favicon for %s: %v", homepage, err)
		}
	}

	return nil
}
