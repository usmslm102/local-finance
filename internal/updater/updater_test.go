package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"  v2.1.3  ", "v2.1.3"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeVersion(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeVersion(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsUpdateAvailable(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "v1.1.0", true},
		{"v1.0.0", "v2.0.0", true},
		{"1.0.0", "1.0.1", true},
		{"v1.1.0", "v1.1.0", false},
		{"v1.2.0", "v1.1.0", false},
		{"v2.0.0", "v1.9.9", false},
	}

	for _, tt := range tests {
		got := IsUpdateAvailable(tt.current, tt.latest)
		if got != tt.expected {
			t.Errorf("IsUpdateAvailable(%q, %q) = %v, expected %v", tt.current, tt.latest, got, tt.expected)
		}
	}
}

func TestSelectAsset(t *testing.T) {
	assets := []GitHubAsset{
		{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
		{Name: "local-finance-darwin-universal.tar.gz", BrowserDownloadURL: "https://example.com/darwin.tar.gz"},
		{Name: "LocalFinance.dmg", BrowserDownloadURL: "https://example.com/darwin.dmg"},
		{Name: "LocalFinance.exe", BrowserDownloadURL: "https://example.com/windows.exe"},
		{Name: "local-finance-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64.tar.gz"},
		{Name: "local-finance-linux-arm64.tar.gz", BrowserDownloadURL: "https://example.com/linux-arm64.tar.gz"},
	}

	// Darwin
	asset, checksum := SelectAsset(assets, "darwin", "arm64")
	if asset == nil || asset.Name != "local-finance-darwin-universal.tar.gz" {
		t.Fatalf("expected darwin asset, got %v", asset)
	}
	if checksum == nil || checksum.Name != "checksums.txt" {
		t.Fatalf("expected checksum asset, got %v", checksum)
	}

	// Windows
	asset, _ = SelectAsset(assets, "windows", "amd64")
	if asset == nil || asset.Name != "LocalFinance.exe" {
		t.Fatalf("expected windows asset, got %v", asset)
	}

	// Linux amd64
	asset, _ = SelectAsset(assets, "linux", "amd64")
	if asset == nil || asset.Name != "local-finance-linux-amd64.tar.gz" {
		t.Fatalf("expected linux-amd64 asset, got %v", asset)
	}

	// Linux arm64
	asset, _ = SelectAsset(assets, "linux", "arm64")
	if asset == nil || asset.Name != "local-finance-linux-arm64.tar.gz" {
		t.Fatalf("expected linux-arm64 asset, got %v", asset)
	}
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := []byte("#!/bin/sh\necho 'hello local finance'")
	hdr := &tar.Header{
		Name: "local-finance",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	_ = tw.Close()
	_ = gzw.Close()

	// Extract
	extractedReader, err := ExtractBinaryFromTarGz(&buf)
	if err != nil {
		t.Fatalf("ExtractBinaryFromTarGz failed: %v", err)
	}

	extractedBytes, err := io.ReadAll(extractedReader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(extractedBytes, content) {
		t.Errorf("extracted content does not match expected, got %s", string(extractedBytes))
	}
}

func TestVerifyChecksumStrictFailsClosed(t *testing.T) {
	sampleData := []byte("binary payload content")
	hash := sha256.Sum256(sampleData)
	hashHex := hex.EncodeToString(hash[:])

	checksumsContent := hashHex + "  local-finance-darwin-universal.tar.gz\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(checksumsContent))
	}))
	defer server.Close()

	svc := NewService()
	svc.client = server.Client()

	// 1. Valid checksum
	err := svc.VerifyChecksum(context.Background(), server.URL, "local-finance-darwin-universal.tar.gz", sampleData)
	if err != nil {
		t.Fatalf("VerifyChecksum failed on valid data: %v", err)
	}

	// 2. Missing checksum URL must FAIL CLOSED
	err = svc.VerifyChecksum(context.Background(), "", "local-finance-darwin-universal.tar.gz", sampleData)
	if err == nil {
		t.Fatal("expected error on empty checksum URL, got nil")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("expected missing manifest error, got %v", err)
	}

	// 3. Manifest missing the asset must FAIL CLOSED
	err = svc.VerifyChecksum(context.Background(), server.URL, "nonexistent-asset.tar.gz", sampleData)
	if err == nil {
		t.Fatal("expected error when asset is not in checksum manifest, got nil")
	}
	if !strings.Contains(err.Error(), "not found in release checksums.txt") {
		t.Errorf("expected not-found error, got %v", err)
	}

	// 4. Invalid/tampered checksum must FAIL
	err = svc.VerifyChecksum(context.Background(), server.URL, "local-finance-darwin-universal.tar.gz", []byte("tampered content"))
	if err == nil {
		t.Fatal("expected error on tampered content, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got %v", err)
	}
}

