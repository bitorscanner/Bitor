package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
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

			// Get group permissions
			permissions := group.Get("permissions").(map[string]interface{})

			// Check permission based on action
			switch action {
			case "read":
				readPerms := permissions["read"].([]interface{})
				hasPermission := false
				for _, perm := range readPerms {
					if perm.(string) == "*" || perm.(string) == collection {
						hasPermission = true
						break
					}
				}
				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "write":
				writePerms := permissions["write"].([]interface{})
				hasPermission := false
				for _, perm := range writePerms {
					if perm.(string) == "*" || perm.(string) == collection {
						hasPermission = true
						break
					}
				}
				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "delete":
				deletePerms := permissions["delete"].([]interface{})
				hasPermission := false
				for _, perm := range deletePerms {
					if perm.(string) == "*" || perm.(string) == collection {
						hasPermission = true
						break
					}
				}
				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "manage_users":
				if !permissions["manage_users"].(bool) {
					return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
				}

			case "settings":
				if !permissions["settings"].(bool) {
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

	// Get group permissions
	permissions := group.Get("permissions").(map[string]interface{})

	// Check permission based on action
	switch action {
	case "read":
		readPerms := permissions["read"].([]interface{})
		for _, perm := range readPerms {
			if perm.(string) == "*" || perm.(string) == collection {
				return true
			}
		}

	case "write":
		writePerms := permissions["write"].([]interface{})
		for _, perm := range writePerms {
			if perm.(string) == "*" || perm.(string) == collection {
				return true
			}
		}

	case "delete":
		deletePerms := permissions["delete"].([]interface{})
		for _, perm := range deletePerms {
			if perm.(string) == "*" || perm.(string) == collection {
				return true
			}
		}

	case "manage_users":
		return permissions["manage_users"].(bool)

	case "settings":
		return permissions["settings"].(bool)
	}

	return false
}
