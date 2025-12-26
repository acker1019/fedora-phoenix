package ops

import (
	"fmt"
	"os"

	"github.com/acker1019/fedora-phoenix/internal/logging"
)

var homeLog = logging.WithSource("ops/home")

// EnsureUserHome ensures the user's home directory exists with correct permissions.
// Returns the home directory path.
// Follows Check-Diff-Act pattern for idempotency.
func EnsureUserHome(username string, uid, gid int) (string, error) {
	homeDir := fmt.Sprintf("/home/%s", username)
	homeLog.Infof("Ensuring home directory: %s", homeDir)

	// Check: Does home directory exist?
	info, err := os.Stat(homeDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Act: Create home directory
			homeLog.Infof("Creating home directory: %s", homeDir)
			if err := os.MkdirAll(homeDir, 0755); err != nil {
				return "", fmt.Errorf("failed to create home directory: %w", err)
			}

			// Set ownership
			if err := os.Chown(homeDir, uid, gid); err != nil {
				return "", fmt.Errorf("failed to set ownership: %w", err)
			}

			homeLog.Infof("Home directory created successfully")
			return homeDir, nil
		}
		return "", fmt.Errorf("failed to stat home directory: %w", err)
	}

	// Check: Is it a directory?
	if !info.IsDir() {
		return "", fmt.Errorf("%s exists but is not a directory", homeDir)
	}

	// Diff: Check permissions (should be at least 0755)
	mode := info.Mode().Perm()
	if mode&0755 != 0755 {
		// Act: Fix permissions
		homeLog.Warnf("Home directory has incorrect permissions: %o, fixing to 0755", mode)
		if err := os.Chmod(homeDir, 0755); err != nil {
			return "", fmt.Errorf("failed to fix permissions: %w", err)
		}
	}

	// Act: Verify ownership (always attempt to set correct ownership)
	if err := os.Chown(homeDir, uid, gid); err != nil {
		homeLog.Warnf("Failed to verify ownership, may already be correct: %v", err)
	}

	homeLog.Infof("Home directory verified: %s", homeDir)
	return homeDir, nil
}
