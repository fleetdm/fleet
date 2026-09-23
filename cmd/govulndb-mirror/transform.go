package main

import (
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/govulndb"
)

// The artifact's shape is govulndb.Artifact, imported from the package that reads it so the two
// sides cannot drift apart. The analyzer rejects an unexpected govulndb.SchemaVersion outright,
// so a format change needs a new version and a Fleet release that reads it before anything here
// changes.
const (
	// goEcosystem is the only OSV ecosystem the Go database publishes. Anything else is a
	// document this publisher was not written for.
	goEcosystem = "Go"

	// toolchainModule holds advisories against the `go` command. They affect the build machine,
	// not the binary it produced, so they are not mirrored.
	toolchainModule = "toolchain"

	// semverRangeType is the only range type the Go database uses. Any other type orders
	// versions by rules this publisher does not implement.
	semverRangeType = "SEMVER"

	// cvePrefix identifies the aliases Fleet can represent. software_cve and every CVE metadata
	// join are keyed on a CVE ID.
	cvePrefix = "CVE-"
)

// transform turns the collected reports into the artifact Fleet reads.
//
// Reports are dropped when they are withdrawn or carry no CVE alias, and the `toolchain` module
// is dropped wherever it appears. Everything else is a straight re-shape: a report affecting
// several modules lands under each of their keys with only that entry's ranges. Module paths are
// kept exactly as upstream reports them, because Fleet matches them by exact string against the
// path fleetd read out of the binary's build info.
func transform(reports []osvReport, generated time.Time) (*govulndb.Artifact, error) {
	modules := make(map[string][]govulndb.Advisory)

	for _, report := range reports {
		// A withdrawn report keeps its ranges, and they are typically "introduced 0" with no
		// fix. Publishing one flags every host running the module for an advisory its
		// maintainers retracted.
		if report.Withdrawn != "" {
			continue
		}

		cves := cveAliases(report.Aliases)
		if len(cves) == 0 {
			continue
		}

		// This report's advisory per module. A report can carry two affected[] entries for the
		// same module (different symbols, different ranges); those belong in one advisory, not
		// two entries with the same ID.
		byModule := make(map[string]*govulndb.Advisory, len(report.Affected))

		for _, affected := range report.Affected {
			module := affected.Package.Name
			if affected.Package.Ecosystem != goEcosystem {
				return nil, suspectf("report %s, module %s: ecosystem %q, want %q",
					report.ID, module, affected.Package.Ecosystem, goEcosystem)
			}
			if module == toolchainModule {
				continue
			}

			ranges, err := affectedRanges(report.ID, module, affected.Ranges)
			if err != nil {
				return nil, err
			}

			if advisory, ok := byModule[module]; ok {
				advisory.Ranges = append(advisory.Ranges, ranges...)
				continue
			}
			byModule[module] = &govulndb.Advisory{ID: report.ID, CVEs: cves, Ranges: ranges}
		}

		for module, advisory := range byModule {
			modules[module] = append(modules[module], *advisory)
		}
	}

	// Reports arrive sorted by ID, but a module's advisories are appended across reports; sort
	// so the same database always produces the same bytes.
	for _, advisories := range modules {
		sort.Slice(advisories, func(i, j int) bool { return advisories[i].ID < advisories[j].ID })
	}

	return &govulndb.Artifact{
		SchemaVersion: govulndb.SchemaVersion,
		Generated:     generated.UTC(),
		Modules:       modules,
	}, nil
}

// affectedRanges pairs up every range on one affected[] entry.
func affectedRanges(reportID, module string, ranges []osvRange) ([]govulndb.VersionRange, error) {
	var out []govulndb.VersionRange

	for _, r := range ranges {
		if r.Type != semverRangeType {
			return nil, suspectf("report %s, module %s: range type %q, want %q; ordering versions by any other rule is a guess",
				reportID, module, r.Type, semverRangeType)
		}

		paired, err := pairEvents(reportID, module, r.Events)
		if err != nil {
			return nil, err
		}
		out = append(out, paired...)
	}

	if len(out) == 0 {
		return nil, suspectf("report %s, module %s: no affected version ranges", reportID, module)
	}

	return out, nil
}

// pairEvents turns an OSV event list into half-open ranges: each "introduced" with the "fixed"
// that follows it, and no "fixed" when the range ends without one.
//
// The events have to alternate. Mis-pairing them shifts every bound in the report and puts wrong
// CVEs on customer hosts, so a list that does not alternate is a suspect database rather than
// something to recover from.
func pairEvents(reportID, module string, events []osvEvent) ([]govulndb.VersionRange, error) {
	var (
		ranges  []govulndb.VersionRange
		current govulndb.VersionRange
		open    bool
	)

	for i, event := range events {
		switch {
		case event.Introduced != "" && event.Fixed == "":
			if open {
				return nil, suspectf("report %s, module %s: event %d introduces %q while the range introduced at %q is still open",
					reportID, module, i, event.Introduced, current.Introduced)
			}
			current = govulndb.VersionRange{Introduced: event.Introduced}
			open = true

		case event.Fixed != "" && event.Introduced == "":
			if !open {
				return nil, suspectf("report %s, module %s: event %d fixes %q with no range open",
					reportID, module, i, event.Fixed)
			}
			current.Fixed = event.Fixed
			ranges = append(ranges, current)
			open = false

		default:
			return nil, suspectf("report %s, module %s: event %d is not a single introduced or fixed bound",
				reportID, module, i)
		}
	}

	// A trailing open range is normal: the report has no fix yet.
	if open {
		ranges = append(ranges, current)
	}

	if len(ranges) == 0 {
		return nil, suspectf("report %s, module %s: range carries no events", reportID, module)
	}

	return ranges, nil
}

// cveAliases returns the report's CVE aliases. GHSA aliases are dropped, and a report left with
// none is not mirrored at all: Fleet keys software_cve and every CVE metadata join on a CVE ID,
// so a GHSA-only report is unrepresentable there.
func cveAliases(aliases []string) []string {
	var cves []string
	for _, alias := range aliases {
		if strings.HasPrefix(alias, cvePrefix) {
			cves = append(cves, alias)
		}
	}
	return cves
}

func countAdvisories(a *govulndb.Artifact) int {
	var n int
	for _, advisories := range a.Modules {
		n += len(advisories)
	}
	return n
}
