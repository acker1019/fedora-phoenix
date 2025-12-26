package config

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/acker1019/fedora-phoenix/internal/logging"
	"gopkg.in/yaml.v3"
)

var log = logging.WithSource("secrets")

// SecretsBook defines the schema for the secrets configuration file.
// We use YAML tags here to map keys from the input file.
type SecretsBook struct {
	LuksPassword string `yaml:"luks_password"`
	// Add other secrets here as needed, e.g.:
	// GitToken     string `yaml:"git_token"`
	// RootPassword string `yaml:"root_password"`
}

// LoadSecrets reads and parses the secrets YAML file from the given path.
func LoadSecrets(path string) (*SecretsBook, error) {
	log.Infof("Loading secrets from local file: %s", path)

	// 1. Sanity check: file existence
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("secret file not found at: %s", path)
	}

	// 2. Read file content
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 3. Parse YAML
	var book SecretsBook
	if err := yaml.Unmarshal(data, &book); err != nil {
		return nil, fmt.Errorf("failed to parse YAML structure: %w", err)
	}

	// Validation: Ensure critical secrets are present
	if book.LuksPassword == "" {
		return nil, fmt.Errorf("invalid secrets file: 'luks_password' is missing or empty")
	}

	return &book, nil
}

// CleanupSecrets safely removes the secrets file from the disk.
// This implements the "Self-Destruct" policy with secure overwrite.
func CleanupSecrets(path string) {
	log.Infof("Destroying secrets file: %s", path)

	// Step 1: Overwrite file with random data before deletion
	if err := secureOverwrite(path); err != nil {
		log.Warnf("Failed to overwrite secrets file: %v", err)
		// Continue to deletion even if overwrite fails
	} else {
		log.Info("Secrets file overwritten with random data")
	}

	// Step 2: Remove the file
	if err := os.Remove(path); err != nil {
		log.Warnf("Failed to delete secrets file: %v", err)
	} else {
		log.Info("Secrets file destroyed successfully")
	}
}

// secureOverwrite overwrites the file with cryptographically secure random data.
// This prevents file recovery from filesystem-level artifacts.
func secureOverwrite(path string) error {
	// Get file size
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := info.Size()
	if fileSize == 0 {
		return nil // Nothing to overwrite
	}

	// Open file for writing (truncate not needed, we'll overwrite in place)
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file for overwrite: %w", err)
	}
	defer file.Close()

	// Generate random data
	randomData := make([]byte, fileSize)
	if _, err := rand.Read(randomData); err != nil {
		return fmt.Errorf("failed to generate random data: %w", err)
	}

	// Write random data to file
	if _, err := file.Write(randomData); err != nil {
		return fmt.Errorf("failed to write random data: %w", err)
	}

	// Sync to disk to ensure data is written
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}
