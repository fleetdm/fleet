package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/goval_dictionary"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	nvdsync "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/sync"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
)

func main() {
	dbDir := flag.String("db_dir", "/tmp/vulndbs", "Path to the vulnerability database")
	debug := flag.Bool("debug", false, "Sets debug mode")
	flag.Parse()

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	vulnPath := *dbDir
	checkNVDVulnerabilities(vulnPath, logger)
	checkGovalDictionaryVulnerabilities(vulnPath)
}

func checkNVDVulnerabilities(vulnPath string, logger *slog.Logger) {
	metaMap := make(map[string]fleet.CVEMeta)
	if err := nvd.CVEMetaFromNVDFeedFiles(context.Background(), metaMap, vulnPath, logger); err != nil {
		panic(err)
	}

	vulns, err := cvefeed.LoadJSONDictionary(filepath.Join(vulnPath, "nvdcve-1.1-2025.json.gz"))
	if err != nil {
		panic(err)
	}

	// make sure VulnCheck enrichment is working
	vulnEntry, ok := vulns["CVE-2025-0938"].(*feednvd.Vuln)
	if !ok {
		panic("failed to cast CVE-2025-0938 to a Vuln")
	}
	// NVD lists CVE-2025-0938 as Deferred with no configurations, so any CPE match
	// here proves VulnCheck enrichment ran. The threshold was previously the historical
	// row count (6), which broke the daily release pipeline whenever VulnCheck dropped
	// a row. Floor of 1 catches a complete enrichment failure for this CVE without
	// breaking on per-CVE drift.
	if len(vulnEntry.Schema().Configurations.Nodes) < 1 || len(vulnEntry.Schema().Configurations.Nodes[0].CPEMatch) < 1 {
		panic(errors.New("enriched vulnerability spot-check failed for Python on CVE-2025-0938"))
	}

	if vulns["CVE-2025-3196"].CVSSv3BaseScore() != 5.5 { // Should pull primary CVSSv3 score, (has primary and secondary)
		panic(fmt.Errorf("cvss v3 spot-check failed for CVE-2025-3196; score was %f instead of 5.5", vulns["CVE-2025-3196"].CVSSv3BaseScore()))
	}

	// Confirm every versionEndExcluding override was applied to the generated feed. This is driven by
	// nvdsync.VersionEndExcludingOverrides, so a new override is validated automatically - it only
	// needs to be added to that table, not here.
	checkVersionEndExcludingOverrides(vulnPath)

	vulns, err = cvefeed.LoadJSONDictionary(filepath.Join(vulnPath, "nvdcve-1.1-2024.json.gz"))
	if err != nil {
		panic(err)
	}

	// make sure VulnCheck enrichment is working on less recent vulns
	vulnEntry, ok = vulns["CVE-2024-6286"].(*feednvd.Vuln)
	if !ok {
		panic("failed to cast CVE-2024-6286 to a Vuln")
	}
	if len(vulnEntry.Schema().Configurations.Nodes) < 1 || len(vulnEntry.Schema().Configurations.Nodes[0].CPEMatch) < 2 ||
		vulnEntry.Schema().Configurations.Nodes[0].CPEMatch[1].VersionEndExcluding != "2403.1" {
		panic(errors.New("enriched vulnerability spot-check failed for Citrix Workstation on CVE-2024-6286"))
	}
	for _, match := range vulnEntry.Schema().Configurations.Nodes[0].CPEMatch {
		// there are a number of matches here with "ltsr" in their cpe23Uri but no versionEndExcluding.
		// We are only interested in confirming that the `versionEndExcluding` for the match whose CPE
		// contains "ltsr", which came from NVD with an incorrect value,has been replaced with "2402"
		if strings.Contains(match.Cpe23Uri, ":ltsr:") && match.VersionEndExcluding != "" && match.VersionEndExcluding != "2402" {
			panic(fmt.Errorf("CVE-2024-6286 LTSR versionEndExcluding spot-check failed: got %q, expected \"2402\"", match.VersionEndExcluding))
		}
	}

	// check CVSS score extraction; confirm that secondary CVSS scores are extracted when primary isn't set
	if vulns["CVE-2024-54559"].CVSSv3BaseScore() != 5.5 { // secondary source CVSS score
		panic(errors.New("cvss v3 spot-check failed for CVE-2024-54559"))
	}
	if vulns["CVE-2024-0450"].CVSSv3BaseScore() != 6.2 { // secondary source CVSS score
		panic(errors.New("cvss v3 spot-check failed for CVE-2024-0450"))
	}
	if vulns["CVE-2024-0540"].CVSSv3BaseScore() != 9.8 { // primary source CVSS score
		panic(errors.New("cvss v3 spot-check failed for CVE-2024-0540"))
	}

	vulns, err = cvefeed.LoadJSONDictionary(filepath.Join(vulnPath, "nvdcve-1.1-2023.json.gz"))
	if err != nil {
		panic(err)
	}

	// make sure we're rewriting docker_desktop to docker
	if vulns["CVE-2023-0626"].Config()[0].Product != "desktop" {
		panic(errors.New("docker_desktop spot-check failed for CVE-2023-0626"))
	}
}

