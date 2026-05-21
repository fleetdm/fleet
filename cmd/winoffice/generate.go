//go:build ignore

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/winoffice"
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

	fmt.Println("Scraping Windows Office security updates from Microsoft Learn...")
	bulletin, err := winoffice.FetchBulletin(context.Background(), client)
	panicif(err)

	fmt.Printf("Found %d versions\n", len(bulletin.Versions))

	fmt.Println("Saving Windows Office bulletin...")
	err = bulletin.Serialize(now, outPath)
	panicif(err)

	fmt.Println("Done.")
}
