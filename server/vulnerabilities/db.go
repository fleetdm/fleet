package vulnerabilities

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func sqliteDB(dbPath string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func applyCPEDatabaseSchema(db *sqlx.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS cpe (
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
	return err
}

func generateCPEItem(item cpedict.CPEItem) ([]interface{}, map[string]string, error) {
	var cpes []interface{}
	deprecations := make(map[string]string)

	targetSW := wfn.StripSlashes(item.CPE23.Name.TargetSW)
	version := wfn.StripSlashes(item.CPE23.Name.Version)
	title := item.Title["en-US"]
	cpe23 := wfn.Attributes(item.CPE23.Name).BindToFmtString()
	cpes = append(cpes, cpe23, title, version, targetSW, item.Deprecated)

	if item.CPE23.Deprecation != nil {
		for _, deprecatedBy := range item.CPE23.Deprecation.DeprecatedBy {
			deprecatedByCPE23 := wfn.Attributes(deprecatedBy.Name).BindToFmtString()
			deprecations[cpe23] = deprecatedByCPE23
		}
	}

	return cpes, deprecations, nil
}

const batchSize = 800

func GenerateCPEDB(path string, items *cpedict.CPEList) error {
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	db, err := sqliteDB(path)
	if err != nil {
		return err
	}
	defer db.Close()

	err = applyCPEDatabaseSchema(db)
	if err != nil {
		return err
	}

	cpesCount := 0
	var allCPEs []interface{}
	deprecationsCount := 0
	var allDeprecations []interface{}

	for _, item := range items.Items {
		cpes, deprecations, err := generateCPEItem(item)
		if err != nil {
			return err
		}
		allCPEs = append(allCPEs, cpes...)
		cpesCount++
		if len(deprecations) > 0 {
			deprecationsCount++
		}
		for key, val := range deprecations {
			allDeprecations = append(allDeprecations, key, val)
		}
		if cpesCount > batchSize {
			err = bulkInsertCPEs(cpesCount, db, allCPEs)
			if err != nil {
				return err
			}
			allCPEs = []interface{}{}
			cpesCount = 0
		}
		if deprecationsCount > batchSize {
			err := bulkInsertDeprecations(deprecationsCount, db, allDeprecations)
			if err != nil {
				return err
			}
			allDeprecations = []interface{}{}
			deprecationsCount = 0
		}
	}
	if cpesCount > 0 {
		err = bulkInsertCPEs(cpesCount, db, allCPEs)
		if err != nil {
			return err
		}
	}
	if deprecationsCount > 0 {
		err := bulkInsertDeprecations(deprecationsCount, db, allDeprecations)
		if err != nil {
			return err
		}
	}

	_, err = db.Exec(`INSERT INTO cpe_search(rowid, title, target_sw) select rowid, title, target_sw from cpe`)
	if err != nil {
		return err
	}
	return nil
}

func bulkInsertDeprecations(deprecationsCount int, db *sqlx.DB, allDeprecations []interface{}) error {
	values := strings.TrimSuffix(strings.Repeat("((SELECT rowid FROM CPE where cpe23=?), ?),", deprecationsCount), ",")
	_, err := db.Exec(
		fmt.Sprintf(`INSERT INTO deprecated_by(cpe_id, cpe23) VALUES %s`, values),
		allDeprecations...,
	)
	return err
}

func bulkInsertCPEs(cpesCount int, db *sqlx.DB, allCPEs []interface{}) error {
	values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?),", cpesCount), ",")
	_, err := db.Exec(
		fmt.Sprintf(`INSERT INTO cpe(cpe23, title, version, target_sw, deprecated) VALUES %s`, values),
		allCPEs...,
	)
	return err
}
