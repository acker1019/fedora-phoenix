package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	CommitSHA = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Phoenix",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Fedora Phoenix v%s (Commit: %s)\n", Version, CommitSHA)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
