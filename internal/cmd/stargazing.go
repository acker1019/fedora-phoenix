package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stargazingCmd represents the stargazing command
var stargazingCmd = &cobra.Command{
	Use:   "stargazing",
	Short: "Observe system state and drift (ADR-0009)",
	Long:  `Monitors system state for drift. Not yet implemented.`,
	Run: func(cmd *cobra.Command, args []string) {
		runStargazing()
	},
}

func init() {
	rootCmd.AddCommand(stargazingCmd)
}

func runStargazing() {
	fmt.Println("🔭 Stargazing is not yet implemented (see ADR-0009).")
}
