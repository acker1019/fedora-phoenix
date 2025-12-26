package config

import (
	"fmt"
	"os"

	"github.com/acker1019/fedora-phoenix/internal/logging"
	"gopkg.in/yaml.v3"
)

var blueprintLog = logging.WithSource("config/blueprint")

// Blueprint defines the schema for the phoenix.yml configuration file.
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
	Packages       []string `yaml:"packages"`
	PinnedPackages []string `yaml:"pinned_packages"`
	Services       []string `yaml:"services"`
}

// IdentityConfig defines target user characteristics
type IdentityConfig struct {
	Username string `yaml:"username"`
	Shell    string `yaml:"shell"`
}

// UserSpaceConfig defines user-level configuration (Block IV)
type UserSpaceConfig struct {
	Stow  StowConfig   `yaml:"stow"`
	Repos []RepoConfig `yaml:"repos"`
}

// StowConfig defines GNU Stow deployment configuration
type StowConfig struct {
	SourceDir string   `yaml:"source_dir"`
	TargetDir string   `yaml:"target_dir"`
	Packages  []string `yaml:"packages"`
}

// RepoConfig defines a git repository to clone
type RepoConfig struct {
	URL  string `yaml:"url"`
	Dest string `yaml:"dest"`
}

// LoadBlueprint reads and parses the phoenix.yml blueprint file.
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

	// 3. Parse YAML
	var bp Blueprint
	if err := yaml.Unmarshal(data, &bp); err != nil {
		return nil, fmt.Errorf("failed to parse YAML structure: %w", err)
	}

	// 4. Validate required fields
	if err := validateBlueprint(&bp); err != nil {
		return nil, fmt.Errorf("invalid blueprint: %w", err)
	}

	blueprintLog.Info("Blueprint loaded successfully")
	return &bp, nil
}

// validateBlueprint ensures critical fields are present
func validateBlueprint(bp *Blueprint) error {
	if bp.Version == "" {
		return fmt.Errorf("version field is required")
	}

	// Validate Infrastructure
	if bp.Infrastructure.Luks.Device == "" {
		return fmt.Errorf("infrastructure.luks.device is required")
	}
	if bp.Infrastructure.Luks.MapperName == "" {
		return fmt.Errorf("infrastructure.luks.mapper_name is required")
	}
	if bp.Infrastructure.Luks.MountPoint == "" {
		return fmt.Errorf("infrastructure.luks.mount_point is required")
	}

	// Validate Identity
	if bp.Identity.Username == "" {
		return fmt.Errorf("identity.username is required")
	}

	return nil
}
