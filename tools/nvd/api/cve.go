package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/pandatix/nvdapi/v2"
)

func main() {
	log.SetFlags(log.LstdFlags)

	apiKey := flag.String("api-key", "", "NVD API key")
	dbDir := flag.String("db-dir", "", "Local directory to store CVEs")

	flag.Parse()

	if *apiKey == "" {
		log.Fatal("Must provide --api-key")
	}
	if *dbDir == "" {
		log.Fatal("Must provide --db-dir")
	}
	if err := os.MkdirAll(*dbDir, 0o777); err != nil {
		log.Fatal(err)
	}

	nvdClient, err := nvdapi.NewNVDClient(fleethttp.NewClient(), *apiKey)
	if err != nil {
		log.Fatal(err)
	}

	var initialSync bool
	lastModStartDatePath := filepath.Join(*dbDir, "last_mod_start_date.txt")
	switch _, err := os.Stat(lastModStartDatePath); {
	case err == nil:
		initialSync = false
	case errors.Is(err, fs.ErrNotExist):
		initialSync = true
	default:
		log.Fatal(err)
	}

	sync(nvdClient, *dbDir, initialSync)
}

func sync(nvdClient *nvdapi.NVDClient, dbDir string, initialSync bool) {
	if initialSync {
		doInitialSync(nvdClient, dbDir)
		return
	}
}

func doInitialSync(nvdClient *nvdapi.NVDClient, dbDir string) {
	log.Println("Performing initial sync...")
	start := time.Now()
	totalResults := 229000 // there are at least this number of CVEs.
	cvesByYear := make(map[int][]nvdapi.CVEItem)
	var year int
	for startIndex := 0; startIndex < totalResults; {
		cveResponse, err := nvdapi.GetCVEs(nvdClient, nvdapi.GetCVEsParams{
			StartIndex: ptr.Int(startIndex),
		})
		if err != nil {
			log.Fatal(err)
		}
		totalResults = cveResponse.TotalResults
		startIndex += cveResponse.ResultsPerPage

		for _, vuln := range cveResponse.Vulnerabilities {
			year, err = strconv.Atoi((*vuln.CVE.ID)[4:8])
			if err != nil {
				log.Fatal(err)
			}
			if _, ok := cvesByYear[year]; !ok {
				if cves, ok := cvesByYear[year-1]; ok {
					storeCVEs(dbDir, year-1, cves)
				}
			}
			cvesByYear[year] = append(cvesByYear[year], vuln)
		}
		time.Sleep(1 * time.Second)
	}
	storeCVEs(dbDir, year, cvesByYear[year])
	log.Printf("Initial sync done, duration: %s\n", time.Since(start))
}

func storeCVEs(dbDir string, year int, cves []nvdapi.CVEItem) {
	log.Printf("Storing vulnerabilities for year %d...\n", year)
	path := filepath.Join(dbDir, fmt.Sprintf("CVE-%d.json", year))
	sort.Slice(cves, func(i, j int) bool {
		return *cves[i].CVE.ID < *cves[j].CVE.ID
	})
	data, err := json.MarshalIndent(cves, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o777); err != nil {
		log.Fatal(err)
	}
}

func convertNewCVEToOld(cves nvdapi.CVEItem) []schema.NVDCVEFeedJSON10DefCVEItem {
}
