package tables

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// a var so tests can point it at a local server
var fmaScriptHashMapBaseURL = "https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/tools/software/fma-script-hashes/"

const (
	installScriptHashMapFile   = "install_script_hash_map.txt"
	uninstallScriptHashMapFile = "uninstall_script_hash_map.txt"
	fmaScriptHashMapTimeout    = 30 * time.Second
	// Some uninstall scripts get substituted with installer metadata, so we can't know
	// whether they were changed. tools/software/fma-script-hashes writes this where the
	// hash would go for those apps, and we mark them as unedited.
	substitutedScriptMarker = "substituted"
)

func init() {
	MigrationClient.AddMigration(Up_20260827170502, Down_20260827170502)
}

func Up_20260827170502(tx *sql.Tx) error {
	if !columnsExists(tx, "software_installers", "install_script_edited", "uninstall_script_edited") {
		if _, err := tx.Exec(`
			ALTER TABLE software_installers
			ADD COLUMN install_script_edited TINYINT(1) NOT NULL DEFAULT 0,
			ADD COLUMN uninstall_script_edited TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("adding script edited columns to software_installers: %w", err)
		}
	}

	step := incrementalMigrationStep(countMaintainedAppInstallers, backfillScriptEditedFlags)
	if err := step(tx); err != nil {
		return fmt.Errorf("backfilling script edited flags: %w", err)
	}
	return nil
}

func countMaintainedAppInstallers(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM software_installers si
		JOIN fleet_maintained_apps fma ON fma.id = si.fleet_maintained_app_id`).Scan(&total)
	return total, err
}

type installerScriptHashes struct {
	ID                  uint   `db:"id"`
	Slug                string `db:"slug"`
	InstallHash         string `db:"install_hash"`
	UninstallHash       string `db:"uninstall_hash"`
	ChangedAfterHashMap bool   `db:"changed_after_hash_map"`
}

func backfillScriptEditedFlags(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// every script Fleet has published; anything else was written by an admin.
	// The cutoff is the same in both files, so either copy will do.
	install, hashMapCutoff, err := fetchScriptHashMap(installScriptHashMapFile)
	if err != nil {
		return err
	}
	uninstall, _, err := fetchScriptHashMap(uninstallScriptHashMapFile)
	if err != nil {
		return err
	}

	const batchSize = 100
	var lastID uint
	for {
		// hash each installer's scripts in MySQL so the bodies never cross the wire
		var rows []installerScriptHashes
		if err := txx.Select(&rows, `
			SELECT
				si.id,
				fma.slug,
				SHA2(sci.contents, 256) AS install_hash,
				SHA2(scu.contents, 256) AS uninstall_hash,
				si.updated_at > ? AS changed_after_hash_map
			FROM software_installers si FORCE INDEX (PRIMARY)
			JOIN fleet_maintained_apps fma ON fma.id = si.fleet_maintained_app_id
			JOIN script_contents sci ON sci.id = si.install_script_content_id
			JOIN script_contents scu ON scu.id = si.uninstall_script_content_id
			WHERE si.id > ?
			ORDER BY si.id
			LIMIT ?`, hashMapCutoff.UTC(), lastID, batchSize); err != nil {
			return fmt.Errorf("selecting fleet-maintained app installers after id %d: %w", lastID, err)
		}
		if len(rows) == 0 {
			return nil
		}

		// a script Fleet never published was written by an admin, so the cron has to
		// keep carrying it forward
		var installerIDsWithEditedInstallScript []uint
		var installerIDsWithEditedUninstallScript []uint
		for _, row := range rows {
			increment()
			// a row changed after the hash map was generated may be carrying a script
			// Fleet published later, which the map can't contain, so a miss proves
			// nothing. Setting the flag would make the cron keep this script instead
			// of the manifest's on every later run, so it stays at the default.
			if row.ChangedAfterHashMap {
				continue
			}
			if _, published := install[row.InstallHash+" "+row.Slug]; !published {
				installerIDsWithEditedInstallScript = append(installerIDsWithEditedInstallScript, row.ID)
			}

			// a substituted uninstall script can't be told apart from an edited one
			_, uninstallSubstituted := uninstall[substitutedScriptMarker+" "+row.Slug]
			_, uninstallPublished := uninstall[row.UninstallHash+" "+row.Slug]
			if !uninstallSubstituted && !uninstallPublished {
				installerIDsWithEditedUninstallScript = append(installerIDsWithEditedUninstallScript, row.ID)
			}
		}

		// only the edited ones are written; the column already defaults to 0
		if len(installerIDsWithEditedInstallScript) > 0 {
			query, args, err := sqlx.In(`UPDATE software_installers SET install_script_edited = 1 WHERE id IN (?)`, installerIDsWithEditedInstallScript)
			if err != nil {
				return fmt.Errorf("building install_script_edited update: %w", err)
			}
			_, err = txx.Exec(query, args...)
			if err != nil {
				return fmt.Errorf("marking install_script_edited on %d installers: %w", len(installerIDsWithEditedInstallScript), err)
			}
		}
		if len(installerIDsWithEditedUninstallScript) > 0 {
			query, args, err := sqlx.In(`UPDATE software_installers SET uninstall_script_edited = 1 WHERE id IN (?)`, installerIDsWithEditedUninstallScript)
			if err != nil {
				return fmt.Errorf("building uninstall_script_edited update: %w", err)
			}
			_, err = txx.Exec(query, args...)
			if err != nil {
				return fmt.Errorf("marking uninstall_script_edited on %d installers: %w", len(installerIDsWithEditedUninstallScript), err)
			}
		}

		lastID = rows[len(rows)-1].ID
	}
}

// A failure here stops the migration: without the hashes every installer would
// look unedited, and the cron would overwrite scripts an admin wrote.
func fetchScriptHashMap(name string) (scriptHashMap, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fmaScriptHashMapTimeout)
	defer cancel()

	url := fmaScriptHashMapBaseURL + name
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("create http request: %w", err)
	}

	res, err := fleethttp.NewClient(fleethttp.WithTimeout(fmaScriptHashMapTimeout)).Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("fetching %s: HTTP status %d", url, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("read http response body: %w", err)
	}

	hashes, hashMapCutoff, err := parseScriptHashMap(string(body))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("parsing %s: %w", url, err)
	}
	return hashes, hashMapCutoff, nil
}

