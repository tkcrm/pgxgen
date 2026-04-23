package ver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tkcrm/pgxgen/pkg/logger"
	"golang.org/x/mod/semver"
)

var requestURL = "https://api.github.com/repos/tkcrm/pgxgen/releases/latest"

// CheckAndUpdateVersion checks the version and updates the binary if necessary
func CheckAndUpdateVersion(ctx context.Context, l logger.Logger, currentVersion string) error {
	l.Infof("Checking for new version (current: %s)", currentVersion)

	// Get information about the latest release
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("client: could not create request: %v", err)
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	githubResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}
	defer func() { _ = githubResp.Body.Close() }()

	body, err := io.ReadAll(githubResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var latestRelease GithubLatestRelesaseResponse
	if err := json.Unmarshal(body, &latestRelease); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	l.Infof("Latest release: %s (published at %s, %d assets)", latestRelease.TagName, latestRelease.PublishedAt.Format(time.RFC3339), len(latestRelease.Assets))

	isLatest := true

	// Compare versions using semver
	if !semver.IsValid(currentVersion) || !semver.IsValid(latestRelease.TagName) {
		return fmt.Errorf("invalid version format: current=%s, latest=%s", currentVersion, latestRelease.TagName)
	}

	cmpResult := semver.Compare(currentVersion, latestRelease.TagName)

	if cmpResult > 0 {
		l.Infof("Current version %s is ahead of latest release %s, skipping update", currentVersion, latestRelease.TagName)
		return nil
	}

	if cmpResult < 0 {
		isLatest = false
	}

	// If version is not up to date, perform update
	if isLatest {
		l.Info("Current version matches latest release, no update needed")
		return nil
	}

	l.Infof("New version available: %s -> %s, starting update", currentVersion, latestRelease.TagName)

	// Get path to current executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get installation directory
	installDir := filepath.Dir(exePath)

	// Create temporary directory for download
	tempDir, err := os.MkdirTemp("", "pgxgen-update")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Determine platform and architecture
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Form asset name
	dirName := fmt.Sprintf("pgxgen_v%s_%s_%s", strings.TrimPrefix(latestRelease.TagName, "v"), platform, arch)
	assetName := dirName + ".tar.gz"

	// Get download URL from release assets
	var downloadURL string
	for _, asset := range latestRelease.Assets {
		if assetName == asset.Name {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("asset %s not found in release", assetName)
	}

	l.Infof("Found asset download URL: %s", downloadURL)

	// Download archive
	l.Info("Downloading archive...")
	downloadResp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download new version: %w", err)
	}
	defer func() { _ = downloadResp.Body.Close() }()

	if downloadResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: status code %d", downloadResp.StatusCode)
	}

	// Save archive
	archivePath := filepath.Join(tempDir, assetName)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer func() { _ = archiveFile.Close() }()

	if _, err := io.Copy(archiveFile, downloadResp.Body); err != nil {
		return fmt.Errorf("failed to save archive: %w", err)
	}

	// Extract archive
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", tempDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract tar.gz archive: %w", err)
	}

	// Find extracted binary
	binaryName := "pgxgen"
	if platform == "windows" {
		binaryName += ".exe"
	}
	newBinaryPath := filepath.Join(tempDir, dirName, binaryName)

	// Check if file exists
	if _, err := os.Stat(newBinaryPath); err != nil {
		return fmt.Errorf("binary not found in archive: %w", err)
	}

	// Make new binary executable
	if err := os.Chmod(newBinaryPath, 0o755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Replace old binary with new one
	oldBinaryPath := filepath.Join(installDir, "pgxgen")
	if platform == "windows" {
		oldBinaryPath += ".exe"
	}

	if err := os.Rename(newBinaryPath, oldBinaryPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Update message about successful update
	l.Infof("Update complete: %s -> %s", currentVersion, latestRelease.TagName)

	return nil
}
