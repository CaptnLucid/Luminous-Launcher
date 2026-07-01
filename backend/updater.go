// backend/updater.go
package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var CurrentVersion = "1.0.0"
const GitHubApiURL = "https://api.github.com/repos/CaptnLucid/Luminous-Launcher/releases/latest"

// GitHubAsset maps the nested asset structure containing the direct download payload
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	HtmlURL string        `json:"html_url"`
	Body    string        `json:"body"`
	Assets  []GitHubAsset `json:"assets"` // 💡 Added to capture attached files
}

type UpdateStatus struct {
	HasUpdate   bool   `json:"has_update"`
	Version     string `json:"version"`
	Changelog   string `json:"changelog"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"` // 💡 Clean, direct binary URL for the frontend
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

	// Format versions cleanly for comparison
	cleanRemoteTag := strings.TrimPrefix(release.TagName, "v")
	cleanLocalTag := strings.TrimPrefix(CurrentVersion, "v")

	if cleanRemoteTag != cleanLocalTag {
		notes := release.Body
		if len(notes) > 500 {
			notes = notes[:500] + "..."
		}

		// Find our compiled binary asset package from the release attachments.
		// ApplyApplicationUpdate (app.go) always unpacks the downloaded file as a
		// ZIP archive and extracts the .exe from inside it, so the asset we grab
		// here must be the .zip package, not a raw .exe.
		downloadURL := ""
		for _, asset := range release.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		if downloadURL == "" {
			return nil, fmt.Errorf("latest release %s has no .zip asset to update from", release.TagName)
		}

		return &UpdateStatus{
			HasUpdate:   true,
			Version:     release.TagName,
			Changelog:   notes,
			URL:         release.HtmlURL,
			DownloadURL: downloadURL, // Send this right to App.tsx
		}, nil
	}

	return &UpdateStatus{HasUpdate: false}, nil
}