package osv

import (
	"compress/gzip"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
)

const (
	artifactExt = ".json.gz"
	dateLayout  = "2006-01-02"
)

// artifactFilename builds the name of one OSV artifact, e.g. osv-ubuntu-2204-2026-03-30.json.gz.
func artifactFilename(prefix, version string, date time.Time) string {
	return prefix + version + "-" + date.Format(dateLayout) + artifactExt
}

// isArtifact reports whether name is a non-delta OSV artifact with the given prefix. Fleet only
// consumes full artifacts; delta assets are excluded from the release listing too.
func isArtifact(name, prefix string) bool {
	return strings.HasPrefix(name, prefix) && strings.HasSuffix(name, artifactExt) && !strings.Contains(name, "delta")
}

// versionFromAssetName extracts the version segment from an OSV asset filename.
// e.g. ("osv-ubuntu-2204-2026-04-27.json.gz", "osv-ubuntu-") -> "2204".
func versionFromAssetName(name, prefix string) string {
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	rest := name[len(prefix):]
	idx := strings.Index(rest, "-")
	if idx <= 0 {
		return ""
	}
	return rest[:idx]
}

// dateFromAssetName extracts the YYYY-MM-DD date suffix from an OSV asset filename.
func dateFromAssetName(name string) (time.Time, bool) {
	s := strings.TrimSuffix(name, artifactExt)
	if len(s) < len(dateLayout) {
		return time.Time{}, false
	}
	t, err := time.Parse(dateLayout, s[len(s)-len(dateLayout):])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// releaseDateFromAssets returns the date encoded in any OSV asset filename in
// the release. All assets in a given release share the same date.
func releaseDateFromAssets(release *vulnrepo.Release) (time.Time, bool) {
	for name := range release.Assets {
		if d, ok := dateFromAssetName(name); ok {
			return d, true
		}
	}
	return time.Time{}, false
}

// readArtifact decodes the artifact for one version of an OSV family into dst and returns the
// path it read. The artifact named for date is preferred; when it is absent (data sync
// disabled, or the release for that day not yet cut) the newest one on disk is used.
func readArtifact(vulnPath, prefix, version string, date time.Time, dst any) (string, error) {
	path := filepath.Join(vulnPath, artifactFilename(prefix, version, date))
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("checking OSV artifact %s: %w", path, err)
		}
		versionPrefix := prefix + version + "-"
		path, err = vulnrepo.NewestLocalAsset(vulnPath, func(name string) bool { return isArtifact(name, versionPrefix) })
		if err != nil {
			return "", err
		}
		if path == "" {
			return "", fmt.Errorf("no OSV artifact found for %s%s", prefix, version)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening OSV artifact: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	if err := json.UnmarshalRead(gz, dst); err != nil {
		return "", fmt.Errorf("decoding OSV artifact %s: %w", filepath.Base(path), err)
	}
	return path, nil
}

// removeOldArtifacts deletes the artifacts of one OSV family in rootPath whose date is not
// date, but only for the versions in upToDateVersions. A version whose download failed or was
// not in the release keeps its last-known-good artifact.
func removeOldArtifacts(prefix string, date time.Time, rootPath string, upToDateVersions []string) error {
	dateSuffix := "-" + date.Format(dateLayout) + artifactExt

	upToDate := make(map[string]struct{}, len(upToDateVersions))
	for _, v := range upToDateVersions {
		upToDate[v] = struct{}{}
	}

	return vulnrepo.RemoveLocalAssets(rootPath, func(name string) bool {
		if !isArtifact(name, prefix) || strings.HasSuffix(name, dateSuffix) {
			return false
		}
		_, ok := upToDate[versionFromAssetName(name, prefix)]
		return ok
	})
}
