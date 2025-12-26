package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Global flags
var secretsPath string
var blueprintPath string
var dotfilesArchive string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "phoenix",
	Short: "A single-binary provisioner for Fedora Workstation",
	Long: `Fedora Phoenix is a specialized tool for restoring a developer environment 
on a Framework Laptop. It handles LUKS unlocking, package installation, 
and dotfiles restoration in a single shot.`,
	// 如果你希望 ./phoenix 直接跑 provision，可以把邏輯寫在 Run 裡，
	// 但通常建議留空，強迫使用者下子命令 (e.g., ./phoenix provision)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// 定義全域 Flag
	// PersistentFlags 代表這個 flag 可以被所有子命令繼承
	rootCmd.PersistentFlags().StringVarP(&secretsPath, "secrets", "s", "", "Path to the secrets YAML file (required)")
	rootCmd.PersistentFlags().StringVarP(&blueprintPath, "blueprint", "b", "phoenix.yml", "Path to the blueprint YAML file")
	rootCmd.PersistentFlags().StringVarP(&dotfilesArchive, "dotfiles-archive", "d", "", "Path to dotfiles tarball (.tgz)")
}
