// Package vulntest provides shared test helpers for vulnerability scanning integration tests.
// Tests here are source-agnostic — the same software fixtures and expected CVE lists can
// verify both OVAL and OSV scanners via the Scanner abstraction.
package vulntest

import (
	"compress/bzip2"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestdataRoot is the path to the shared testdata directory, relative to vulntest/.
var TestdataRoot = filepath.Join("..", "testdata")

// Scanner abstracts a vulnerability scanning backend so the same test fixtures
// can be run against OVAL, OSV, or goval-dictionary without modification.
type Scanner struct {
	// Name identifies this scanner for subtest naming (e.g. "oval", "osv").
	Name string
	// Source is the VulnerabilitySource this scanner writes to the DB.
	Source fleet.VulnerabilitySource
	// Setup loads scanner-specific definition artifacts into vulnPath
	// for the given OS version. Called before Analyze.
	Setup func(t *testing.T, vulnPath string, ver fleet.OSVersion)
	// Analyze runs the vulnerability scanner against the datastore.
	Analyze func(ctx context.Context, ds fleet.Datastore, ver fleet.OSVersion, vulnPath string) ([]fleet.SoftwareVulnerability, error)
}

// SoftwareFixture represents a single software entry in a test fixture JSON file.
type SoftwareFixture struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Release string `json:"release"`
	Arch    string `json:"arch"`
}

// ExpectedVuln maps a software package to the CVEs it should trigger.
type ExpectedVuln struct {
	Software SoftwareFixture `json:"software"`
	CVEs     []string        `json:"cves"`
}

// VulnFixture maps software packages to the CVEs each should produce.
// A single fixture file is shared across all scanners for a given OS version.
type VulnFixture struct {
	Vulnerabilities []ExpectedVuln `json:"vulnerabilities"`
}

// AllCVEs returns a sorted, deduplicated list of all expected CVE IDs.
func (f *VulnFixture) AllCVEs() []string {
	seen := make(map[string]struct{})
	for _, v := range f.Vulnerabilities {
		for _, cve := range v.CVEs {
			seen[cve] = struct{}{}
		}
	}
	cves := make([]string, 0, len(seen))
	for cve := range seen {
		cves = append(cves, cve)
	}
	sort.Strings(cves)
	return cves
}

// Software returns all software entries from the fixture.
func (f *VulnFixture) Software() []SoftwareFixture {
	sw := make([]SoftwareFixture, len(f.Vulnerabilities))
	for i, v := range f.Vulnerabilities {
		sw[i] = v.Software
	}
	return sw
}

// LoadFixture loads a VulnFixture from a bzip2-compressed JSON file.
func LoadFixture(fixturePath string, t require.TestingT) *VulnFixture {
	f, err := os.Open(fixturePath)
	require.NoError(t, err)
	defer f.Close()

	r := bzip2.NewReader(f)
	var fixture VulnFixture
	require.NoError(t, json.NewDecoder(r).Decode(&fixture))
	require.NotEmpty(t, fixture.Vulnerabilities, "fixture %s has no vulnerabilities", fixturePath)
	return &fixture
}

// createHostWithSoftware creates a host in the database and populates it with the given software.
func createHostWithSoftware(
	ds *mysql.Datastore,
	platformStr string,
	ver fleet.OSVersion,
	software []fleet.Software,
	t require.TestingT,
) *fleet.Host {
	osqueryHostID, err := server.GenerateRandomText(10)
	require.NoError(t, err)

	ctx := context.Background()

	h, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        platformStr,
		NodeKey:         ptr.String(platformStr),
		UUID:            platformStr,
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   &osqueryHostID,
		Platform:        ver.Platform,
		OSVersion:       ver.Name,
	})
	require.NoError(t, err)

	_, err = ds.UpdateHostSoftware(ctx, h.ID, software)
	require.NoError(t, err)

	err = ds.LoadHostSoftware(ctx, h, false)
	require.NoError(t, err)

	var cpes []fleet.SoftwareCPE
	for _, s := range h.Software {
		cpes = append(cpes, fleet.SoftwareCPE{SoftwareID: s.ID, CPE: fmt.Sprintf("%s-%s", s.Name, s.Version)})
	}
	_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	return h
}

