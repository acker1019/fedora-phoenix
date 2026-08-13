package ops

import (
	"fmt"

	"github.com/acker1019/fedora-trisolaran/internal/logging"
	"github.com/acker1019/fedora-trisolaran/internal/utils"
)

var flatpakLog = logging.WithSource("ops/flatpak")

// flathubRepoURL is Flathub's own flatpakrepo file, the standard way to
// add it as a remote.
const flathubRepoURL = "https://flathub.org/repo/flathub.flatpakrepo"

// EnsureFlatpaks adds the Flathub remote (user-scope: no root needed,
// unlike Snap which requires a system-level /snap symlink on Fedora) and
// installs the given Flathub app IDs as the target user. No separate
// check step: both `flatpak remote-add --if-not-exists` and `flatpak
// install` are themselves idempotent.
func EnsureFlatpaks(appIDs []string, username string) error {
	if len(appIDs) == 0 {
		return nil
	}

	flatpakLog.Info("Ensuring Flathub remote...")
	if err := utils.RunCommandAsUser(username, "flatpak", "remote-add", "--if-not-exists", "--user", "flathub", flathubRepoURL); err != nil {
		return fmt.Errorf("failed to add flathub remote: %w", err)
	}

	flatpakLog.Infof("Ensuring %d Flatpak app(s)...", len(appIDs))
	args := append([]string{"install", "-y", "--user", "flathub"}, appIDs...)
	if err := utils.RunCommandAsUser(username, "flatpak", args...); err != nil {
		return fmt.Errorf("failed to install flatpaks: %w", err)
	}

	flatpakLog.Info("Flatpaks ensured")
	return nil
}
