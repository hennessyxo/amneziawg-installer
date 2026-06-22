package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const repoReleasesBase = "https://github.com/hennessyxo/amneziawg-installer/releases"

// UpdateInfo is the result of an update check (JSON-bound to the frontend).
type UpdateInfo struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	Available   bool   `json:"available"`
	DownloadURL string `json:"downloadUrl"`
	ReleaseURL  string `json:"releaseUrl"`
}

// CheckUpdate looks up the latest published release and reports whether it is
// newer than this build. The app never downloads or replaces itself — the UI
// just offers a link. Uses the /releases/latest redirect (no API, no rate limit).
func (a *App) CheckUpdate() (UpdateInfo, error) {
	latest, err := latestReleaseTag()
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("не удалось проверить обновления: %w", err)
	}
	return UpdateInfo{
		Current:     appVersion,
		Latest:      latest,
		Available:   isNewer(latest, appVersion),
		DownloadURL: guiDownloadURL(),
		ReleaseURL:  repoReleasesBase + "/tag/" + latest,
	}, nil
}

// latestReleaseTag returns the newest release tag by reading the redirect GitHub
// serves for /releases/latest (e.g. → /releases/tag/v1.2.3).
func latestReleaseTag() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // capture the 302, don't follow it
		},
	}
	resp, err := client.Get(repoReleasesBase + "/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no release found")
	}
	tag := loc[strings.LastIndex(loc, "/")+1:]
	if tag == "" {
		return "", fmt.Errorf("could not parse release tag")
	}
	return tag, nil
}

// guiDownloadURL is the stable latest-release download URL for this OS.
func guiDownloadURL() string {
	base := repoReleasesBase + "/latest/download/"
	if runtime.GOOS == "windows" {
		return base + "awg-gui-windows-amd64.exe"
	}
	return base + "awg-gui-macos.zip"
}

// isNewer reports whether semver `latest` is greater than `current`. If either is
// not a valid vX.Y.Z (e.g. a "dev" build), it returns false (no nag).
func isNewer(latest, current string) bool {
	lv, ok1 := parseVer(latest)
	cv, ok2 := parseVer(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if lv[i] != cv[i] {
			return lv[i] > cv[i]
		}
	}
	return false
}

// parseVer parses "v1.2.3" (or "1.2.3", "1.2.3-rc1") into [major, minor, patch].
func parseVer(s string) ([3]int, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var v [3]int
	for i, p := range parts {
		if idx := strings.IndexFunc(p, func(r rune) bool { return r < '0' || r > '9' }); idx >= 0 {
			p = p[:idx] // trim a pre-release/build suffix
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		v[i] = n
	}
	return v, true
}
