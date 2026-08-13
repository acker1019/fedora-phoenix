package utils

import (
	"os"
	"strings"
)

// WriteNoticesReport writes one notice per line to path, so a CLI command
// with a lot of notices can print a short summary instead of flooding the
// terminal. Does nothing if notices is empty -- no report file for a run
// that had nothing to flag.
func WriteNoticesReport(notices []string, path string) error {
	if len(notices) == 0 {
		return nil
	}

	return os.WriteFile(path, []byte(strings.Join(notices, "\n")+"\n"), 0644)
}
