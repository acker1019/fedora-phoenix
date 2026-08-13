package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/acker1019/fedora-trisolaran/internal/logging"
	"gopkg.in/yaml.v3"
)

var blueprintLog = logging.WithSource("config/blueprint")

// Blueprint defines the schema for the trisolaran.yml configuration file.
// It represents the declarative system restoration plan.
type Blueprint struct {
	Version string `yaml:"version"`

	// Infrastructure: Hardware and storage configuration
	Infrastructure InfrastructureConfig `yaml:"infrastructure"`

	// System: OS-level packages and services
	System SystemConfig `yaml:"system"`

	// Identity: Target user configuration
	Identity IdentityConfig `yaml:"identity"`

	// UserSpace: User-level configuration (Block IV)
	UserSpace UserSpaceConfig `yaml:"userspace"`
}

// InfrastructureConfig defines storage and hardware mappings
type InfrastructureConfig struct {
	Luks LuksConfig `yaml:"luks"`
}

// LuksConfig defines LUKS partition configuration
type LuksConfig struct {
	Device     string `yaml:"device"`
	MapperName string `yaml:"mapper_name"`
	MountPoint string `yaml:"mount_point"`
}

// SystemConfig defines OS-level state
type SystemConfig struct {
	SkipUpdate     bool            `yaml:"skip_update"`
	Pkgs           []string        `yaml:"pkgs"`
	PinnedPackages []string        `yaml:"pinned_packages"`
	PkgRepos       []PkgRepoConfig `yaml:"pkg_repos"`
	Services       []string        `yaml:"services"`
	// Tmpfiles are paths (may use "~", expanded against the target user's
	// home) removed on every boot via systemd-tmpfiles -- e.g. a stale app
	// lock file left behind by an unclean shutdown. Declarative: no custom
	// systemd unit needed, since systemd-tmpfiles-setup.service already
	// ships enabled on any systemd system.
	Tmpfiles []string      `yaml:"tmpfiles"`
	Users    []UserConfig  `yaml:"users"`
	Groups   []GroupConfig `yaml:"groups"`
}

// PkgRepoConfig declares the desired state of a single dnf repo, so
// packages that live outside Fedora's default repos (e.g. VS Code, Chrome)
// can be ensured before System.Pkgs installs them. Two shapes:
//   - id only: an already-shipped repo (e.g. Fedora's google-chrome.repo,
//     provided by fedora-workstation-repositories) whose `enabled=` flag
//     just needs flipping.
//   - id + baseurl: a repo file that doesn't exist yet and must be created
//     (GPGKey, if set, is imported via rpm --import).
type PkgRepoConfig struct {
	ID      string `yaml:"id"`
	BaseURL string `yaml:"baseurl,omitempty"`
	GPGKey  string `yaml:"gpgkey,omitempty"`
	Enabled bool   `yaml:"enabled"`
}

// UserConfig defines a user account to ensure exists.
// Only the name is recorded; UID is read back from the OS after creation.
type UserConfig struct {
	Name   string   `yaml:"name"`
	System bool     `yaml:"system"`
	Groups []string `yaml:"groups"`
}

// GroupConfig defines a group to ensure exists.
// Only the name is recorded; GID is read back from the OS after creation.
type GroupConfig struct {
	Name   string `yaml:"name"`
	System bool   `yaml:"system"`
}

// IdentityConfig defines target user characteristics
type IdentityConfig struct {
	Username string `yaml:"username"`
	Shell    string `yaml:"shell"`
}

// UserSpaceConfig defines user-level configuration (Block IV)
type UserSpaceConfig struct {
	Dehydration DehydrationConfig `yaml:"dehydration"`
	Repos       []RepoConfig      `yaml:"repos"`

	// Pipeline is the ordered sequence of Block IV steps, run as the
	// target user. By default (no step in Pipeline has a Run of
	// RunDehydration or RunRepos), restoring the artifact and cloning
	// Repos both happen automatically before Pipeline runs, and every
	// Script step then runs in order -- the original fixed "restore,
	// clone, then scripts last" behavior. Explicitly placing a
	// RunDehydration or RunRepos step in Pipeline opts out of that
	// default and lets you interleave custom commands around them (e.g. a
	// dotfile-clobbering installer that must run *before* the artifact
	// restore, so the restore has the last word instead of the
	// installer).
	//
	// A step's Run keyword is deliberately the same word as the section
	// it triggers (RunDehydration -> the Dehydration section above,
	// RunRepos -> the Repos section above) rather than a separate
	// "restore_x" verb, so there's only one name per concept to remember.
	//
	// Script entries are opaque to Trisolaran: making a given line safe
	// to rerun is the script author's responsibility, not the engine's.
	Pipeline []PipelineStep `yaml:"pipeline"`
}

