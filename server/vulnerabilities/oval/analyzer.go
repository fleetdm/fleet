package oval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
	utils "github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

const (
	hostsBatchSize = 500
	vulnBatchSize  = 500
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

// Analyze scans all hosts for vulnerabilities based on the OVAL definitions for their platform,
// inserting any new vulnerabilities and deleting anything patched. Returns nil, nil when
// the platform isn't supported.
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
		return nil, ErrUnsupportedPlatform
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
		hostIDs, err := ds.HostIDsByOSVersion(ctx, ver, offset, hostsBatchSize)
		if err != nil {
			return nil, err
		}

		if len(hostIDs) == 0 {
			break
		}
		offset += hostsBatchSize

		foundInBatch := make(map[uint][]fleet.SoftwareVulnerability)
		for _, hostID := range hostIDs {
			hostID := hostID
			software, err := ds.ListSoftwareForVulnDetection(ctx, fleet.VulnSoftwareFilter{HostID: &hostID})
			// we have id, name, version, cpe available for us
			// release should be populated as well
			if err != nil {
				return nil, err
			}

			evalR, err := defs.Eval(ver, software)
			if err != nil {
				return nil, err
			}
			foundInBatch[hostID] = evalR

			evalU, err := defs.EvalKernel(software)
			if err != nil {
				return nil, err
			}
			foundInBatch[hostID] = append(foundInBatch[hostID], evalU...)

			// We only know about software in this scope
			fmt.Println("------------------------- oval batch -------------------------")
			fmt.Println(foundInBatch[hostID])
			fmt.Println()
			// we can filter by os version as well
			// we probably need at least name and version range (SemVerConstraint)
			// toFilter := make(map[uint][]fleet.SoftwareVulnerability)

			// First idea:
			// For each software, evaluate if it is a filtered software
			//    - We get software as a slice, so this makes it O(n)
			//    - Or we could make software a map O(n)
			//            then search each filter if its in software
			// Get its ID
			// For each pair in foundInBatch
			// delete pairs with ID and CVE match from filter

			// Should we match software or cpe?
			// My thought is software because RpmInfoTests don't use cpe

			// How the heck do we optimize this?

			rules, err := GetKnownOVALBugRules()
			if err != nil {
				return nil, err
			}

			softwareIDs := make(map[uint]fleet.Software)
			for _, s := range software {
				softwareIDs[s.ID] = s
			}

			updatedBatch := make([]fleet.SoftwareVulnerability, 0, len(foundInBatch[hostID]))
			fmt.Println("updatedBatch: ", updatedBatch)
			for _, v := range foundInBatch[hostID] {
				sName := softwareIDs[v.SoftwareID].Name
				sVer := softwareIDs[v.SoftwareID].Version
				skip := rules.MatchesAny(sName, sVer, v.CVE)
				if !skip {
					updatedBatch = append(updatedBatch, v)
				}
			}

			foundInBatch[hostID] = updatedBatch
			fmt.Println("------------------------- modified -------------------------")
			fmt.Println(foundInBatch[hostID])
			fmt.Println()

			// This is hoooooooorrible
			// we need to make a map of ID: Software
			// then just remove things from that
			// var newVulns []fleet.SoftwareVulnerability
			// for _, s := range software {
			// 	for _, r := range ruleList {
			// 		if s.Name == r.Name {
			// 			for _, pair := range foundInBatch[s.ID] { // this doesnt even work it should be hostID
			// 				if cve, found := r.CVEs[pair.CVE]; !found {
			// 					newVulns = append(newVulns, cve)
			// 				}
			// 			}
			// 		}
			// 	}
			// }

			// guh
		}

		existingInBatch, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, hostIDs, source)
		if err != nil {
			return nil, err
		}

		for _, hostID := range hostIDs {
			insrt, del := utils.VulnsDelta(foundInBatch[hostID], existingInBatch[hostID])
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

	err = utils.BatchProcess(toInsertSet, func(vulns []fleet.SoftwareVulnerability) error {
		for _, v := range vulns {
			ok, err := ds.InsertSoftwareVulnerability(ctx, v, source)
			if err != nil {
				return err
			}

			if collectVulns && ok {
				inserted = append(inserted, v)
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
	payload, err := os.ReadFile(latest)
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
