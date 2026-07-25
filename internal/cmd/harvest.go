package cmd

import (
	"fmt"
	"os"

	"github.com/acker1019/fedora-phoenix/internal/artifact"
	"github.com/acker1019/fedora-phoenix/internal/config"
	"github.com/acker1019/fedora-phoenix/internal/utils"

	"github.com/spf13/cobra"
)

var harvestOutput string

// harvestCmd represents the harvest command
var harvestCmd = &cobra.Command{
	Use:   "harvest",
	Short: "Collect configured paths into a single artifact",
	Long:  `Reads userspace.harvest.paths from the blueprint and packs them into an artifact tgz (ADR-0008).`,
	Run: func(cmd *cobra.Command, args []string) {
		runHarvest()
	},
}

func init() {
	rootCmd.AddCommand(harvestCmd)
	harvestCmd.Flags().StringVarP(&harvestOutput, "output", "o", "phoenix-backup.tgz", "Path to write the artifact")
}

func runHarvest() {
	blueprint, err := config.LoadBlueprint(blueprintPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load blueprint: %v", err))
	}

	if len(blueprint.UserSpace.Harvest.Paths) == 0 {
		fmt.Println("❌ Error: no paths configured under userspace.harvest.paths")
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("Failed to determine home directory: %v", err))
	}

	paths := make([]string, len(blueprint.UserSpace.Harvest.Paths))
	for i, p := range blueprint.UserSpace.Harvest.Paths {
		paths[i] = utils.ExpandPath(p, homeDir)
	}

	fmt.Printf("🌾 Harvesting %d paths...\n", len(paths))
	if err := artifact.Pack(paths, harvestOutput); err != nil {
		panic(fmt.Sprintf("Failed to pack artifact: %v", err))
	}

	fmt.Printf("✓ Artifact written to %s\n", harvestOutput)
}
