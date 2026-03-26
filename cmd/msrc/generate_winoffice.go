//go:build ignore

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	wd, err := os.Getwd()
	panicif(err)

	outPath := filepath.Join(wd, "winoffice_out")
	err = os.MkdirAll(outPath, 0o755)
	panicif(err)

	now := time.Now()

	client := &http.Client{Timeout: 60 * time.Second}

	// Scrape Windows Office security updates from Microsoft Learn
	fmt.Println("Scraping Windows Office security updates from Microsoft Learn...")
	winOfficeBulletin, err := msrc.FetchWinOfficeBulletin(client)
	panicif(err)

	fmt.Printf("Found %d CVEs across %d supported versions\n",
		len(winOfficeBulletin.CVEToFixedBuilds), len(winOfficeBulletin.SupportedVersions))
	fmt.Printf("Supported versions: %v\n", winOfficeBulletin.SupportedVersions)
	fmt.Printf("Build prefixes: %d mappings\n", len(winOfficeBulletin.BuildPrefixToVersion))

	// Convert to AppBulletinFile format
	bulletinFile := msrc.ConvertWinOfficeToAppBulletin(winOfficeBulletin)

	fmt.Printf("Created bulletin with %d version branches\n", len(bulletinFile.Versions))

	fmt.Println("Saving Windows Office bulletins...")
	err = bulletinFile.SerializeAsWinOffice(now, outPath)
	panicif(err)

	// Show CVE counts per supported version
	fmt.Println("\nCVEs per supported version:")
	for _, version := range winOfficeBulletin.SupportedVersions {
		if vb, ok := bulletinFile.Versions[version]; ok {
			fmt.Printf("  %s: %d CVEs\n", version, len(vb.SecurityUpdates))
		}
	}

	fmt.Println("\nDone processing Office security updates.")
}
