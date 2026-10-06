package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/minio/selfupdate"
	"golang.org/x/mod/semver"
	"local-finance/internal/db"
)

// CurrentVersion is the current version of the application.
// This can be set via go build -ldflags="-X 'local-finance/internal/updater.CurrentVersion=v1.2.0'".
var CurrentVersion = "v1.2.0"

// Default repository coordinates on GitHub
const (
	defaultOwner = "usmslm102"
	defaultRepo  = "local-finance"
)

// RestartHook is an optional callback to close the HTTP listener before the child process is launched.
type RestartHook func(ctx context.Context) error

var (
	serverPort      int
	serverDBPath    string
	restartHook     RestartHook
	serverCtxMu     sync.RWMutex
	lastTargetExe   string
	lastTargetExeMu sync.RWMutex

	restarting   atomic.Bool
	relaunchDone = make(chan struct{})
)

// IsRestarting reports whether a server restart has been triggered by the updater.
func IsRestarting() bool {
	return restarting.Load()
}

// WaitForRelaunch blocks until the replacement process has been started or a safety timeout expires.
func WaitForRelaunch() {
	select {
	case <-relaunchDone:
	case <-time.After(10 * time.Second):
	}
}

// RegisterServerContext stores running server details and a shutdown hook to cleanly release the port upon update restart.
func RegisterServerContext(port int, dbPath string, hook RestartHook) {
	serverCtxMu.Lock()
	defer serverCtxMu.Unlock()
	serverPort = port
	serverDBPath = dbPath
	restartHook = hook
}

// GetRepoCoordinates returns the GitHub owner and repository name.
func GetRepoCoordinates() (string, string) {
	owner := os.Getenv("LOCAL_FINANCE_REPO_OWNER")
	if owner == "" {
		owner = defaultOwner
	}
	repo := os.Getenv("LOCAL_FINANCE_REPO_NAME")
	if repo == "" {
		repo = defaultRepo
	}
	return owner, repo
}

