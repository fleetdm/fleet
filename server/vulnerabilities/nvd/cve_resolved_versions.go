package nvd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/google/go-github/v37/github"
)

const cveResolvedVersionsFilename = "cve_resolved_versions.json"

// CVEResolvedVersion supplies a known upstream fix version for a CVE whose NVD record only provides
// a versionEndIncluding constraint (so the resolved version can't be derived from the feed). Like
// the CPE translations feed, this works around bad/incomplete NVD data, but it lives in the
// vulnerability feed rather than in Go source so entries can be added or removed without a Fleet
// release. It is consulted only as a fallback - authoritative NVD data always takes precedence (see
// getMatchingVersionEndExcluding). See https://github.com/fleetdm/fleet/issues/44800.
//
// Example cve_resolved_versions.json:
//
//	[
//	  {
//	    "cve": "CVE-2025-63389",
//	    "vendor": "ollama",
//	    "product": "ollama",
//	    "resolved_in_version": "0.12.4"
//	  }
//	]
type CVEResolvedVersion struct {
	CVE               string `json:"cve"`
	Vendor            string `json:"vendor"`
	Product           string `json:"product"`
	ResolvedInVersion string `json:"resolved_in_version"`
}

// CVEResolvedVersions indexes resolved-version overrides by CVE for fast lookup during matching. A
// single CVE may have more than one entry when it affects multiple products.
type CVEResolvedVersions map[string][]CVEResolvedVersion

// Exported for testing.
func LoadCVEResolvedVersions(path string) (CVEResolvedVersions, error) {
	return loadCVEResolvedVersions(path)
}

// loadCVEResolvedVersions loads the resolved-version overrides from the given feed file. If the
// file does not exist (e.g. the feed predates this file), it returns an empty set and no error so
// callers can proceed without the overrides.
func loadCVEResolvedVersions(path string) (CVEResolvedVersions, error) {
	f, err := os.Open(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return CVEResolvedVersions{}, nil
	case err != nil:
		return nil, err
	}
	defer f.Close()

	var entries []CVEResolvedVersion
	if err := json.NewDecoder(f).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	overrides := make(CVEResolvedVersions, len(entries))
	for _, entry := range entries {
		overrides[entry.CVE] = append(overrides[entry.CVE], entry)
	}
	return overrides, nil
}

// FindResolvedVersion returns the known resolved version for the given CVE and host software, if
// one is configured and applicable, or an empty string otherwise. The caller must already have
// confirmed that the host is affected by the CVE; the version guard here only avoids reporting a
// resolved version for a host that is already at or above the fix version.
func (c CVEResolvedVersions) FindResolvedVersion(cve string, hostSoftwareMeta *wfn.Attributes) string {
	if hostSoftwareMeta == nil {
		return ""
	}
	for _, override := range c[cve] {
		if hostSoftwareMeta.Vendor != override.Vendor || hostSoftwareMeta.Product != override.Product {
			continue
		}
		if feednvd.SmartVerCmp(wfn.StripSlashes(hostSoftwareMeta.Version), override.ResolvedInVersion) != -1 {
			continue
		}
		return override.ResolvedInVersion
	}
	return ""
}

// DownloadCVEResolvedVersionsFromGithub downloads the CVE resolved-versions feed to the given
// vulnPath. If cveResolvedVersionsURL is empty, attempts to download it from the latest release of
// github.com/fleetdm/nvd. Skips downloading if the local file is newer than the release. Mirrors
// DownloadCPETranslationsFromGithub.
func DownloadCVEResolvedVersionsFromGithub(vulnPath string, cveResolvedVersionsURL string) error {
	path := filepath.Join(vulnPath, cveResolvedVersionsFilename)

	if cveResolvedVersionsURL == "" {
		stat, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// okay
		case err != nil:
			return err
		case stat.ModTime().Truncate(24 * time.Hour).Equal(time.Now().Truncate(24 * time.Hour)):
			// Vulnerability assets are published once per day - if the asset in question has a
			// mod date of 'today', then we can assume that is already up to date.
			return nil
		}

		release, asset, err := GetGithubNVDAsset(func(asset *github.ReleaseAsset) bool {
			return cveResolvedVersionsFilename == asset.GetName()
		})
		if err != nil {
			// GetGithubNVDAsset returns an error when no release contains this asset. Because this
			// feed is optional (it may not be published yet), the caller in Sync treats this error
			// as non-fatal and continues the rest of the sync.
			return err
		}
		if asset == nil {
			// Defensive: no asset and no error.
			return nil
		}
		if stat != nil && stat.ModTime().After(release.CreatedAt.Time) {
			// file is newer than release, do nothing
			return nil
		}
		cveResolvedVersionsURL = asset.GetBrowserDownloadURL()
	}

	u, err := url.Parse(cveResolvedVersionsURL)
	if err != nil {
		return err
	}
	client := fleethttp.NewGithubClient()
	if err := download.Download(client, u, path); err != nil {
		return err
	}

	return nil
}
