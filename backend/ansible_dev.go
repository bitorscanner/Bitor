//go:build !production

package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

// getAnsibleFS is a development-only function that provides access to the ansible files.
// It is used during development to serve ansible files directly from the filesystem
// instead of using embedded files like in production.
// Currently unused but retained for future development environment support.
// nolint:unused
func getAnsibleFS() fs.FS {
	ansiblePath := filepath.Join("..", "ansible")
	return os.DirFS(ansiblePath)
}
