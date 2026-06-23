// backend/updater.go
package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const CurrentVersion = "1.0.0"
const GitHubApiURL = "https://api.github.com/repos/CaptnLucid/<add-repo>/releases/latest"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HtmlURL string `json:"html_url"`
	Body    string `json:"body"`
}

type UpdateStatus struct {
	HasUpdate bool   `json:"has_update"`
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	URL       string `json:"url"`
}

func CheckForUpdates() (*UpdateStatus, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", GitHubApiURL, nil)
	req.Header.Set("User-Agent", "BDOLauncher/"+CurrentVersion)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	if release.TagName != "v"+CurrentVersion && release.TagName != CurrentVersion {
		notes := release.Body
		if len(notes) > 500 {
			notes = notes[:500] + "..."
		}
		return &UpdateStatus{
			HasUpdate: true,
			Version:   release.TagName,
			Changelog: notes,
			URL:       release.HtmlURL,
		}, nil
	}

	return &UpdateStatus{HasUpdate: false}, nil
}