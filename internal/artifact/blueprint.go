package artifact

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BlueprintFileName is the fixed name, alongside ManifestFileName inside the
// protective top-level folder, of the blueprint that produced the artifact.
// Living at a fixed location (not under fs/) lets rehydra read it straight
// out of an artifact tgz without needing an external --blueprint file first.
const BlueprintFileName = "blueprint.yml"

// ErrBlueprintNotEmbedded is returned by ExtractBlueprint when the archive
// was packed without an embedded blueprint (e.g. produced before this
// feature existed).
var ErrBlueprintNotEmbedded = errors.New("artifact does not contain an embedded blueprint")

// ExtractBlueprint reads BlueprintFileName out of an artifact tgz without
// extracting anything else, so callers don't pay for unpacking fs/ just to
// read a few KB of YAML.
func ExtractBlueprint(archivePath string) ([]byte, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open artifact: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			unpackLog.Warnf("failed to close artifact %s: %v", archivePath, cerr)
		}
	}()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to open gzip stream: %w", err)
	}
	defer func() {
		if cerr := gr.Close(); cerr != nil {
			unpackLog.Warnf("failed to close gzip reader: %v", cerr)
		}
	}()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil, ErrBlueprintNotEmbedded
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read artifact: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Must be directly under the protective top-level folder
		// ("<name>/blueprint.yml"), not somewhere inside fs/.
		parts := strings.Split(filepath.ToSlash(header.Name), "/")
		if len(parts) != 2 || parts[1] != BlueprintFileName {
			continue
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded blueprint: %w", err)
		}
		return data, nil
	}
}
