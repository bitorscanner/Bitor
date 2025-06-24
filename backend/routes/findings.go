package routes

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"

	"bitor/services"
)

// RegisterFindingsRoutes registers all findings-related routes
func RegisterFindingsRoutes(app *pocketbase.PocketBase, e *core.ServeEvent, findingManager *services.FindingManager) {
	// Get user findings
	e.Router.GET("/api/findings/my", func(c echo.Context) error {
		// Get the current user
		user, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "User not found")
		}

		// Get query parameters
		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}

		perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
		if perPage < 1 {
			perPage = 50
		}

		filter := c.QueryParam("filter")
		sort := c.QueryParam("sort")
		if sort == "" {
			sort = "-created"
		}

		// Get findings
		findings, err := findingManager.GetUserFindings(user.Id, filter, sort, page, perPage)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// Get total count
		total, err := findingManager.GetUserFindingsCount(user.Id, filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// Get summary
		summary, err := findingManager.GetUserFindingsSummary(user.Id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":     page,
			"per_page": perPage,
			"total":    total,
			"items":    findings,
			"summary":  summary,
		})
	}, apis.RequireAdminOrRecordAuth())
}
