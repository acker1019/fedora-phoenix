package utils

import (
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ to user's home directory.
// It uses /home/{username} convention without relying on environment variables.
func ExpandPath(path string, homeDir string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	if path == "~" {
		return homeDir
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}

	// Path like ~username is not supported, return as-is
	return path
}
