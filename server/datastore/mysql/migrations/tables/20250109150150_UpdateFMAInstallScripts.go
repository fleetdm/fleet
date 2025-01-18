package tables

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250109150150, Down_20250109150150)
}

const quitApplicationFunc = `
quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    if osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1; then
      if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
        echo "Application '$bundle_id' quit successfully."
        quit_success=true
        break
      fi
    fi
    sleep 1
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
  fi
}
`

// This is a map from tokens to known app filenames. These app names differ from the name field we pull
// from fleet_library_apps.
var knownGoodAppFilenames = map[string]string{"visual-studio-code": "Visual Studio Code.app", "firefox": "Firefox.app", "brave-browser": "Brave Browser.app"}

func Up_20250109150150(tx *sql.Tx) error {
	var scriptsToModify []struct {
		InstallScriptContents string `db:"contents"`
		AppName               string `db:"name"`
		BundleID              string `db:"bundle_identifier"`
		ScriptContentID       uint   `db:"script_content_id"`
		Token                 string `db:"token"`
	}

	// Note: we're not updating any install scripts that have been edited by users, only the
	// "original" script contents for FMAs that are created when the fleet_library_apps table is populated.
	selectStmt := `
SELECT 
	sc.contents AS contents,
	fla.name AS name,
	fla.bundle_identifier AS bundle_identifier,
	sc.id AS script_content_id,
	fla.token AS token
FROM 
	fleet_library_apps fla 
	JOIN script_contents sc 
	ON fla.install_script_content_id = sc.id
WHERE fla.token IN (?)
`

	// This is the list of Fleet-maintained apps we want to update ("token" is an ID found on the brew
	// metadata)
	appTokens := []string{"1password", "brave-browser", "docker", "figma", "google-chrome", "visual-studio-code", "firefox", "notion", "slack", "whatsapp", "postman"}

	stmt, args, err := sqlx.In(selectStmt, appTokens)
	if err != nil {
		return fmt.Errorf("building SQL in statement for selecting fleet maintained apps: %w", err)
	}

	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	if err := txx.Select(&scriptsToModify, stmt, args...); err != nil {
		// if this migration is running on a brand-new Fleet deployment, then there won't be
		// anything in the fleet_library_apps table, so we can just exit.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("selecting script contents: %w", err)
	}

	for _, sc := range scriptsToModify {
		lines := strings.Split(sc.InstallScriptContents, "\n")
		// Find the line where we copy the new .app file into the Applications folder. We want to
		// add our changes right before that line.
		var copyLineNumber int
		for i, l := range lines {
			if strings.Contains(l, `sudo cp -R "$TMPDIR/`) {
				copyLineNumber = i
				break
			}
		}

		// Default to using the name we pulled + ".app". We know that is incorrect for some apps
		// though, so look them up in our map of known good names and use that if it exists.
		appFileName := fmt.Sprintf("%s.app", sc.AppName)
		if knownName, ok := knownGoodAppFilenames[sc.Token]; ok {
			appFileName = knownName
		}

		// This line will move the old version of the .app (if it exists) to the temporary directory
		lines = slices.Insert(lines, copyLineNumber, fmt.Sprintf(`sudo [ -d "$APPDIR/%[1]s" ] && sudo mv "$APPDIR/%[1]s" "$TMPDIR/%[1]s.bkp"`, appFileName))
		// Add a call to our "quit_application" function
		lines = slices.Insert(lines, copyLineNumber, fmt.Sprintf("quit_application %s", sc.BundleID))
		// Add the "quit_application" function to the script
		lines = slices.Insert(lines, 2, quitApplicationFunc)

		updatedScript := strings.Join(lines, "\n")

		checksum := md5ChecksumScriptContent(updatedScript)

		if _, err = tx.Exec(`UPDATE script_contents SET contents = ?, md5_checksum = UNHEX(?) WHERE id = ?`, strings.Join(lines, "\n"), checksum, sc.ScriptContentID); err != nil {
			return fmt.Errorf("updating fma install script contents: %w", err)
		}
	}

	return nil
}

func Down_20250109150150(tx *sql.Tx) error {
	return nil
}
