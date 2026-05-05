package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

// Generates Mac Office release notes metadata in JSON format, to be used by our vulnerability process.
func main() {
	wd, err := os.Getwd()
	panicif(err)

	outPath := filepath.Join(wd, "macoffice_rel_notes")
	err = os.MkdirAll(outPath, 0o755)
	panicif(err)

	fmt.Println("Downloading and parsing Mac Office rel notes...")

	relNotes, err := macoffice.GetReleaseNotes(false)
	panicif(err)

	err = relNotes.Serialize(time.Now(), outPath)
	panicif(err)

	fmt.Println("Done.")
}