// keyed by the generated line, "<sha256> <slug>".
type scriptHashMap map[string]struct{}

// a "do not edit" note, then the commit it was cut at with that commit's date
const scriptHashMapHeaderLines = 2

func parseScriptHashMap(contents string) (scriptHashMap, time.Time, error) {
	lines := strings.Split(strings.TrimSpace(contents), "\n")
	if len(lines) < scriptHashMapHeaderLines {
		return nil, time.Time{}, errors.New("too short to hold the generated header")
	}

	// the cutoff bounds which installers these hashes can speak to. Parsed here
	// rather than in MySQL, which can't compare RFC 3339 against a TIMESTAMP.
	header := strings.Fields(lines[1])
	if len(header) != 4 || header[0] != "#" || header[1] != "to" {
		return nil, time.Time{}, fmt.Errorf(`expected a "# to <commit> <date>" header, got %q`, lines[1])
	}
	hashMapCutoff, err := time.Parse(time.RFC3339, header[3])
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("parsing cutoff %q: %w", header[3], err)
	}

	// the rest of the file is one "<sha256> <slug>" per line, which is the key
	hashes := scriptHashMap{}
	for _, line := range lines[scriptHashMapHeaderLines:] {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, time.Time{}, fmt.Errorf("unexpected line %q", line)
		}
		hashes[fields[0]+" "+fields[1]] = struct{}{}
	}
	return hashes, hashMapCutoff, nil
}

func Down_20260827170502(tx *sql.Tx) error {
	return nil
}
