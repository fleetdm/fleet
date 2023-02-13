package macoffice

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
)

func latestReleaseNotes(vulnPath string) ([]ReleaseNote, error) {
	fs := io.NewFSClient(vulnPath)

	files, err := fs.MacOfficeReleaseNotes()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil
	}

	sort.Slice(files, func(i, j int) bool { return files[j].Before(files[i]) })

	payload, err := os.ReadFile(files[0].String())
	if err != nil {
		return nil, err
	}

	relNotes := []ReleaseNote{}
	err = json.Unmarshal(payload, &relNotes)
	if err != nil {
		return nil, err
	}
	return relNotes, nil
}

func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	collectVulns bool,
) ([]fleet.SoftwareVulnerability, error) {
	relNotes, err := latestReleaseNotes(vulnPath)
	if err != nil {
		return nil, err
	}

	if len(relNotes) == 0 {
		return nil, nil
	}

	// Ensure the release notes are sorted by release date, this is because the vuln. processing
	// algo. will stop when a release note older than the current software version is found.
	sort.Slice(relNotes, func(i, j int) bool { return relNotes[j].Date.Before(relNotes[i].Date) })

	iter, err := ds.ListSoftwareBySourceIter(ctx, []string{"apps"})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var vulnerabilities []fleet.SoftwareVulnerability
	for iter.Next() {
		software, err := iter.Value()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting software from iterator")
		}

		// If we have an Office Product ...
		if product, ok := OfficeProductFromBundleId(software.BundleIdentifier); ok {
			for _, relNote := range relNotes {
				// We only care about release notes with set versions and with security updates
				if !relNote.Valid() {
					continue
				}

				cmp := relNote.CmpVersion(software.Version)
				if cmp == -1 || cmp == 0 {
					break
				}

				for _, cve := range relNote.CollectVulnerabilities(product) {
					vulnerabilities = append(vulnerabilities, fleet.SoftwareVulnerability{
						SoftwareID: software.ID,
						CVE:        cve,
					})
				}
			}
		}
	}

	// Determine what to delete and what to insert

	return nil, nil
}
