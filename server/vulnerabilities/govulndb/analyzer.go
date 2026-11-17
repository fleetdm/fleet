package govulndb

import (
	"compress/gzip"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
	"golang.org/x/mod/semver"
)

// vulnBatchSize is how many matched vulnerabilities are written per INSERT.
const vulnBatchSize = 500

// Analyze matches Go binaries and Go toolchain installs against the most recent mirrored Go
// vulnerability database artifact in vulnPath. startTime is when the vulnerability run began:
// matched rows have their updated_at refreshed, and anything older is treated as remediated.
// toolchains must be the rules the NVD pass of the same run skipped, so that every toolchain
// install is matched by exactly one of them; with nil rules, toolchain installs are left to NVD.
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	toolchains *ToolchainRules,
	collectVulns bool,
	startTime time.Time,
	logger *slog.Logger,
) ([]fleet.SoftwareVulnerability, error) {
	analysisStart := time.Now()

	artifact, err := loadLatestArtifact(ctx, vulnPath)
	if err != nil {
		return nil, err
	}
	if artifact == nil {
		logger.DebugContext(ctx, "no Go vulnerability database artifact found", "path", vulnPath)
		return nil, nil
	}

	// Matching nothing would mark every stored Go vulnerability as remediated, so refuse an
	// unknown schema or an empty module list (a corrupted or partial download).
	if artifact.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("Go vulnerability database artifact has schema version %q, want %q", artifact.SchemaVersion, SchemaVersion)
	}
	if len(artifact.Modules) == 0 {
		return nil, errors.New("Go vulnerability database artifact contains no modules (possible corrupted feed)")
	}

	var (
		found     []fleet.SoftwareVulnerability
		newVulns  []fleet.SoftwareVulnerability
		softwareN int
	)

	insertBatch := func(batch []fleet.SoftwareVulnerability) error {
		inserted, err := ds.InsertSoftwareVulnerabilities(ctx, batch, fleet.GoVulnDBSource)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting Go vulnerabilities")
		}
		if collectVulns {
			newVulns = append(newVulns, inserted...)
		}
		return nil
	}

	scan := func(
		what string,
		opts fleet.SoftwareIterQueryOptions,
		match func(*fleet.Software, *Artifact) []fleet.SoftwareVulnerability,
	) error {
		iter, err := ds.AllSoftwareIterator(ctx, opts)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "%s software iterator", what)
		}
		defer iter.Close()

		for iter.Next() {
			software, err := iter.Value()
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting software from iterator")
			}
			softwareN++

			found = append(found, match(software, artifact)...)
			if len(found) >= vulnBatchSize {
				if err := insertBatch(found); err != nil {
					return err
				}
				// A fresh slice rather than found[:0]: the datastore may hand the
				// batch's backing array back as its result.
				found = nil
			}
		}
		if err := iter.Err(); err != nil {
			return ctxerr.Wrapf(ctx, err, "iterating %s software", what)
		}
		return nil
	}

	// From here on, errors still return newVulns: those rows are inserted, so the next run sees
	// them as known and dropping them would suppress their automations forever.
	if err := scan("go binaries", fleet.SoftwareIterQueryOptions{
		IncludedSources: []string{goBinariesSource},
	}, matchSoftware); err != nil {
		return newVulns, err
	}
	if opts, ok := toolchains.iterOptions(); ok {
		if err := scan("go toolchain", opts, toolchains.match); err != nil {
			return newVulns, err
		}
	}

	if len(found) > 0 {
		if err := insertBatch(found); err != nil {
			return newVulns, err
		}
	}

	// Only delete once every insert has refreshed updated_at on the rows that still match.
	if err := ds.DeleteOutOfDateVulnerabilities(ctx, fleet.GoVulnDBSource, startTime); err != nil {
		return newVulns, ctxerr.Wrap(ctx, err, "deleting out of date Go vulnerabilities")
	}

	logger.DebugContext(ctx, "go-vulndb-analysis-done",
		"generated", artifact.Generated,
		"modules", len(artifact.Modules),
		"software", softwareN,
		"elapsed", time.Since(analysisStart),
		"found new", len(newVulns))

	return newVulns, nil
}

// matchSoftware returns the vulnerabilities the artifact reports for one Go binary row: module
// advisories by module path (extension_id) against the binary's own version, and stdlib
// advisories by the toolchain it was built with (release), since that code is compiled in.
func matchSoftware(software *fleet.Software, artifact *Artifact) []fleet.SoftwareVulnerability {
	vulns := newVulnSet(software.ID)

	if modulePath := software.ExtensionID; modulePath != "" && modulePath != ToolchainModule {
		if version, ok := moduleVersion(software.Version); ok {
			// The fix is a newer release of the same module, so it is the upgrade target.
			vulns.addAffected(artifact.Modules[modulePath], version, upgradeTo("v"))
		}
	}

	if goVersion, ok := toolchainVersion(software.Release); ok {
		// Left unset: the fix is a rebuild with a newer toolchain, not a newer release of this
		// software, which is how the UI renders the field.
		vulns.addAffected(artifact.Modules[StdlibModule], goVersion, func(string) *string { return nil })
	}

	return vulns.vulns
}

