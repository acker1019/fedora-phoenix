package cmd

import (
	"fmt"
	"os"

	"github.com/acker1019/fedora-trisolaran/internal/artifact"
	"github.com/acker1019/fedora-trisolaran/internal/config"
	"github.com/acker1019/fedora-trisolaran/internal/ops"
	"github.com/acker1019/fedora-trisolaran/internal/session"
	"github.com/acker1019/fedora-trisolaran/internal/utils"

	"github.com/spf13/cobra"
)

// rehydraCmd represents the rehydra command
var rehydraCmd = &cobra.Command{
	Use:   "rehydra",
	Short: "Start the full rehydration protocol",
	Long:  `Unlock LUKS, mount data, install packages, and restore user space from an artifact.`,
	Run: func(cmd *cobra.Command, args []string) {
		runRehydra(cmd)
	},
}

func init() {
	rootCmd.AddCommand(rehydraCmd)
	// 如果 rehydra 有自己專屬的 flag，可以在這裡加
	// rehydraCmd.Flags().BoolP("dry-run", "d", false, "Preview changes only")
}

func runRehydra(cmd *cobra.Command) {
	// 1. Validate Flags
	if secretsPath == "" {
		fmt.Println("❌ Error: --secrets flag is required.")
		fmt.Println("Usage: sudo tri rehydra --secrets=/path/to/secrets.yml")
		os.Exit(1)
	}

	// 2. Root Check
	if os.Geteuid() != 0 {
		fmt.Println("❌ Error: This command must be run as root (sudo).")
		os.Exit(1)
	}

	fmt.Println("💧 Initiating Rehydration Protocol...")

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

	// Load Blueprint. If --blueprint was not explicitly passed but an
	// artifact was given, prefer the blueprint embedded in that artifact
	// (see artifact.ExtractBlueprint) over the --blueprint default, so a
	// single artifact tgz can be self-sufficient for rehydration.
	sess.Blueprint = nil
	if !cmd.Flags().Changed("blueprint") && artifactPath != "" {
		if data, embedErr := artifact.ExtractBlueprint(artifactPath); embedErr == nil {
			sess.Blueprint, err = config.ParseBlueprint(data)
			if err != nil {
				panic(fmt.Sprintf("Failed to parse blueprint embedded in artifact: %v", err))
			}
			fmt.Println("📦 Using blueprint embedded in artifact (no --blueprint given)")
		} else if embedErr != artifact.ErrBlueprintNotEmbedded {
			panic(fmt.Sprintf("Failed to read artifact: %v", embedErr))
		}
	}
	if sess.Blueprint == nil {
		sess.Blueprint, err = config.LoadBlueprint(blueprintPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to load blueprint: %v", err))
		}
	}

	// Load Secrets
	sess.Secrets, err = config.LoadSecrets(secretsPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load secrets: %v", err))
	}

	hasLuks := sess.Blueprint.HasLuks()
	if hasLuks && sess.Secrets.LuksPassword == "" {
		panic("secrets.luks_password is required because infrastructure.luks is configured in the blueprint")
	}

	// Store artifact path
	sess.ArtifactPath = artifactPath

	// ============================================================================
	// Block II: Infrastructure
	// ============================================================================
	fmt.Println("🔧 Step 2/5: Setting up infrastructure...")

	if hasLuks {
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
	} else {
		fmt.Println("↷ No infrastructure.luks configured, skipping LUKS unlock/mount.")
	}

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

	fmt.Println("✨ Rehydration Complete. Welcome back, Commander.")
}
