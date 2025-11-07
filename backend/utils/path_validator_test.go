package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSecurePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path-validator-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		baseDir     string
		userPath    string
		shouldError bool
		description string
	}{
		{
			name:        "Valid relative path",
			baseDir:     tempDir,
			userPath:    "templates/test.yaml",
			shouldError: false,
			description: "Normal relative path should be allowed",
		},
		{
			name:        "Valid single file",
			baseDir:     tempDir,
			userPath:    "test.yaml",
			shouldError: false,
			description: "Single filename should be allowed",
		},
		{
			name:        "Valid current directory",
			baseDir:     tempDir,
			userPath:    ".",
			shouldError: false,
			description: "Current directory reference should be allowed",
		},
		{
			name:        "Directory traversal with ../",
			baseDir:     tempDir,
			userPath:    "../../etc/passwd",
			shouldError: true,
			description: "Path traversal should be blocked",
		},
		{
			name:        "Directory traversal to root",
			baseDir:     tempDir,
			userPath:    "../../../../../../../../etc/passwd",
			shouldError: true,
			description: "Deep path traversal should be blocked",
		},
		{
			name:        "Absolute path attempt",
			baseDir:     tempDir,
			userPath:    "/etc/passwd",
			shouldError: true,
			description: "Absolute paths should be rejected",
		},
		{
			name:        "Windows-style path traversal",
			baseDir:     tempDir,
			userPath:    "..\\..\\windows\\system32\\config\\sam",
			shouldError: true,
			description: "Windows-style traversal should be blocked",
		},
		{
			name:        "Mixed separators",
			baseDir:     tempDir,
			userPath:    "..//.//../etc/passwd",
			shouldError: true,
			description: "Mixed separator traversal should be blocked",
		},
		{
			name:        "URL encoded traversal",
			baseDir:     tempDir,
			userPath:    "..%2F..%2Fetc%2Fpasswd",
			shouldError: true, // The path contains special characters and may traverse
			description: "URL encoded paths with traversal patterns should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateSecurePath(tt.baseDir, tt.userPath)

			if tt.shouldError {
				if err == nil {
					t.Errorf("%s: Expected error but got none. Result: %s", tt.description, result)
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.description, err)
				} else {
					// Verify the result is within the base directory
					absBaseDir, _ := filepath.Abs(tt.baseDir)
					if !strings.HasPrefix(result, absBaseDir) {
						t.Errorf("%s: Result path '%s' is not within base dir '%s'", tt.description, result, absBaseDir)
					}
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expected    string
		shouldError bool
		description string
	}{
		{
			name:        "Valid simple filename",
			filename:    "test.yaml",
			expected:    "test.yaml",
			shouldError: false,
			description: "Simple filename should pass",
		},
		{
			name:        "Valid filename with extension",
			filename:    "my-template.yml",
			expected:    "my-template.yml",
			shouldError: false,
			description: "Filename with dashes and extension should pass",
		},
		{
			name:        "Filename with directory separator",
			filename:    "subdir/test.yaml",
			expected:    "",
			shouldError: true,
			description: "Filename with forward slash should be rejected",
		},
		{
			name:        "Filename with Windows separator",
			filename:    "subdir\\test.yaml",
			expected:    "",
			shouldError: true,
			description: "Filename with backslash should be rejected",
		},
		{
			name:        "Filename with traversal",
			filename:    "../test.yaml",
			expected:    "",
			shouldError: true,
			description: "Filename with .. should be rejected",
		},
		{
			name:        "Filename with multiple traversals",
			filename:    "../../etc/passwd",
			expected:    "",
			shouldError: true,
			description: "Filename with multiple .. should be rejected",
		},
		{
			name:        "Just dot",
			filename:    ".",
			expected:    "",
			shouldError: true,
			description: "Single dot should be rejected",
		},
		{
			name:        "Just double dot",
			filename:    "..",
			expected:    "",
			shouldError: true,
			description: "Double dot should be rejected",
		},
		{
			name:        "Empty filename",
			filename:    "",
			expected:    "",
			shouldError: true,
			description: "Empty filename should be rejected",
		},
		{
			name:        "Hidden file",
			filename:    ".hidden",
			expected:    ".hidden",
			shouldError: false,
			description: "Hidden files (starting with .) should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeFilename(tt.filename)

			if tt.shouldError {
				if err == nil {
					t.Errorf("%s: Expected error but got none. Result: %s", tt.description, result)
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.description, err)
				}
				if result != tt.expected {
					t.Errorf("%s: Expected '%s' but got '%s'", tt.description, tt.expected, result)
				}
			}
		})
	}
}

func TestValidateSecurePathWithFilename(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path-validator-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "templates")
	os.MkdirAll(subDir, 0755)

	tests := []struct {
		name        string
		baseDir     string
		dirPath     string
		filename    string
		shouldError bool
		description string
	}{
		{
			name:        "Valid directory and filename",
			baseDir:     tempDir,
			dirPath:     "templates",
			filename:    "test.yaml",
			shouldError: false,
			description: "Valid directory and filename should be allowed",
		},
		{
			name:        "Root directory with valid filename",
			baseDir:     tempDir,
			dirPath:     ".",
			filename:    "test.yaml",
			shouldError: false,
			description: "Root directory with valid filename should be allowed",
		},
		{
			name:        "Valid directory with traversal in filename",
			baseDir:     tempDir,
			dirPath:     "templates",
			filename:    "../evil.yaml",
			shouldError: true,
			description: "Traversal in filename should be rejected",
		},
		{
			name:        "Traversal in directory",
			baseDir:     tempDir,
			dirPath:     "../../etc",
			filename:    "passwd",
			shouldError: true,
			description: "Traversal in directory should be rejected",
		},
		{
			name:        "Directory separator in filename",
			baseDir:     tempDir,
			dirPath:     "templates",
			filename:    "subdir/evil.yaml",
			shouldError: true,
			description: "Directory separator in filename should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateSecurePathWithFilename(tt.baseDir, tt.dirPath, tt.filename)

			if tt.shouldError {
				if err == nil {
					t.Errorf("%s: Expected error but got none. Result: %s", tt.description, result)
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.description, err)
				} else {
					// Verify the result is within the base directory
					absBaseDir, _ := filepath.Abs(tt.baseDir)
					if !strings.HasPrefix(result, absBaseDir) {
						t.Errorf("%s: Result path '%s' is not within base dir '%s'", tt.description, result, absBaseDir)
					}
				}
			}
		})
	}
}

// Benchmark tests to ensure validation doesn't significantly impact performance
func BenchmarkValidateSecurePath(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "benchmark-*")
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateSecurePath(tempDir, "templates/test.yaml")
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SanitizeFilename("test-template.yaml")
	}
}

