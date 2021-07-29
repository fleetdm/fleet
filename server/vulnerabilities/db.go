package vulnerabilities

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
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

func applyCPEDatabaseSchema(dbPath string) error {
	db, err := sqliteDB(dbPath)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
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
	if err != nil {
		return err
	}
	return nil
}

func applyCVEDatabaseSchema(dbPath string) error {
	db, err := sqliteDB(dbPath)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS cve (
		product TEXT NOT NULL,
		cve_data TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_product ON cve (product);
	`)
	if err != nil {
		return err
	}
	return nil
}

func InsertCPEItem(db *sqlx.DB, item cpedict.CPEItem) ([]interface{}, map[string]string, error) {
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
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	db, err := sqliteDB(path)
	if err != nil {
		return err
	}
	err = applyCPEDatabaseSchema(path)
	if err != nil {
		return err
	}

	cpesCount := 0
	var allCPEs []interface{}
	deprecationsCount := 0
	var allDeprecations []interface{}

	for _, item := range items.Items {
		cpes, deprecations, err := InsertCPEItem(db, item)
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

// Based on nvdtools code
// TODO: check whether we need to post
func parseCVEJSON(in io.Reader) ([]*schema.NVDCVEFeedJSON10DefCVEItem, error) {
	feed, err := getFeed(in)
	if err != nil {
		return nil, fmt.Errorf("cvefeed.ParseJSON: %v", err)
	}
	return feed.CVEItems, nil
}

func getFeed(in io.Reader) (*schema.NVDCVEFeedJSON10, error) {
	reader, err := setupReader(in)
	if err != nil {
		return nil, fmt.Errorf("can't setup reader: %v", err)
	}
	defer reader.Close()

	var feed schema.NVDCVEFeedJSON10
	if err := json.NewDecoder(reader).Decode(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func setupReader(in io.Reader) (src io.ReadCloser, err error) {
	r := bufio.NewReader(in)
	header, err := r.Peek(2)
	if err != nil {
		return nil, err
	}
	// assume plain text first
	src = ioutil.NopCloser(r)
	// replace with gzip.Reader if gzip'ed
	if header[0] == 0x1f && header[1] == 0x8b { // file is gzip'ed
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		src = zr
	} else if header[0] == 'B' && header[1] == 'Z' {
		// or with bzip2.Reader if bzip2'ed
		src = ioutil.NopCloser(bzip2.NewReader(r))
	}
	return src, nil
}

func GenerateCVEDB(dbPath string, cveFeedReaders ...io.Reader) error {
	err := os.Remove(dbPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	db, err := sqliteDB(dbPath)
	if err != nil {
		return err
	}
	err = applyCVEDatabaseSchema(dbPath)
	if err != nil {
		return err
	}

	cveCount := 0
	var cveArgs []interface{}
	for _, reader := range cveFeedReaders {
		cves, err := parseCVEJSON(reader)
		if err != nil {
			return err
		}
		for _, cve := range cves {
			vuln := nvd.ToVuln(cve)
			for _, cpe := range vuln.Config() {
				if cpe == nil {
					continue
				}
				product := cpe.Product
				if wfn.HasWildcard(product) {
					// we ignore wildcards for now because we don't want
					// to check the whole db for now
					// product = wfn.Any
					continue
				}
				//cve.Impact = nil
				//if cve.CVE != nil {
				//	cve.CVE.Affects = nil
				//	cve.CVE.Description = nil
				//	cve.CVE.Problemtype = nil
				//	cve.CVE.References = nil
				//}
				cveBytes, err := json.Marshal(cve.Configurations.Nodes)
				//cveBytes, err := json.Marshal(cve)
				if err != nil {
					return err
				}
				//var compressedBytes bytes.Buffer
				//w := gzip.NewWriter(&compressedBytes)
				//w.Write(cveBytes)
				//w.Close()
				cveArgs = append(cveArgs, product, string(cveBytes))
				cveCount++
			}
			if cveCount > batchSize {
				err = bulkInsertCVEs(cveCount, db, cveArgs)
				if err != nil {
					return err
				}
				cveCount = 0
				cveArgs = []interface{}{}
			}
		}
		if cveCount > 0 {
			err = bulkInsertCVEs(cveCount, db, cveArgs)
			if err != nil {
				return err
			}
			cveCount = 0
			cveArgs = []interface{}{}
		}
	}

	return nil
}

func bulkInsertCVEs(cveCount int, db *sqlx.DB, cveArgs []interface{}) error {
	values := strings.TrimSuffix(strings.Repeat("(?, ?),", cveCount), ",")
	_, err := db.Exec(
		fmt.Sprintf(`INSERT INTO cve(product, cve_data) VALUES %s`, values),
		cveArgs...,
	)
	return err
}
