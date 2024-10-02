package tables

import (
	"crypto/md5"
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
	SELECT DISTINCT uninstall_script_content_id
	FROM software_installers
	WHERE platform = "darwin" AND extension = "pkg"
`
	var ids []uint
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	if err := txx.Select(&ids, getUninstallScriptIDs); err != nil {
		return fmt.Errorf("failed to find uninstall script IDs: %w", err)
	}

	updateScriptContents := func(id uint, contents string) error {
		const stmt = `
		UPDATE script_contents
		SET contents = ?, md5_checksum = UNHEX(?)
		WHERE id = ?
		`
		rawChecksum := md5.Sum([]byte(contents)) //nolint:gosec
		md5Checksum := []byte(strings.ToUpper(hex.EncodeToString(rawChecksum[:])))
		_, err := tx.Exec(stmt, contents, md5Checksum, id)
		if err != nil {
			return fmt.Errorf("update script contents: %w", err)
		}

		return nil
	}

	// Go to script contents and check if it is the default uninstall script
	for _, id := range ids {
		getUninstallScript := `
		SELECT contents
		FROM script_contents
		WHERE id = ?
	`
		var contents string
		if err := txx.Get(&contents, getUninstallScript, id); err != nil {
			return fmt.Errorf("failed to find uninstall script content: %w", err)
		}

		// Check if it matches the regex
		matches := existingUninstallScript.FindStringSubmatch(contents)
		if matches != nil {
			packageIDs := matches[existingUninstallScript.SubexpIndex("packageIDs")]

			// Prepare new script
			newContents := strings.ReplaceAll(newScript, "$PACKAGE_ID", fmt.Sprintf("(%s)", packageIDs))
			// Write new script
			if err := updateScriptContents(id, newContents); err != nil {
				return fmt.Errorf("failed to update uninstall script content for script ID %d: %w", id, err)
			}
		}
	}

	return nil
}

func Down_20241002104104(_ *sql.Tx) error {
	return nil
}
