package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultOwner = "cisco-ai-defense"
	defaultRepo  = "ai-defense-cli"
	binaryName   = "aid-cisco-cli"
)

// Options controls a self-update install.
type Options struct {
	Owner   string
	Repo    string
	Version string // "latest" or "X.Y.Z"
	Prefix  string // install directory
	Client  *http.Client
}

// Result describes an install or check outcome.
type Result struct {
	CurrentVersion string
	TargetVersion  string
	Artifact       string
	InstallPath    string
	Changed        bool
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

// Platform returns os-arch suffix for release artifacts or an error if unsupported.
func Platform() (osName, arch, suffix string, err error) {
	osName = runtime.GOOS
	arch = runtime.GOARCH
	switch {
	case osName == "linux" && arch == "amd64":
		return osName, arch, "linux-amd64", nil
	case osName == "darwin" && arch == "arm64":
		return osName, arch, "darwin-arm64", nil
	case osName == "darwin" && arch == "amd64":
		return osName, arch, "darwin-amd64", nil
	default:
		return osName, arch, "", fmt.Errorf("unsupported platform %s/%s (release binaries: linux-amd64, darwin-amd64, darwin-arm64)", osName, arch)
	}
}

func (o Options) ownerRepo() (owner, repo string) {
	owner, repo = o.Owner, o.Repo
	if owner == "" {
		owner = defaultOwner
	}
	if repo == "" {
		repo = defaultRepo
	}
	return owner, repo
}

func (o Options) httpClient() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

// Check reports whether a newer release exists compared to currentVersion.
func Check(opts Options, currentVersion string) (*Result, error) {
	_, _, suffix, err := Platform()
	if err != nil {
		return nil, err
	}
	release, err := fetchRelease(opts, opts.Version)
	if err != nil {
		return nil, err
	}
	target := strings.TrimPrefix(release.TagName, "v")
	artifact := artifactName(target, suffix)
	if !hasAsset(release, artifact) {
		return nil, fmt.Errorf("release %s has no asset %s", release.TagName, artifact)
	}
	installPath := filepath.Join(opts.Prefix, binaryName)
	return &Result{
		CurrentVersion: currentVersion,
		TargetVersion:  target,
		Artifact:       artifact,
		InstallPath:    installPath,
		Changed:        normalizeVersion(currentVersion) != normalizeVersion(target),
	}, nil
}

// Install downloads, verifies, and replaces the CLI binary.
func Install(opts Options) (*Result, error) {
	if opts.Prefix == "" {
		return nil, fmt.Errorf("install prefix is required")
	}
	_, _, suffix, err := Platform()
	if err != nil {
		return nil, err
	}
	release, err := fetchRelease(opts, opts.Version)
	if err != nil {
		return nil, err
	}
	target := strings.TrimPrefix(release.TagName, "v")
	artifact := artifactName(target, suffix)
	checksumsURL, err := assetURL(release, "checksums.txt")
	if err != nil {
		return nil, err
	}
	binaryURL, err := assetURL(release, artifact)
	if err != nil {
		return nil, err
	}

	client := opts.httpClient()
	checksums, err := downloadBytes(client, checksumsURL)
	if err != nil {
		return nil, fmt.Errorf("download checksums: %w", err)
	}
	expected, err := parseChecksumLine(string(checksums), artifact)
	if err != nil {
		return nil, err
	}

	payload, err := downloadBytes(client, binaryURL)
	if err != nil {
		return nil, fmt.Errorf("download binary: %w", err)
	}
	if err := verifySHA256(payload, expected); err != nil {
		return nil, err
	}

	installPath := filepath.Join(opts.Prefix, binaryName)
	if err := atomicWriteExecutable(installPath, payload); err != nil {
		return nil, err
	}
	clearMacOSQuarantine(installPath)

	return &Result{
		TargetVersion: target,
		Artifact:      artifact,
		InstallPath:   installPath,
		Changed:       true,
	}, nil
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if idx := strings.IndexByte(v, '('); idx >= 0 {
		v = strings.TrimSpace(v[:idx])
	}
	return v
}

func artifactName(version, suffix string) string {
	return fmt.Sprintf("%s-%s-%s", binaryName, version, suffix)
}

func fetchRelease(opts Options, version string) (*ghRelease, error) {
	owner, repo := opts.ownerRepo()
	var url string
	v := strings.TrimSpace(version)
	if v == "" || strings.EqualFold(v, "latest") {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	} else {
		v = strings.TrimPrefix(v, "v")
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/v%s", owner, repo, v)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "aid-cisco-cli-install")

	resp, err := opts.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release JSON: %w", err)
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("release response missing tag_name")
	}
	return &release, nil
}

func hasAsset(release *ghRelease, name string) bool {
	_, err := assetURL(release, name)
	return err == nil
}

func assetURL(release *ghRelease, name string) (string, error) {
	for _, a := range release.Assets {
		if a.Name == name && a.BrowserDownloadURL != "" {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("asset %q not found in release %s", name, release.TagName)
}

func downloadBytes(client *http.Client, rawURL string) ([]byte, error) {
	if err := validateDownloadURL(rawURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "aid-cisco-cli-install")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, rawURL, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 512<<20))
}

func validateDownloadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	host := strings.ToLower(u.Hostname())
	switch u.Scheme {
	case "https":
		if host != "github.com" && host != "objects.githubusercontent.com" {
			return fmt.Errorf("refusing download from disallowed host %q", host)
		}
	case "http":
		// httptest only; production release assets use HTTPS on github.com.
		if host != "127.0.0.1" && host != "localhost" {
			return fmt.Errorf("refusing insecure download from %q", host)
		}
	default:
		return fmt.Errorf("unsupported download URL scheme %q", u.Scheme)
	}
	return nil
}

func parseChecksumLine(checksums, artifact string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimPrefix(parts[1], "*")
		if name == artifact {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum line for %s not found in checksums.txt", artifact)
}

func verifySHA256(data []byte, expectedHex string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, strings.TrimSpace(expectedHex)) {
		return fmt.Errorf("checksum mismatch for downloaded binary (expected %s, got %s)", expectedHex, got)
	}
	return nil
}

func atomicWriteExecutable(dest string, data []byte) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".aid-cisco-cli-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp binary: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp binary: %w", err)
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("replace binary at %s: %w", dest, err)
	}
	cleanup = false
	return nil
}

func clearMacOSQuarantine(path string) {
	if runtime.GOOS != "darwin" {
		return
	}
	cmd := exec.Command("xattr", "-d", "com.apple.quarantine", path)
	_ = cmd.Run()
}

// DefaultPrefix returns the directory containing the running executable.
func DefaultPrefix() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}
