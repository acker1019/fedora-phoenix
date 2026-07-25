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
	"syscall"
	"time"

	"github.com/acker1019/fedora-trisolaran/internal/logging"
)

var packLog = logging.WithSource("artifact/pack")

// Pack collects the given absolute paths into a single artifact tgz file.
// Each path is mirrored under fs/<absolute path> inside the archive, and
// its permission metadata (mode, owner, group, hash) is recorded in
// filemeta.yml. See ADR-0008 (Artifact Storage Format).
//
// Paths must already be expanded to absolute form (no "~").
func Pack(paths []string, outputPath string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no paths to pack")
	}

	packLog.Infof("Packing %d paths into %s", len(paths), outputPath)

	stageDir, err := os.MkdirTemp("", "phoenix-pack-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer func() {
		if rerr := os.RemoveAll(stageDir); rerr != nil {
			packLog.Warnf("failed to remove staging directory %s: %v", stageDir, rerr)
		}
	}()

	protectName := fmt.Sprintf("phoenix-backup-%s", time.Now().Format("20060102"))
	root := filepath.Join(stageDir, protectName)
	fsRoot := filepath.Join(root, "fs")
	if err := os.MkdirAll(fsRoot, 0755); err != nil {
		return fmt.Errorf("failed to create fs mirror root: %w", err)
	}

	manifest := &Manifest{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, p := range paths {
		if err := collectPath(p, fsRoot, manifest); err != nil {
			return fmt.Errorf("failed to collect %s: %w", p, err)
		}
	}

	if err := SaveManifest(filepath.Join(root, ManifestFileName), manifest); err != nil {
		return err
	}

	if err := createTarGz(stageDir, protectName, outputPath); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	packLog.Infof("Artifact created: %s (%d entries)", outputPath, len(manifest.Entries))
	return nil
}

// collectPath walks a single source path (file or directory) and mirrors
// it under fsRoot, appending metadata entries to the manifest.
func collectPath(src, fsRoot string, manifest *Manifest) error {
	src = filepath.Clean(src)

	return filepath.Walk(src, func(walkedPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		mirrorPath := filepath.Join(fsRoot, walkedPath)
		owner, group, err := lookupOwnerGroup(info)
		if err != nil {
			return err
		}
		mode := fmt.Sprintf("%04o", info.Mode().Perm())

		if info.IsDir() {
			if err := os.MkdirAll(mirrorPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("failed to create mirror directory: %w", err)
			}
			manifest.Entries = append(manifest.Entries, FileMeta{
				Path:  walkedPath,
				Mode:  mode,
				Owner: owner,
				Group: group,
				Type:  "directory",
			})
			return nil
		}

		// Symlinks and other special files are skipped; only regular files are mirrored.
		if !info.Mode().IsRegular() {
			packLog.Warnf("Skipping non-regular file: %s", walkedPath)
			return nil
		}

		if err := copyFile(walkedPath, mirrorPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to copy %s: %w", walkedPath, err)
		}

		sum, err := sha256File(walkedPath)
		if err != nil {
			return err
		}

		manifest.Entries = append(manifest.Entries, FileMeta{
			Path:   walkedPath,
			Mode:   mode,
			Owner:  owner,
			Group:  group,
			SHA256: sum,
		})

		return nil
	})
}

// lookupOwnerGroup resolves the owner/group *names* for a file, per
// ADR-0008: UID/GID are never persisted, only names.
func lookupOwnerGroup(info os.FileInfo) (owner, group string, err error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", "", fmt.Errorf("unable to read owner/group metadata")
	}

	u, err := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10))
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve owner name: %w", err)
	}

	g, err := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10))
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve group name: %w", err)
	}

	return u.Username, g.Name, nil
}

// copyFile copies src to dest. The destination's Close() error is checked
// and returned: a failed Close can mean buffered data was never flushed to
// disk, which would silently corrupt the copy.
func copyFile(src, dest string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := in.Close(); cerr != nil {
			packLog.Warnf("failed to close source file %s: %v", src, cerr)
		}
	}()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close() // best-effort; the copy error is the one that matters here
		return err
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", dest, err)
	}

	return nil
}

// createTarGz packs baseDir/name into a gzip-compressed tarball at outputPath.
//
// Closing tw/gw/outFile (in that order) is what actually flushes the tar
// footer and compressed stream to disk, so a failed Close here means a
// truncated/corrupt archive. Those errors are surfaced via the named
// return instead of being silently deferred away.
func createTarGz(baseDir, name, outputPath string) (err error) {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := outFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close archive file: %w", cerr)
		}
	}()

	gw := gzip.NewWriter(outFile)
	defer func() {
		if cerr := gw.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close gzip writer: %w", cerr)
		}
	}()

	tw := tar.NewWriter(gw)
	defer func() {
		if cerr := tw.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close tar writer: %w", cerr)
		}
	}()

	root := filepath.Join(baseDir, name)
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				packLog.Warnf("failed to close source file %s: %v", path, cerr)
			}
		}()

		_, err = io.Copy(tw, f)
		return err
	})

	return err
}
