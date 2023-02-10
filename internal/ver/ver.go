package ver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var requestURL = "https://api.github.com/repos/tkcrm/pgxgen/releases/latest"

func CheckLastestReleaseVersion(currentVersion string) (*CheckLastestReleaseVersionResponse, error) {
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
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

	// re := regexp.MustCompile(`\d[\d.]+`)
	// parsedCurrentVersion := re.FindString(currentVersion)
	// parsedGHLVersion := re.FindString(latestRelease.TagName)

	resp := CheckLastestReleaseVersionResponse{
		IsLatest:                    true,
		CurrentVersion:              currentVersion,
		GithubLatestRelesaseVersion: latestRelease.TagName,
	}

	if currentVersion != latestRelease.TagName {
		resp.IsLatest = false
		resp.Message = fmt.Sprintf(MessageTemplate, currentVersion, latestRelease.TagName)
	}

	return &resp, nil
}
