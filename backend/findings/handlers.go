package findings

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"

	"github.com/pocketbase/dbx"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/models"
)

// GroupedFindings represents the grouped findings.
type GroupedFindings struct {
	TemplateID    string                   `db:"template_id" json:"template_id"`
	SeverityOrder int                      `db:"severity_order" json:"severity_order"`
	Count         int                      `db:"count" json:"count"`
	Findings      []map[string]interface{} `json:"findings"`
}

// Utility function to calculate total pages
func totalPages(totalItems, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	return (totalItems + perPage - 1) / perPage
}

// HandleGroupedFindings handles the /api/findings/grouped endpoint.
func HandleGroupedFindings(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Parse query parameters for pagination
		pageParam := c.QueryParam("page")
		perPageParam := c.QueryParam("perPage")

		// Default values
		page := 1
		perPage := 10

		// Parse page parameter
		if pageParam != "" {
			if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
				page = p
			}
		}

		// Parse perPage parameter
		if perPageParam != "" {
			if pp, err := strconv.Atoi(perPageParam); err == nil && pp > 0 {
				perPage = pp
			}
		}

		// Get sorting parameters
		sortField := c.QueryParam("sortField")
		sortDirection := c.QueryParam("sortDirection")
		// Default sorting
		if sortField == "" {
			sortField = "severity_order"
		}
		if sortDirection == "" {
			sortDirection = "asc"
		}

		// Get filter parameters as slices
		severityFilters := c.QueryParams()["severity"]
		clientFilters := c.QueryParams()["client"]

		// Get search parameters
		searchTerm := c.QueryParam("search")
		searchField := c.QueryParam("searchField")

		// Get status filters
		statusFilters := c.QueryParams()["status"]

		// Initialize conditions as a slice of dbx.Expression
		var conditions []dbx.Expression

		// Build the query conditions
		if len(severityFilters) > 0 {
			conditions = append(conditions, dbx.In("severity", stringSliceToInterfaceSlice(severityFilters)...))
		}

		if len(clientFilters) > 0 {
			conditions = append(conditions, dbx.In("client", stringSliceToInterfaceSlice(clientFilters)...))
		}

		if searchTerm != "" && searchField != "" {
			switch searchField {
			case "template_id":
				conditions = append(conditions, dbx.NewExp("LOWER(template_id) LIKE LOWER({:pattern})", dbx.Params{"pattern": "%" + searchTerm + "%"}))
			case "name":
				conditions = append(conditions, dbx.NewExp("LOWER(json_extract(info, '$.name')) LIKE LOWER({:pattern})", dbx.Params{"pattern": "%" + searchTerm + "%"}))
			case "host":
				conditions = append(conditions, dbx.NewExp("LOWER(host) LIKE LOWER({:pattern})", dbx.Params{"pattern": "%" + searchTerm + "%"}))
			case "ip":
				conditions = append(conditions, dbx.NewExp("LOWER(ip) LIKE LOWER({:pattern})", dbx.Params{"pattern": "%" + searchTerm + "%"}))
			default:
				conditions = append(conditions, dbx.Like(searchField, "%"+searchTerm+"%"))
			}
		}

		// Add conditions for status filters
		if len(statusFilters) > 0 {
			var statusConditions []dbx.Expression

			for _, status := range statusFilters {
				switch status {
				case "acknowledged":
					statusConditions = append(statusConditions, dbx.HashExp{"acknowledged": true})
				case "false_positive":
					statusConditions = append(statusConditions, dbx.HashExp{"false_positive": true})
				case "remediated":
					statusConditions = append(statusConditions, dbx.HashExp{"remediated": true})
				case "no_status":
					statusConditions = append(statusConditions, dbx.And(
						dbx.HashExp{"acknowledged": false},
						dbx.HashExp{"false_positive": false},
						dbx.HashExp{"remediated": false},
					))
				}
			}

			if len(statusConditions) > 0 {
				conditions = append(conditions, dbx.Or(statusConditions...))
			}
		}

		// Combine all conditions
		var whereCond dbx.Expression
		if len(conditions) > 0 {
			whereCond = dbx.And(conditions...)
		}

		// Build the base query for counting total groups
		countQuery := app.DB().
			Select("COUNT(DISTINCT template_id) as total").
			From("nuclei_results")

		if whereCond != nil {
			countQuery.Where(whereCond)
		}

		// Get total number of groups
		var totalGroups int
		err := countQuery.Row(&totalGroups)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Failed to get total groups",
				"details": err.Error(),
			})
		}

		// Build the main query for fetching findings
		query := app.DB().
			Select(
				"template_id",
				"severity_order",
				"COUNT(*) as count",
				"GROUP_CONCAT(id) as finding_ids",
			).
			From("nuclei_results")

		if whereCond != nil {
			query.Where(whereCond)
		}

		query.GroupBy("template_id", "severity_order")

		// Apply sorting
		if sortField == "severity_order" {
			if sortDirection == "desc" {
				query.OrderBy("severity_order DESC")
			} else {
				query.OrderBy("severity_order ASC")
			}
		} else if sortField == "template_id" {
			if sortDirection == "desc" {
				query.OrderBy("template_id DESC")
			} else {
				query.OrderBy("template_id ASC")
			}
		}

		// Apply pagination
		query.Limit(int64(perPage)).
			Offset(int64((page - 1) * perPage))

		// Execute the query
		var results []struct {
			TemplateID    string `db:"template_id"`
			SeverityOrder int    `db:"severity_order"`
			Count         int    `db:"count"`
			FindingIDs    string `db:"finding_ids"`
		}

		if err := query.All(&results); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Failed to fetch findings",
				"details": err.Error(),
			})
		}

		// Process the results
		var groups []GroupedFindings
		for _, result := range results {
			// Split the finding IDs
			findingIDs := strings.Split(result.FindingIDs, ",")

			// Fetch the findings for this group
			records, err := app.Dao().FindRecordsByExpr("nuclei_results", dbx.In("id", stringSliceToInterfaceSlice(findingIDs)...))
			if err != nil {
				continue
			}

			var findings []map[string]interface{}
			for _, record := range records {
				// Fetch client data if available
				clientID := record.GetString("client")
				var clientData map[string]interface{}
				if clientID != "" {
					if clientRecord, err := app.Dao().FindRecordById("clients", clientID); err == nil {
						clientData = map[string]interface{}{
							"id":   clientRecord.Id,
							"name": clientRecord.GetString("name"),
						}
					}
				}

				// Convert record to map
				finding := recordToMap(record)
				finding["id"] = record.Id
				finding["client"] = clientData

				findings = append(findings, finding)
			}

			groups = append(groups, GroupedFindings{
				TemplateID:    result.TemplateID,
				SeverityOrder: result.SeverityOrder,
				Count:         result.Count,
				Findings:      findings,
			})
		}

		// Prepare the response
		response := map[string]interface{}{
			"page":       page,
			"perPage":    perPage,
			"totalPages": totalPages(totalGroups, perPage),
			"totalItems": totalGroups,
			"items":      groups,
		}

		return c.JSON(http.StatusOK, response)
	}
}

