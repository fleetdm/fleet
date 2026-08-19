package vulntest_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulntest"
	"github.com/stretchr/testify/require"
)

// ovalDef mirrors the parsed OVAL JSON structure enough to extract package names and EVR states.
type ovalDef struct {
	RpmInfoTests map[string]struct {
		Objects []string `json:"Objects"`
		States  []struct {
			Evr  string `json:"Evr,omitempty"`
			Arch string `json:"Arch,omitempty"`
		} `json:"States"`
	} `json:"RpmInfoTests"`
}

// extractVulnerablePackages parses a bzip2-compressed OVAL definition JSON and returns
// a map of package name → oldest fixed EVR version. For each package, we pick the oldest
// fixed version so we can generate a software entry just below it.
func extractVulnerablePackages(defPath string, t *testing.T) map[string]string {
	t.Helper()

	var def ovalDef

	// Decompress bzip2 inline
	tmpJSON := filepath.Join(t.TempDir(), "oval_def.json")
	vulntest.ExtractBzip2(defPath, tmpJSON, t)

	data, err := os.ReadFile(tmpJSON)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &def))

	packages := make(map[string]string) // name → oldest fixed EVR
	for _, test := range def.RpmInfoTests {
		if test.States == nil {
			continue
		}
		for _, obj := range test.Objects {
			for _, state := range test.States {
				evr := state.Evr
				if !strings.HasPrefix(evr, "less than|") {
					continue
				}
				fixedVer := strings.TrimPrefix(evr, "less than|")

				// Only include x86_64-compatible packages
				arch := state.Arch
				if arch != "" && !strings.Contains(arch, "x86_64") {
					continue
				}

				existing, ok := packages[obj]
				if !ok || utils.Rpmvercmp(fixedVer, existing) < 0 {
					packages[obj] = fixedVer
				}
			}
		}
	}

	return packages
}

// makeVulnerableVersion takes an EVR fixed version like "0:7.76.1-26.el9_3.3"
// and returns a version/release pair that is older (vulnerable).
// Strategy: strip epoch, use the version as-is, decrement the release minor.
func makeVulnerableVersion(fixedEVR string) (version, release string) {
	// Strip epoch: "0:7.76.1-26.el9_3.3" → "7.76.1-26.el9_3.3"
	evr := fixedEVR
	if _, after, ok := strings.Cut(evr, ":"); ok {
		evr = after
	}

	// Split version-release: "7.76.1-26.el9_3.3" → "7.76.1", "26.el9_3.3"
	if ver, rel, ok := strings.Cut(evr, "-"); ok {
		version = ver
		release = rel

		// Decrement the first numeric component of release to make it older
		// "26.el9_3.3" → "25.el9_3.3"
		for i, c := range release {
			if c >= '1' && c <= '9' {
				release = release[:i] + string(c-1) + release[i+1:]
				return version, release
			}
			if c == '0' && i+1 < len(release) && release[i+1] != '.' {
				continue // skip leading zeros
			}
		}
	} else {
		version = evr
	}

	return version, release
}

