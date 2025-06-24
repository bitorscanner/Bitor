package providers

import (
	"fmt"
	"net/http"
	"bitor/providers/digitalocean"
	"bitor/providers/s3"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

// checkManageProvidersPermission creates a middleware that checks for the manage_providers permission
func checkManageProvidersPermission(app *pocketbase.PocketBase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the auth context
			authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)

			// Allow PocketBase admins
			if admin != nil {
				return next(c)
			}

			// If not an admin, check for user permissions
			if authRecord == nil {
				return apis.NewForbiddenError("User not found", nil)
			}

			// Get the group ID
			groupID := authRecord.GetString("group")
			if groupID == "" {
				return apis.NewForbiddenError("User has no group assigned", nil)
			}

			// Fetch the group record directly
			group, err := app.Dao().FindRecordById("groups", groupID)
			if err != nil {
				return apis.NewForbiddenError("Failed to find group", err)
			}

			// Check if group name is "admin"
			if group.GetString("name") == "admin" {
				return next(c)
			}

			// Check for manage_providers permission
			permissions, ok := group.Get("permissions").(map[string]interface{})
			if ok {
				if manageProviders, ok := permissions["manage_providers"].(bool); ok && manageProviders {
					return next(c)
				}
			}

			return apis.NewForbiddenError("User does not have manage_providers permission", nil)
		}
	}
}

func RegisterRoutes(app *pocketbase.PocketBase, router *echo.Group) {
	fmt.Printf("Starting provider routes registration...\n")

	// Create a group for provider routes with authentication middleware
	providerGroup := router.Group("/providers",
		apis.LoadAuthContext(app),           // Apply LoadAuthContext middleware first
		apis.RequireAdminOrRecordAuth(),     // Allow both admin and user authentication
		checkManageProvidersPermission(app), // Check for manage_providers permission
		apis.ActivityLogger(app),            // Optional: log activities
	)

	fmt.Printf("Provider group created with path: /providers\n")

	// DigitalOcean routes
	providerGroup.GET("/digitalocean/projects", func(c echo.Context) error {
		fmt.Printf("Handling /digitalocean/projects request\n")
		fmt.Printf("Request headers: %v\n", c.Request().Header)
		fmt.Printf("Request URL: %s\n", c.Request().URL.String())
		fmt.Printf("Request Method: %s\n", c.Request().Method)

		providerID := c.QueryParam("providerId")
		fmt.Printf("Provider ID: %s\n", providerID)

		if providerID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Provider ID is required",
			})
		}

		projects, err := digitalocean.FetchProjects(app, providerID)
		if err != nil {
			fmt.Printf("Error fetching projects: %v\n", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to fetch projects: %v", err),
			})
		}

		return c.JSON(http.StatusOK, projects)
	})

	providerGroup.GET("/digitalocean/regions", func(c echo.Context) error {
		fmt.Printf("Handling /digitalocean/regions request\n")
		fmt.Printf("Request headers: %v\n", c.Request().Header)

		providerID := c.QueryParam("providerId")
		fmt.Printf("Provider ID: %s\n", providerID)

		if providerID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Provider ID is required",
			})
		}

		regions, err := digitalocean.FetchRegions(app, providerID)
		if err != nil {
			fmt.Printf("Error fetching regions: %v\n", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to fetch regions: %v", err),
			})
		}

		return c.JSON(http.StatusOK, regions)
	})

	providerGroup.GET("/digitalocean/sizes", func(c echo.Context) error {
		providerID := c.QueryParam("providerId")
		region := c.QueryParam("region")
		if providerID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Provider ID is required",
			})
		}
		if region == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Region is required",
			})
		}

		sizes, err := digitalocean.FetchSizes(app, providerID, region)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to fetch sizes: %v", err),
			})
		}

		return c.JSON(http.StatusOK, sizes)
	})

	providerGroup.GET("/digitalocean/domains", func(c echo.Context) error {
		providerID := c.QueryParam("providerId")
		if providerID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Provider ID is required",
			})
		}

		domains, err := digitalocean.FetchDomains(app, providerID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to fetch domains: %v", err),
			})
		}

		return c.JSON(http.StatusOK, domains)
	})

	providerGroup.GET("/digitalocean/price", func(c echo.Context) error {
		providerID := c.QueryParam("providerId")
		region := c.QueryParam("region")
		size := c.QueryParam("size")

		if providerID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Provider ID is required",
			})
		}
		if region == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Region is required",
			})
		}
		if size == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Size is required",
			})
		}

		provider, err := app.Dao().FindRecordById("providers", providerID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to find provider: %v", err),
			})
		}

		apiKey, err := digitalocean.GetAPIKey(app, provider)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to get API key: %v", err),
			})
		}

		price, err := digitalocean.GetSizePrice(apiKey, region, size)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to get price: %v", err),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"price_hourly": price,
		})
	})

	// S3 routes
	providerGroup.POST("/s3/test", func(c echo.Context) error {
		providerID := c.QueryParam("provider")
		testPath := c.QueryParam("path")
		
		if providerID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "provider parameter is required",
			})
		}
		
		if testPath == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "path parameter is required",
			})
		}

		err := s3.TestS3Connection(app, providerID, testPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "success",
			"message": "S3 connection test passed successfully",
		})
	})

	providerGroup.GET("/s3/validate", func(c echo.Context) error {
		providerID := c.QueryParam("provider")
		
		if providerID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "provider parameter is required",
			})
		}

		err := s3.ValidateCredentials(app, providerID)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": err.Error(),
			})
		}
		
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "success",
			"message": "S3 credentials are valid",
		})
	})

	fmt.Printf("Provider routes registered successfully\n")
}
