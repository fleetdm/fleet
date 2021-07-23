package vulnerabilities

import (
	"path"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	CPEDBName = "cpe.sqlite"
)

func CPEDB(dbPath string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", path.Join(dbPath, CPEDBName))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GenerateCPEDatabaseSkeleton(dbPath string) error {
	db, err := CPEDB(dbPath)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cpe (
		cpe23 TEXT NOT NULL,
		title TEXT NOT NULL,
		version TEXT,
		target_sw TEXT,
		deprecated BOOLEAN DEFAULT FALSE
	);
	CREATE TABLE IF NOT EXISTS deprecated_by (
		cpe_id INTEGER,
		cpe23 TEXT NOT NULL,
		FOREIGN KEY(cpe_id) REFERENCES cpe(rowid)
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS cpe_search USING fts5(title, target_sw);
	CREATE INDEX IF NOT EXISTS idx_version ON cpe (version);
	CREATE INDEX IF NOT EXISTS idx_cpe23 ON cpe (cpe23);
	CREATE INDEX IF NOT EXISTS idx_target_sw ON cpe (target_sw);
	CREATE INDEX IF NOT EXISTS idx_deprecated_by ON deprecated_by (cpe23);
	`)
	if err != nil {
		return err
	}
	return nil
}

func InsertCPEItem(db *sqlx.DB, item cpedict.CPEItem) error {
	targetSW := wfn.StripSlashes(item.CPE23.Name.TargetSW)
	version := wfn.StripSlashes(item.CPE23.Name.Version)
	title := item.Title["en-US"]
	cpe23 := wfn.Attributes(item.CPE23.Name).BindToFmtString()
	res, err := db.Exec(
		`INSERT INTO cpe(cpe23, title, version, target_sw, deprecated) VALUES (?, ?, ?, ?, ?)`,
		cpe23, title, version, targetSW, item.Deprecated,
	)
	if err != nil {
		return err
	}
	rowid, err := res.LastInsertId()
	if err != nil {
		return err
	}

	if item.CPE23.Deprecation != nil {
		for _, deprecatedBy := range item.CPE23.Deprecation.DeprecatedBy {
			deprecatedByCPE23 := wfn.Attributes(deprecatedBy.Name).BindToFmtString()
			_, err := db.Exec(`INSERT INTO deprecated_by(cpe_id, cpe23) VALUES (?, ?)`, rowid, deprecatedByCPE23)
			if err != nil {
				return err
			}
		}
	}

	_, err = db.Exec(`INSERT INTO cpe_search(rowid, title, target_sw) VALUES (?, ?, ?)`, rowid, title, targetSW)
	if err != nil {
		return err
	}
	return nil
}