// checkVersionEndExcludingOverrides confirms every override in nvdsync.VersionEndExcludingOverrides
// was applied to the generated feed: for each one the target CVE must exist and have a CPE match
// (containing the override's CPESubstr) whose versionEndExcluding equals the override's To value.
// It panics on the first mismatch, failing the daily release pipeline before a bad feed ships.
func checkVersionEndExcludingOverrides(vulnPath string) {
	// Load each year's feed at most once.
	dicts := make(map[int]cvefeed.Dictionary)
	for _, override := range nvdsync.VersionEndExcludingOverrides {
		year, err := strconv.Atoi(override.CVE[4:8])
		if err != nil {
			panic(fmt.Errorf("versionEndExcluding override: parsing year from %q: %w", override.CVE, err))
		}

		dict, ok := dicts[year]
		if !ok {
			dict, err = cvefeed.LoadJSONDictionary(filepath.Join(vulnPath, fmt.Sprintf("nvdcve-1.1-%d.json.gz", year)))
			if err != nil {
				panic(err)
			}
			dicts[year] = dict
		}

		vulnEntry, ok := dict[override.CVE].(*feednvd.Vuln)
		if !ok {
			panic(fmt.Errorf("versionEndExcluding override spot-check failed: %s not found in feed", override.CVE))
		}

		var found bool
		for _, node := range vulnEntry.Schema().Configurations.Nodes {
			for _, match := range node.CPEMatch {
				if strings.Contains(match.Cpe23Uri, override.CPESubstr) && match.VersionEndExcluding == override.To {
					found = true
				}
			}
		}
		if !found {
			panic(fmt.Errorf("versionEndExcluding override spot-check failed for %s: expected %q on a CPE containing %q",
				override.CVE, override.To, override.CPESubstr))
		}
	}
}

func checkGovalDictionaryVulnerabilities(vulnPath string) {
	for _, p := range oval.SupportedGovalPlatforms {
		platform := platformFromString(p)

		destFilename := platform.ToGovalDictionaryFilename()
		filename := platform.ToGovalDatabaseFilename()

		// Renaming these files from amzn_%d.sqlite3 to fleet_goval_dictionary_amzn_%d.sqlite3
		// In the vulnerabilities repository the `goval-dictionary fetch` downloads these files with the shorter name
		// However, the goval_dictionary/sync.go#Refresh method download these files, extracts them, and uses the longer name
		// See in specific the `downloadDatabase` function where it sets the `dstPath` to use `platform.ToGovalDictionaryFilename`
		// LoadDb then expect the path to include the `ToGovalDictionaryFilename`
		err := os.Rename(fmt.Sprintf("%s/%s", vulnPath, filename), fmt.Sprintf("%s/%s", vulnPath, destFilename))
		if err != nil {
			panic(fmt.Sprintf("failed to move file from %s/%s to %s/%s: %v", vulnPath, filename, vulnPath, destFilename, err))
		}

		db, err := goval_dictionary.LoadDb(platform, vulnPath)
		if err != nil {
			panic(err)
		}

		err = db.Verfiy()
		if err != nil {
			panic(err)
		}

		err = os.Rename(fmt.Sprintf("%s/%s", vulnPath, destFilename), fmt.Sprintf("%s/%s", vulnPath, filename))
		if err != nil {
			panic(fmt.Sprintf("failed to move file from %s/%s to %s/%s: %v", vulnPath, destFilename, vulnPath, filename, err))
		}
	}
}

func platformFromString(platform string) oval.Platform {
	parts := strings.Split(platform, "_")
	host, version := parts[0], fmt.Sprintf("Amazon Linux %s.0", parts[1]) // this is due to how oval.Platform#getMajorMinorVer works
	return oval.NewPlatform(host, version)
}
