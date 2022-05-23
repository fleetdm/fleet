package oval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
)

// Analyze scans all hosts for vulnerabilities based on the OVAL definitions for their platform,
// inserting any new vulnerabilities and deleting anything patched.
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	versions *fleet.OSVersions,
	vulnPath string,
) ([]fleet.SoftwareVulnerability, error) {
	for _, v := range versions.OSVersions {
		platform := NewPlatform(v.Platform, v.Name)
		if !platform.IsSupported() {
			continue
		}

		defs, err := loadDef(platform, vulnPath)
		if err != nil {
			return nil, err
		}

		hIds, err := ds.HostIDsByPlatform(ctx, &v.Platform, &v.Name)
		if err != nil {
			return nil, err
		}

		for _, hId := range hIds {
			h := fleet.Host{ID: hId}
			opts := fleet.SoftwareListOptions{
				SkipLoadingCVEs: true,
				VulnerableOnly:  false,
				WithHostCounts:  false,
			}
			err := ds.LoadHostSoftware(ctx, &h, opts)
			if err != nil {
				return nil, err
			}

			found := defs.Eval(h.Software)
			if len(found) == 0 {
				return nil, nil
			}

			existing, err := ds.ListSoftwareVulnerabilities(ctx, hId)
			if err != nil {
				return nil, err
			}

			toInsert, toDelete := vulnsDelta(found, existing)
			if len(toDelete) > 0 {
				if err := ds.DeleteVulnerabilitiesByCPECVE(ctx, toDelete); err != nil {
					return nil, err
				}
			}
			if len(toInsert) > 0 {
				if _, err = ds.InsertVulnerabilities(ctx, toInsert, fleet.OVAL); err != nil {
					return nil, err
				}
				return toInsert, nil
			}
		}
	}

	return nil, nil
}

// vulnsDelta compares what vulnerabilities already exists with what new vulnerabilities were found
// and returns what to insert and what to delete.
func vulnsDelta(
	found []fleet.SoftwareVulnerability,
	existing []fleet.SoftwareVulnerability,
) ([]fleet.SoftwareVulnerability, []fleet.SoftwareVulnerability) {
	toDelete := make([]fleet.SoftwareVulnerability, 0)
	toInsert := make([]fleet.SoftwareVulnerability, 0)

	existingSet := make(map[string]bool)
	for _, e := range existing {
		existingSet[e.CVE] = true
	}

	foundSet := make(map[string]bool)
	for _, f := range found {
		foundSet[f.CVE] = true
	}

	for _, e := range existing {
		if _, ok := foundSet[e.CVE]; !ok {
			toDelete = append(toDelete, e)
		}
	}

	for _, f := range found {
		if _, ok := existingSet[f.CVE]; !ok {
			toInsert = append(toInsert, f)
		}
	}

	return toInsert, toDelete
}

// loadDef returns the latest oval Definition for the given platform.
func loadDef(platform Platform, vulnPath string) (oval_parsed.Result, error) {
	if !platform.IsUbuntu() {
		return nil, fmt.Errorf("don't know how to load OVAL file for '%s' platform", platform)
	}

	latest, err := latestOvalDefFor(platform, vulnPath, time.Now())
	if err != nil {
		return nil, err
	}
	payload, err := ioutil.ReadFile(latest)
	if err != nil {
		return nil, err
	}

	result := oval_parsed.UbuntuResult{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// latestOvalDefFor returns the contents of the OVAL definition for the given 'platform' in
// 'vulnPath' for the given 'date'.
// If not found, returns the most up to date OVAL definition for the given 'platform'
func latestOvalDefFor(platform Platform, vulnPath string, date time.Time) (string, error) {
	ext := "json"
	fileName := platform.ToFilename(date, ext)
	target := path.Join(vulnPath, fileName)

	_, err := os.Stat(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			files, err := os.ReadDir(vulnPath)
			if err != nil {
				return "", err
			}

			prefix := strings.Split(fileName, "-")[0]
			var latest os.FileInfo
			for _, f := range files {
				if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ext) {
					info, err := f.Info()
					if err != nil {
						continue
					}
					if latest == nil || info.ModTime().After(latest.ModTime()) {
						latest = info
					}
				}
			}

			if latest != nil {
				return path.Join(vulnPath, latest.Name()), nil
			}
		}
		return "", fmt.Errorf("file not found for platform '%s' in '%s'", platform, vulnPath)
	}
	return target, nil
}
