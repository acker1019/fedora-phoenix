package cmd

import (
	"fmt"
	"os"

	"github.com/acker1019/fedora-phoenix/internal/artifact"
	"github.com/acker1019/fedora-phoenix/internal/config"
	"github.com/acker1019/fedora-phoenix/internal/ops"
	"github.com/acker1019/fedora-phoenix/internal/session"
	"github.com/acker1019/fedora-phoenix/internal/utils"

	"github.com/spf13/cobra"
)

// provisionCmd represents the provision command
var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Start the full restoration protocol",
	Long:  `Unlock LUKS, mount data, install packages, and restore user space from an artifact.`,
	Run: func(cmd *cobra.Command, args []string) {
		runProvision()
	},
}

func init() {
	rootCmd.AddCommand(provisionCmd)
	// 如果 provision 有自己專屬的 flag，可以在這裡加
	// provisionCmd.Flags().BoolP("dry-run", "d", false, "Preview changes only")
}

func runProvision() {
	// 1. Validate Flags
	if secretsPath == "" {
		fmt.Println("❌ Error: --secrets flag is required.")
		fmt.Println("Usage: sudo phoenix provision --secrets=/path/to/secrets.yml")
		os.Exit(1)
	}

	// 2. Root Check
	if os.Geteuid() != 0 {
		fmt.Println("❌ Error: This command must be run as root (sudo).")
		os.Exit(1)
	}

	fmt.Println("🔥 Initiating Phoenix Protocol...")

	// ============================================================================
	// Initialize Session
	// ============================================================================
	sess := &session.Session{}

	// 3. Real User Detection (supports X11 & Wayland)
	realUser, realUID, realGID, err := utils.GetRealUser()
	if err != nil {
		panic(fmt.Sprintf("Failed to detect real user: %v", err))
	}
	sess.Username = realUser
	sess.UID = realUID
	sess.GID = realGID
	sess.UserHome, err = ops.EnsureUserHome(sess.Username, sess.UID, sess.GID)
	if err != nil {
		panic(fmt.Sprintf("Failed to ensure home directory: %v", err))
	}
	fmt.Printf("✓ Detected real user: %s (UID %d, GID %d) -> %s\n", sess.Username, sess.UID, sess.GID, sess.UserHome)

	// ============================================================================
	// Block I: Identity & Configuration
	// ============================================================================
	fmt.Println("🔑 Step 1/5: Loading configuration...")

	// Load Blueprint (phoenix.yml)
	sess.Blueprint, err = config.LoadBlueprint(blueprintPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load blueprint: %v", err))
	}

	// Load Secrets
	sess.Secrets, err = config.LoadSecrets(secretsPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load secrets: %v", err))
	}
	// Self-destruct logic
	config.CleanupSecrets(secretsPath)

	// Store artifact path
	sess.ArtifactPath = artifactPath

	// ============================================================================
	// Block II: Infrastructure
	// ============================================================================
	fmt.Println("🔧 Step 2/5: Setting up infrastructure...")

	// Store infrastructure info in session
	sess.LuksMapperName = sess.Blueprint.Infrastructure.Luks.MapperName
	sess.LuksMountPoint = sess.Blueprint.Infrastructure.Luks.MountPoint

	// LUKS Unlock
	err = ops.UnlockLuks(
		sess.Blueprint.Infrastructure.Luks.Device,
		sess.LuksMapperName,
		sess.Secrets.LuksPassword,
	)
	if err != nil {
		panic(err)
	}
	sess.LuksUnlocked = true

	// Mount Device
	if err := ops.MountDevice(
		sess.LuksMapperName,
		sess.LuksMountPoint,
	); err != nil {
		panic(err)
	}
	sess.LuksMounted = true

	// ============================================================================
	// Block III: System State
	// ============================================================================
	fmt.Println("📦 Step 3/5: Configuring system state...")

	// Ensure Groups (before Users, since users may reference them)
	if len(sess.Blueprint.System.Groups) > 0 {
		if err := ops.EnsureGroups(sess.Blueprint.System.Groups); err != nil {
			panic(err)
		}
	}

	// Ensure Users
	if len(sess.Blueprint.System.Users) > 0 {
		if err := ops.EnsureUsers(sess.Blueprint.System.Users); err != nil {
			panic(err)
		}
	}

	// Install Packages
	if len(sess.Blueprint.System.Packages) > 0 {
		if err := ops.EnsurePackages(sess.Blueprint.System.Packages); err != nil {
			panic(err)
		}
	}

	// Install Pinned Packages
	if len(sess.Blueprint.System.PinnedPackages) > 0 {
		if err := ops.EnsurePinnedPackages(sess.Blueprint.System.PinnedPackages); err != nil {
			panic(err)
		}
	}

	// Enable Services
	if len(sess.Blueprint.System.Services) > 0 {
		if err := ops.EnsureServices(sess.Blueprint.System.Services); err != nil {
			panic(err)
		}
	}

	// Set User Shell
	if sess.Blueprint.Identity.Shell != "" {
		if err := ops.EnsureUserShell(sess.Blueprint.Identity.Username, sess.Blueprint.Identity.Shell); err != nil {
			panic(err)
		}
	}

	// ============================================================================
	// Block IV: User Space
	// ============================================================================
	fmt.Println("👤 Step 4/5: Restoring user space...")

	// Restore Artifact (ADR-0008)
	if sess.ArtifactPath != "" {
		if err := artifact.Restore(sess.ArtifactPath); err != nil {
			panic(err)
		}
	}

	// Clone Git Repositories
	for _, repo := range sess.Blueprint.UserSpace.Repos {
		expandedDest := utils.ExpandPath(repo.Dest, sess.UserHome)
		if err := ops.GitClone(repo.URL, expandedDest, sess.Blueprint.Identity.Username); err != nil {
			panic(err)
		}
	}

	fmt.Println("✨ Phoenix Protocol Complete. Welcome back, Commander.")
}
