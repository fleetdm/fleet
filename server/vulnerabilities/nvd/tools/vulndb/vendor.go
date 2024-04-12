// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vulndb

import (
	"context"
	"database/sql"
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/debug"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/sqlutil"
)

// VendorRecord represents a db record of the `vendor` table.
type VendorRecord struct {
	Version  int64     `sql:"version"`
	TS       time.Time `sql:"ts"`
	Ready    bool      `sql:"ready"`
	Owner    string    `sql:"owner"`
	Provider string    `sql:"provider"`
}

// VendorDataRecord represents a db record of the `vendor_data` table.
type VendorDataRecord struct {
	Version   int64     `sql:"version"`
	CVE       string    `sql:"cve_id"`
	Published time.Time `sql:"published"`
	Modified  time.Time `sql:"modified"`
	BaseScore float64   `sql:"base_score"`
	Summary   string    `sql:"summary"`
	JSON      []byte    `sql:"cve_json"`
}

// VendorDataImporter is a helper for importing an entire dataset
// from multiple files.
type VendorDataImporter struct {
	DB       *sql.DB
	Owner    string
	Provider string
	OnFile   func(filename string)
}

// ImportFiles creates a new dataset version and imports all files into it
// Files must be formatted as NVD CVE JSON 1.0 optionally gzipped.
func (v VendorDataImporter) ImportFiles(ctx context.Context, files ...string) (*VendorRecord, error) {
	vendor, err := v.newVersion(ctx, v.Owner, v.Provider)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if v.OnFile != nil {
			v.OnFile(file)
		}
		records, err := VendorDataFromFile(vendor, file)
		if err != nil {
			return nil, err
		}

		err = v.importData(ctx, records)
		if err != nil {
			return nil, err
		}
	}

	err = v.enableVersion(ctx, vendor)
	if err != nil {
		return nil, err
	}

	return vendor, nil
}

func (v VendorDataImporter) newVersion(ctx context.Context, owner, provider string) (*VendorRecord, error) {
	vendor := VendorRecord{
		TS:       time.Now().UTC(),
		Ready:    false,
		Owner:    owner,
		Provider: provider,
	}

	r := sqlutil.NewRecordType(vendor).Subset(
		"ts",
		"ready",
		"owner",
		"provider",
	)

	q := sqlutil.Insert().
		Into("vendor").
		Fields(r.Fields()...).
		Values(r)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	res, err := v.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "cannot insert vendor record")
	}

	version, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get last id from vendor record")
	}

	vendor.Version = version
	return &vendor, nil
}

func (v VendorDataImporter) replaceVendorData(ctx context.Context, records sqlutil.Records) error {
	q := sqlutil.Replace().
		Into("vendor_data").
		Fields(records.Fields()...).
		Values(records...)

	query, args := q.String(), q.QueryArgs()

	if debug.V(2) {
		flog.Infof("running: %q", query)
	}

	_, err := v.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot insert vendor data records")
	}
	return nil
}

func (v VendorDataImporter) replaceVendorDataBatch(ctx context.Context, records sqlutil.Records) error {
	// when sub-batch gets inserted, but some other fails, don't insert the succeeded one
	from := 0

OuterLoop:
	// start with full size and gradually double the size down
	for batchSize := len(records); batchSize > 0; batchSize /= 2 {
		for idx := from; idx < len(records); idx += batchSize {
			limit := idx + batchSize
			if limit > len(records) {
				limit = len(records)
			}
			if err := v.replaceVendorData(ctx, records[idx:limit]); err != nil {
				continue OuterLoop
			}
			// succeeded, move the from to the new location
			from = limit
		}
		// if it didn't continue before here, then all inserted
		return nil
	}

	// if it came to here, means it didn't insert
	return errors.New("can't insert batch")
}

func (v VendorDataImporter) importData(ctx context.Context, data []VendorDataRecord) error {
	records := sqlutil.NewRecords(data)

	const batchSize = 100

	// the next few lines insert records into vendor_data in batches
	// if inserting a batch fails, then we subdivide the batch into half and try to insert that
	// repeat the process until the batch size comes to 0

	for i := 0; i < len(records); i += batchSize {
		limit := i + batchSize
		if limit > len(records) {
			limit = len(records)
		}

		if err := v.replaceVendorDataBatch(ctx, records[i:limit]); err != nil {
			return errors.Wrap(err, "cannot insert vendor data records")
		}
	}

	return nil
}

