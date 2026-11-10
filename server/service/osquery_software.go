package service

import (
	"context"

	"github.com/Masterminds/semver/v3"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// orbitSoftwareQuery is an extra software query backed by a fleetd table whose sources no other
// query reports. While fleetd's extension is not yet registered with osqueryd, as right after a
// reboot, its discovery finds no table and osquery sends nothing for it, which the full replace
// in directIngestSoftware would read as "none installed".
type orbitSoftwareQuery struct {
	sources []string
	// An older fleetd never ran the query, so there is nothing to keep.
	minFleetdVersion *semver.Version
}

// orbitSoftwareQueries lists, by detail query name, the queries whose rows are kept when the
// query did not run. Only the sources a query alone reports are listed: ai_tools rows with its
// pseudo-source are never stored, and the rows it flags on other sources survive on their own.
var orbitSoftwareQueries = map[string]orbitSoftwareQuery{
	"software_ai_tools": {
		sources:          []string{"ai_clis", "mcp_servers", "ai_skills"},
		minFleetdVersion: semver.MustParse("1.59.0"),
	},
}

// preservedSoftwareSources returns the sources of the skipped orbit software queries that were
// sent to the host and whose table its fleetd ships. The fleetd version is read from
// host_orbit_info because orbit_info is itself an orbit table and misses the same batches.
func (svc *Service) preservedSoftwareSources(ctx context.Context, host *fleet.Host, cfg *hostDetailQueryConfig, skipped []string) []string {
	var candidates []string
	for _, name := range skipped {
		if _, ok := orbitSoftwareQueries[name]; !ok {
			continue
		}
		if dq, sent := cfg.detailQueries[name]; sent && dq.RunsForPlatform(host.Platform) {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	// A read failure keeps the rows: keeping is a no-op on a host that lacks the table.
	var fleetdVersion *semver.Version
	fleetdVersionString := "unknown"
	info, err := svc.ds.GetHostOrbitInfo(ctx, host.ID)
	switch {
	case fleet.IsNotFound(err):
		return nil
	case err != nil:
		svc.logger.ErrorContext(ctx, "reading host orbit info", "host_id", host.ID, "err", err)
	default:
		fleetdVersion, err = fleet.VersionToSemverVersion(info.Version)
		if err != nil {
			svc.logger.DebugContext(ctx, "unparseable fleetd version", "host_id", host.ID, "version", info.Version)
			return nil
		}
		fleetdVersionString = info.Version
	}

	var kept, sources []string
	for _, name := range candidates {
		oq := orbitSoftwareQueries[name]
		if fleetdVersion != nil && fleetdVersion.LessThan(oq.minFleetdVersion) {
			continue
		}
		kept = append(kept, name)
		sources = append(sources, oq.sources...)
	}
	if len(kept) > 0 {
		svc.logger.InfoContext(ctx, "orbit software queries did not run, keeping previously reported rows",
			"queries", kept,
			"sources", sources,
			"fleetd_version", fleetdVersionString,
			"host_id", host.ID,
		)
	}
	return sources
}

// mainSoftwareQueries receive the merged extra query rows and reach directIngestSoftware. Only
// one of them runs on a host.
var mainSoftwareQueries = []string{
	hostDetailQueryPrefix + "software_macos",
	hostDetailQueryPrefix + "software_windows",
	hostDetailQueryPrefix + "software_linux",
}

func hasMainSoftwareResults(results fleet.OsqueryDistributedQueryResults) bool {
	for _, query := range mainSoftwareQueries {
		if _, ok := results[query]; ok {
			return true
		}
	}
	return false
}
