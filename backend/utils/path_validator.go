package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateSecurePath validates that a user-provided path stays within the allowed base directory.
// It prevents directory traversal attacks by:
// 1. Cleaning the path to remove redundant separators and resolve . and .. elements
// 2. Joining with the base directory
// 3. Resolving to absolute path
// 4. Verifying the final path is within the base directory
//
// Parameters:
//   - baseDir: The allowed base directory (must be absolute)
//   - userPath: The user-provided path (can be relative)
//
// Returns:
//   - The validated absolute path
//   - An error if the path escapes the base directory or is invalid
func ValidateSecurePath(baseDir, userPath string) (string, error) {
	// Ensure base directory is absolute
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("invalid base directory: %w", err)
	}

	// Clean the user path to normalize it
	cleanUserPath := filepath.Clean(userPath)

	// Prevent absolute paths from user input
	if filepath.IsAbs(cleanUserPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	// Join base directory with user path
	fullPath := filepath.Join(absBaseDir, cleanUserPath)

	// Resolve to absolute path (handles symlinks and relative components)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Ensure the resolved path is within the base directory
	// Use filepath.Rel to check if the path is within baseDir
	relPath, err := filepath.Rel(absBaseDir, absFullPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	// If the relative path starts with "..", it's outside the base directory
	if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected: path escapes base directory")
	}

	return absFullPath, nil
}

// SanitizeFilename removes directory components from a filename to prevent path traversal.
// This is useful for validating filenames that should not contain directory separators.
//
// Parameters:
//   - filename: The filename to sanitize
//
// Returns:
//   - The sanitized filename (base name only)
//   - An error if the filename contains directory separators or traversal sequences
func SanitizeFilename(filename string) (string, error) {
	// Check for empty filename
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	// Clean the filename
	cleaned := filepath.Clean(filename)

	// Check for directory traversal sequences
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("filename contains directory traversal sequence")
	}

	// Check for directory separators (both Unix and Windows)
	if strings.ContainsAny(cleaned, "/\\") {
		return "", fmt.Errorf("filename cannot contain directory separators")
	}

	// Get base name to remove any directory components
	baseName := filepath.Base(cleaned)

	// Additional check: ensure base name doesn't start with . (hidden files can be security risk)
	// But allow legitimate file extensions
	if baseName == "." || baseName == ".." {
		return "", fmt.Errorf("invalid filename")
	}

	return baseName, nil
}

// ValidateSecurePathWithFilename validates a directory path and ensures a filename is safe.
// This is useful for operations like rename where you need to validate both the directory
// and the new filename.
//
// Parameters:
//   - baseDir: The allowed base directory (must be absolute)
//   - dirPath: The directory path within baseDir
//   - filename: The filename to be used
//
// Returns:
//   - The validated absolute path (directory + filename)
//   - An error if validation fails
func ValidateSecurePathWithFilename(baseDir, dirPath, filename string) (string, error) {
	// Sanitize the filename first
	safeFilename, err := SanitizeFilename(filename)
	if err != nil {
		return "", fmt.Errorf("invalid filename: %w", err)
	}

	// Validate the directory path
	_, err = ValidateSecurePath(baseDir, dirPath)
	if err != nil {
		return "", fmt.Errorf("invalid directory path: %w", err)
	}

	// Final validation to ensure the complete path is still within baseDir
	finalPath, err := ValidateSecurePath(baseDir, filepath.Join(dirPath, safeFilename))
	if err != nil {
		return "", fmt.Errorf("final path validation failed: %w", err)
	}

	return finalPath, nil
}