func (v VendorDataImporter) enableVersion(ctx context.Context, vendor *VendorRecord) error {
	q := sqlutil.Update("vendor").Set(
		sqlutil.Assign().Equal("ready", true),
	).Where(
		sqlutil.Cond().Equal("version", vendor.Version),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	_, err := v.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot update vendor record")
	}

	return nil
}

// VendorDataFromFile loads vendor data from NVD CVE JSON files.
func VendorDataFromFile(vendor *VendorRecord, name string) ([]VendorDataRecord, error) {
	feed, err := readNVDCVEJSON(name)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load vendor file")
	}

	records := make([]VendorDataRecord, len(feed.CVEItems))

	for i, item := range feed.CVEItems {
		cve := cveItem{item}
		records[i] = VendorDataRecord{
			Version:   vendor.Version,
			CVE:       cve.ID(),
			Published: cve.Published(),
			Modified:  cve.Modified(),
			BaseScore: cve.BaseScore(),
			Summary:   cve.Summary(),
			JSON:      cve.JSON(),
		}
	}

	return records, nil
}

// VendorDataExporter is a helper for exporting vendor data.
type VendorDataExporter struct {
	DB         *sql.DB
	Provider   string
	FilterCVEs []string
}

func (v VendorDataExporter) condition() *sqlutil.QueryConditionSet {
	cond := sqlutil.Cond().InSelect("vendor.version",
		sqlutil.Select("latest.version").
			From().
			SelectGroup("latest", latestVendorVersion()).
			Where(
				sqlutil.Cond().Equal("provider", v.Provider),
			),
	)

	if len(v.FilterCVEs) > 0 {
		cond = cond.And().In("vendor_data.cve_id", v.FilterCVEs)
	}

	return cond
}

// CSV writes vendor data records to w.
func (v VendorDataExporter) CSV(ctx context.Context, w io.Writer, header bool) error {
	q := sqlutil.Select(
		"vendor.version AS version",
		"vendor.ts AS ts",
		"vendor.owner AS owner",
		"vendor.provider AS provider",
		"vendor_data.cve_id AS cve_id",
		"vendor_data.published AS published",
		"vendor_data.modified AS modified",
		"vendor_data.base_score AS base_score",
		"vendor_data.summary AS summary",
	).From(
		"vendor_data",
	).Literal(
		"LEFT JOIN vendor ON vendor.version = vendor_data.version",
	).Where(
		v.condition(),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := v.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot query vendor data")
	}

	defer rows.Close()

	record := struct {
		Version   string    `sql:"version"`
		TS        time.Time `sql:"ts"`
		Owner     string    `sql:"owner"`
		Provider  string    `sql:"provider"`
		CVE       string    `sql:"cve_id"`
		Published time.Time `sql:"published"`
		Modified  time.Time `sql:"modified"`
		BaseScore float64   `sql:"base_score"`
		Summary   string    `sql:"summary"`
	}{}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	if header {
		fields := sqlutil.NewRecordType(record).Fields()
		cw.Write(fields)
	}

	for rows.Next() {
		v := record
		err = rows.Scan(sqlutil.NewRecordType(&v).Values()...)
		if err != nil {
			return errors.Wrap(err, "cannot scan vendor data")
		}

		cw.Write([]string{
			v.Version,
			v.TS.Format(TimeLayout),
			v.Owner,
			v.Provider,
			v.CVE,
			v.Published.Format(TimeLayout),
			v.Modified.Format(TimeLayout),
			strconv.FormatFloat(v.BaseScore, 'f', 3, 64),
			v.Summary,
		})
	}

	return nil
}

// JSON writes NVD CVE JSON to w.
func (v VendorDataExporter) JSON(ctx context.Context, w io.Writer, indent string) error {
	q := sqlutil.Select(
		"cve_id",
		"cve_json",
	).From(
		"vendor_data",
	).Literal(
		"LEFT JOIN vendor ON vendor.version = vendor_data.version",
	).Where(
		v.condition(),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := v.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot query vendor data")
	}

	defer rows.Close()

	record := struct {
		CVE  string
		JSON []byte
	}{}

	f := &cveFile{}
	for rows.Next() {
		v := record
		err = rows.Scan(sqlutil.NewRecordType(&v).Values()...)
		if err != nil {
			return errors.Wrap(err, "cannot scan vendor data")
		}

		err = f.Add(v.CVE, v.JSON)
		if err != nil {
			return err
		}
	}

	if indent == "" {
		return f.EncodeJSON(w)
	}

	const prefix = ""
	return f.EncodeIndentedJSON(w, prefix, indent)
}

