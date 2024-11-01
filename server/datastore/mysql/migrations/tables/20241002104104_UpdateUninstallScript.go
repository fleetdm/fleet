package tables

import (
	"crypto/md5" //nolint:gosec
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

//go:embed data/uninstall_pkg.sh
var newScript string

func init() {
	MigrationClient.AddMigration(Up_20241002104104, Down_20241002104104)
}

func Up_20241002104104(tx *sql.Tx) error {
	existingUninstallScript := regexp.MustCompile(
		`(?sU)^#!/bin/sh\n\n` +
			`# Fleet extracts and saves package IDs.\n` +
			`pkg_ids=\((?P<packageIDs>.*)\)\n\n` +
			`# Get all files associated with package and remove them\n` +
			`for pkg_id in "\$\{pkg_ids\[@]}"\n` +
			`do\n` +
			`  # Get volume and location of package\n` +
			`  volume=\$\(pkgutil --pkg-info "\$pkg_id" \| grep -i "volume" \| awk '\{for \(i=2; i<NF; i\+\+\) printf \$i " "; print \$NF}'\)\n` +
			`  location=\$\(pkgutil --pkg-info "\$pkg_id" \| grep -i "location" \| awk '\{for \(i=2; i<NF; i\+\+\) printf \$i " "; print \$NF}'\)\n` +
			`  # Check if this package id corresponds to a valid/installed package\n` +
			`  if \[\[ ! -z "\$volume" && ! -z "\$location" ]]; then\n` +
			`    # Remove individual files/directories belonging to package\n` +
			`    pkgutil --files "\$pkg_id" \| sed -e 's@\^@'"\$volume""\$location"'/@' \| tr '\\n' '\\0' \| xargs -n 1 -0 rm -rf\n` +
			`    # Remove receipts\n` +
			`    pkgutil --forget "\$pkg_id"\n` +
			`  else\n` +
			`    echo "WARNING: volume or location are empty for package ID \$pkg_id"\n` +
			`  fi\n` +
			`done\n$`)

	// Get script ids for uninstall scripts from software_installers platform = "darwin" and extension = "pkg"
	getUninstallScriptIDs := `
	SELECT id, uninstall_script_content_id
	FROM software_installers
	WHERE platform = "darwin" AND extension = "pkg"
`
	type scripts struct {
		ID              uint `db:"id"`
		ScriptContentID uint `db:"uninstall_script_content_id"`
	}

	var uninstallScripts []scripts
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	if err := txx.Select(&uninstallScripts, getUninstallScriptIDs); err != nil {
		return fmt.Errorf("failed to find uninstall script IDs: %w", err)
	}

	insertScriptContents := func(contents string) (uint, error) {
		const stmt = `
INSERT INTO
  script_contents (
	  md5_checksum, contents
  )
VALUES (UNHEX(?),?)
ON DUPLICATE KEY UPDATE
  id=LAST_INSERT_ID(id)
		`
		rawChecksum := md5.Sum([]byte(contents)) //nolint:gosec
		md5Checksum := []byte(strings.ToUpper(hex.EncodeToString(rawChecksum[:])))
		res, err := tx.Exec(stmt, md5Checksum, contents)
		if err != nil {
			return 0, fmt.Errorf("update script contents: %w", err)
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("get last insert ID: %w", err)
		}

		return uint(newID), nil //nolint:gosec // dismiss G115
	}

	// Go to script contents and check if it is the default uninstall script
	for _, script := range uninstallScripts {
		scriptContentID := script.ScriptContentID
		getUninstallScript := `
		SELECT contents
		FROM script_contents
		WHERE id = ?
	`
		var contents string
		// The script id must exist due to FK constraint on software_installers.uninstall_script_content_id
		if err := txx.Get(&contents, getUninstallScript, scriptContentID); err != nil {
			return fmt.Errorf("failed to find uninstall script content: %w", err)
		}

		// Check if script contents match the regex
		matches := existingUninstallScript.FindStringSubmatch(contents)
		if matches != nil {
			packageIDs := matches[existingUninstallScript.SubexpIndex("packageIDs")]

			// Prepare new script
			newContents := strings.ReplaceAll(newScript, "$PACKAGE_ID", fmt.Sprintf("(%s)", packageIDs))
			// Write new script
			newID, err := insertScriptContents(newContents)
			if err != nil {
				return fmt.Errorf("failed to update uninstall script content for script ID %d: %w", scriptContentID, err)
			}

			// Update software_installers to point to new script
			updateUninstallScript := `
			UPDATE software_installers
			SET uninstall_script_content_id = ?
			WHERE id = ?`
			if _, err := tx.Exec(updateUninstallScript, newID, script.ID); err != nil {
				return fmt.Errorf("failed to update uninstall script ID %d: %w", script.ID, err)
			}

		}
	}

	return nil
}

func Down_20241002104104(_ *sql.Tx) error {
	return nil
}
