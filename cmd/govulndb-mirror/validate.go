package main

import (
	"errors"
	"fmt"
	"slices"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/govulndb"
)

// minNamePrefixLen keeps a toolchain install rule from claiming unrelated software by a short
// prefix.
const minNamePrefixLen = 3

// deniedToolchainSources never hold toolchain installs a version comparison can judge:
// distributions backport fixes to deb and rpm packages without changing the upstream version,
// and Go binaries have their own matcher.
var deniedToolchainSources = []string{"deb_packages", "rpm_packages", "go_binaries"}

// validate reports whether a snapshot is safe to publish.
//
// Fleet reads a database with fewer advisories as those advisories having been remediated and
// deletes the matching software_cve rows, so a snapshot that is empty, missing its most
// load-bearing key, or noticeably smaller than the last published one has to be held back rather
// than shipped. The failure mode of holding back is stale data; the failure mode of shipping is
// vulnerabilities disappearing off customer hosts.
//
// previous is the previously published artifact, or nil on the first run ever.
func validate(artifact, previous *govulndb.Artifact, maxDropPercent float64) error {
	if len(artifact.Modules) == 0 {
		return suspectf("artifact has no modules at all")
	}

	// Every snapshot of the database carries standard-library reports, and they are the ones
	// that match the most hosts. Their absence means the collection went wrong upstream of any
	// count check.
	if len(artifact.Modules[govulndb.StdlibModule]) == 0 {
		return suspectf("artifact has no %q advisories; every snapshot of the database carries some", govulndb.StdlibModule)
	}

	if err := validateToolchainInstalls(artifact.ToolchainInstalls); err != nil {
		return suspectf("artifact has unsafe toolchain install rules: %v", err)
	}

	if previous == nil {
		return nil
	}

	if err := checkDrop("module count", len(previous.Modules), len(artifact.Modules), maxDropPercent); err != nil {
		return err
	}

	return checkDrop("advisory count", countAdvisories(previous), countAdvisories(artifact), maxDropPercent)
}

// checkDrop fails when a count fell further below the previously published artifact than
// maxDropPercent allows. The message names the check and the size of the drop, because an alert
// that only says a run was skipped tells whoever reads it nothing about whether to worry.
func checkDrop(what string, was, now int, maxPercent float64) error {
	if was == 0 || now >= was {
		return nil
	}

	dropped := was - now
	percent := float64(dropped) / float64(was) * 100
	if percent <= maxPercent {
		return nil
	}

	return suspectf("%s dropped %.1f%% against the last published artifact (%d to %d, %d fewer), past the %.1f%% allowed",
		what, percent, was, now, dropped, maxPercent)
}

// validateToolchainInstalls checks the rules before they reach servers, which apply whatever is
// published. NVD skips everything the rules claim, so a broad or misdirected rule would remove
// vulnerabilities from customer hosts, and rules a server cannot apply leave Go toolchain
// installs to NVD's name matching.
func validateToolchainInstalls(installs *govulndb.ToolchainInstalls) error {
	if installs == nil {
		return errors.New("none published")
	}
	if len(installs.Rules) == 0 {
		return errors.New("no rules")
	}
	for i, r := range installs.Rules {
		if err := validateToolchainInstallRule(r); err != nil {
			return fmt.Errorf("rule %d (%s): %w", i, r.Source, err)
		}
	}
	_, err := govulndb.NewToolchainRules(installs)
	return err
}

func validateToolchainInstallRule(r govulndb.ToolchainInstallRule) error {
	switch {
	case r.Source == "":
		return errors.New("no source")
	case slices.Contains(deniedToolchainSources, r.Source):
		return fmt.Errorf("source %s cannot hold toolchain installs", r.Source)
	case len(r.Names) == 0 && len(r.NamePrefixes) == 0:
		return errors.New("no names or name prefixes")
	case slices.Contains(r.Names, ""):
		return errors.New("empty name")
	case len(r.Versions) == 0:
		return errors.New("no versions")
	}
	for _, prefix := range r.NamePrefixes {
		if len(prefix) < minNamePrefixLen {
			return fmt.Errorf("name prefix %q is shorter than %d characters", prefix, minNamePrefixLen)
		}
	}
	return nil
}
