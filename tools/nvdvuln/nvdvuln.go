package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/go-kit/log"
)

func main() {
	sync := flag.Bool("sync", false, "...")
	vulnDBDir := flag.String("vuln_db_dir", "/tmp/vulndbs", "...")

	softwareName := flag.String("software_name", "", "Name of the software as ingested by Fleet")
	softwareVersion := flag.String("software_version", "", "Version of the software as ingested by Fleet")
	softwareSource := flag.String("software_source", "", "Source for this software (e.g. 'apps' for macOS applications)")
	softwareBundleIdentifier := flag.String("software_bundle_identifier", "", "Bundle identifier of the software as ingested by Fleet (for macOS apps only)")
	cveToLog := flag.String("cve_to_log", "", "CVE Identifier to log (e.g. CVE-2021-1234)")

	flag.Parse()

	if *softwareName == "" {
		fmt.Println("Must set -software_name flag.")
		return
	}
	if *softwareVersion == "" {
		fmt.Println("Must set -software_version flag.")
		return
	}
	if *softwareSource == "" {
		fmt.Println("Must set -software_source flag.")
		return
	}

	if err := os.MkdirAll(*vulnDBDir, os.ModePerm); err != nil {
		panic(err)
	}

	logger := log.NewJSONLogger(os.Stdout)

	if *sync {
		fmt.Printf("Syncing into %s...\n", *vulnDBDir)
		if err := vulnDBSync(*vulnDBDir, logger); err != nil {
			panic(err)
		}
	}

	ctx := context.Background()

	ds := new(mock.Store)

	ds.AllSoftwareIteratorFunc = func(ctx context.Context, query fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
		return &softwareIterator{
			software: []fleet.Software{
				{
					Name:             *softwareName,
					Version:          *softwareVersion,
					Source:           *softwareSource,
					BundleIdentifier: *softwareBundleIdentifier,
				},
			},
		}, nil
	}
	var softwareCPEs []fleet.SoftwareCPE
	ds.UpsertSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
		softwareCPEs = cpes
		for _, cpe := range cpes {
			fmt.Printf("Matched CPE: %s\n", cpe.CPE)
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
	err := nvd.TranslateSoftwareToCPE(ctx, ds, *vulnDBDir, logger)
	if err != nil {
		panic(err)
	}
	if len(softwareCPEs) == 0 {
		fmt.Println("Unable to match a CPE for the software...")
		return
	}
	fmt.Println("Translating CPEs to CVEs...")
	vulns, err := nvd.TranslateCPEToCVE(ctx, ds, *vulnDBDir, logger, true, 1*time.Hour, *cveToLog)
	if err != nil {
		panic(err)
	}
	var cves []string
	for _, vuln := range vulns {
		cves = append(cves, vuln.CVE)
	}
	fmt.Printf("CVEs found for %s (%s): %s\n", *softwareName, *softwareVersion, strings.Join(cves, ", "))
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

func vulnDBSync(vulnDBDir string, logger log.Logger) error {
	opts := nvd.SyncOptions{
		VulnPath: vulnDBDir,
	}
	err := nvd.Sync(opts, logger)
	if err != nil {
		return err
	}
	return nil
}