// Assuming `record` is of type *models.Record
func recordToMap(record *models.Record) map[string]interface{} {
	result := make(map[string]interface{})
	for _, field := range record.Collection().Schema.Fields() {
		result[field.Name] = record.Get(field.Name)
	}
	return result
}

// HandleFindings handles the /api/findings/findings endpoint.
func HandleFindings(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Implement your logic here
		return c.JSON(http.StatusOK, map[string]string{
			"message": "HandleFindings endpoint",
		})
	}
}

func HandleBulkUpdateFindings(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		var payload struct {
			IDs        []string               `json:"ids"`
			UpdateData map[string]interface{} `json:"updateData"`
		}

		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":   "Invalid request payload",
				"details": err.Error(),
			})
		}

		if len(payload.IDs) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "No IDs provided for bulk update",
			})
		}

		// Validate update data fields
		validFields := []string{"acknowledged", "false_positive", "remediated"}
		for field := range payload.UpdateData {
			if !contains(validFields, field) {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": fmt.Sprintf("Invalid field: %s", field),
				})
			}
		}

		// Update records in batch
		for _, id := range payload.IDs {
			record, err := app.Dao().FindRecordById("nuclei_results", id)
			if err != nil {
				continue // Skip if not found
			}

			for field, value := range payload.UpdateData {
				record.Set(field, value)
			}

			if err := app.Dao().Save(record); err != nil {
				continue // Skip if update fails
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Findings updated successfully",
		})
	}
}

// Helper function to check if a slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// Helper function to convert []string to []interface{}
func stringSliceToInterfaceSlice(strSlice []string) []interface{} {
	ifaces := make([]interface{}, len(strSlice))
	for i, v := range strSlice {
		ifaces[i] = v
	}
	return ifaces
}