// vulnSet collects one software row's matches. A CVE matched twice keeps its first fix.
type vulnSet struct {
	softwareID uint
	seen       map[string]struct{}
	vulns      []fleet.SoftwareVulnerability
}

func newVulnSet(softwareID uint) *vulnSet {
	return &vulnSet{softwareID: softwareID, seen: make(map[string]struct{})}
}

// addAffected adds the CVEs of every advisory affecting version. resolvedIn turns the matching
// range's fix, empty when the range has none, into the row's resolved_in_version.
func (v *vulnSet) addAffected(advisories []Advisory, version string, resolvedIn func(fixed string) *string) {
	for _, advisory := range advisories {
		fixed, affected := advisory.affects(version)
		if !affected {
			continue
		}
		for _, cve := range advisory.CVEs {
			if _, ok := v.seen[cve]; ok {
				continue
			}
			v.seen[cve] = struct{}{}
			v.vulns = append(v.vulns, fleet.SoftwareVulnerability{
				SoftwareID:        v.softwareID,
				CVE:               cve,
				ResolvedInVersion: resolvedIn(fixed),
			})
		}
	}
}

// upgradeTo reports a range's fix, with prefix prepended, as the version to upgrade to.
func upgradeTo(prefix string) func(fixed string) *string {
	return func(fixed string) *string {
		if fixed == "" {
			return nil
		}
		return new(prefix + fixed)
	}
}

// affects reports whether version is inside any of the advisory's ranges, and the matching
// range's fix. version is canonical semver with a leading "v"; bounds carry none. An advisory
// with no CVE alias never matches: software_cve is keyed on CVE ID.
//
// An introduced bound of "0" means every version, not v0.0.0: `go install` records
// v0.0.0-<pseudo> for untagged modules and commits, which sorts below v0.0.0.
func (a Advisory) affects(version string) (fixed string, affected bool) {
	if len(a.CVEs) == 0 {
		return "", false
	}

	for _, r := range a.Ranges {
		if r.Introduced != "" && r.Introduced != "0" && semver.Compare(version, "v"+r.Introduced) < 0 {
			continue
		}
		if r.Fixed != "" && semver.Compare(version, "v"+r.Fixed) >= 0 {
			continue
		}
		return r.Fixed, true
	}

	return "", false
}

// moduleVersion canonicalizes a Go binary's own version. Fleet stores it as reported, leading
// "v" included, and "(devel)" for a working-tree build, which has nothing to compare against.
func moduleVersion(version string) (string, bool) {
	if version == "" || version == develVersion {
		return "", false
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return "", false
	}
	return version, true
}

// toolchainPrerelease matches the release-candidate and beta forms of a Go toolchain tag, e.g.
// "1.22rc1" or "1.22beta1".
var toolchainPrerelease = regexp.MustCompile(`^(\d+\.\d+(?:\.\d+)?)(rc|beta)(\d+)$`)

// toolchainVersion canonicalizes a Go toolchain version ("go1.21.12") into semver as
// govulncheck does: a GOEXPERIMENT suffix ("go1.21.12 X:boringcrypto") is dropped, release
// candidates and betas become prereleases ("go1.22rc1" -> "v1.22.0-rc.1") that order against
// the database's "1.22.0-0" lower bounds, and "devel ..." toolchains are skipped.
func toolchainVersion(release string) (string, bool) {
	fields := strings.Fields(release)
	if len(fields) == 0 {
		return "", false
	}
	tag := strings.TrimPrefix(fields[0], "go")
	if m := toolchainPrerelease.FindStringSubmatch(tag); m != nil {
		tag = strings.TrimPrefix(semver.Canonical("v"+m[1]), "v") + "-" + m[2] + "." + m[3]
	}
	version := "v" + tag
	if !semver.IsValid(version) {
		return "", false
	}
	return version, true
}

// loadLatestArtifact reads the newest mirrored artifact in vulnPath, or nil if the directory
// holds none.
func loadLatestArtifact(ctx context.Context, vulnPath string) (*Artifact, error) {
	latest, err := vulnrepo.NewestLocalAsset(vulnPath, isArtifactAsset)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding newest Go vulnerability database artifact")
	}
	if latest == "" {
		return nil, nil
	}
	return loadArtifact(ctx, latest)
}

// loadArtifact reads a gzipped Go vulnerability database artifact from disk.
func loadArtifact(ctx context.Context, path string) (*Artifact, error) {
	// #nosec G304 -- path is built from a Fleet-controlled vuln directory listing
	f, err := os.Open(path)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "opening %s", path)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "gunzipping %s", path)
	}
	defer gz.Close()

	var artifact Artifact
	if err := json.UnmarshalRead(gz, &artifact); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "decoding %s", path)
	}

	return &artifact, nil
}
