package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
)

// RequirePermission creates middleware that checks if the user has the required permission
func RequirePermission(app *pocketbase.PocketBase, action string, collection string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the token from the Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header")
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Skip for admin users
			admin, _ := app.Dao().FindAdminByToken(token, app.Settings().RecordAuthToken.Secret)
			if admin != nil {
				return next(c)
			}

			// Get authenticated user
			record, err := app.Dao().FindAuthRecordByToken(token, app.Settings().RecordAuthToken.Secret)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
			}

			// Get user's group
			groupId := record.GetString("group")
			if groupId == "" {
				return echo.NewHTTPError(http.StatusForbidden, "User has no assigned group")
			}

			// Get group record
			group, err := app.Dao().FindRecordById("groups", groupId)
			if err != nil {
				return echo.NewHTTPError(http.StatusForbidden, "Invalid group")
			}

			// Get and parse group permissions
			rawPermissions := group.Get("permissions")
			var permissions map[string]interface{}

			switch p := rawPermissions.(type) {
			case types.JsonRaw:
				if err := json.Unmarshal(p, &permissions); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse permissions")
				}
			case map[string]interface{}:
				permissions = p
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "Invalid permissions format")
			}

			// Check permission based on action
			switch action {
			case "read":
				readPerms, ok := permissions["read"].([]interface{})
				if !ok {
					return echo.NewHTTPError(http.StatusForbidden, "Invalid read permissions format")
				}
				hasPermission := false
				for _, perm := range readPerms {
					if permStr, ok := perm.(string); ok {
						if permStr == "*" || permStr == collection {
							hasPermission = true
							break
						}
					}
				}
				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "write":
				writePerms, ok := permissions["write"].([]interface{})
				if !ok {
					return echo.NewHTTPError(http.StatusForbidden, "Invalid write permissions format")
				}
				hasPermission := false
				for _, perm := range writePerms {
					if permStr, ok := perm.(string); ok {
						if permStr == "*" || permStr == collection {
							hasPermission = true
							break
						}
					}
				}
				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "delete":
				deletePerms, ok := permissions["delete"].([]interface{})
				if !ok {
					return echo.NewHTTPError(http.StatusForbidden, "Invalid delete permissions format")
				}
				hasPermission := false
				for _, perm := range deletePerms {
					if permStr, ok := perm.(string); ok {
						if permStr == "*" || permStr == collection {
							hasPermission = true
							break
						}
					}
				}
				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "manage_users":
				manageUsers, ok := permissions["manage_users"].(bool)
				if !ok {
					return echo.NewHTTPError(http.StatusForbidden, "Invalid manage_users permission format")
				}
				if !manageUsers {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "settings":
				settings, ok := permissions["settings"].(bool)
				if !ok {
					return echo.NewHTTPError(http.StatusForbidden, "Invalid settings permission format")
				}
				if !settings {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}
			}

			return next(c)
		}
	}
}

// HasPermission checks if a user has a specific permission
func HasPermission(app *pocketbase.PocketBase, user *models.Record, action string, collection string) bool {
	if user == nil {
		return false
	}

	// Get user's group
	groupId := user.GetString("group")
	if groupId == "" {
		return false
	}

	// Get group record
	group, err := app.Dao().FindRecordById("groups", groupId)
	if err != nil {
		return false
	}

	// Get and parse group permissions
	rawPermissions := group.Get("permissions")
	var permissions map[string]interface{}

	switch p := rawPermissions.(type) {
	case types.JsonRaw:
		if err := json.Unmarshal(p, &permissions); err != nil {
			return false
		}
	case map[string]interface{}:
		permissions = p
	default:
		return false
	}

	// Check permission based on action
	switch action {
	case "read":
		readPerms, ok := permissions["read"].([]interface{})
		if !ok {
			return false
		}
		for _, perm := range readPerms {
			if permStr, ok := perm.(string); ok {
				if permStr == "*" || permStr == collection {
					return true
				}
			}
		}

	case "write":
		writePerms, ok := permissions["write"].([]interface{})
		if !ok {
			return false
		}
		for _, perm := range writePerms {
			if permStr, ok := perm.(string); ok {
				if permStr == "*" || permStr == collection {
					return true
				}
			}
		}

	case "delete":
		deletePerms, ok := permissions["delete"].([]interface{})
		if !ok {
			return false
		}
		for _, perm := range deletePerms {
			if permStr, ok := perm.(string); ok {
				if permStr == "*" || permStr == collection {
					return true
				}
			}
		}

	case "manage_users":
		manageUsers, ok := permissions["manage_users"].(bool)
		return ok && manageUsers

	case "settings":
		settings, ok := permissions["settings"].(bool)
		return ok && settings
	}

	return false
}
