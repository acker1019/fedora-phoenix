package ops

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/acker1019/fedora-phoenix/internal/logging"
)

var pkgLog = logging.WithSource("ops/pkg")

// EnsurePackages is the idempotent function to install packages.
// It filters out already installed packages using rpm -q for speed.
func EnsurePackages(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}

	pkgLog.Infof("Checking status for %d packages...", len(pkgs))

	// Filter out already installed packages
	// Use rpm -q to check each package individually for idempotency
	var missingPkgs []string

	for _, pkg := range pkgs {
		// rpm -q returns exit code 0 if installed, non-zero if not
		cmd := exec.Command("rpm", "-q", pkg)
		if err := cmd.Run(); err != nil {
			missingPkgs = append(missingPkgs, pkg)
		}
	}

	if len(missingPkgs) == 0 {
		pkgLog.Info("All packages are already installed")
		return nil
	}

	pkgLog.Infof("Found %d missing packages: %v", len(missingPkgs), missingPkgs)

	// Construct DNF command
	// -y: assume yes
	// --refresh: force metadata update
	args := append([]string{"install", "-y", "--refresh"}, missingPkgs...)

	cmd := exec.Command("dnf", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	pkgLog.Info("Starting DNF transaction...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dnf install failed: %w", err)
	}

	pkgLog.Info("Packages installed successfully")
	return nil
}

// EnsurePinnedPackages installs and locks specific package versions.
// Follows Check-Diff-Act pattern for idempotency.
func EnsurePinnedPackages(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}

	pkgLog.Infof("Processing %d pinned packages...", len(pkgs))

	// Ensure versionlock plugin is installed
	pkgLog.Info("Ensuring dnf-plugin-versionlock is installed...")
	if err := EnsurePackages([]string{"python3-dnf-plugin-versionlock"}); err != nil {
		return fmt.Errorf("failed to install versionlock plugin: %w", err)
	}

	// Process each pinned package
	for _, pkg := range pkgs {
		pkgLog.Infof("Checking pinned package: %s", pkg)

		// Check: Is package already installed?
		checkCmd := exec.Command("rpm", "-q", pkg)
		isInstalled := checkCmd.Run() == nil

		// Check: Is package already locked?
		listCmd := exec.Command("dnf", "versionlock", "list")
		output, err := listCmd.Output()
		isLocked := err == nil && strings.Contains(string(output), pkg)

		// Diff: If both installed and locked, skip
		if isInstalled && isLocked {
			pkgLog.Infof("Package %s already installed and locked. Skipping.", pkg)
			continue
		}

		// Act: Install if needed
		if !isInstalled {
			pkgLog.Infof("Installing pinned package: %s", pkg)
			cmd := exec.Command("dnf", "install", "-y", pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install pinned package %s: %w", pkg, err)
			}
		}

		// Act: Lock if needed
		if !isLocked {
			pkgLog.Infof("Locking package version: %s", pkg)
			cmd := exec.Command("dnf", "versionlock", "add", pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to lock version for %s: %w", pkg, err)
			}
		}
	}

	pkgLog.Info("All pinned packages verified")
	return nil
}