// fixturesToSoftware converts SoftwareFixtures to fleet.Software.
func fixturesToSoftware(fixtures []SoftwareFixture) []fleet.Software {
	software := make([]fleet.Software, len(fixtures))
	for i, fi := range fixtures {
		software[i] = fleet.Software{
			Name:    fi.Name,
			Version: fi.Version,
			Release: fi.Release,
			Arch:    fi.Arch,
		}
	}
	return software
}

// LoadSoftwareFromFixture creates a host and populates it with software from a VulnFixture.
func LoadSoftwareFromFixture(
	ds *mysql.Datastore,
	platformStr string,
	ver fleet.OSVersion,
	fixture *VulnFixture,
	t require.TestingT,
) *fleet.Host {
	return createHostWithSoftware(ds, platformStr, ver, fixturesToSoftware(fixture.Software()), t)
}

// RunAndAssert loads software from a VulnFixture, runs the scanner, and asserts that
// each package's detected CVEs match the expected per-package mapping.
func RunAndAssert(
	t *testing.T,
	ds *mysql.Datastore,
	scanner Scanner,
	ver fleet.OSVersion,
	platformStr string,
	fixturePath string,
	vulnPath string,
) {
	ctx := t.Context()

	fixture := LoadFixture(fixturePath, t)

	h := LoadSoftwareFromFixture(ds, platformStr, ver, fixture, t)
	require.NoError(t, ds.UpdateOSVersions(ctx))

	_, err := scanner.Analyze(ctx, ds, ver, vulnPath)
	require.NoError(t, err)

	storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{h.ID}, scanner.Source)
	require.NoError(t, err)

	// Assert total CVE count matches
	actualCVEs := make(map[string]struct{})
	for _, v := range storedVulns[h.ID] {
		actualCVEs[v.CVE] = struct{}{}
	}
	actualList := make([]string, 0, len(actualCVEs))
	for cve := range actualCVEs {
		actualList = append(actualList, cve)
	}
	require.ElementsMatch(t, actualList, fixture.AllCVEs())

	// Build softwareID→name lookup for per-package assertions
	err = ds.LoadHostSoftware(ctx, h, false)
	require.NoError(t, err)
	softwareIDToName := make(map[uint]string)
	for _, s := range h.Software {
		softwareIDToName[s.ID] = s.Name
	}

	// Assert per-package CVE mapping
	actualByPackage := make(map[string][]string)
	for _, v := range storedVulns[h.ID] {
		name := softwareIDToName[v.SoftwareID]
		actualByPackage[name] = append(actualByPackage[name], v.CVE)
	}
	expectedPackages := make(map[string]struct{})
	for _, expected := range fixture.Vulnerabilities {
		expectedPackages[expected.Software.Name] = struct{}{}
		if len(expected.CVEs) == 0 {
			continue
		}
		assert.ElementsMatch(t, actualByPackage[expected.Software.Name], expected.CVEs,
			"CVE mismatch for package %s (%s-%s)",
			expected.Software.Name, expected.Software.Version, expected.Software.Release)
	}
	for pkg, cves := range actualByPackage {
		if _, ok := expectedPackages[pkg]; !ok {
			assert.Failf(t, "unexpected package with vulnerabilities",
				"package %q has CVEs %v but is not in the fixture", pkg, cves)
		}
	}

	require.NoError(t, ds.DeleteHost(ctx, h.ID))
}

// ExtractBzip2 decompresses a bzip2-compressed file from src to dst.
func ExtractBzip2(src, dst string, t require.TestingT) {
	srcF, err := os.Open(src)
	require.NoError(t, err)
	defer srcF.Close()

	dstF, err := os.Create(dst)
	require.NoError(t, err)
	defer dstF.Close()

	r := bzip2.NewReader(srcF)
	_, err = io.Copy(dstF, r) //nolint:gosec // ignoring "G110: Potential DoS vulnerability via decompression bomb", as this is test code.
	require.NoError(t, err)
}

