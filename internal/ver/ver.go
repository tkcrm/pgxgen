package ver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var requestURL = "https://api.github.com/repos/tkcrm/pgxgen/releases/latest"

func CheckLastestReleaseVersion(ctx context.Context, currentVersion string) (*CheckLastestReleaseVersionResponse, error) {
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

	parsedCurrentVersion, err := getFloatFromTag(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("get float from tag '%s' error: %w", currentVersion, err)
	}

	parsedGHLVersion, err := getFloatFromTag(latestRelease.TagName)
	if err != nil {
		return nil, fmt.Errorf("get float from tag '%s' error: %w", latestRelease.TagName, err)
	}

	if parsedCurrentVersion >= parsedGHLVersion {
		return resp, nil
	}

	if currentVersion != latestRelease.TagName {
		resp.IsLatest = false
		resp.Message = fmt.Sprintf(MessageTemplate, currentVersion, latestRelease.TagName)
	}

	return resp, nil
}

func getFloatFromTag(tag string) (float64, error) {
	re := regexp.MustCompile(`[\d\.]+`)

	var strFloat string
	for index, item := range strings.Split(re.FindString(tag), ".") {
		strFloat += item
		if index == 0 {
			strFloat += "."
		}
	}

	parsedFloat, err := strconv.ParseFloat(strFloat, 64)
	if err != nil {
		return 0, err
	}

	return parsedFloat, nil
}
