package macoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
)

func getLatestReleaseNotes(vulnPath string) ([]ReleaseNote, error) {
	fs := io.NewFSClient(vulnPath)

	files, err := fs.MacOfficeReleaseNotes()
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Before(files[j]) })

	if len(files) != 0 {
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

	return nil, nil
}

func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	collectVulns bool,
) ([]fleet.SoftwareVulnerability, error) {
	relNotes, err := getLatestReleaseNotes(vulnPath)
	if err != nil {
		return nil, err
	}

	if len(relNotes) == 0 {
		return nil, nil
	}

	// TODO Make sure relNotes are sorted

	iter, err := ds.ListSoftwareBySourceIter(ctx, []string{"apps"})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.Next() {
		software, err := iter.Value()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting software from iterator")
		}

		if pType, ok := GetProductTypeFromBundleId(software.BundleIdentifier); ok {
			fmt.Println(pType)
			fmt.Println(ok)
		}
	}

	return nil, nil
}
