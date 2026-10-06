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

// CheckExecutableWritable verifies if the running application can be modified in place.
func CheckExecutableWritable() (bool, string) {
	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Sprintf("cannot resolve executable path: %v", err)
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = realPath
	}

	// On macOS, if running directly from a mounted volume (like a read-only DMG)
	if runtime.GOOS == "darwin" && strings.HasPrefix(exePath, "/Volumes/") {
		return false, "Application is running from a mounted disk image (DMG). Please copy LocalFinance.app to /Applications to enable one-click updates."
	}

	// Test if directory of the binary is writable (needed to create temp update file)
	dir := filepath.Dir(exePath)
	testFile := filepath.Join(dir, fmt.Sprintf(".perm_test_%d", time.Now().UnixNano()))
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return false, fmt.Sprintf("application directory is not writable (%s): %v", dir, err)
	}
	_ = f.Close()
	_ = os.Remove(testFile)

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
func (s *Service) CheckForUpdate(ctx context.Context, forceRefresh bool) (*UpdateInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !forceRefresh && s.cachedInfo != nil && time.Since(s.cachedAt) < s.cacheTTL {
		return s.cachedInfo, nil
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
func (s *Service) VerifyChecksum(ctx context.Context, checksumURL string, assetName string, assetData []byte) error {
	if checksumURL == "" {
		return nil // No checksum file present in release, skip
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

	computedHash := sha256.Sum256(assetData)
	computedHex := hex.EncodeToString(computedHash[:])

	// Format of checksums.txt: "<hash>  <filename>" or "<hash> *<filename>"
	lines := strings.Split(string(checksumContent), "\n")
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

	log.Printf("⚠️ Asset %s not found in checksums.txt, skipping strict checksum validation", assetName)
	return nil
}

// ApplyUpdateResult holds the outcome of an update application.
type ApplyUpdateResult struct {
	PreviousVersion string `json:"previous_version"`
	NewVersion      string `json:"new_version"`
	BackupPath      string `json:"backup_path"`
	Message         string `json:"message"`
}

// ApplyUpdate downloads the release asset, backs up the database, and replaces the binary.
func (s *Service) ApplyUpdate(ctx context.Context, database *db.DB) (*ApplyUpdateResult, error) {
	// 1. Fetch latest release info
	info, err := s.CheckForUpdate(ctx, true)
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

	// 2. Perform safe SQLite backup before changing binary
	var backupPath string
	if database != nil {
		backupPath, err = BackupDatabaseBeforeUpdate(database)
		if err != nil {
			return nil, fmt.Errorf("update aborted: database backup failed: %w", err)
		}
	}

	// 3. Download asset
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
		return nil, fmt.Errorf("failed to read downloaded update: %w", err)
	}

	// 4. Verify checksum
	if err := s.VerifyChecksum(ctx, info.ChecksumURL, info.AssetName, assetBytes); err != nil {
		return nil, fmt.Errorf("security check failed: %w", err)
	}

	// 5. Extract binary if tar.gz
	var binaryReader io.Reader = bytes.NewReader(assetBytes)
	if strings.HasSuffix(info.AssetName, ".tar.gz") {
		extracted, err := ExtractBinaryFromTarGz(bytes.NewReader(assetBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to extract binary: %w", err)
		}
		binaryReader = extracted
	}

	// 6. Apply atomic update to the running binary
	log.Println("🔄 Applying binary update to current executable...")
	if err := selfupdate.Apply(binaryReader, selfupdate.Options{}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return nil, fmt.Errorf("update failed and rollback failed: %v (rollback err: %v)", err, rerr)
		}
		return nil, fmt.Errorf("update failed (rolled back): %w", err)
	}

	log.Printf("✅ LocalFinance updated successfully from %s to %s", CurrentVersion, info.LatestVersion)

	return &ApplyUpdateResult{
		PreviousVersion: CurrentVersion,
		NewVersion:      info.LatestVersion,
		BackupPath:      backupPath,
		Message:         fmt.Sprintf("Successfully updated to %s. Server is restarting...", info.LatestVersion),
	}, nil
}

// RestartServer re-launches the updated executable and terminates the current process.
func RestartServer() {
	go func() {
		// Allow HTTP response to be completely sent to the browser
		time.Sleep(1200 * time.Millisecond)

		exe, err := os.Executable()
		if err != nil {
			log.Fatalf("❌ Failed to find executable for restart: %v", err)
		}

		// Resolve any symlinks
		if realPath, err := filepath.EvalSymlinks(exe); err == nil {
			exe = realPath
		}

		log.Printf("🚀 Relaunching server: %s %v", exe, os.Args[1:])
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			log.Fatalf("❌ Failed to restart server: %v", err)
		}

		// Exit current process
		os.Exit(0)
	}()
}
