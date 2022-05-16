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

// Analyze scans all hosts for vulnerabilities based on the OVAL definitions for their platform.
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	versions *fleet.OSVersions,
	vulnPath string,
) error {
	for _, v := range versions.OSVersions {
		platform := NewPlatform(v.Platform, v.Name)
		if !platform.IsSupported() {
			continue
		}

		defs, err := loadDef(platform, vulnPath)
		if err != nil {
			return err
		}

		hIds, err := ds.HostIDsByPlatform(ctx, v.Platform, v.Name)
		if err != nil {
			return err
		}

		for _, hId := range hIds {
			software, err := ds.ListSoftwareByHostIDShort(ctx, hId)
			if err != nil {
				return err
			}

			found := defs.Eval(software)
			existing, err := ds.ListSoftwareVulnerabilities(ctx, hId)
			if err != nil {
				return err
			}

			toDelete := vulnsToDelete(found, existing)
			if err := ds.DeleteVulnerabilitiesByCPECVE(ctx, toDelete); err != nil {
				return err
			}

			for sId, vulns := range found {
				if len(vulns) == 0 {
					continue
				}
				_, err = ds.InsertVulnerabilitiesForSoftwareID(ctx, sId, vulns)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func vulnsToDelete(found map[uint][]string, existing []fleet.SoftwareVulnerability) []fleet.SoftwareVulnerability {
	toDelete := make([]fleet.SoftwareVulnerability, 0)

	foundSet := make(map[string]bool)
	for _, vulns := range found {
		for _, v := range vulns {
			foundSet[v] = true
		}
	}

	for _, sv := range existing {
		if _, ok := foundSet[sv.CVE]; !ok {
			toDelete = append(toDelete, sv)
		}
	}

	return toDelete
}

// loadDef returns the latest oval Definition for the given platform.
func loadDef(platform Platform, vulnPath string) (oval_parsed.Result, error) {
	latest, err := latestOvalDefFor(platform, vulnPath, time.Now())
	if err != nil {
		return nil, err
	}
	paylaod, err := ioutil.ReadFile(latest)
	if err != nil {
		return nil, err
	}

	if platform.IsUbuntu() {
		result := oval_parsed.UbuntuResult{}
		if err := json.Unmarshal(paylaod, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, fmt.Errorf("don't know how to load OVAL file for '%s' platform", platform)
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
