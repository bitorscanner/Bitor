package notifications

import (
	"fmt"
	"log"
	"net/http"
	"bitor/providers/jira"
	"bitor/services"
	"bitor/types"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

type JiraHandler struct {
	jiraService *services.JiraService
	app         *pocketbase.PocketBase
}

func NewJiraHandler(app *pocketbase.PocketBase, jiraService *services.JiraService) *JiraHandler {
	return &JiraHandler{
		jiraService: jiraService,
		app:         app,
	}
}

func (h *JiraHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/jira/projects", h.GetProjects)
	g.POST("/jira/issuetypes", h.GetIssueTypes)
	g.POST("/jira/organizations", h.GetOrganizations)
}

func (h *JiraHandler) GetProjects(c echo.Context) error {
	var req struct {
		URL string `json:"url"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	providerID := c.Get("provider_id").(string)
	provider, err := h.app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to find provider: %v", err),
		})
	}

	username, apiKey, err := jira.GetCredentials(h.app, provider)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get credentials: %v", err),
		})
	}

	projects, err := h.jiraService.GetProjects(types.JiraSettings{
		JiraURL:  req.URL,
		Username: username,
		APIKey:   apiKey,
	})
	if err != nil {
		log.Printf("Error fetching Jira projects: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to fetch projects: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"projects": projects,
	})
}

func (h *JiraHandler) GetIssueTypes(c echo.Context) error {
	var req struct {
		URL        string `json:"url"`
		ProjectKey string `json:"project_key"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	providerID := c.Get("provider_id").(string)
	provider, err := h.app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to find provider: %v", err),
		})
	}

	username, apiKey, err := jira.GetCredentials(h.app, provider)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get credentials: %v", err),
		})
	}

	issueTypes, err := h.jiraService.GetIssueTypes(types.JiraSettings{
		JiraURL:  req.URL,
		Username: username,
		APIKey:   apiKey,
	}, req.ProjectKey)
	if err != nil {
		log.Printf("Error fetching Jira issue types: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to fetch issue types: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"issueTypes": issueTypes,
	})
}

func (h *JiraHandler) GetOrganizations(c echo.Context) error {
	var req struct {
		URL        string `json:"url"`
		ProjectKey string `json:"project_key"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	providerID := c.Get("provider_id").(string)
	provider, err := h.app.Dao().FindRecordById("providers", providerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to find provider: %v", err),
		})
	}

	username, apiKey, err := jira.GetCredentials(h.app, provider)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get credentials: %v", err),
		})
	}

	organizations, err := h.jiraService.GetOrganizations(types.JiraSettings{
		JiraURL:  req.URL,
		Username: username,
		APIKey:   apiKey,
	}, req.ProjectKey)
	if err != nil {
		log.Printf("Error fetching Jira organizations: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to fetch organizations: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"organizations": organizations,
	})
}
