package profiles

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

// HandleAddOfficialProfiles handles the request to add official Nuclei profiles to the database
func HandleAddOfficialProfiles(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Create a new profile manager
		pm := NewProfileManager(".")

		// Add profiles to database
		if err := pm.AddProfilesToDatabase(app); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Official Nuclei profiles added successfully",
		})
	}
}
