package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"bitor/services/notification"
)

type ScanHandler struct {
	pb                  *pocketbase.PocketBase
	notificationService *notification.NotificationService
}

func NewScanHandler(pb *pocketbase.PocketBase, notificationService *notification.NotificationService) *ScanHandler {
	return &ScanHandler{
		pb:                  pb,
		notificationService: notificationService,
	}
}

func (h *ScanHandler) StartScan(c echo.Context) error {
	var req struct {
		ScanID string `json:"scan_id"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Get scan details
	record, err := h.pb.Dao().FindRecordById("nuclei_scans", req.ScanID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Scan not found"})
	}

	// Update scan status
	record.Set("status", "Started")

	if err := h.pb.Dao().SaveRecord(record); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update scan"})
	}

	// Send notification
	if err := h.notificationService.NotifyScanStarted(context.Background(), req.ScanID, record.GetString("name")); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to send notification: %v\n", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *ScanHandler) StopScan(c echo.Context) error {
	var req struct {
		ScanID string `json:"scan_id"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Get scan details
	record, err := h.pb.Dao().FindRecordById("nuclei_scans", req.ScanID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Scan not found"})
	}

	// Update scan status
	record.Set("status", "Stopped")

	if err := h.pb.Dao().SaveRecord(record); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update scan"})
	}

	// Send notification
	if err := h.notificationService.NotifyScanFailed(context.Background(), req.ScanID, record.GetString("name"), "Scan stopped by user"); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to send notification: %v\n", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *ScanHandler) ImportManualScan(c echo.Context) error {
	var req struct {
		Name     string          `json:"name"`
		ClientID string          `json:"client_id"`
		Results  json.RawMessage `json:"results"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Get the collection
	collection, err := h.pb.Dao().FindCollectionByNameOrId("nuclei_scans")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to find collection"})
	}

	// Create manual scan record
	record := models.NewRecord(collection)
	record.Set("name", req.Name)
	record.Set("client", req.ClientID)
	record.Set("status", "Manual")
	record.Set("results", req.Results)

	if err := h.pb.Dao().SaveRecord(record); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create scan"})
	}

	return c.JSON(http.StatusOK, record)
}