func TestOfflineOptOutCheck(t *testing.T) {
	svc := NewService()

	// When offline is requested, CheckForUpdate must NOT make any network calls
	info, err := svc.CheckForUpdate(context.Background(), false, true)
	if err != nil {
		t.Fatalf("unexpected error on offline check: %v", err)
	}
	if info.CheckedAt != "" {
		t.Errorf("expected CheckedAt to be empty for offline check, got %q", info.CheckedAt)
	}
	if info.UpdateAvailable {
		t.Errorf("expected UpdateAvailable to be false for offline check, got true")
	}
}

func TestCheckForUpdateWithMock(t *testing.T) {
	mockRelease := GitHubRelease{
		TagName:     "v1.3.0",
		Name:        "v1.3.0 New Feature",
		Body:        "Awesome update notes",
		HTMLURL:     "https://github.com/usmslm102/local-finance/releases/tag/v1.3.0",
		PublishedAt: "2026-10-06T12:00:00Z",
		Assets: []GitHubAsset{
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
			{Name: "local-finance-darwin-universal.tar.gz", BrowserDownloadURL: "https://example.com/darwin.tar.gz", Size: 12345},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockRelease)
	}))
	defer server.Close()

	CurrentVersion = "v1.2.0"

	svc := NewService()
	svc.client = server.Client()

	transport := server.Client().Transport
	svc.client.Transport = &mockTransport{
		targetURL: server.URL,
		base:      transport,
	}

	info, err := svc.CheckForUpdate(context.Background(), true, false)
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}

	if !info.UpdateAvailable {
		t.Errorf("expected update to be available, got false")
	}
	if info.LatestVersion != "v1.3.0" {
		t.Errorf("expected latest version v1.3.0, got %s", info.LatestVersion)
	}
	if info.ReleaseName != "v1.3.0 New Feature" {
		t.Errorf("expected release name 'v1.3.0 New Feature', got %s", info.ReleaseName)
	}
}

type mockTransport struct {
	targetURL string
	base      http.RoundTripper
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	mockReq, _ := http.NewRequestWithContext(req.Context(), req.Method, m.targetURL, req.Body)
	mockReq.Header = req.Header
	if m.base != nil {
		return m.base.RoundTrip(mockReq)
	}
	return http.DefaultTransport.RoundTrip(mockReq)
}

func TestCheckExecutableWritableCurrentEnv(t *testing.T) {
	writable, reason := CheckExecutableWritable()
	// In the test runner environment, either it's writable or has a clear error explanation
	t.Logf("CheckExecutableWritable returned writable=%v, reason=%q", writable, reason)
}

func TestRestartStateSync(t *testing.T) {
	if IsRestarting() {
		t.Error("expected IsRestarting to initially be false")
	}
}

func TestMacOSAppBundleCannotReplaceOnlyExecutable(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "LocalFinance.app", "Contents", "MacOS", "LocalFinance")
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	original := []byte("existing signed executable")
	if err := os.WriteFile(exe, original, 0755); err != nil {
		t.Fatal(err)
	}
	writable, reason := checkExecutableWritable(exe, "darwin")
	if writable {
		t.Fatal("one-click binary replacement must not be offered for a macOS app bundle")
	}
	if !strings.Contains(reason, "DMG") {
		t.Fatalf("expected complete-app download guidance, got %q", reason)
	}
	after, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("capability check modified the installed app")
	}
}

func TestStandaloneExecutableCanStillUpdate(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			exe := filepath.Join(t.TempDir(), "local-finance")
			if err := os.WriteFile(exe, []byte("standalone executable"), 0755); err != nil {
				t.Fatal(err)
			}
			writable, reason := checkExecutableWritable(exe, goos)
			if !writable {
				t.Fatalf("standalone update disabled: %s", reason)
			}
		})
	}
}

func TestSelectInstallationAsset(t *testing.T) {
	assets := []GitHubAsset{{Name: "local-finance-darwin-universal.tar.gz"}, {Name: "LocalFinance.dmg"}, {Name: "checksums.txt"}}
	bundleExe := filepath.Join("/Applications", "LocalFinance.app", "Contents", "MacOS", "LocalFinance")
	asset, checksum := SelectInstallationAsset(assets, "darwin", "arm64", bundleExe)
	if asset == nil || asset.Name != "LocalFinance.dmg" {
		t.Fatalf("expected complete DMG for app bundle, got %v", asset)
	}
	if checksum == nil {
		t.Fatal("missing checksum manifest")
	}
	asset, _ = SelectInstallationAsset(assets[:1], "darwin", "arm64", bundleExe)
	if asset != nil {
		t.Fatal("must not fall back to an executable archive for an app bundle")
	}
	asset, _ = SelectInstallationAsset(assets, "darwin", "amd64", filepath.Join("/usr/local/bin", "local-finance"))
	if asset == nil || asset.Name != "local-finance-darwin-universal.tar.gz" {
		t.Fatalf("standalone CLI must retain binary archive, got %v", asset)
	}
}
