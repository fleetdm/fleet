//go:build ignore

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
	msrcapps "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/apps"
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

	// Convert to AppBulletinFile format
	bulletinFile := msrc.ConvertWinOfficeToAppBulletin(winOfficeBulletin)
	bulletinFile = bulletinFile.WithMappings(msrcapps.DefaultMappings())

	fmt.Printf("Created %d product bulletins\n", len(bulletinFile.Products))

	fmt.Println("Saving Windows Office bulletins...")
	err = bulletinFile.SerializeAsWinOffice(now, outPath)
	panicif(err)

	// Show sample of CVEs from first product
	if len(bulletinFile.Products) > 0 {
		fmt.Println("\nSample CVEs (first 10 from first product):")
		for i, su := range bulletinFile.Products[0].SecurityUpdates {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(bulletinFile.Products[0].SecurityUpdates)-10)
				break
			}
			fmt.Printf("  %s -> %s\n", su.CVE, su.FixedVersion)
		}
	}

	fmt.Println("\nDone processing Office security updates.")
}
