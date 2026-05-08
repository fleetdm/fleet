package main

import (
	"context"
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
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
)

// anyVulnerableCPEMatch reports whether any node (or descendant) carries a
// CPEMatch that's both marked vulnerable and has a non-empty CPE 2.3 URI.
// Non-vulnerable matches (typical of AND-operator context entries that
// describe the platform a vulnerable component runs on) and empty URIs are
// excluded so the canary measures real enrichment, not just slice length.
func anyVulnerableCPEMatch(nodes []*schema.NVDCVEFeedJSON10DefNode) bool {
	for _, n := range nodes {
		for _, m := range n.CPEMatch {
			if m.Vulnerable && m.Cpe23Uri != "" {
				return true
			}
		}
		if anyVulnerableCPEMatch(n.Children) {
			return true
		}
	}
	return false
}

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

	// NVD leaves many recent CVEs without CPE matches (Deferred, Awaiting Analysis), so
	// most populated configurations in this feed come from VulnCheck enrichment. Counting
	// CVEs with any CPE match catches a broken enrichment without pegging the canary to a
	// single CVE's row count — VulnCheck drifts per-CVE and that broke this pipeline
	// repeatedly when the canary was CVE-2025-0938.
	//
	// Two thresholds catch different failure modes:
	//   - Absolute floor (10k): trips if the 2025 feed itself shrinks pathologically
	//     (corrupt download, partial sync) regardless of ratio.
	//   - Ratio floor (30%): trips if enrichment quality regresses while the feed size
	//     stays normal. Last healthy run was 31,197 / 42,477 = 73.4%; 30% leaves wide
	//     margin while still firing on a wholesale enrichment regression.
	const (
		minEnrichedCVEs2025  = 10000
		minEnrichedRatio2025 = 0.30
	)
	total2025, enriched2025 := 0, 0
	for id, v := range vulns {
		if !strings.HasPrefix(id, "CVE-2025-") {
			continue
		}
		total2025++
		entry, ok := v.(*feednvd.Vuln)
		if !ok || entry.Schema().Configurations == nil {
			continue
		}
		if anyVulnerableCPEMatch(entry.Schema().Configurations.Nodes) {
			enriched2025++
		}
	}
	ratio2025 := 0.0
	if total2025 > 0 {
		ratio2025 = float64(enriched2025) / float64(total2025)
	}
	if enriched2025 < minEnrichedCVEs2025 || ratio2025 < minEnrichedRatio2025 {
		panic(fmt.Errorf("2025 enrichment canary failed: %d/%d (%.1f%%) CVEs have CPE matches, expected >= %d AND >= %.0f%%",
			enriched2025, total2025, ratio2025*100, minEnrichedCVEs2025, minEnrichedRatio2025*100))
	}

	if vulns["CVE-2025-3196"].CVSSv3BaseScore() != 5.5 { // Should pull primary CVSSv3 score, (has primary and secondary)
		panic(fmt.Errorf("cvss v3 spot-check failed for CVE-2025-3196; score was %f instead of 5.5", vulns["CVE-2025-3196"].CVSSv3BaseScore()))
	}

	vulns, err = cvefeed.LoadJSONDictionary(filepath.Join(vulnPath, "nvdcve-1.1-2024.json.gz"))
	if err != nil {
		panic(err)
	}

	// make sure VulnCheck enrichment is working on less recent vulns
	vulnEntry, ok := vulns["CVE-2024-6286"].(*feednvd.Vuln)
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

func platformFromString(platform string) oval.Platform {
	parts := strings.Split(platform, "_")
	host, version := parts[0], fmt.Sprintf("Amazon Linux %s.0", parts[1]) // this is due to how oval.Platform#getMajorMinorVer works
	return oval.NewPlatform(host, version)
}
