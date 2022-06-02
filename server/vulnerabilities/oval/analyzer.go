package oval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
)

const (
	hostsBatchSize = 500
	vulnBatchSize  = 500
)

// Analyze scans all hosts for vulnerabilities based on the OVAL definitions for their platform,
// inserting any new vulnerabilities and deleting anything patched.
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	ver fleet.OSVersion,
	vulnPath string,
	collectVulns bool,
) ([]fleet.SoftwareVulnerability, error) {
	platform := NewPlatform(ver.Platform, ver.Name)

	if !platform.IsSupported() {
		return nil, nil
	}

	defs, err := loadDef(platform, vulnPath)
	if err != nil {
		return nil, err
	}

	// Since hosts and software have a M:N relationship, the following sets are used to
	// avoid doing duplicated inserts/delete operations (a vulnerable software might be
	// present in many hosts).
	toInsertSet := make(map[string]fleet.SoftwareVulnerability)
	toDeleteSet := make(map[string]fleet.SoftwareVulnerability)

	var offset int
	for {
		hIds, err := ds.HostIDsByOSVersion(ctx, ver, offset, hostsBatchSize)
		offset += hostsBatchSize

		if err != nil {
			return nil, err
		}

		if len(hIds) == 0 {
			break
		}

		foundInBatch := make(map[uint][]fleet.SoftwareVulnerability)
		for _, hId := range hIds {
			software, err := ds.ListSoftwareForVulnDetection(ctx, hId)
			if err != nil {
				return nil, err
			}
			foundInBatch[hId] = defs.Eval(software)
		}

		existingInBatch, err := ds.ListSoftwareVulnerabilities(ctx, hIds)
		if err != nil {
			return nil, err
		}

		for _, hId := range hIds {
			insrt, del := vulnsDelta(foundInBatch[hId], existingInBatch[hId])
			for _, i := range insrt {
				toInsertSet[i.String()] = i
			}
			for _, d := range del {
				toDeleteSet[d.String()] = d
			}
		}
	}

	err = batchProcess(toDeleteSet, func(v []fleet.SoftwareVulnerability) error {
		return ds.DeleteVulnerabilitiesByCPECVE(ctx, v)
	})
	if err != nil {
		return nil, err
	}

	var inserted []fleet.SoftwareVulnerability
	if collectVulns {
		inserted = make([]fleet.SoftwareVulnerability, 0, len(toInsertSet))
	}

	err = batchProcess(toInsertSet, func(v []fleet.SoftwareVulnerability) error {
		n, err := ds.InsertVulnerabilities(ctx, v, fleet.OVAL)
		if err != nil {
			return err
		}

		if collectVulns && n > 0 {
			inserted = append(inserted, v...)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return inserted, nil
}

func batchProcess(
	values map[string]fleet.SoftwareVulnerability,
	dsFunc func(v []fleet.SoftwareVulnerability) error,
) error {
	if len(values) == 0 {
		return nil
	}

	bSize := vulnBatchSize
	if bSize > len(values) {
		bSize = len(values)
	}

	buffer := make([]fleet.SoftwareVulnerability, bSize)
	var offset, i int
	for _, v := range values {
		buffer[offset] = v
		offset++
		i++

		// Consume buffer if full or if we are at the last iteration
		if offset == bSize || i >= len(values) {
			err := dsFunc(buffer[:offset])
			if err != nil {
				return err
			}
			offset = 0
		}
	}
	return nil
}

// vulnsDelta compares what vulnerabilities already exists with what new vulnerabilities were found
// and returns what to insert and what to delete.
func vulnsDelta(
	found []fleet.SoftwareVulnerability,
	existing []fleet.SoftwareVulnerability,
) (toInsert []fleet.SoftwareVulnerability, toDelete []fleet.SoftwareVulnerability) {
	toDelete = make([]fleet.SoftwareVulnerability, 0)
	toInsert = make([]fleet.SoftwareVulnerability, 0)

	existingSet := make(map[string]bool)
	for _, e := range existing {
		existingSet[e.String()] = true
	}

	foundSet := make(map[string]bool)
	for _, f := range found {
		foundSet[f.String()] = true
	}

	for _, e := range existing {
		if _, ok := foundSet[e.String()]; !ok {
			toDelete = append(toDelete, e)
		}
	}

	for _, f := range found {
		if _, ok := existingSet[f.String()]; !ok {
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

// latestOvalDefFor returns the path of the OVAL definition for the given 'platform' in
// 'vulnPath' for the given 'date'.
// If not found, returns the most up to date OVAL definition for the given 'platform'
func latestOvalDefFor(platform Platform, vulnPath string, date time.Time) (string, error) {
	ext := "json"
	fileName := platform.ToFilename(date, ext)
	target := filepath.Join(vulnPath, fileName)

	switch _, err := os.Stat(target); {
	case err == nil:
		return target, nil
	case errors.Is(err, fs.ErrNotExist):
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
		if latest == nil {
			return "", fmt.Errorf("file not found for platform '%s' in '%s'", platform, vulnPath)
		}
		return filepath.Join(vulnPath, latest.Name()), nil
	default:
		return "", fmt.Errorf("failed to stat %q: %w", target, err)
	}
}
