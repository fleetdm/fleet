package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/go-cmp/cmp"
)

func main() {
	sync := flag.Bool("sync", false, "If set, it will synchronize the vulnerability database before running vulnerability processing")
	dbDir := flag.String("db_dir", "/tmp/vulndbs", "Path to the vulnerability database")
	debug := flag.Bool("debug", false, "Sets debug mode")

	// Fields that allow setting a specific software.
	softwareName := flag.String("software_name", "", "Name of the software as ingested by Fleet")
	softwareVersion := flag.String("software_version", "", "Version of the software as ingested by Fleet")
	softwareSource := flag.String("software_source", "", "Source for this software (e.g. 'apps' for macOS applications)")
	softwareBundleIdentifier := flag.String("software_bundle_identifier", "", "Bundle identifier of the software as ingested by Fleet (for macOS apps only)")

	// Fields to fetch software (and the found vulnerabilities) from a Fleet instance.
	// This mode of operation then compares the CVEs found by the Fleet instance with the CVEs found by this new run of vulnerability processing.
	softwareFromURL := flag.String("software_from_url", "", "URL to get software from")
	softwareFromAPIToken := flag.String("software_from_api_token", "", "API token to authenticate to get the software")

	flag.Parse()

	if *debug {
		go func() {
			for {
				select {
				case <-time.After(5 * time.Second):
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					fmt.Printf("Memory usage: Alloc = %v MiB, TotalAlloc = %v MiB, Sys = %v MiB\n", m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024)
				}
			}
		}()
	}

	singleSoftwareSet := *softwareName != ""
	softwareFromURLSet := *softwareFromURL != ""

	if !*sync && !singleSoftwareSet && !softwareFromURLSet {
		fmt.Printf("Must either set --sync, --software_name or --software_from_url")
		return
	}

	if singleSoftwareSet && softwareFromURLSet {
		fmt.Printf("Cannot set both --software_name and --software_from_url")
		return
	}

	if singleSoftwareSet {
		if *softwareVersion == "" {
			fmt.Printf("Must set --software_version")
			return
		}
		if *softwareSource == "" {
			fmt.Printf("Must set --software_source")
			return
		}
	}

	if softwareFromURLSet {
		if *softwareFromAPIToken == "" {
			fmt.Printf("Must set --software_from_api_token")
			return
		}
	}

	if err := os.MkdirAll(*dbDir, os.ModePerm); err != nil {
		panic(err)
	}

	logger := log.NewJSONLogger(os.Stdout)
	if *debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	if *sync {
		fmt.Printf("Syncing into %s...\n", *dbDir)
		if err := vulnDBSync(*dbDir, *debug, logger); err != nil {
			panic(err)
		}
		if !singleSoftwareSet && !softwareFromURLSet {
			return
		}
	}

	ctx := context.Background()

	var software []fleet.Software
	if singleSoftwareSet {
		software = []fleet.Software{
			{
				Name:             *softwareName,
				Version:          *softwareVersion,
				Source:           *softwareSource,
				BundleIdentifier: *softwareBundleIdentifier,
			},
		}
	} else { // softwareFromURLSet
		software = getSoftwareFromURL(*softwareFromURL, *softwareFromAPIToken, *debug)
		if *debug {
			fmt.Printf("Retrieved software:\n")
			for _, s := range software {
				fmt.Printf("%+v\n", s)
			}
		}
		// Set CPE to empty to trigger CPE matching.
		for i := range software {
			software[i].GenerateCPE = ""
		}
	}

	ds := new(mock.Store)

	ds.AllSoftwareIteratorFunc = func(ctx context.Context, query fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
		return &softwareIterator{software: software}, nil
	}
	var softwareCPEs []fleet.SoftwareCPE
	ds.UpsertSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
		for _, cpe := range cpes {
			var found bool
			for _, storedCPEs := range softwareCPEs {
				if storedCPEs == cpe {
					found = true
					break
				}
			}
			if !found {
				softwareCPEs = append(softwareCPEs, cpe)
			}
		}
		if singleSoftwareSet || *debug {
			for _, cpe := range cpes {
				fmt.Printf("Matched CPE: %d: %s\n", cpe.SoftwareID, cpe.CPE)
			}
		}
		return int64(len(cpes)), nil
	}
	ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
		return softwareCPEs, nil
	}
	ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, source fleet.VulnerabilitySource) (bool, error) {
		return true, nil
	}
	ds.DeleteOutOfDateVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, duration time.Duration) error {
		return nil
	}

	fmt.Println("Translating software to CPE...")
	err := nvd.TranslateSoftwareToCPE(ctx, ds, *dbDir, logger)
	if err != nil {
		panic(err)
	}
	if len(softwareCPEs) == 0 {
		fmt.Println("Unable to match a CPE for the software...")
		return
	}
	fmt.Println("Translating CPEs to CVEs...")
	vulns, err := nvd.TranslateCPEToCVE(ctx, ds, *dbDir, logger, true, 1*time.Hour)
	if err != nil {
		panic(err)
	}

	if singleSoftwareSet {
		var cves []string
		for _, vuln := range vulns {
			cves = append(cves, vuln.CVE)
		}
		fmt.Printf("CVEs found for %s (%s): %s\n", *softwareName, *softwareVersion, strings.Join(cves, ", "))
	} else { // softwareFromURLSet
		expectedSoftwareMap := make(map[uint][]string)
		for _, s := range software {
			var vulnerabilities []string
			for _, vulnerability := range s.Vulnerabilities {
				vulnerabilities = append(vulnerabilities, vulnerability.CVE)
			}
			if len(vulnerabilities) == 0 {
				continue
			}
			sort.Strings(vulnerabilities)
			expectedSoftwareMap[s.ID] = vulnerabilities
		}

		foundSoftwareCVEs := make(map[uint][]string)
		for _, vuln := range vulns {
			foundSoftwareCVEs[vuln.SoftwareID] = append(foundSoftwareCVEs[vuln.SoftwareID], vuln.CVE)
		}
		for softwareID := range foundSoftwareCVEs {
			sort.Strings(foundSoftwareCVEs[softwareID])
		}
		if *debug {
			fmt.Printf("Found vulnerabilities:\n")
			for softwareID, cves := range foundSoftwareCVEs {
				fmt.Printf("%s (%d): %s\n", getSoftwareName(software, softwareID), softwareID, cves)
			}
		}
		if cmp.Equal(expectedSoftwareMap, foundSoftwareCVEs) {
			fmt.Printf("CVEs found and expected matched!\n")
			return
		}
		for s, expectedVulns := range expectedSoftwareMap {
			if vulnsFound, ok := foundSoftwareCVEs[s]; !ok || !cmp.Equal(expectedVulns, vulnsFound) {
				fmt.Printf("Mismatched software %s (%d): expected=%+v vs found=%+v\n", getSoftwareName(software, s), s, expectedVulns, vulnsFound)
				if ok {
					delete(foundSoftwareCVEs, s)
				}
			}
		}
		for s, vulnsFound := range foundSoftwareCVEs {
			if expectedVulns, ok := expectedSoftwareMap[s]; !ok || !cmp.Equal(expectedVulns, vulnsFound) {
				fmt.Printf("Mismatched software %s (%d): expected=%+v vs found=%+v\n", getSoftwareName(software, s), s, expectedVulns, vulnsFound)
			}
		}
	}
}

