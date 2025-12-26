package ops

import (
	"fmt"
	"os"
	"os/exec"
)

// EnsurePackages is the idempotent function to install packages.
// It filters out already installed packages to speed up execution.
func EnsurePackages(pkgs ...string) {
	if len(pkgs) == 0 {
		return
	}

	fmt.Printf("📦 [DNF] Checking status for %d packages...\n", len(pkgs))

	// Optimization: Instead of looping rpm -q (N processes),
	// use a loop here for simplicity and robustness.
	// Future optimization: batch query with `rpm -q pkg1 pkg2 ...`

	var missingPkgs []string

	for _, pkg := range pkgs {
		// rpm -q returns exit code 0 if installed, non-zero if not.
		cmd := exec.Command("rpm", "-q", pkg)
		if err := cmd.Run(); err != nil {
			missingPkgs = append(missingPkgs, pkg)
		}
	}

	if len(missingPkgs) == 0 {
		fmt.Println("✅ All packages are already installed.")
		return
	}

	fmt.Printf("⬇️ Found %d missing packages. Installing: %v\n", len(missingPkgs), missingPkgs)

	// Construct DNF command
	// -y: assume yes
	// --refresh: force metadata update
	args := append([]string{"install", "-y", "--refresh"}, missingPkgs...)

	// Execute via RunOrDie helper (assuming it's defined in common util)
	cmd := exec.Command("dnf", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("🚀 Starting DNF transaction...")
	if err := cmd.Run(); err != nil {
		// Fail fast if package installation fails
		panic(fmt.Sprintf("❌ DNF install failed: %v", err))
	}

	fmt.Println("✨ Packages installed successfully.")
}
