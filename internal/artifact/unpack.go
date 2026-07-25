package artifact

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/acker1019/fedora-trisolaran/internal/logging"
)

var unpackLog = logging.WithSource("artifact/unpack")

// Restore extracts an artifact tgz and applies its contents back onto the
// system: files are copied to their recorded absolute paths, then
// owner/group/mode are applied from filemeta.yml and file content is
// verified against the recorded SHA256. See ADR-0008 (Artifact Storage Format).
func Restore(archivePath string) error {
	unpackLog.Infof("Restoring artifact: %s", archivePath)

	stageDir, err := os.MkdirTemp("", "trisolaran-unpack-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer func() {
		if rerr := os.RemoveAll(stageDir); rerr != nil {
			unpackLog.Warnf("failed to remove staging directory %s: %v", stageDir, rerr)
		}
	}()

	if err := extractTarGz(archivePath, stageDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	root, err := findArtifactRoot(stageDir)
	if err != nil {
		return err
	}

	manifest, err := LoadManifest(filepath.Join(root, ManifestFileName))
	if err != nil {
		return err
	}

	fsRoot := filepath.Join(root, "fs")

	for _, entry := range manifest.Entries {
		if err := applyEntry(entry, fsRoot); err != nil {
			return fmt.Errorf("failed to restore %s: %w", entry.Path, err)
		}
	}

	unpackLog.Infof("Artifact restored: %d entries applied", len(manifest.Entries))
	return nil
}

// findArtifactRoot locates the single "trisolaran-backup-<date>" directory
// produced by Pack inside the extracted archive.
func findArtifactRoot(stageDir string) (string, error) {
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return "", fmt.Errorf("failed to read staging directory: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			return filepath.Join(stageDir, e.Name()), nil
		}
	}

	return "", fmt.Errorf("archive does not contain a backup directory")
}

// applyEntry restores a single manifest entry: content, mode, and ownership.
func applyEntry(entry FileMeta, fsRoot string) error {
	mode, err := parseMode(entry.Mode)
	if err != nil {
		return err
	}

	if entry.Type == "directory" {
		if err := os.MkdirAll(entry.Path, mode); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	} else {
		src := filepath.Join(fsRoot, entry.Path)

		if err := copyFile(src, entry.Path, mode); err != nil {
			return fmt.Errorf("failed to restore file content: %w", err)
		}

		if entry.SHA256 != "" {
			sum, err := sha256File(entry.Path)
			if err != nil {
				return err
			}
			if sum != entry.SHA256 {
				return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", entry.Path, entry.SHA256, sum)
			}
		}
	}

	if err := os.Chmod(entry.Path, mode); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	uid, gid, err := lookupUIDGID(entry.Owner, entry.Group)
	if err != nil {
		return err
	}
	if err := os.Chown(entry.Path, uid, gid); err != nil {
		return fmt.Errorf("failed to chown: %w", err)
	}

	return nil
}

func parseMode(mode string) (os.FileMode, error) {
	v, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid mode %q: %w", mode, err)
	}
	return os.FileMode(v), nil
}

// lookupUIDGID resolves owner/group *names* back to numeric IDs on the
// current system. Per ADR-0008, IDs are never trusted from the artifact
// itself — only names, resolved fresh at restore time.
func lookupUIDGID(owner, group string) (int, int, error) {
	u, err := user.Lookup(owner)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve owner %q: %w", owner, err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid uid for %q: %w", owner, err)
	}

	g, err := user.LookupGroup(group)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve group %q: %w", group, err)
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid gid for %q: %w", group, err)
	}

	return uid, gid, nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			unpackLog.Warnf("failed to close archive %s: %v", archivePath, cerr)
		}
	}()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
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
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close() // best-effort; the copy error is the one that matters here
				return err
			}
			if err := out.Close(); err != nil {
				return fmt.Errorf("failed to close %s: %w", target, err)
			}
		}
	}

	return nil
}
