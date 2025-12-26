package session

import (
	"github.com/acker1019/fedora-phoenix/internal/config"
)

// Session holds all runtime state for a single provision execution.
// This is created locally in runProvision() and passed to Acts as needed.
//
// Design: Local instance, not global singleton. Explicit parameter passing.
type Session struct {
	// Configuration (loaded from files)
	Blueprint *config.Blueprint
	Secrets   *config.Secrets

	// User Identity (discovered at runtime)
	Username string // Real user who invoked sudo (e.g., "ack")
	UID      int    // User's UID
	GID      int    // User's GID
	UserHome string // User's home directory path (e.g., "/home/ack")

	// Infrastructure State (from Block II)
	LuksMapperName string // LUKS mapper name (e.g., "company_data")
	LuksMountPoint string // LUKS mount point (e.g., "/mnt/company_data")
	LuksUnlocked   bool   // Whether LUKS device is currently unlocked
	LuksMounted    bool   // Whether LUKS device is currently mounted

	// Expanded Paths (from Block IV)
	StowSourceDir string // Expanded stow source directory
	StowTargetDir string // Expanded stow target directory

	// Temporary Variables
	DotfilesArchive string // Path to dotfiles tarball (from --dotfiles-archive flag)
}
