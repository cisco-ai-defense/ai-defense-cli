//go:build embed_python

package python

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// The aid-py bundle is a relocatable CPython interpreter with cisco-aibom[all]
// installed into its site-packages, packed as a gzip-compressed tar. It is
// produced by aid_cli/Dockerfile and embedded here.
//
//go:embed bin/aid-py-bundle.tar.gz
var bundleFS embed.FS

// bundleID is a short content hash of the bundle, used as the on-disk cache key
// so a new build extracts to a fresh directory and stale trees are never reused.
//
//go:embed bin/aid-py-bundle.id
var bundleID string

var (
	extractOnce sync.Once
	pythonPath  string
	extractErr  error
)

// cacheDirName namespaces the extracted interpreter under the user cache dir.
const cacheDirName = "aid-cisco-cli"

// Interpreter returns the path to a Python interpreter with cisco-aibom
// installed. By default it extracts the embedded bundle once per content id;
// set AID_CLI_PYTHON to a local interpreter to bypass the bundle for dev.
func Interpreter() (string, error) {
	if py := os.Getenv("AID_CLI_PYTHON"); py != "" {
		if _, err := os.Stat(py); err != nil {
			return "", fmt.Errorf("AID_CLI_PYTHON %q: %w", py, err)
		}
		return py, nil
	}
	return ensureExtracted()
}

// ensureExtracted unpacks the embedded bundle once per content id and returns
// the path to the bundled python interpreter. Extraction is skipped if a
// completed extraction for this id already exists.
func ensureExtracted() (string, error) {
	extractOnce.Do(func() {
		id := strings.TrimSpace(bundleID)
		if id == "" {
			id = "current"
		}

		root := cacheRoot()
		dest := filepath.Join(root, cacheDirName, "py", id)
		pyBin := filepath.Join(dest, "python", "bin", "python3")
		marker := filepath.Join(dest, ".complete")

		if _, err := os.Stat(marker); err == nil {
			pythonPath = pyBin
			return
		}

		parent := filepath.Dir(dest)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			extractErr = fmt.Errorf("create cache dir: %w", err)
			return
		}

		tmp, err := os.MkdirTemp(parent, ".extract-*")
		if err != nil {
			extractErr = fmt.Errorf("create temp extract dir: %w", err)
			return
		}
		cleanupTmp := true
		defer func() {
			if cleanupTmp {
				_ = os.RemoveAll(tmp)
			}
		}()

		if err := extractBundle(tmp); err != nil {
			extractErr = fmt.Errorf("extract aid-py bundle: %w", err)
			return
		}

		// Mark the temp tree complete before publishing it, so the marker is
		// present atomically once the directory is renamed into place.
		if err := os.WriteFile(filepath.Join(tmp, ".complete"), []byte(id), 0o644); err != nil {
			extractErr = fmt.Errorf("write completion marker: %w", err)
			return
		}

		if err := os.Rename(tmp, dest); err != nil {
			// Another process may have extracted the same id concurrently. If a
			// completed tree now exists, use it; otherwise surface the error.
			if _, statErr := os.Stat(marker); statErr == nil {
				cleanupTmp = true
				pythonPath = pyBin
				return
			}
			extractErr = fmt.Errorf("publish extracted bundle: %w", err)
			return
		}
		cleanupTmp = false
		pythonPath = pyBin
	})

	return pythonPath, extractErr
}

// cacheRoot returns the directory under which the interpreter is extracted,
// falling back to the OS temp dir when no user cache dir is available.
func cacheRoot() string {
	if root, err := os.UserCacheDir(); err == nil && root != "" {
		return root
	}
	return os.TempDir()
}

// extractBundle unpacks the embedded gzip-tar bundle into dest, preserving file
// modes and symlinks and guarding against path traversal.
func extractBundle(dest string) error {
	f, err := bundleFS.Open("bin/aid-py-bundle.tar.gz")
	if err != nil {
		return fmt.Errorf("open embedded bundle: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create dir %s: %w", hdr.Name, err)
			}
		case tar.TypeReg:
			if err := writeFile(target, tr, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if filepath.IsAbs(hdr.Linkname) || strings.Contains(hdr.Linkname, "..") {
				return fmt.Errorf("illegal symlink in bundle: %q -> %q", hdr.Name, hdr.Linkname)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent for symlink %s: %w", hdr.Name, err)
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return fmt.Errorf("create symlink %s: %w", hdr.Name, err)
			}
		case tar.TypeLink:
			source, err := safeJoin(dest, hdr.Linkname)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent for hardlink %s: %w", hdr.Name, err)
			}
			_ = os.Remove(target)
			if err := os.Link(source, target); err != nil {
				return fmt.Errorf("create hardlink %s: %w", hdr.Name, err)
			}
		default:
			// Skip unsupported entry types (devices, fifos, etc.).
		}
	}
	return nil
}

func writeFile(target string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent for %s: %w", target, err)
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode|0o600)
	if err != nil {
		return fmt.Errorf("create file %s: %w", target, err)
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return fmt.Errorf("write file %s: %w", target, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close file %s: %w", target, err)
	}
	return nil
}

// safeJoin joins name onto base, rejecting entries that would escape base.
func safeJoin(base, name string) (string, error) {
	target := filepath.Join(base, name)
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("illegal path in bundle: %q", name)
	}
	return target, nil
}
