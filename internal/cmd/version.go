package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Phoenix",
	Run: func(cmd *cobra.Command, args []string) {
		version := "dev"
		commit := "unknown"
		dirty := false

		if info, ok := debug.ReadBuildInfo(); ok {
			// Try to get version from VCS info
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					commit = setting.Value
					if len(commit) > 7 {
						commit = commit[:7] // Short SHA
					}
				case "vcs.modified":
					dirty = setting.Value == "true"
				}
			}
		}

		// Format output
		if dirty {
			fmt.Printf("Fedora Phoenix %s (commit: %s-dirty)\n", version, commit)
		} else {
			fmt.Printf("Fedora Phoenix %s (commit: %s)\n", version, commit)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
