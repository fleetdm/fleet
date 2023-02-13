package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
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
	res, err := http.Get(macoffice.RelNotesURL)
	panicif(err)

	parsed, err := macoffice.ParseReleaseHTML(res.Body)
	panicif(err)

	var relNotes []macoffice.ReleaseNote
	for _, rn := range parsed {
		// We only care about release notes that have a version set (because we need that for
		// matching software entries) and also that contain some
		// security updates (because we only intented to use the release notes for vulnerability processing).
		if rn.Valid() {
			relNotes = append(relNotes, rn)
		}
	}

	err = serialize(relNotes, time.Now(), outPath)
	panicif(err)

	fmt.Println("Done.")
}

func serialize(relNotes []macoffice.ReleaseNote, d time.Time, dir string) error {
	payload, err := json.Marshal(relNotes)
	if err != nil {
		return err
	}

	fileName := io.MacOfficeRelNotesFileName(d)
	filePath := filepath.Join(dir, fileName)

	return os.WriteFile(filePath, payload, 0o644)
}
