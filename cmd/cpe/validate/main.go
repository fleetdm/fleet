package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/jmoiron/sqlx"
)

func main() {
	dbDir := flag.String("db_dir", "/tmp/vulndbs", "Path to the vulnerability database")
	flag.Parse()

	vulnPath := *dbDir
	checkCPETranslations(vulnPath)
	checkMacOfficeNotes(vulnPath)
	checkMSRCVulnerabilities(vulnPath)
	checkSqliteDb(vulnPath)
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
				panic(fmt.Sprintf("failed to read MacOffice release notes file %s: %v", file.Name(), err))
			}
			// Attempt to parse the file as a MacOffice release notes.
			relNotes := macoffice.ReleaseNotes{}
			err = json.Unmarshal(payload, &relNotes)
			if err != nil {
				panic(fmt.Sprintf("failed to parse MacOffice release notes %s: %v", file.Name(), err))
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
				panic(fmt.Sprintf("failed to parse MSRC feed %s: %v", file.Name(), err))
			}
		}
	}
}

func checkSqliteDb(vulnPath string) {
	// Iterate over each file in the vulnPath directory to find the sqlite.gz file
	files, err := os.ReadDir(vulnPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read directory: %v", err))
	}
	var sqliteFilename string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sqlite.gz") {
			sqliteFilename = file.Name()
			break
		}
	}
	if sqliteFilename == "" {
		panic(fmt.Sprintf("no sqlite.gz file found: %v", err))
	}
	// Unzip the sqlite.gz file and create a new sqlite.db file
	gzFile, err := os.Open(filepath.Join(vulnPath, sqliteFilename))
	if err != nil {
		panic(fmt.Sprintf("error opening sqlite.gz file: %v", err))
	}
	defer gzFile.Close()
	sqliteFile, err := os.Create(filepath.Join(vulnPath, "sqlite.db"))
	if err != nil {
		panic(fmt.Sprintf("error creating test sqlite.db file: %v", err))
	}
	defer sqliteFile.Close()
	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		panic(fmt.Sprintf("error creating new gzip reader: %v", err))
	}
	defer gzReader.Close()
	for {
		_, err := io.CopyN(sqliteFile, gzReader, 100*1024*1024)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(fmt.Sprintf("error unzipping sqlite file: %v", err))
		}
	}
	db, err := sqlx.Open("sqlite3", filepath.Join(vulnPath, "sqlite.db"))
	if err != nil {
		panic(fmt.Sprintf("error opening sqlite db: %v", err))
	}
	// Check that the database is valid
	_, err = db.Exec(`SELECT * FROM cpe_2 LIMIT 1`)
	if err != nil {
		panic(fmt.Sprintf("error executing query on sqlite db: %v", err))
	}
}
