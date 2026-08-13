package ops

import (
	"fmt"
	"os"
	"strings"

	"github.com/acker1019/fedora-trisolaran/internal/logging"
	"github.com/acker1019/fedora-trisolaran/internal/utils"
)

var tmpfilesLog = logging.WithSource("ops/tmpfiles")

// tmpfilesConfPath is where EnsureTmpfiles writes its systemd-tmpfiles
// rules. systemd-tmpfiles-setup.service reads every file under
// /etc/tmpfiles.d/ on every boot, so no custom unit is needed to apply
// this on startup.
const tmpfilesConfPath = "/etc/tmpfiles.d/trisolaran.conf"

// EnsureTmpfiles declares paths to be removed on every boot (e.g. a stale
// app lock file). Idempotent: the desired file content is computed and
// only written if it differs from what's already on disk.
func EnsureTmpfiles(paths []string, userHome string) error {
	if len(paths) == 0 {
		return nil
	}

	var desired strings.Builder
	for _, p := range paths {
		fmt.Fprintf(&desired, "r %s\n", utils.ExpandPath(p, userHome))
	}

	existing, err := os.ReadFile(tmpfilesConfPath)
	if err == nil && string(existing) == desired.String() {
		tmpfilesLog.Info("tmpfiles config already up to date. Skipping.")
		return nil
	}

	tmpfilesLog.Infof("Writing tmpfiles config: %s", tmpfilesConfPath)
	if err := os.WriteFile(tmpfilesConfPath, []byte(desired.String()), 0644); err != nil {
		return fmt.Errorf("failed to write tmpfiles config: %w", err)
	}

	return nil
}
