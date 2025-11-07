package templates

import (
	"bitor/utils"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v5"
)

const (
	templatesBaseDir = "nuclei-templates"
	repoURL          = "https://github.com/projectdiscovery/nuclei-templates.git"
	publicTemplates  = "public"
	customTemplates  = "custom"
)

type FileItem struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"isDir"`
	IsCustom bool       `json:"isCustom"`
	Children []FileItem `json:"children,omitempty"`
}

// InitializeTemplates clones the repository if needed
func InitializeTemplates() error {
	publicDir := filepath.Join(templatesBaseDir, publicTemplates)
	customDir := filepath.Join(templatesBaseDir, customTemplates)

	// Check if the public directory exists
	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		// Directory doesn't exist, perform sparse checkout
		fmt.Println("Cloning public templates repository with sparse checkout...")

		// Create the base templates directory
		err := os.MkdirAll(templatesBaseDir, os.ModePerm)
		if err != nil {
			log.Println("Error creating templates base directory:", err)
			return err
		}

		// Prepare the git clone command with sparse checkout
		cloneArgs := []string{
			"clone",
			"--depth", "1",
			"--filter=blob:none",
			"--sparse",
			repoURL,
			publicDir,
		}

		cmd := exec.Command("git", cloneArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Println("Error cloning repository:", err)
			return err
		}

		// Set the sparse-checkout directories
		dirs := []string{
			"/cloud",
			"/code",
			"/dast",
			"/dns",
			"/file",
			"/headless",
			"/http",
			"/javascript",
			"/network",
			"/passive",
			"/ssl",
			"/workflows",
		}

		// Change directory to publicDir
		cmd.Dir = publicDir

		// Configure sparse-checkout to include only the specified directories
		sparseCheckoutArgs := append([]string{"sparse-checkout", "set", "--no-cone"}, dirs...)
		cmd = exec.Command("git", sparseCheckoutArgs...)
		cmd.Dir = publicDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Println("Error setting sparse-checkout directories:", err)
			return err
		}

		log.Println("Public templates repository cloned successfully with sparse checkout.")
	} else {
		log.Println("Public templates repository already exists.")
	}

	// Ensure custom templates directory exists
	if _, err := os.Stat(customDir); os.IsNotExist(err) {
		err := os.MkdirAll(customDir, os.ModePerm)
		if err != nil {
			log.Println("Error creating custom templates directory:", err)
			return err
		}
	}

	return nil
}

// StartTemplateUpdater schedules daily updates for the repository
func StartTemplateUpdater() {
	go func() {
		for {
			// Sleep until the next day
			now := time.Now()
			next := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)
			duration := next.Sub(now)
			time.Sleep(duration)

			// Pull the latest changes
			publicDir := filepath.Join(templatesBaseDir, publicTemplates)
			log.Println("Updating public templates repository...")

			// Run 'git sparse-checkout reapply' before pulling
			cmd := exec.Command("git", "sparse-checkout", "reapply")
			cmd.Dir = publicDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Println("Error reapplying sparse-checkout:", err)
				continue
			}

			// Run 'git pull'
			cmd = exec.Command("git", "pull", "--depth", "1")
			cmd.Dir = publicDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Println("Error updating repository:", err)
				continue
			}

			log.Println("Public templates repository updated successfully.")
		}
	}()
}

func ListTemplatesHandler(c echo.Context) error {
	dir := c.QueryParam("dir")
	if dir == "" {
		dir = "."
	}

	isCustom := c.QueryParam("custom") == "true"

	var baseDir string
	if isCustom {
		baseDir = filepath.Join(templatesBaseDir, customTemplates)
	} else {
		baseDir = filepath.Join(templatesBaseDir, publicTemplates)
	}

	// Validate path to prevent directory traversal
	fullPath, err := utils.ValidateSecurePath(baseDir, dir)
	if err != nil {
		log.Printf("Path validation failed for dir '%s': %v", dir, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid directory path",
		})
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	items := []FileItem{}

	for _, entry := range entries {
		item := FileItem{
			Name:     entry.Name(),
			Path:     filepath.Join(dir, entry.Name()),
			IsDir:    entry.IsDir(),
			IsCustom: isCustom,
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, items)
}

func GetTemplateContentHandler(c echo.Context) error {
	filePath := c.QueryParam("path")
	if filePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "path parameter is required",
		})
	}

	isCustom := c.QueryParam("custom") == "true"

	var baseDir string
	if isCustom {
		baseDir = filepath.Join(templatesBaseDir, customTemplates)
	} else {
		baseDir = filepath.Join(templatesBaseDir, publicTemplates)
	}

	// Validate path to prevent directory traversal
	fullPath, err := utils.ValidateSecurePath(baseDir, filePath)
	if err != nil {
		log.Printf("Path validation failed for file '%s': %v", filePath, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid file path",
		})
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.String(http.StatusOK, string(content))
}

func SaveTemplateContentHandler(c echo.Context) error {
	var request struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		IsCustom bool   `json:"isCustom"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Only allow modification of custom templates
	if !request.IsCustom {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Public templates cannot be modified",
		})
	}

	var baseDir string
	if request.IsCustom {
		baseDir = filepath.Join(templatesBaseDir, customTemplates)
	} else {
		baseDir = filepath.Join(templatesBaseDir, publicTemplates)
	}

	// Validate path to prevent directory traversal
	fullPath, err := utils.ValidateSecurePath(baseDir, request.Path)
	if err != nil {
		log.Printf("Path validation failed for file '%s': %v", request.Path, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid file path",
		})
	}

	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create directories",
		})
	}

	if err := os.WriteFile(fullPath, []byte(request.Content), 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save file",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "File saved successfully",
	})
}