// LoadSoftware creates a host from a JSON fixture file at <vulnPath>/<platformStr>-software.json.
func LoadSoftware(
	ds *mysql.Datastore,
	platformStr string,
	ver fleet.OSVersion,
	vulnPath string,
	t require.TestingT,
) *fleet.Host {
	var fixtures []SoftwareFixture
	contents, err := os.ReadFile(filepath.Join(vulnPath, fmt.Sprintf("%s-software.json", platformStr)))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(contents, &fixtures))

	return createHostWithSoftware(ds, platformStr, ver, fixturesToSoftware(fixtures), t)
}

// loadExpectedCVEs reads expected CVEs from a CSV fixture file.
func loadExpectedCVEs(csvPath string, t require.TestingT) []string {
	f, err := os.Open(csvPath)
	require.NoError(t, err)
	defer f.Close()

	r := csv.NewReader(f)
	var expected []string
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		if len(row) < 1 {
			continue
		}

		if len(row) > 1 && row[1] == "#ignore:" || strings.Contains(row[0], "ignore") {
			continue
		}

		if !strings.HasPrefix(strings.ToLower(row[0]), "cve") {
			continue
		}

		expected = append(expected, row[0])
	}
	require.NotEmpty(t, expected)
	return expected
}

// assertVulnsMatch queries the DB for stored vulnerabilities and asserts they match expected CVEs.
func assertVulnsMatch(
	t require.TestingT,
	ds *mysql.Datastore,
	hostID uint,
	source fleet.VulnerabilitySource,
	expected []string,
) {
	ctx := context.Background()
	storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{hostID}, source)
	require.NoError(t, err)

	uniq := make(map[string]bool)
	for _, v := range storedVulns[hostID] {
		uniq[v.CVE] = true
	}
	actual := make([]string, 0, len(uniq))
	for k := range uniq {
		actual = append(actual, k)
	}

	require.ElementsMatch(t, actual, expected)
}

// ExtractSoftwareAndCVEFixtures extracts software and CVE fixture files from bzip2 archives.
func ExtractSoftwareAndCVEFixtures(
	platformStr string,
	softwareFixPath string,
	vulnPath string,
	t require.TestingT,
) {
	srcSoftPath := filepath.Join(softwareFixPath, fmt.Sprintf("%s-software.json.bz2", platformStr))
	dstSoftPath := filepath.Join(vulnPath, fmt.Sprintf("%s-software.json", platformStr))
	ExtractBzip2(srcSoftPath, dstSoftPath, t)

	srcCvesPath := filepath.Join(softwareFixPath, fmt.Sprintf("%s-software_cves.csv.bz2", platformStr))
	dstCvesPath := filepath.Join(vulnPath, fmt.Sprintf("%s-software_cves.csv", platformStr))
	ExtractBzip2(srcCvesPath, dstCvesPath, t)
}

// WithTestFixture orchestrates loading a test fixture: extracts archives, creates host with software,
// runs UpdateOSVersions, calls the callback, then cleans up the host.
func WithTestFixture(
	version fleet.OSVersion,
	platformStr string,
	extractFn func(vulnPath string),
	vulnPath string,
	ds *mysql.Datastore,
	afterLoad func(h *fleet.Host),
	t require.TestingT,
) {
	ctx := context.Background()

	extractFn(vulnPath)

	h := LoadSoftware(ds, platformStr, version, vulnPath, t)
	err := ds.UpdateOSVersions(ctx)
	require.NoError(t, err)
	afterLoad(h)
	err = ds.DeleteHost(ctx, h.ID)
	require.NoError(t, err)
}

// AssertVulns loads expected CVEs from a CSV fixture file and asserts they match
// the vulnerabilities stored in the database for the given host and source.
func AssertVulns(
	t require.TestingT,
	ds *mysql.Datastore,
	vulnPath string,
	h *fleet.Host,
	platformStr string,
	source fleet.VulnerabilitySource,
) {
	csvPath := filepath.Join(vulnPath, fmt.Sprintf("%s-software_cves.csv", platformStr))
	expected := loadExpectedCVEs(csvPath, t)
	assertVulnsMatch(t, ds, h.ID, source, expected)
}