// VendorDataTrimmer is a helper for trimming vendor data.
//
// It deletes all versions but the latest.
//
// Deleting would be easier in common scenarions, but we have some hard
// constraints:
//
//   - Vendor data is versioned
//   - No foreign key between vendor_data and vendor tables
//   - MySQL in safe mode forbids deleting from SELECT queries, wants values
//   - Must keep the binlog smaller than 500M, not enough for the NVD database
//
// Therefore, deletion works as follows:
//
//   - Select versions from the vendor table based on the provided settings
//   - Operate on vendor records with ready=true or older versions
//   - By default, delete all versions but the latest, for each provider
//   - Delete from vendor table first, effectively making data records orphans
//   - Delete any orphan records from vendor_data, effectively crowd sourcing deletions
//   - Delete data in chunks, keeping binlog small
//
// Deletion operations are expensive.
type VendorDataTrimmer struct {
	DB                  *sql.DB
	FilterProviders     []string
	DeleteLatestVersion bool // TODO: support keeping up to N versions
}

// Trim deletes vendor data versions from the database.
func (v VendorDataTrimmer) Trim(ctx context.Context) error {
	err := v.deleteVendors(ctx)
	if err != nil {
		return err
	}

	return v.deleteOrphanData(ctx)
}

func (v VendorDataTrimmer) deleteVendors(ctx context.Context) error {
	versions, err := v.selectVendorVersions(ctx)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return nil
	}

	q := sqlutil.Delete().From(
		"vendor",
	).Where(
		sqlutil.Cond().In("version", versions),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	_, err = v.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot delete vendor data")
	}

	return nil
}

func (v VendorDataTrimmer) deleteOrphanData(ctx context.Context) error {
	versions, err := v.selectOrphanDataVersions(ctx)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return nil
	}

	q := sqlutil.Delete().From(
		"vendor_data",
	).Where(
		sqlutil.Cond().In("version", versions),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	q = q.Literal("LIMIT 100")

	for {
		res, err := v.DB.ExecContext(ctx, q.String(), q.QueryArgs()...)
		if err != nil {
			return errors.Wrap(err, "cannot delete vendor data")
		}

		n, err := res.RowsAffected()
		if err != nil {
			return errors.Wrap(err, "cannot get rows affected")
		}

		if n == 0 {
			break
		}
	}

	return nil
}

func (v VendorDataTrimmer) selectVendorVersions(ctx context.Context) ([]int64, error) {
	cond := sqlutil.Cond().Group(
		sqlutil.Cond().
			Equal("ready", true).
			Or().
			Group( // this is to delete stale data from failed imports
				sqlutil.Cond().
					Equal("ready", false).
					And().
					Literal("ts < DATE_SUB(NOW(), INTERVAL 1 DAY)"),
			),
	)

	if len(v.FilterProviders) > 0 {
		cond = cond.And().In("provider", v.FilterProviders)
	}

	if !v.DeleteLatestVersion {
		cond = cond.And().Not().InSelect(
			"version",
			sqlutil.Select("latest.version").
				From().
				SelectGroup("latest", latestVendorVersion()),
		)
	}

	q := sqlutil.Select(
		"version",
	).From(
		"vendor",
	).Where(
		cond,
	)

	return v.selectVersions(ctx, q)
}

func (v VendorDataTrimmer) selectOrphanDataVersions(ctx context.Context) ([]int64, error) {
	q := sqlutil.Select(
		"vendor_data.version",
	).From(
		"vendor_data",
	).Literal(
		"LEFT JOIN vendor ON vendor.version = vendor_data.version",
	).Literal(
		"WHERE vendor.version IS NULL",
	).Literal(
		"GROUP BY vendor_data.version",
	)

	return v.selectVersions(ctx, q)
}

func (v VendorDataTrimmer) selectVersions(ctx context.Context, q *sqlutil.SelectStmt) ([]int64, error) {
	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := v.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "cannot query data")
	}

	defer rows.Close()

	var versions []int64
	for rows.Next() {
		var version int64
		err = rows.Scan(&version)
		if err != nil {
			return nil, errors.Wrap(err, "cannot scan data")
		}

		versions = append(versions, version)
	}

	return versions, nil
}

func latestVendorVersion() *sqlutil.SelectStmt {
	return sqlutil.Select(
		"MAX(version) AS version",
		"provider",
	).From(
		"vendor",
	).Where(
		sqlutil.Cond().Equal("ready", true),
	).Literal(
		"GROUP BY provider",
	)
}