// Add a new handler function
func HandleVulnerabilitiesByClient(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get filter parameters if needed
		acknowledgedFilter := c.QueryParam("acknowledged")
		falsePositiveFilter := c.QueryParam("false_positive")

		// Build conditions
		conditions := dbx.HashExp{}

		if acknowledgedFilter != "" {
			acknowledged, _ := strconv.ParseBool(acknowledgedFilter)
			conditions["acknowledged"] = acknowledged
		}
		if falsePositiveFilter != "" {
			falsePositive, _ := strconv.ParseBool(falsePositiveFilter)
			conditions["false_positive"] = falsePositive
		}

		// Exclude 'info' severity from the query
		severityCondition := dbx.NotIn(
			"LOWER(COALESCE(NULLIF(TRIM(severity), ''), 'unknown'))",
			"info",
		)

		// Ensure 'client' is not NULL
		clientCondition := dbx.NewExp("client IS NOT NULL")

		// Combine all conditions into a single expression
		allConditions := dbx.And(
			clientCondition,
			severityCondition,
		)

		// If there are additional conditions, include them
		if len(conditions) > 0 {
			allConditions = dbx.And(allConditions, conditions)
		}

		// Query counts per client per severity
		query := app.DB().
			Select(
				"client",
				"LOWER(COALESCE(NULLIF(TRIM(severity), ''), 'unknown')) as severity",
				"COUNT(*) as count",
			).
			From("nuclei_results").
			Where(allConditions).
			GroupBy("client", "severity")

		var results []struct {
			ClientID string `db:"client"`
			Severity string `db:"severity"`
			Count    int    `db:"count"`
		}

		if err := query.All(&results); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Process results to build data per client
		clientDataMap := make(map[string]*VulnerabilityItem)
		for _, result := range results {
			// Fetch or initialize the client data
			item, exists := clientDataMap[result.ClientID]
			if !exists {
				clientRecord, err := app.Dao().FindRecordById("clients", result.ClientID)
				clientName := "Unknown Client"
				if err == nil {
					clientName = clientRecord.GetString("name")
				}
				item = &VulnerabilityItem{
					ClientID:   result.ClientID,
					ClientName: clientName,
					Critical:   0,
					High:       0,
					Medium:     0,
					Low:        0,
					Unknown:    0,
					Total:      0,
				}
				clientDataMap[result.ClientID] = item
			}
			// Update counts based on severity
			switch result.Severity {
			case "critical":
				item.Critical += result.Count
			case "high":
				item.High += result.Count
			case "medium":
				item.Medium += result.Count
			case "low":
				item.Low += result.Count
			case "unknown":
				item.Unknown += result.Count
			default:
				// Handle any other severities as 'unknown'
				item.Unknown += result.Count
			}
			// Update total count (excluding 'info' since 'info' is excluded from the query)
			item.Total += result.Count
		}

		// Convert map to slice and sort by Total descending
		var data []VulnerabilityItem
		for _, item := range clientDataMap {
			data = append(data, *item)
		}

		// Sort clients by total vulnerabilities in descending order
		sort.Slice(data, func(i, j int) bool {
			return data[i].Total > data[j].Total
		})

		return c.JSON(http.StatusOK, data)
	}
}

// Define the VulnerabilityItem struct
type VulnerabilityItem struct {
	ClientID   string `json:"clientId"`
	ClientName string `json:"clientName"`
	Critical   int    `json:"critical"`
	High       int    `json:"high"`
	Medium     int    `json:"medium"`
	Low        int    `json:"low"`
	Unknown    int    `json:"unknown"`
	Total      int    `json:"total"`
}

// HandleRecentFindings handles the /api/findings/recent endpoint.
func HandleRecentFindings(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Calculate the date 30 days ago
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

		// Build the query to fetch findings from the last 30 days, excluding 'info' severity
		findingsQuery := app.DB().
			Select(
				"severity",
				"id",
				"json_extract(info, '$.name') as info_name",
				"host",
				"ip",
				"timestamp",
			).
			From("nuclei_results").
			Where(dbx.And(
				dbx.NewExp("timestamp >= {:thirtyDaysAgo}", dbx.Params{"thirtyDaysAgo": thirtyDaysAgo}),
				// Exclude 'info' severity (case-insensitive)
				dbx.Not(dbx.NewExp("LOWER(severity) = {:severity}", dbx.Params{"severity": "info"})),
			))

		// Fetch the findings
		var findings []struct {
			Severity  string `db:"severity"`
			ID        string `db:"id"`
			InfoName  string `db:"info_name"`
			Host      string `db:"host"`
			IP        string `db:"ip"`
			Timestamp string `db:"timestamp"`
		}

		err := findingsQuery.All(&findings)
		if err != nil {
			log.Printf("Error executing findings query: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Failed to get recent findings",
				"details": err.Error(),
			})
		}

		// Group findings by severity
		groupedFindings := make(map[string][]map[string]interface{})
		for _, finding := range findings {
			severity := strings.Title(strings.ToLower(finding.Severity))
			infoName := finding.InfoName

			groupedFindings[severity] = append(groupedFindings[severity], map[string]interface{}{
				"id": finding.ID,
				"info": map[string]string{
					"name": infoName,
				},
				"host":      finding.Host,
				"ip":        finding.IP,
				"timestamp": finding.Timestamp,
				"severity":  severity,
			})
		}

		// Convert the grouped findings to a slice for JSON response
		var result []map[string]interface{}
		for severity, findings := range groupedFindings {
			result = append(result, map[string]interface{}{
				"severity":       severity,
				"severity_order": getSeverityOrder(severity), // Include severity_order
				"findings":       findings,
			})
		}

		// Sort the result by severity order
		sort.Slice(result, func(i, j int) bool {
			return result[i]["severity_order"].(int) < result[j]["severity_order"].(int)
		})

		return c.JSON(http.StatusOK, result)
	}
}

// Helper function to map severity to an order
func getSeverityOrder(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	case "low":
		return 4
	case "info":
		return 5
	default:
		return 6 // Unknown severity
	}
}
