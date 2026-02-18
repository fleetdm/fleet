package tables

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260218165545, Down_20260218165545)
}

func Up_20260218165545(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	const findMismatchedSoftwareStmt = `
	SELECT 
		software.id id,
		software.source source,
		software.name name.
		software.bundle_identifier bundle_identifier,
		software.title_id title_id, 
		software_titles.source title_source
	FROM software 
		JOIN software_titles ON software.title_id = software_titles.id
	WHERE
		software.source != software_titles.source 
		AND software.bundle_identifier != '' -- limit scope to apple software
`

	type badSoftware struct {
		SoftwareID       uint   `db:"id"`
		SoftwareName     string `db:"name"`
		SoftwareSource   string `db:"source"`
		BundleIdentifier string `db:"bundle_identifier"`
		TitleID          uint   `db:"title_id"`
		TitleSource      string `db:"title_source"`
	}
	softwareList := make([]badSoftware, 0)

	// find the mismatched software
	err := txx.Select(&softwareList, findMismatchedSoftwareStmt)
	if err != nil {
		return errors.Wrap(err, "find mismatched software")
	}

	if len(softwareList) == 0 {
		return nil // nothing to do
	}

	for _, s := range softwareList {
		// find or create a title with the correct source
		newID, err := getOrInsertTitleID(txx, s.SoftwareName, s.BundleIdentifier, s.SoftwareSource)
		if err != nil {
			return errors.Wrap(err, "getting or inserting software title")
		}

		// update the software entry to use the correct title id
		const updateSoftwareTitleIDStmt = `UPDATE software SET title_id = ? WHERE id = ?`
		_, err = txx.Exec(updateSoftwareTitleIDStmt, newID, s.SoftwareID)
		if err != nil {
			return errors.Wrap(err, "updating software to use correct title id")
		}
	}

	return nil
}

// should be similar to ds.getOrGenerateSoftwareInstallerTitleID
func getOrInsertTitleID(txx sqlx.Tx, name, bundleIdentifier, source string) (uint, error) {
	const findTitleStmt = `
	SELECT id
	FROM software_titles 
	WHERE bundle_identifier = ? AND source = ?;
`
	const insertTitleStmt = `
	INSERT INTO software_titles (name, source, extension_for, bundle_identifier) VALUES (?, ?, ?, ?)
`
	var titleID uint
	err := txx.Get(&titleID, findTitleStmt, bundleIdentifier, source)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res, err := txx.Exec(insertTitleStmt, name, source, "", bundleIdentifier)
			if err != nil {
				return 0, errors.Wrap(err, "inserting new software title")
			}
			id, _ := res.LastInsertId()
			return uint(id), nil

		}
		return 0, errors.Wrapf(err, "find title for software with bundle identifier %s", bundleIdentifier)
	}
	return titleID, nil
}

func Down_20260218165545(tx *sql.Tx) error {
	return nil
}
