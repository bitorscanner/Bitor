// File: backend/providers/digitalocean/routes.go
package digitalocean

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes registers all DigitalOcean provider routes
func RegisterRoutes(e *core.ServeEvent, apiGroup *echo.Group) {
	doGroup := apiGroup.Group("/providers/digitalocean")

	// Register routes
	doGroup.GET("/regions", func(c echo.Context) error {
		providerId := c.QueryParam("providerId")
		if providerId == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "providerId is required"})
		}

		app := e.App.(*pocketbase.PocketBase)
		regions, err := FetchRegions(app, providerId)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, regions)
	})

	doGroup.GET("/sizes", func(c echo.Context) error {
		providerId := c.QueryParam("providerId")
		region := c.QueryParam("region")
		if providerId == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "providerId is required"})
		}
		if region == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "region is required"})
		}

		app := e.App.(*pocketbase.PocketBase)
		sizes, err := FetchSizes(app, providerId, region)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, sizes)
	})

	doGroup.GET("/projects", func(c echo.Context) error {
		providerId := c.QueryParam("providerId")
		if providerId == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "providerId is required"})
		}

		app := e.App.(*pocketbase.PocketBase)
		projects, err := FetchProjects(app, providerId)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, projects)
	})
}