// Known values for PipelineStep.Run.
const (
	RunDehydration = "dehydration"
	RunRepos       = "repos"
)

// PipelineStep is exactly one of: a custom one-liner (Script), or a named
// built-in action (Run -- one of RunDehydration, RunRepos). Run is a
// keyword, not a bool, so future built-in actions can be added as new
// valid values without growing the struct every time.
type PipelineStep struct {
	Script string `yaml:"script,omitempty"`
	Run    string `yaml:"run,omitempty"`
}

// DehydrationConfig defines which paths are collected into an artifact.
// See ADR-0008 (Artifact Storage Format).
type DehydrationConfig struct {
	Paths []string `yaml:"paths"`
}

// RepoConfig defines a git repository to clone
type RepoConfig struct {
	URL  string `yaml:"url"`
	Dest string `yaml:"dest"`
}

// LoadBlueprint reads and parses the trisolaran.yml blueprint file.
func LoadBlueprint(path string) (*Blueprint, error) {
	blueprintLog.Infof("Loading blueprint from: %s", path)

	// 1. Check file existence
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("blueprint file not found at: %s", path)
	}

	// 2. Read file content
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read blueprint file: %w", err)
	}

	bp, err := ParseBlueprint(data)
	if err != nil {
		return nil, err
	}

	blueprintLog.Info("Blueprint loaded successfully")
	return bp, nil
}

// ParseBlueprint parses and validates blueprint YAML content regardless of
// where it came from — a file on disk, or one embedded in an artifact tgz
// (see artifact.ExtractBlueprint).
func ParseBlueprint(data []byte) (*Blueprint, error) {
	var bp Blueprint

	// KnownFields rejects unrecognized keys instead of silently dropping
	// them. Without this, a stale field left over from a schema rename
	// (e.g. system.packages -> system.pkgs) parses successfully but the
	// new field just never gets set, with no error anywhere.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&bp); err != nil {
		return nil, fmt.Errorf("failed to parse YAML structure: %w", err)
	}

	if err := validateBlueprint(&bp); err != nil {
		return nil, fmt.Errorf("invalid blueprint: %w", err)
	}

	return &bp, nil
}

// HasLuks reports whether infrastructure.luks is configured in the blueprint.
// LUKS is optional: an absent `infrastructure` block or an empty `luks`
// sub-block means the rehydration protocol skips LUKS unlock/mount entirely.
func (bp *Blueprint) HasLuks() bool {
	l := bp.Infrastructure.Luks
	return l.Device != "" || l.MapperName != "" || l.MountPoint != ""
}

// validateBlueprint ensures critical fields are present
func validateBlueprint(bp *Blueprint) error {
	if bp.Version == "" {
		return fmt.Errorf("version field is required")
	}

	// Validate Infrastructure: LUKS is optional, but if any field is set,
	// all three are required (a partially configured LUKS block is an error).
	if bp.HasLuks() {
		if bp.Infrastructure.Luks.Device == "" {
			return fmt.Errorf("infrastructure.luks.device is required when infrastructure.luks is configured")
		}
		if bp.Infrastructure.Luks.MapperName == "" {
			return fmt.Errorf("infrastructure.luks.mapper_name is required when infrastructure.luks is configured")
		}
		if bp.Infrastructure.Luks.MountPoint == "" {
			return fmt.Errorf("infrastructure.luks.mount_point is required when infrastructure.luks is configured")
		}
	}

	// Validate Identity
	if bp.Identity.Username == "" {
		return fmt.Errorf("identity.username is required")
	}

	// Validate Pipeline: each step must set exactly one of Script/Run, and
	// a Run value must be a known keyword, so there's no ambiguity about
	// what a step does (or silent no-ops from an empty/misspelled step).
	for i, step := range bp.UserSpace.Pipeline {
		set := 0
		if step.Script != "" {
			set++
		}
		if step.Run != "" {
			set++
		}
		if set != 1 {
			return fmt.Errorf("userspace.pipeline[%d] must set exactly one of script or run", i)
		}
		if step.Run != "" && step.Run != RunDehydration && step.Run != RunRepos {
			return fmt.Errorf("userspace.pipeline[%d]: unknown run value %q (expected %q or %q)", i, step.Run, RunDehydration, RunRepos)
		}
	}

	return nil
}
