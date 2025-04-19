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

// CheckAndUpdateVersion проверяет версию и при необходимости обновляет бинарник
func CheckAndUpdateVersion(ctx context.Context, currentVersion string) (*CheckLastestReleaseVersionResponse, error) {
	// Получаем информацию о последнем релизе
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

	// Сравниваем версии используя semver
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

	// Если версия не актуальна, выполняем обновление
	if !resp.IsLatest {
		// Получаем путь к текущему исполняемому файлу
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to get executable path: %w", err)
		}

		// Получаем директорию установки
		installDir := filepath.Dir(exePath)

		// Создаем временную директорию для загрузки
		tempDir, err := os.MkdirTemp("", "pgxgen-update")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Определяем платформу и архитектуру
		platform := runtime.GOOS
		arch := runtime.GOARCH

		// Формируем имя ассета
		assetName := fmt.Sprintf("pgxgen_v%s_%s_%s.tar.gz", strings.TrimPrefix(latestRelease.TagName, "v"), platform, arch)

		// Получаем URL для скачивания из ассетов релиза
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

		// Скачиваем архив
		downloadResp, err := http.Get(downloadURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download new version: %w", err)
		}
		defer downloadResp.Body.Close()

		if downloadResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download: status code %d", downloadResp.StatusCode)
		}

		// Сохраняем архив
		archivePath := filepath.Join(tempDir, assetName)
		archiveFile, err := os.Create(archivePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create archive file: %w", err)
		}
		defer archiveFile.Close()

		if _, err := io.Copy(archiveFile, downloadResp.Body); err != nil {
			return nil, fmt.Errorf("failed to save archive: %w", err)
		}

		// Распаковываем архив
		cmd := exec.Command("tar", "-xzf", archivePath, "-C", tempDir)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to extract tar.gz archive: %w", err)
		}

		// Находим распакованный бинарник
		binaryName := fmt.Sprintf("pgxgen_%s_%s", platform, arch)
		if platform == "windows" {
			binaryName += ".exe"
		}
		newBinaryPath := filepath.Join(tempDir, binaryName)

		// Проверяем, что файл существует
		if _, err := os.Stat(newBinaryPath); err != nil {
			return nil, fmt.Errorf("binary not found in archive: %w", err)
		}

		// Делаем новый бинарник исполняемым
		if err := os.Chmod(newBinaryPath, 0o755); err != nil {
			return nil, fmt.Errorf("failed to make binary executable: %w", err)
		}

		// Заменяем старый бинарник новым
		oldBinaryPath := filepath.Join(installDir, "pgxgen")
		if platform == "windows" {
			oldBinaryPath += ".exe"
		}
		if err := os.Rename(newBinaryPath, oldBinaryPath); err != nil {
			return nil, fmt.Errorf("failed to replace binary: %w", err)
		}

		// Обновляем сообщение о том, что обновление прошло успешно
		resp.Message = fmt.Sprintf("Successfully updated from %s to %s", currentVersion, latestRelease.TagName)
	}

	return resp, nil
}
