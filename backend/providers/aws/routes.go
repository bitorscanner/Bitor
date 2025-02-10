package aws

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes registers all routes for the AWS provider
func RegisterRoutes(e *core.ServeEvent, g *echo.Group) {
	app := e.App.(*pocketbase.PocketBase)

	// Create a group for AWS routes
	awsGroup := g.Group("/aws")

	awsGroup.GET("/regions", func(c echo.Context) error {
		regions, err := FetchRegions(app, c.QueryParam("provider"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, regions)
	})

	awsGroup.GET("/vpcs", func(c echo.Context) error {
		vpcs, err := FetchVPCs(app, c.QueryParam("provider"), c.QueryParam("region"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, vpcs)
	})

	awsGroup.GET("/subnets", func(c echo.Context) error {
		subnets, err := FetchSubnets(app, c.QueryParam("provider"), c.QueryParam("region"), c.QueryParam("vpc"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, subnets)
	})

	awsGroup.GET("/security-groups", func(c echo.Context) error {
		securityGroups, err := FetchSecurityGroups(app, c.QueryParam("provider"), c.QueryParam("region"), c.QueryParam("vpc"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, securityGroups)
	})

	awsGroup.GET("/validate", func(c echo.Context) error {
		accountID, err := ValidateCredentials(app, c.QueryParam("provider"))
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]string{"account_id": accountID})
	})

	awsGroup.GET("/instance-types", func(c echo.Context) error {
		instanceTypes, err := FetchInstanceTypes(app, c.QueryParam("provider"), c.QueryParam("region"))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, instanceTypes)
	})
}
