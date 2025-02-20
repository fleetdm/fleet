package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/shirou/gopsutil/v3/process"
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

	singleSoftwareSet := *softwareName != ""
	softwareFromURLSet := *softwareFromURL != ""

	if !*sync && !singleSoftwareSet && !softwareFromURLSet {
		printf("Must either set --sync, --software_name or --software_from_url\n")
		return
	}

	if singleSoftwareSet && softwareFromURLSet {
		printf("Cannot set both --software_name and --software_from_url\n")
		return
	}

	if singleSoftwareSet {
		if *softwareVersion == "" {
			printf("Must set --software_version\n")
			return
		}
		if *softwareSource == "" {
			printf("Must set --software_source\n")
			return
		}
	}

	if softwareFromURLSet {
		if *softwareFromAPIToken == "" {
			printf("Must set --software_from_api_token\n")
			return
		}
	}

	// All macOS apps are expected to have a bundle identifier, which influences CPE generation.
	if softwareSource != nil && *softwareSource == "apps" && softwareBundleIdentifier != nil && *softwareBundleIdentifier == "" {
		printf("Must set --software_bundle_identifier for macOS apps when specifying -software_source apps\n")
		return
	}

	if err := os.MkdirAll(*dbDir, os.ModePerm); err != nil {
		panic(err)
	}

	if *debug {
		// Sample the process CPU and memory usage every second
		// and store it on a file under the dbDir.
		myProcess, err := process.NewProcess(int32(os.Getpid())) //nolint:gosec // dismiss G115
		if err != nil {
			panic(err)
		}
		cpuAndMemFile, err := os.Create(filepath.Join(*dbDir, "cpu_and_mem.dat"))
		if err != nil {
			panic(err)
		}
		defer cpuAndMemFile.Close()
		go func() {
			for {
				time.Sleep(time.Second)
				cpuPercent, err := myProcess.CPUPercent()
				if err != nil {
					panic(err)
				}
				memInfo, err := myProcess.MemoryInfo()
				if err != nil {
					panic(err)
				}
				now := time.Now().UTC().Format("15:04:05")
				fmt.Fprintf(cpuAndMemFile, "%s %.2f %.2f\n", now, cpuPercent, float64(memInfo.RSS)/1024.0/1024.0)
			}
		}()
	}

	logger := log.NewJSONLogger(os.Stdout)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	if *debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	if *sync {
		printf("Syncing into %s...\n", *dbDir)
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
			printf("Retrieved software:\n")
			for _, s := range software {
				printf("%+v\n", s)
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
				printf("Matched CPE: %d: %s\n", cpe.SoftwareID, cpe.CPE)
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

	ds.ListOperatingSystemsForPlatformFunc = func(ctx context.Context, platform string) ([]fleet.OperatingSystem, error) {
		return nil, nil
	}

	ds.DeleteOutOfDateOSVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, duration time.Duration) error {
		return nil
	}

	printf("Translating software to CPE...\n")
	err := nvd.TranslateSoftwareToCPE(ctx, ds, *dbDir, logger)
	if err != nil {
		panic(err)
	}
	if len(softwareCPEs) == 0 {
		printf("Unable to match a CPE for the software...\n")
		return
	}
	printf("Translating CPEs to CVEs...\n")
	vulns, err := nvd.TranslateCPEToCVE(ctx, ds, *dbDir, logger, true, 1*time.Hour)
	if err != nil {
		panic(err)
	}

	if singleSoftwareSet {
		var cves []string
		for _, vuln := range vulns {
			cves = append(cves, vuln.CVE)
		}
		printf("CVEs found for %s (%s): %s\n", *softwareName, *softwareVersion, strings.Join(cves, ", "))
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
			printf("Found vulnerabilities:\n")
			for softwareID, cves := range foundSoftwareCVEs {
				printf("%s (%d): %s\n", getSoftwareName(software, softwareID), softwareID, cves)
			}
		}
		if cmp.Equal(expectedSoftwareMap, foundSoftwareCVEs) {
			printf("CVEs found and expected matched!\n")
			return
		}
		for s, expectedVulns := range expectedSoftwareMap {
			if vulnsFound, ok := foundSoftwareCVEs[s]; !ok || !cmp.Equal(expectedVulns, vulnsFound) {
				printf("Mismatched software %s (%d): expected=%+v vs found=%+v\n", getSoftwareName(software, s), s, expectedVulns, vulnsFound)
				if ok {
					delete(foundSoftwareCVEs, s)
				}
			}
		}
		for s, vulnsFound := range foundSoftwareCVEs {
			if expectedVulns, ok := expectedSoftwareMap[s]; !ok || !cmp.Equal(expectedVulns, vulnsFound) {
				printf("Mismatched software %s (%d): expected=%+v vs found=%+v\n", getSoftwareName(software, s), s, expectedVulns, vulnsFound)
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
	return s.i < len(s.software)
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

	software, err := apiClient.ListSoftwareVersions("")
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

func printf(format string, a ...any) {
	fmt.Printf(time.Now().UTC().Format("2006-01-02T15:04:05Z")+": "+format, a...)
}
