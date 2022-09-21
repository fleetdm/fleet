package oval

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
	utils "github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
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

	source := fleet.UbuntuOVALSource
	if platform.IsRedHat() {
		source = fleet.RHELOVALSource
	}

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

			evalR, err := defs.Eval(ver, software)
			if err != nil {
				return nil, err
			}
			foundInBatch[hId] = evalR
		}

		existingInBatch, err := ds.ListSoftwareVulnerabilities(ctx, hIds)
		if err != nil {
			return nil, err
		}

		for _, hId := range hIds {
			insrt, del := utils.VulnsDelta(foundInBatch[hId], existingInBatch[hId])
			for _, i := range insrt {
				toInsertSet[i.Key()] = i
			}
			for _, d := range del {
				toDeleteSet[d.Key()] = d
			}
		}
	}

	err = utils.BatchProcess(toDeleteSet, func(v []fleet.SoftwareVulnerability) error {
		return ds.DeleteSoftwareVulnerabilities(ctx, v)
	}, vulnBatchSize)
	if err != nil {
		return nil, err
	}

	var inserted []fleet.SoftwareVulnerability
	if collectVulns {
		inserted = make([]fleet.SoftwareVulnerability, 0, len(toInsertSet))
	}

	err = utils.BatchProcess(toInsertSet, func(v []fleet.SoftwareVulnerability) error {
		n, err := ds.InsertSoftwareVulnerabilities(ctx, v, source)
		if err != nil {
			return err
		}

		if collectVulns && n > 0 {
			for _, e := range v {
				inserted = append(inserted, e)
			}
		}

		return nil
	}, vulnBatchSize)
	if err != nil {
		return nil, err
	}

	return inserted, nil
}

// loadDef returns the latest oval Definition for the given platform.
func loadDef(platform Platform, vulnPath string) (oval_parsed.Result, error) {
	if !platform.IsSupported() {
		return nil, fmt.Errorf("platform %q not supported", platform)
	}

	fileName := platform.ToFilename(time.Now(), "json")
	latest, err := utils.LatestFile(fileName, vulnPath)
	if err != nil {
		return nil, err
	}
	payload, err := ioutil.ReadFile(latest)
	if err != nil {
		return nil, err
	}

	if platform.IsUbuntu() {
		result := oval_parsed.UbuntuResult{}
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	if platform.IsRedHat() {
		result := oval_parsed.RhelResult{}
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, fmt.Errorf("don't know how to parse file %q for %q platform", latest, platform)
}