// TestGenerateVulnFixtures builds VulnFixture files by:
// 1. Extracting all vulnerable packages from OVAL definitions
// 2. Creating software entries at versions just below the fixed version
// 3. Running the OVAL analyzer to get per-package CVE mappings
// 4. Writing the result as vulns.json.bz2
//
// Run manually:
//
//	GENERATE_FIXTURES=1 MYSQL_TEST=1 go test -v -run TestGenerateVulnFixtures -timeout 300s ./server/vulnerabilities/vulntest/...
func TestGenerateVulnFixtures(t *testing.T) {
	if os.Getenv("GENERATE_FIXTURES") == "" {
		t.Skip("set GENERATE_FIXTURES=1 to run this test")
	}

	ds := mysqltest.CreateMySQLDS(t)
	defer mysqltest.TruncateTables(t, ds)

	testdataRoot := vulntest.TestdataRoot

	systems := []struct {
		defDir      string
		platformStr string
		version     fleet.OSVersion
		outPath     string
	}{
		{
			defDir:      filepath.Join("rhel", "2026"),
			platformStr: "rhel_08",
			version:     fleet.OSVersion{Platform: "rhel", Name: "Red Hat Enterprise Linux 8.10.0"},
			outPath:     filepath.Join(testdataRoot, "rhel", "software", "0810", "rhel_08-vulns.json.bz2"),
		},
		{
			defDir:      filepath.Join("rhel", "2026"),
			platformStr: "rhel_09",
			version:     fleet.OSVersion{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0"},
			outPath:     filepath.Join(testdataRoot, "rhel", "software", "0904", "rhel_09-vulns.json.bz2"),
		},
	}

	for _, sys := range systems {
		t.Run(sys.platformStr, func(t *testing.T) {
			ctx := t.Context()
			vulnPath := t.TempDir()

			p := oval.NewPlatform(sys.version.Platform, sys.version.Name)

			// Extract OVAL definitions
			defSrc := filepath.Join(testdataRoot, sys.defDir, fmt.Sprintf("%s-oval_def.json.bz2", p))
			defDst := filepath.Join(vulnPath, p.ToFilename(time.Now(), "json"))
			vulntest.ExtractBzip2(defSrc, defDst, t)

			// Extract vulnerable packages from OVAL data
			vulnPackages := extractVulnerablePackages(defSrc, t)
			t.Logf("Found %d vulnerable packages in OVAL definitions", len(vulnPackages))

			// Build software fixtures from OVAL data
			var softwareFixtures []vulntest.SoftwareFixture
			for name, fixedEVR := range vulnPackages {
				version, release := makeVulnerableVersion(fixedEVR)
				softwareFixtures = append(softwareFixtures, vulntest.SoftwareFixture{
					Name:    name,
					Version: version,
					Release: release,
					Arch:    "x86_64",
				})
			}
			sort.Slice(softwareFixtures, func(i, j int) bool {
				return softwareFixtures[i].Name < softwareFixtures[j].Name
			})

			// Write software JSON for LoadSoftware
			softwareBytes, err := json.Marshal(softwareFixtures)
			require.NoError(t, err)
			dstSoft := filepath.Join(vulnPath, fmt.Sprintf("%s-software.json", sys.platformStr))
			require.NoError(t, os.WriteFile(dstSoft, softwareBytes, 0o644))

			// Load into DB and run analyzer
			h := vulntest.LoadSoftware(ds, sys.platformStr, sys.version, vulnPath, t)
			require.NoError(t, ds.UpdateOSVersions(ctx))

			_, err = oval.Analyze(ctx, ds, sys.version, vulnPath, true)
			require.NoError(t, err)

			// Query stored vulns
			storedVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{h.ID}, fleet.RHELOVALSource)
			require.NoError(t, err)

			// Build softwareID → name lookup
			err = ds.LoadHostSoftware(ctx, h, false)
			require.NoError(t, err)
			idToName := make(map[uint]string)
			for _, s := range h.Software {
				idToName[s.ID] = s.Name
			}

			// Build per-package CVE map
			pkgCVEs := make(map[string]map[string]struct{})
			for _, v := range storedVulns[h.ID] {
				name := idToName[v.SoftwareID]
				if pkgCVEs[name] == nil {
					pkgCVEs[name] = make(map[string]struct{})
				}
				pkgCVEs[name][v.CVE] = struct{}{}
			}

			// Build fixture — only include packages that triggered CVEs
			var fixture vulntest.VulnFixture
			for _, sf := range softwareFixtures {
				cveSet := pkgCVEs[sf.Name]
				if len(cveSet) == 0 {
					continue
				}
				cves := make([]string, 0, len(cveSet))
				for cve := range cveSet {
					cves = append(cves, cve)
				}
				sort.Strings(cves)
				fixture.Vulnerabilities = append(fixture.Vulnerabilities, vulntest.ExpectedVuln{
					Software: sf,
					CVEs:     cves,
				})
			}

			// Write JSON then bzip2
			jsonPath := sys.outPath + ".tmp.json"
			jsonBytes, err := json.MarshalIndent(&fixture, "", "  ")
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(filepath.Dir(sys.outPath), 0o755))
			require.NoError(t, os.WriteFile(jsonPath, jsonBytes, 0o644))

			cmd := exec.Command("bzip2", "-f", jsonPath)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "bzip2 failed: %s", out)
			require.NoError(t, os.Rename(jsonPath+".bz2", sys.outPath))

			fmt.Printf("  %s: %d packages with vulns (of %d total), %d CVEs → %s\n",
				sys.platformStr, len(fixture.Vulnerabilities), len(softwareFixtures),
				len(fixture.AllCVEs()), sys.outPath)

			require.NoError(t, ds.DeleteHost(ctx, h.ID))
		})
	}
}