// ListAllTemplatesHandler handles the request to get all templates recursively
func ListAllTemplatesHandler(c echo.Context) error {
	isCustom := c.QueryParam("custom") == "true"

	var baseDir string
	if isCustom {
		baseDir = filepath.Join(templatesBaseDir, customTemplates)
	} else {
		baseDir = filepath.Join(templatesBaseDir, publicTemplates)
	}

	// Validate that the base directory exists and is accessible
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Templates directory not found",
		})
	}

	// Start recursive traversal from baseDir
	items, err := getAllTemplates(baseDir, "", isCustom)
	if err != nil {
		log.Println("Error getting all templates:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, items)
}

// getAllTemplates recursively fetches all templates and builds a nested structure
func getAllTemplates(currentDir, relativePath string, isCustom bool) ([]FileItem, error) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, err
	}

	var items []FileItem

	for _, entry := range entries {
		itemPath := filepath.Join(currentDir, entry.Name())
		relPath := filepath.Join(relativePath, entry.Name())
		fileItem := FileItem{
			Name:     entry.Name(),
			Path:     relPath,
			IsDir:    entry.IsDir(),
			IsCustom: isCustom,
		}

		if entry.IsDir() {
			children, err := getAllTemplates(itemPath, relPath, isCustom)
			if err != nil {
				return nil, err
			}
			fileItem.Children = children
		}

		items = append(items, fileItem)
	}

	return items, nil
}

// RenameRequest represents the expected request body for renaming a template
type RenameRequest struct {
	OldPath  string `json:"oldPath"`
	NewName  string `json:"newName"`
	IsCustom bool   `json:"isCustom"`
}

// RenameTemplateHandler handles renaming templates (files or directories)
func RenameTemplateHandler(c echo.Context) error {
	// Parse the request body
	var req RenameRequest
	if err := c.Bind(&req); err != nil {
		log.Println("Error parsing rename request:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate inputs
	if req.OldPath == "" || req.NewName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Old path and new name are required"})
	}

	// Only allow modification of custom templates
	if !req.IsCustom {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Public templates cannot be modified",
		})
	}

	// Determine base directory
	var baseDir string
	if req.IsCustom {
		baseDir = filepath.Join(templatesBaseDir, customTemplates)
	} else {
		baseDir = filepath.Join(templatesBaseDir, publicTemplates)
	}

	// Validate the old path
	oldFullPath, err := utils.ValidateSecurePath(baseDir, req.OldPath)
	if err != nil {
		log.Printf("Path validation failed for old path '%s': %v", req.OldPath, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid old path"})
	}

	// Sanitize the new name to ensure it's just a filename without directory components
	safeNewName, err := utils.SanitizeFilename(req.NewName)
	if err != nil {
		log.Printf("Filename sanitization failed for new name '%s': %v", req.NewName, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid new name: must be a filename without directory separators"})
	}

	// Construct the new full path in the same directory as the old path
	parentDir := filepath.Dir(oldFullPath)
	newFullPath := filepath.Join(parentDir, safeNewName)

	// Validate that the new path is still within the base directory
	if _, err := utils.ValidateSecurePath(baseDir, filepath.Join(filepath.Dir(req.OldPath), safeNewName)); err != nil {
		log.Printf("Path validation failed for new path: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid new path"})
	}

	// Check if the old file/directory exists
	if _, err := os.Stat(oldFullPath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Old file or directory does not exist"})
	}

	// Check if the new name already exists in the same directory
	if _, err := os.Stat(newFullPath); err == nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "A file or directory with the new name already exists"})
	}

	// Perform the rename operation
	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		log.Println("Error renaming file or directory:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to rename file or directory"})
	}

	// Optionally, log the renaming action
	log.Printf("Renamed '%s' to '%s'\n", oldFullPath, newFullPath)

	// Return success response
	return c.JSON(http.StatusOK, map[string]string{"message": "File or directory renamed successfully"})
}

// DeleteRequest represents the expected request body for deleting a template
type DeleteRequest struct {
	Path     string `json:"path"`
	IsCustom bool   `json:"isCustom"`
}

// DeleteTemplateHandler handles deleting templates (files or directories)
func DeleteTemplateHandler(c echo.Context) error {
	// Parse the request body
	var req DeleteRequest
	if err := c.Bind(&req); err != nil {
		log.Println("Error parsing delete request:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate inputs
	if req.Path == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Path is required"})
	}

	// Only allow deletion of custom templates
	if !req.IsCustom {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Public templates cannot be deleted",
		})
	}

	// Determine base directory
	var baseDir string
	if req.IsCustom {
		baseDir = filepath.Join(templatesBaseDir, customTemplates)
	} else {
		baseDir = filepath.Join(templatesBaseDir, publicTemplates)
	}

	// Validate path to prevent directory traversal
	fullPath, err := utils.ValidateSecurePath(baseDir, req.Path)
	if err != nil {
		log.Printf("Path validation failed for path '%s': %v", req.Path, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid path"})
	}

	// Prevent deletion of root directories
	cleanPath := filepath.Clean(req.Path)
	if cleanPath == "." || cleanPath == "/" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot delete root directory"})
	}

	// Check if the file/directory exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "File or directory does not exist"})
	}

	// Remove the file or directory
	err = os.RemoveAll(fullPath)
	if err != nil {
		log.Println("Error deleting file or directory:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete file or directory"})
	}

	// Optionally, log the deletion action
	log.Printf("Deleted '%s'\n", fullPath)

	// Return success response
	return c.JSON(http.StatusOK, map[string]string{"message": "File or directory deleted successfully"})
}