// GitHubRelease represents the structure returned by GitHub's releases/latest API.
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt string        `json:"published_at"`
	Prerelease  bool          `json:"prerelease"`
	Draft       bool          `json:"draft"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a binary or file attached to a GitHub release.
type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// UpdateInfo holds the result of an update check.
type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	CanAutoUpdate   bool   `json:"can_auto_update"`
	AutoUpdateError string `json:"auto_update_error,omitempty"`
	ReleaseName     string `json:"release_name"`
	ReleaseNotes    string `json:"release_notes"`
	ReleaseURL      string `json:"release_url"`
	PublishedAt     string `json:"published_at"`
	AssetName       string `json:"asset_name,omitempty"`
	AssetURL        string `json:"asset_url,omitempty"`
	AssetSize       int64  `json:"asset_size,omitempty"`
	ChecksumURL     string `json:"checksum_url,omitempty"`
	CheckedAt       string `json:"checked_at"`
}

// Service handles checking and applying updates.
type Service struct {
	client     *http.Client
	mu         sync.Mutex
	applyMu    sync.Mutex
	cachedInfo *UpdateInfo
	cachedAt   time.Time
	cacheTTL   time.Duration
}

// NewService creates a new updater service.
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheTTL: 4 * time.Hour,
	}
}

// NormalizeVersion ensures a version string starts with 'v' and is valid semver.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// IsUpdateAvailable compares current and latest versions.
func IsUpdateAvailable(current, latest string) bool {
	normCurrent := NormalizeVersion(current)
	normLatest := NormalizeVersion(latest)

	if !semver.IsValid(normCurrent) || !semver.IsValid(normLatest) {
		// If current is "dev", allow testing when an explicit env var is set
		if current == "dev" && os.Getenv("LOCAL_FINANCE_DEV_UPDATE") == "true" {
			return true
		}
		// Fallback: If not valid semver, compare strings directly if latest != current
		return normLatest != "" && normLatest != normCurrent && semver.IsValid(normLatest)
	}

	return semver.Compare(normLatest, normCurrent) > 0
}

// GetExecutablePath resolves the canonical absolute path of the current running executable.
// Crucially, this must be evaluated BEFORE replacing the executable on disk (especially on Linux
// where /proc/self/exe points to the unlinked/renamed .old binary after selfupdate.Apply).
func GetExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot resolve executable path: %w", err)
	}

	realPath, err := filepath.EvalSymlinks(exePath)
	if err == nil && realPath != "" {
		return realPath, nil
	}
	return exePath, nil
}

// CheckExecutableWritable verifies if the running application can be modified in place.
func CheckExecutableWritable() (bool, string) {
	exePath, err := GetExecutablePath()
	if err != nil {
		return false, fmt.Sprintf("cannot resolve executable path: %v", err)
	}

	// Test if directory of the binary is writable (needed to create temp update file and replace binary)
	dir := filepath.Dir(exePath)
	testFile := filepath.Join(dir, fmt.Sprintf(".perm_test_%d", time.Now().UnixNano()))
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		// Provide a helpful hint on macOS if running from a mounted read-only volume
		if runtime.GOOS == "darwin" && strings.HasPrefix(exePath, "/Volumes/") {
			return false, "Application appears to be running from a read-only mounted disk image (DMG). Please copy LocalFinance.app to /Applications to enable one-click updates."
		}
		return false, fmt.Sprintf("application directory is not writable (%s): %v", dir, err)
	}
	_ = f.Close()
	_ = os.Remove(testFile)

	// Verify that the executable file itself has write permissions
	fExe, err := os.OpenFile(exePath, os.O_WRONLY, 0)
	if err != nil {
		if os.IsPermission(err) {
			return false, fmt.Sprintf("executable file is not writable: %v", err)
		}
	} else {
		_ = fExe.Close()
	}

	return true, ""
}

// SelectAsset matches the target platform OS/Arch with the release assets.
func SelectAsset(assets []GitHubAsset, goos, goarch string) (asset *GitHubAsset, checksumAsset *GitHubAsset) {
	for i := range assets {
		if assets[i].Name == "checksums.txt" {
			checksumAsset = &assets[i]
			break
		}
	}

	var targetName string
	switch goos {
	case "darwin":
		// Universal binary archive covers both arm64 and amd64
		targetName = "local-finance-darwin-universal.tar.gz"
	case "windows":
		targetName = "LocalFinance.exe"
	case "linux":
		if goarch == "arm64" {
			targetName = "local-finance-linux-arm64.tar.gz"
		} else {
			targetName = "local-finance-linux-amd64.tar.gz"
		}
	default:
		return nil, checksumAsset
	}

	for i := range assets {
		if assets[i].Name == targetName {
			return &assets[i], checksumAsset
		}
	}

	return nil, checksumAsset
}

// CheckForUpdate queries the GitHub Releases API for the latest release.
// Release checks are cached in memory for 4 hours to avoid rate limits and unnecessary network access.
// If offline is true, no network calls are made and only local version or cache is returned.
// When offline is false, it uses cached data within TTL, or queries GitHub Releases when cache is nil/expired.
func (s *Service) CheckForUpdate(ctx context.Context, forceRefresh bool, offline bool) (*UpdateInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return cached information if available and within TTL (unless forceRefresh)
	if !forceRefresh && s.cachedInfo != nil && time.Since(s.cachedAt) < s.cacheTTL {
		return s.cachedInfo, nil
	}

	// If offline mode is requested (e.g. user opted out of automatic background checks), do not initiate network calls
	if offline {
		if s.cachedInfo != nil {
			return s.cachedInfo, nil
		}
		writable, writeErr := CheckExecutableWritable()
		return &UpdateInfo{
			CurrentVersion:  CurrentVersion,
			LatestVersion:   "",
			UpdateAvailable: false,
			CanAutoUpdate:   writable,
			AutoUpdateError: writeErr,
			ReleaseName:     "",
			ReleaseNotes:    "",
			ReleaseURL:      "",
			PublishedAt:     "",
			CheckedAt:       "", // Empty indicates not checked
		}, nil
	}

	owner, repo := GetRepoCoordinates()
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "LocalFinance-App-Updater")

	resp, err := s.client.Do(req)
	if err != nil {
		if s.cachedInfo != nil {
			return s.cachedInfo, nil // gracefully fall back to cache on network failure
		}
		return nil, fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no releases found for repository")
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden && s.cachedInfo != nil {
			return s.cachedInfo, nil // GitHub rate limit reached, reuse cache
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub release: %w", err)
	}

	writable, writeErr := CheckExecutableWritable()
	updateAvailable := IsUpdateAvailable(CurrentVersion, release.TagName)

	matchedAsset, checksumAsset := SelectAsset(release.Assets, runtime.GOOS, runtime.GOARCH)

	info := &UpdateInfo{
		CurrentVersion:  CurrentVersion,
		LatestVersion:   release.TagName,
		UpdateAvailable: updateAvailable,
		CanAutoUpdate:   writable && matchedAsset != nil,
		AutoUpdateError: writeErr,
		ReleaseName:     release.Name,
		ReleaseNotes:    release.Body,
		ReleaseURL:      release.HTMLURL,
		PublishedAt:     release.PublishedAt,
		CheckedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	if matchedAsset != nil {
		info.AssetName = matchedAsset.Name
		info.AssetURL = matchedAsset.BrowserDownloadURL
		info.AssetSize = matchedAsset.Size
	} else if updateAvailable {
		info.CanAutoUpdate = false
		if info.AutoUpdateError == "" {
			info.AutoUpdateError = fmt.Sprintf("no pre-compiled binary available for %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	}

	if checksumAsset != nil {
		info.ChecksumURL = checksumAsset.BrowserDownloadURL
	}

	s.cachedInfo = info
	s.cachedAt = time.Now()

	return info, nil
}

// BackupDatabaseBeforeUpdate creates an automatic database backup before applying an update.
func BackupDatabaseBeforeUpdate(database *db.DB) (string, error) {
	if database == nil {
		return "", errors.New("database instance is nil")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home dir: %w", err)
	}

	backupDir := filepath.Join(homeDir, ".localfinance", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("could not create backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("local_finance_pre_update_%s_%s.db", CurrentVersion, timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	if err := database.BackupTo(backupPath); err != nil {
		return "", fmt.Errorf("failed to create database backup: %w", err)
	}

	log.Printf("📦 Pre-update database backup saved to: %s", backupPath)
	return backupPath, nil
}

// ExtractBinaryFromTarGz extracts the executable binary from a .tar.gz stream.
func ExtractBinaryFromTarGz(r io.Reader) (io.Reader, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("invalid gzip stream: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar reading error: %w", err)
		}

		base := filepath.Base(header.Name)
		if base == "local-finance" || base == "LocalFinance" || base == "LocalFinance.exe" {
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, tr); err != nil {
				return nil, fmt.Errorf("failed to extract binary from tar: %w", err)
			}
			return buf, nil
		}
	}

	return nil, errors.New("target executable not found in tar.gz archive")
}

// VerifyChecksum downloads checksums.txt and validates the SHA256 of the downloaded asset.
// Strictly fails closed: if checksum URL is missing, or asset is not listed, or hash differs, it returns an error.
func (s *Service) VerifyChecksum(ctx context.Context, checksumURL string, assetName string, assetData []byte) error {
	if checksumURL == "" {
		return errors.New("security check failed: release manifest (checksums.txt) is missing")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create checksum request: %w", err)
	}
	req.Header.Set("User-Agent", "LocalFinance-App-Updater")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download checksums.txt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums download returned status %d", resp.StatusCode)
	}

	checksumContent, err := io.ReadAll(io.LimitReader(resp.Body, 1024*64))
	if err != nil {
		return fmt.Errorf("failed to read checksums.txt: %w", err)
	}

	return VerifyChecksumFromManifest(string(checksumContent), assetName, assetData)
}

// VerifyChecksumFromManifest checks assetData against the manifest content string.
func VerifyChecksumFromManifest(manifestText string, assetName string, assetData []byte) error {
	computedHash := sha256.Sum256(assetData)
	computedHex := hex.EncodeToString(computedHash[:])

	lines := strings.Split(manifestText, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			expectedHash := strings.ToLower(fields[0])
			fileName := strings.TrimPrefix(fields[1], "*")
			if fileName == assetName {
				if !strings.EqualFold(expectedHash, computedHex) {
					return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expectedHash, computedHex)
				}
				log.Printf("🔒 Checksum verified successfully for %s", assetName)
				return nil
			}
		}
	}

	return fmt.Errorf("security check failed: asset %q not found in release checksums.txt manifest", assetName)
}

// ApplyUpdateResult holds the outcome of an update application.
type ApplyUpdateResult struct {
	PreviousVersion string `json:"previous_version"`
	NewVersion      string `json:"new_version"`
	BackupPath      string `json:"backup_path"`
	Message         string `json:"message"`
}

// ApplyUpdate downloads the release asset, backs up the database, and replaces the binary.
// Serialized via applyMu to prevent concurrent updates from corrupting binary files or scheduling duplicate restarts.
func (s *Service) ApplyUpdate(ctx context.Context, database *db.DB) (*ApplyUpdateResult, error) {
	if restarting.Load() {
		return nil, errors.New("an update is already in progress")
	}
	if !s.applyMu.TryLock() {
		return nil, errors.New("an update is already in progress")
	}
	keepLocked := false
	defer func() {
		if !keepLocked {
			s.applyMu.Unlock()
		}
	}()

	// 1. Resolve canonical target executable path BEFORE replacing anything.
	// On Linux, /proc/self/exe will point to .old after selfupdate.Apply.
	targetExe, err := GetExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve target executable path before update: %w", err)
	}

	// 2. Fetch latest release info
	info, err := s.CheckForUpdate(ctx, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to check for update: %w", err)
	}

	if !info.UpdateAvailable {
		return nil, errors.New("application is already up to date")
	}

	if !info.CanAutoUpdate {
		if info.AutoUpdateError != "" {
			return nil, fmt.Errorf("cannot auto-update: %s", info.AutoUpdateError)
		}
		return nil, errors.New("one-click update is not available for this platform or installation")
	}

	// 3. Strict verification prerequisite: release MUST have checksum manifest
	if info.ChecksumURL == "" {
		return nil, errors.New("update aborted: release does not contain a checksums.txt verification manifest")
	}

	// 4. Perform safe SQLite backup before changing binary
	var backupPath string
	if database != nil {
		backupPath, err = BackupDatabaseBeforeUpdate(database)
		if err != nil {
			return nil, fmt.Errorf("update aborted: database backup failed: %w", err)
		}
	}

	// 5. Download asset
	log.Printf("⬇️ Downloading update asset: %s", info.AssetURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "LocalFinance-App-Updater")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	assetBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download content: %w", err)
	}

	// 6. Verify checksum strictly before executing or extracting
	if err := s.VerifyChecksum(ctx, info.ChecksumURL, info.AssetName, assetBytes); err != nil {
		return nil, fmt.Errorf("update aborted due to checksum verification failure: %w", err)
	}

	// 7. Prepare binary stream (extract if .tar.gz)
	var binaryReader io.Reader
	if strings.HasSuffix(info.AssetName, ".tar.gz") {
		extracted, err := ExtractBinaryFromTarGz(bytes.NewReader(assetBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to extract binary archive: %w", err)
		}
		binaryReader = extracted
	} else {
		binaryReader = bytes.NewReader(assetBytes)
	}

	// 8. Atomically apply self-update to the pre-resolved target executable path
	opts := selfupdate.Options{
		TargetPath: targetExe,
	}

	if err := selfupdate.Apply(binaryReader, opts); err != nil {
		log.Printf("❌ Failed to apply binary update: %v, rolling back...", err)
		_ = selfupdate.RollbackError(err)
		return nil, fmt.Errorf("failed to apply binary update: %w", err)
	}

	log.Printf("✅ Application binary successfully updated in place: %s", targetExe)

	// Save target executable path for RestartServer
	lastTargetExeMu.Lock()
	lastTargetExe = targetExe
	lastTargetExeMu.Unlock()

	// Keep applyMu locked through process exit so no concurrent requests can interfere during restart
	keepLocked = true

	return &ApplyUpdateResult{
		PreviousVersion: CurrentVersion,
		NewVersion:      info.LatestVersion,
		BackupPath:      backupPath,
		Message:         fmt.Sprintf("Successfully updated to %s. Server is restarting...", info.LatestVersion),
	}, nil
}

// RestartServer safely shuts down the existing HTTP server (releasing the bound port) and relaunches
// the updated executable at the pre-resolved path with the exact same port.
func RestartServer() {
	restarting.Store(true)
	go func() {
		// 1. Allow HTTP response to be completely written and flushed to the network socket
		time.Sleep(600 * time.Millisecond)

		lastTargetExeMu.RLock()
		exe := lastTargetExe
		lastTargetExeMu.RUnlock()

		if exe == "" {
			var err error
			exe, err = GetExecutablePath()
			if err != nil {
				log.Fatalf("❌ Failed to find executable for restart: %v", err)
			}
		}

		serverCtxMu.RLock()
		port := serverPort
		dbPath := serverDBPath
		hook := restartHook
		serverCtxMu.RUnlock()

		// 2. Shut down parent HTTP server if registered, releasing port listener so child can rebind
		if hook != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = hook(shutdownCtx)
			cancel()
		}

		// 3. Construct arguments preserving current port, database path, and disable duplicate browser open
		args := []string{}
		if port > 0 {
			args = append(args, fmt.Sprintf("-port=%d", port))
		}
		if dbPath != "" {
			args = append(args, fmt.Sprintf("-db=%s", dbPath))
		}
		args = append(args, "-open=false")

		log.Printf("🚀 Relaunching server: %s %v", exe, args)
		cmd := exec.Command(exe, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			log.Fatalf("❌ Failed to restart server: %v", err)
		}

		// Signal to parent/main goroutine that child process has started
		close(relaunchDone)

		// 4. Terminate current process
		os.Exit(0)
	}()
}
