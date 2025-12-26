package cmd

import (
	"fmt"
	"os"

	"github.com/acker1019/fedora-phoenix/internal/config"
	"github.com/acker1019/fedora-phoenix/internal/ops"

	"github.com/spf13/cobra"
)

// provisionCmd represents the provision command
var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Start the full restoration protocol",
	Long:  `Unlock LUKS, mount data, install packages, and link dotfiles.`,
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

	// 3. Load Secrets
	secrets, err := config.LoadSecrets(secretsPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load secrets: %v", err))
	}
	// Self-destruct logic
	config.CleanupSecrets(secretsPath)

	// 4. LUKS Unlock
	err = ops.UnlockLuks("/dev/nvme0n1p4", "company_data", secrets.LuksPassword)
	if err != nil {
		panic(err)
	}

	// 5. Mount
	ops.MountDevice("/dev/mapper/company_data", "/mnt/company_data")

	// 6. Packages
	ops.EnsurePackages("git", "zsh", "docker", "fprintd")

	// ... 其他邏輯 ...

	fmt.Println("✨ World Line Restored.")
}
