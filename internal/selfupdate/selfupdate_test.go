package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPlatformSupported(t *testing.T) {
	_, _, suffix, err := Platform()
	if err != nil {
		t.Skipf("platform not supported in this test environment: %v", err)
	}
	if suffix == "" {
		t.Fatal("expected non-empty suffix")
	}
}

func TestParseChecksumLine(t *testing.T) {
	checksums := "abc123  aid-cisco-cli-1.0.0-linux-amd64\n"
	got, err := parseChecksumLine(checksums, "aid-cisco-cli-1.0.0-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if got != "abc123" {
		t.Fatalf("got %q", got)
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("hello")
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := verifySHA256(data, want); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(data, "deadbeef"); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestInstallFromRelease(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("install integration test runs on linux-amd64 only")
	}

	const version = "0.9.0"
	artifact := artifactName(version, "linux-amd64")
	payload := []byte("#!/bin/sh\necho test\n")
	sum := sha256Hex(payload)

	mux := http.NewServeMux()
	assetHost := ""
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v0.9.0",
			"assets": [
				{"name": "checksums.txt", "browser_download_url": "http://` + assetHost + `/checksums.txt"},
				{"name": "` + artifact + `", "browser_download_url": "http://` + assetHost + `/bin"}
			]
		}`))
	})
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sum + "  " + artifact + "\n"))
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	assetHost = srv.Listener.Addr().String()

	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = srv.Listener.Addr().String()
		if req.URL.Path == "/repos/cisco-ai-defense/ai-defense-cli/releases/latest" {
			req.URL.Path = "/repos/o/r/releases/latest"
		}
		return http.DefaultTransport.RoundTrip(req)
	})
	client := &http.Client{Transport: transport}

	dir := t.TempDir()
	res, err := Install(Options{
		Version: "latest",
		Prefix:  dir,
		Client:  client,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.TargetVersion != version {
		t.Fatalf("version %q", res.TargetVersion)
	}
	dest := filepath.Join(dir, binaryName)
	st, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&0o111 == 0 {
		t.Fatal("expected executable bit")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestValidateDownloadURL(t *testing.T) {
	if err := validateDownloadURL("https://github.com/o/r/releases/download/v1/bin"); err != nil {
		t.Fatal(err)
	}
	if err := validateDownloadURL("https://objects.githubusercontent.com/github-production-release-asset-2e65be/o/r/1/bin"); err != nil {
		t.Fatal(err)
	}
	if err := validateDownloadURL("https://evil.example/bin"); err == nil {
		t.Fatal("expected reject for evil host")
	}
	if err := validateDownloadURL("http://127.0.0.1/bin"); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeVersion(t *testing.T) {
	if got := normalizeVersion("0.1.2 (abc123)"); got != "0.1.2" {
		t.Fatalf("got %q", got)
	}
}
