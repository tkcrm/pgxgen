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

	"golang.org/x/mod/semver"
)

var requestURL = "https://api.github.com/repos/tkcrm/pgxgen/releases/latest"

// CheckAndUpdateVersion checks the version and updates the binary if necessary
func CheckAndUpdateVersion(ctx context.Context, currentVersion string) (*CheckLastestReleaseVersionResponse, error) {
	// Get information about the latest release
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("client: could not create request: %v", err)
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	githubResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer githubResp.Body.Close()

	body, err := io.ReadAll(githubResp.Body)
	if err != nil {
		return nil, err
	}

	var latestRelease GithubLatestRelesaseResponse
	if err := json.Unmarshal(body, &latestRelease); err != nil {
		return nil, err
	}

	resp := &CheckLastestReleaseVersionResponse{
		IsLatest:                    true,
		CurrentVersion:              currentVersion,
		GithubLatestRelesaseVersion: latestRelease.TagName,
	}

	// Compare versions using semver
	if !semver.IsValid(currentVersion) || !semver.IsValid(latestRelease.TagName) {
		return nil, fmt.Errorf("invalid version format: current=%s, latest=%s", currentVersion, latestRelease.TagName)
	}

	if semver.Compare(currentVersion, latestRelease.TagName) > 0 {
		return resp, nil
	}

	if semver.Compare(currentVersion, latestRelease.TagName) < 0 {
		resp.IsLatest = false
		resp.Message = fmt.Sprintf(MessageTemplate, currentVersion, latestRelease.TagName)
	}

	// If version is not up to date, perform update
	if !resp.IsLatest {
		// Get path to current executable
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get executable path: %w", err)
		}

		// Get installation directory
		installDir := filepath.Dir(exePath)

		// Create temporary directory for download
		tempDir, err := os.MkdirTemp("", "pgxgen-update")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Determine platform and architecture
		platform := runtime.GOOS
		arch := runtime.GOARCH

		// Form asset name
		assetName := fmt.Sprintf("pgxgen_v%s_%s_%s.tar.gz", strings.TrimPrefix(latestRelease.TagName, "v"), platform, arch)

		// Get download URL from release assets
		var downloadURL string
		for _, asset := range latestRelease.Assets {
			if assetName == asset.Name {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		if downloadURL == "" {
			return nil, fmt.Errorf("asset %s not found in release", assetName)
		}

		// Download archive
		downloadResp, err := http.Get(downloadURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download new version: %w", err)
		}
		defer downloadResp.Body.Close()

		if downloadResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download: status code %d", downloadResp.StatusCode)
		}

		// Save archive
		archivePath := filepath.Join(tempDir, assetName)
		archiveFile, err := os.Create(archivePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create archive file: %w", err)
		}
		defer archiveFile.Close()

		if _, err := io.Copy(archiveFile, downloadResp.Body); err != nil {
			return nil, fmt.Errorf("failed to save archive: %w", err)
		}

		// Extract archive
		cmd := exec.Command("tar", "-xzf", archivePath, "-C", tempDir)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to extract tar.gz archive: %w", err)
		}

		// Find extracted binary
		binaryName := fmt.Sprintf("pgxgen_%s_%s", platform, arch)
		if platform == "windows" {
			binaryName += ".exe"
		}
		newBinaryPath := filepath.Join(tempDir, binaryName)

		// Check if file exists
		if _, err := os.Stat(newBinaryPath); err != nil {
			return nil, fmt.Errorf("binary not found in archive: %w", err)
		}

		// Make new binary executable
		if err := os.Chmod(newBinaryPath, 0o755); err != nil {
			return nil, fmt.Errorf("failed to make binary executable: %w", err)
		}

		// Replace old binary with new one
		oldBinaryPath := filepath.Join(installDir, "pgxgen")
		if platform == "windows" {
			oldBinaryPath += ".exe"
		}
		if err := os.Rename(newBinaryPath, oldBinaryPath); err != nil {
			return nil, fmt.Errorf("failed to replace binary: %w", err)
		}

		// Update message about successful update
		resp.Message = fmt.Sprintf("Successfully updated from %s to %s", currentVersion, latestRelease.TagName)
	}

	return resp, nil
}
