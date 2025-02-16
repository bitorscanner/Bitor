package templates

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
)

type Template struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Path        string `json:"path"`
}

type CategoryInfo struct {
	Templates []Template `json:"templates"`
	Total     int        `json:"total"`
}

// HandleListTemplates returns a list of all available Nuclei templates with pagination
func HandleListTemplates() echo.HandlerFunc {
	return func(c echo.Context) error {
		templatesDir := "nuclei-templates"
		searchQuery := strings.ToLower(c.QueryParam("search"))
		limit := 10 // Default limit per category
		if limitParam := c.QueryParam("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
				limit = parsedLimit
			}
		}

		// Categories based on template types
		categories := map[string]string{
			"dns":       "DNS",
			"file":      "File",
			"headless":  "Headless",
			"http":      "HTTP",
			"network":   "Network",
			"ssl":       "SSL/TLS",
			"workflows": "Workflows",
		}

		// Use maps to store templates and totals by category
		officialTemplatesByCategory := make(map[string]CategoryInfo)
		customTemplates := CategoryInfo{
			Templates: make([]Template, 0),
			Total:     0,
		}

		// Initialize category info
		for _, category := range categories {
			officialTemplatesByCategory[category] = CategoryInfo{
				Templates: make([]Template, 0),
				Total:     0,
			}
		}

		// First pass: count totals
		err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
				content, err := ioutil.ReadFile(path)
				if err != nil {
					return nil
				}

				description := extractDescription(string(content))
				relPath, _ := filepath.Rel(templatesDir, path)
				pathParts := strings.Split(relPath, string(os.PathSeparator))

				template := Template{
					Name:        info.Name(),
					Description: description,
					Path:        path,
				}

				// If search query exists, check if template matches
				if searchQuery != "" {
					nameMatch := strings.Contains(strings.ToLower(template.Name), searchQuery)
					descMatch := strings.Contains(strings.ToLower(template.Description), searchQuery)
					if !nameMatch && !descMatch {
						return nil
					}
				}

				isCustom := strings.HasPrefix(relPath, "custom/")
				if isCustom {
					customTemplates.Total++
				} else if len(pathParts) > 2 && pathParts[0] == "public" {
					categoryKey := pathParts[1]
					if category, ok := categories[categoryKey]; ok {
						info := officialTemplatesByCategory[category]
						info.Total++
						officialTemplatesByCategory[category] = info
					}
				}
			}
			return nil
		})

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to count templates",
			})
		}

		// Second pass: collect paginated templates
		err = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
				content, err := ioutil.ReadFile(path)
				if err != nil {
					return nil
				}

				description := extractDescription(string(content))
				relPath, _ := filepath.Rel(templatesDir, path)
				pathParts := strings.Split(relPath, string(os.PathSeparator))

				template := Template{
					Name:        info.Name(),
					Description: description,
					Path:        path,
				}

				// If search query exists, check if template matches
				if searchQuery != "" {
					nameMatch := strings.Contains(strings.ToLower(template.Name), searchQuery)
					descMatch := strings.Contains(strings.ToLower(template.Description), searchQuery)
					if !nameMatch && !descMatch {
						return nil
					}
				}

				isCustom := strings.HasPrefix(relPath, "custom/")
				if isCustom {
					if len(customTemplates.Templates) < limit {
						customTemplates.Templates = append(customTemplates.Templates, template)
					}
				} else if len(pathParts) > 2 && pathParts[0] == "public" {
					categoryKey := pathParts[1]
					if category, ok := categories[categoryKey]; ok {
						info := officialTemplatesByCategory[category]
						if len(info.Templates) < limit {
							info.Templates = append(info.Templates, template)
							officialTemplatesByCategory[category] = info
						}
					}
				}
			}
			return nil
		})

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to read templates",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"official": officialTemplatesByCategory,
			"custom":   customTemplates,
		})
	}
}

// extractDescription attempts to extract the description from a template file
func extractDescription(content string) string {
	// Look for the description field in the template
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "description:") {
			description := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			// Remove quotes if present
			description = strings.Trim(description, `"'`)
			return description
		}
	}
	return ""
}