func getSoftwareName(software []fleet.Software, softwareID uint) string {
	for _, s := range software {
		if s.ID == softwareID {
			return s.Name + ":" + s.Version
		}
	}
	panic(fmt.Sprintf("software %d not found", softwareID))
}

type softwareIterator struct {
	software []fleet.Software
	i        int
}

func (s *softwareIterator) Next() bool {
	if s.i >= len(s.software) {
		return false
	}
	return true
}

func (s *softwareIterator) Value() (*fleet.Software, error) {
	ss := &s.software[s.i]
	s.i += 1
	return ss, nil
}

func (s *softwareIterator) Err() error {
	return nil
}

func (s *softwareIterator) Close() error {
	return nil
}

func vulnDBSync(vulnDBDir string, debug bool, logger log.Logger) error {
	opts := nvd.SyncOptions{
		VulnPath: vulnDBDir,
		Debug:    debug,
	}
	err := nvd.Sync(opts, logger)
	if err != nil {
		return err
	}
	return nil
}

func getSoftwareFromURL(url, apiToken string, debug bool) []fleet.Software {
	var clientOpts []service.ClientOption
	if debug {
		clientOpts = append(clientOpts, service.EnableClientDebug())
	}
	apiClient, err := service.NewClient(url, true, "", "", clientOpts...)
	if err != nil {
		panic(err)
	}
	apiClient.SetToken(apiToken)

	software, err := apiClient.ListSoftware("")
	if err != nil {
		panic(err)
	}
	var filteredSoftware []fleet.Software
	for _, s := range software {
		if s.Source == "deb_packages" || s.Source == "rpm_packages" {
			continue
		}
		filteredSoftware = append(filteredSoftware, s)
	}
	return filteredSoftware
}
