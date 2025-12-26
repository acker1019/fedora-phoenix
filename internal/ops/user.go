package ops

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/acker1019/fedora-phoenix/internal/logging"
	"github.com/acker1019/fedora-phoenix/internal/utils"
)

var userLog = logging.WithSource("ops/user")

// EnsureUserShell changes the user's default shell if it doesn't match.
// Idempotent: checks /etc/passwd before executing usermod.
func EnsureUserShell(username, targetShell string) error {
	userLog.Infof("Checking shell for user: %s", username)

	// Read /etc/passwd to get current shell
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return fmt.Errorf("failed to open /etc/passwd: %w", err)
	}
	defer file.Close()

	var currentShell string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, username+":") {
			fields := strings.Split(line, ":")
			if len(fields) >= 7 {
				currentShell = fields[6]
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read /etc/passwd: %w", err)
	}

	// Check if shell already matches
	if currentShell == targetShell {
		userLog.Infof("User %s already has shell %s. Skipping.", username, targetShell)
		return nil
	}

	userLog.Infof("Changing shell for %s: %s -> %s", username, currentShell, targetShell)

	// Execute usermod
	cmd := exec.Command("usermod", "-s", targetShell, username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to change shell for %s: %w", username, err)
	}

	userLog.Infof("Shell changed successfully for user %s", username)
	return nil
}

// EnsureSymlink creates a symlink from src to dest as the specified user.
// Idempotent: checks if symlink already exists and points to correct target.
func EnsureSymlink(src, dest, username string) error {
	userLog.Infof("Ensuring symlink: %s -> %s (as %s)", dest, src, username)

	// Check if dest exists and is correct
	if target, err := os.Readlink(dest); err == nil {
		if target == src {
			userLog.Infof("Symlink already correct. Skipping.")
			return nil
		}
	}

	// Create/update symlink using RunCommandAsUser
	if err := utils.RunCommandAsUser(username, "ln", "-sfn", src, dest); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	userLog.Info("Symlink created successfully")
	return nil
}

// ExtractTarball extracts a tarball to the destination directory as the specified user.
// Follows Check-Diff-Act pattern: checks if destination is non-empty before extracting.
func ExtractTarball(archivePath, destDir, username string) error {
	userLog.Infof("Checking tarball extraction: %s -> %s (as %s)", archivePath, destDir, username)

	// Ensure destination directory exists
	if err := utils.RunCommandAsUser(username, "mkdir", "-p", destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Check: Is destination directory non-empty?
	entries, err := os.ReadDir(destDir)
	if err == nil && len(entries) > 0 {
		userLog.Infof("Destination %s is non-empty (contains %d items). Skipping extraction.", destDir, len(entries))
		return nil
	}

	// Act: Extract tarball
	userLog.Infof("Extracting %s to %s", archivePath, destDir)
	if err := utils.RunCommandAsUser(username, "tar", "-xzf", archivePath, "-C", destDir); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}

	userLog.Info("Tarball extracted successfully")
	return nil
}

// RunStow deploys dotfiles using GNU Stow as the specified user.
// Idempotent: stow -R (restow) is inherently idempotent - it will recreate
// correct symlinks even if they already exist, and fix broken ones.
func RunStow(sourceDir, targetDir string, packages []string, username string) error {
	if len(packages) == 0 {
		return nil
	}

	userLog.Infof("Running Stow to deploy %d packages...", len(packages))

	for _, pkg := range packages {
		userLog.Infof("Deploying package: %s", pkg)

		// Act: Execute stow -R (restow)
		// The -R flag ensures idempotency by recreating all symlinks
		if err := utils.RunCommandAsUser(username, "stow", "-d", sourceDir, "-t", targetDir, "-R", pkg); err != nil {
			return fmt.Errorf("failed to deploy package %s: %w", pkg, err)
		}
	}

	userLog.Info("Stow deployment completed successfully")
	return nil
}

// GitClone clones a git repository to the destination as the specified user.
// Idempotent: checks if destination already exists.
func GitClone(url, dest, username string) error {
	userLog.Infof("Cloning %s to %s (as %s)", url, dest, username)

	// Check if destination exists
	if _, err := os.Stat(dest); err == nil {
		userLog.Infof("Destination %s already exists. Skipping clone.", dest)
		return nil
	}

	// Clone repository
	if err := utils.RunCommandAsUser(username, "git", "clone", url, dest); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	userLog.Info("Repository cloned successfully")
	return nil
}
