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

	relNotes, err := macoffice.ParseReleaseHTML(res.Body)
	panicif(err)

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
