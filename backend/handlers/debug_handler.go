package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterDebugRoutes registers debug endpoints for development purposes.
// This endpoint returns the current authenticated record for debugging.
func RegisterDebugRoutes(e *core.ServeEvent) {
	e.Router.GET("/api/debug/auth", func(c echo.Context) error {
		record := c.Get("record")
		if record == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "No auth record found"})
		}
		return c.JSON(http.StatusOK, record)
	}, apis.RequireRecordAuth())
}
