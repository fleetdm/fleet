package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/goval_dictionary"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	dbDir := flag.String("db_dir", "/tmp/vulndbs", "Path to the vulnerability database")
	debug := flag.Bool("debug", false, "Sets debug mode")
	flag.Parse()

	logger := log.NewJSONLogger(os.Stdout)
	if *debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	vulnPath := *dbDir
	checkNVDVulnerabilities(vulnPath, logger)
	checkGovalDictionaryVulnerabilities(vulnPath)
}

func checkNVDVulnerabilities(vulnPath string, logger log.Logger) {
	metaMap := make(map[string]fleet.CVEMeta)
	if err := nvd.CVEMetaFromNVDFeedFiles(metaMap, vulnPath, logger); err != nil {
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
	if len(vulnEntry.Schema().Configurations.Nodes) < 1 || len(vulnEntry.Schema().Configurations.Nodes[0].CPEMatch) < 6 {
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
