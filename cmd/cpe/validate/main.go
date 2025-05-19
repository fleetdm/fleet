package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
)

func main() {
	dbDir := flag.String("db_dir", "/tmp/vulndbs", "Path to the vulnerability database")
	flag.Parse()

	vulnPath := *dbDir
	checkCPETranslations(vulnPath)
	checkMacOfficeNotes(vulnPath)
	checkMSRCVulnerabilities(vulnPath)
}

func checkCPETranslations(vulnPath string) {
	// Check that the CPE translations file is parseable into an array of CPETranslationItem
	_, err := nvd.LoadCPETranslations(filepath.Join(vulnPath, "cpe_translations.json"))
	if err != nil {
		panic(fmt.Sprintf("failed to load CPE translations: %v", err))
	}
}

func checkMacOfficeNotes(vulnPath string) {
	// Iterate over each file in the vulnPath directory that starts with `fleet_macoffice`
	files, err := os.ReadDir(vulnPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read directory: %v", err))
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "fleet_macoffice") && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(vulnPath, file.Name())

			payload, err := os.ReadFile(filePath)
			if err != nil {
				panic(fmt.Sprintf("failed to read MacOffice release notes file: %v", err))
			}
			// Attempt to parse the file as a MacOffice release notes.
			relNotes := macoffice.ReleaseNotes{}
			err = json.Unmarshal(payload, &relNotes)
			if err != nil {
				panic(fmt.Sprintf("failed to parse MacOffice release notes: %v", err))
			}
		}
	}
}

func checkMSRCVulnerabilities(vulnPath string) {
	// Iterate over each file in the vulnPath directory that starts with `fleet_msrc`
	files, err := os.ReadDir(vulnPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read directory: %v", err))
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "fleet_msrc") && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(vulnPath, file.Name())
			// Attempt to parse the file as a MSRC feed.
			_, err := parsed.UnmarshalBulletin(filePath)
			if err != nil {
				panic(fmt.Sprintf("failed to parse MSRC feed: %v", err))
			}
		}
	}
}
