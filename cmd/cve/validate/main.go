package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/goval_dictionary"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/osv"
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
	checkOSVVulnerabilities(vulnPath, logger)
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

func checkOSVVulnerabilities(vulnPath string, logger *slog.Logger) {
	entries, err := os.ReadDir(vulnPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read vuln path for OSV validation: %v", err))
	}

	var osvFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, osv.OSVFilePrefix) && strings.HasSuffix(name, ".json.gz") && !strings.Contains(name, "delta") {
			osvFiles = append(osvFiles, name)
		}
	}

	if len(osvFiles) == 0 {
		logger.Warn("no OSV artifact files found in vuln path, skipping OSV validation")
		return
	}

	logger.Info("validating OSV artifacts", "count", len(osvFiles))

	for _, fileName := range osvFiles {
		filePath := filepath.Join(vulnPath, fileName)

		f, err := os.Open(filePath)
		if err != nil {
			panic(fmt.Sprintf("failed to open OSV artifact %s: %v", fileName, err))
		}

		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			panic(fmt.Sprintf("failed to create gzip reader for %s: %v", fileName, err))
		}

		var artifact osv.OSVArtifact
		if err := json.NewDecoder(gz).Decode(&artifact); err != nil {
			gz.Close()
			f.Close()
			panic(fmt.Sprintf("failed to decode OSV artifact %s: %v", fileName, err))
		}
		gz.Close()
		f.Close()

		// Validate required fields
		if artifact.SchemaVersion == "" {
			panic(fmt.Sprintf("OSV artifact %s has empty schema_version", fileName))
		}
		if artifact.UbuntuVersion == "" {
			panic(fmt.Sprintf("OSV artifact %s has empty ubuntu_version", fileName))
		}
		if artifact.TotalCVEs == 0 {
			panic(fmt.Sprintf("OSV artifact %s has zero total_cves", fileName))
		}
		if artifact.TotalPackages == 0 {
			panic(fmt.Sprintf("OSV artifact %s has zero total_packages", fileName))
		}
		if len(artifact.Vulnerabilities) == 0 {
			panic(fmt.Sprintf("OSV artifact %s has empty vulnerabilities map", fileName))
		}

		// Validate that at least one vulnerability entry has a CVE
		foundCVE := false
		for _, vulns := range artifact.Vulnerabilities {
			for _, v := range vulns {
				if strings.HasPrefix(v.CVE, "CVE-") {
					foundCVE = true
					break
				}
			}
			if foundCVE {
				break
			}
		}
		if !foundCVE {
			panic(fmt.Sprintf("OSV artifact %s has no entries with valid CVE identifiers", fileName))
		}

		logger.Info("OSV artifact validated",
			"file", fileName,
			"ubuntu_version", artifact.UbuntuVersion,
			"total_cves", artifact.TotalCVEs,
			"total_packages", artifact.TotalPackages,
		)
	}
}

func platformFromString(platform string) oval.Platform {
	parts := strings.Split(platform, "_")
	host, version := parts[0], fmt.Sprintf("Amazon Linux %s.0", parts[1]) // this is due to how oval.Platform#getMajorMinorVer works
	return oval.NewPlatform(host, version)
}
