package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const releasesAPIURL = "https://api.github.com/repos/leboerF/OpenAvionicsBridge/releases?per_page=10"

type githubRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type updateResult struct {
	Available bool
	Version   string
	URL       string
}

type semVersion struct {
	major, minor, patch int
	preKind             string
	preNum              int
}

func parseSemVersion(s string) (semVersion, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "v"))
	if s == "" {
		return semVersion{}, false
	}
	parts := strings.SplitN(s, "-", 2)
	nums := strings.Split(parts[0], ".")
	if len(nums) != 3 {
		return semVersion{}, false
	}
	v := semVersion{}
	var err error
	if v.major, err = strconv.Atoi(nums[0]); err != nil {
		return semVersion{}, false
	}
	if v.minor, err = strconv.Atoi(nums[1]); err != nil {
		return semVersion{}, false
	}
	if v.patch, err = strconv.Atoi(nums[2]); err != nil {
		return semVersion{}, false
	}
	if len(parts) == 1 {
		return v, true
	}
	pre := strings.ToLower(strings.TrimSpace(parts[1]))
	if pre == "" {
		return semVersion{}, false
	}
	v.preKind = pre
	for _, prefix := range []string{"rc", "beta", "alpha"} {
		if strings.HasPrefix(pre, prefix) {
			v.preKind = prefix
			n := strings.TrimPrefix(pre, prefix)
			if n != "" {
				v.preNum, _ = strconv.Atoi(n)
			}
			break
		}
	}
	return v, true
}

func compareSemVersion(a, b semVersion) int {
	for _, p := range [][2]int{{a.major, b.major}, {a.minor, b.minor}, {a.patch, b.patch}} {
		if p[0] < p[1] {
			return -1
		}
		if p[0] > p[1] {
			return 1
		}
	}
	// A final release sorts after every prerelease of the same numeric version.
	if a.preKind == "" && b.preKind != "" {
		return 1
	}
	if a.preKind != "" && b.preKind == "" {
		return -1
	}
	if a.preKind == b.preKind {
		if a.preNum < b.preNum {
			return -1
		}
		if a.preNum > b.preNum {
			return 1
		}
		return 0
	}
	// Known prerelease stages sort in their natural progression.
	rank := func(kind string) int {
		switch kind {
		case "alpha":
			return 1
		case "beta":
			return 2
		case "rc":
			return 3
		case "":
			return 4
		default:
			return 0
		}
	}
	ra, rb := rank(a.preKind), rank(b.preKind)
	if ra < rb {
		return -1
	}
	if ra > rb {
		return 1
	}
	return strings.Compare(a.preKind, b.preKind)
}

func queryUpdates(current string) (updateResult, error) {
	currentVersion, ok := parseSemVersion(current)
	if !ok {
		return updateResult{}, fmt.Errorf("version %q is not a release version", current)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, releasesAPIURL, nil)
	if err != nil {
		return updateResult{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "OpenAvionicsBridge/"+current)

	resp, err := client.Do(req)
	if err != nil {
		return updateResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return updateResult{}, fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return updateResult{}, err
	}

	var best updateResult
	var bestVersion semVersion
	haveBest := false
	for _, rel := range releases {
		if rel.Draft || strings.TrimSpace(rel.TagName) == "" || strings.TrimSpace(rel.HTMLURL) == "" {
			continue
		}
		candidate, ok := parseSemVersion(rel.TagName)
		if !ok || compareSemVersion(candidate, currentVersion) <= 0 {
			continue
		}
		if !haveBest || compareSemVersion(candidate, bestVersion) > 0 {
			best = updateResult{Available: true, Version: strings.TrimPrefix(rel.TagName, "v"), URL: rel.HTMLURL}
			bestVersion = candidate
			haveBest = true
		}
	}
	return best, nil
}
