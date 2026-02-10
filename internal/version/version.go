package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Version is set at build time via -ldflags "-X github.com/madicen/jj-tui/internal/version.Version=v1.0.0"
var Version = "dev"

// GitHubRepo is the repository to check for updates
const GitHubRepo = "madicen/jj-tui"

// UpdateInfo holds information about available updates
type UpdateInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
	CheckedAt       time.Time
	Error           error
}

var (
	cachedUpdateInfo *UpdateInfo
	updateMutex      sync.RWMutex
	checkInProgress  bool
)

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// GetUpdateInfo returns cached update information (non-blocking)
func GetUpdateInfo() *UpdateInfo {
	updateMutex.RLock()
	defer updateMutex.RUnlock()
	return cachedUpdateInfo
}

// CheckForUpdates checks GitHub for a newer version (async, caches result)
func CheckForUpdates(ctx context.Context) {
	updateMutex.Lock()
	if checkInProgress {
		updateMutex.Unlock()
		return
	}
	checkInProgress = true
	updateMutex.Unlock()

	go func() {
		defer func() {
			updateMutex.Lock()
			checkInProgress = false
			updateMutex.Unlock()
		}()

		info := &UpdateInfo{
			CurrentVersion: Version,
			CheckedAt:      time.Now(),
		}

		// Don't check for updates in dev mode
		if Version == "dev" {
			info.LatestVersion = "dev"
			info.UpdateAvailable = false
			updateMutex.Lock()
			cachedUpdateInfo = info
			updateMutex.Unlock()
			return
		}

		// Create request with timeout
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo)
		req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
		if err != nil {
			info.Error = err
			updateMutex.Lock()
			cachedUpdateInfo = info
			updateMutex.Unlock()
			return
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "jj-tui/"+Version)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			info.Error = err
			updateMutex.Lock()
			cachedUpdateInfo = info
			updateMutex.Unlock()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			info.Error = fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
			updateMutex.Lock()
			cachedUpdateInfo = info
			updateMutex.Unlock()
			return
		}

		var release struct {
			TagName string `json:"tag_name"`
			HTMLURL string `json:"html_url"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			info.Error = err
			updateMutex.Lock()
			cachedUpdateInfo = info
			updateMutex.Unlock()
			return
		}

		info.LatestVersion = release.TagName
		info.ReleaseURL = release.HTMLURL
		info.UpdateAvailable = isNewerVersion(release.TagName, Version)

		updateMutex.Lock()
		cachedUpdateInfo = info
		updateMutex.Unlock()
	}()
}

// isNewerVersion compares two semantic version strings
// Returns true if latest is newer than current
func isNewerVersion(latest, current string) bool {
	// Strip 'v' prefix if present
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	// Simple string comparison for now (works for semver)
	// For more robust comparison, could use a semver library
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	// Compare each part
	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	// If all compared parts are equal, longer version is newer
	return len(latestParts) > len(currentParts)
}

