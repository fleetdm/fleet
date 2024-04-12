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
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/debug"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/sqlutil"
)

// CustomDataRecord represents a db record of the `custom_data` table.
type CustomDataRecord struct {
	Owner     string    `sql:"owner"`
	Provider  string    `sql:"provider"`
	CVE       string    `sql:"cve_id"`
	Published time.Time `sql:"published"`
	Modified  time.Time `sql:"modified"`
	BaseScore float64   `sql:"base_score"`
	Summary   string    `sql:"summary"`
	JSON      []byte    `sql:"cve_json"`
}

// CustomDataImporter is a helper for importing custom data.
type CustomDataImporter struct {
	DB       *sql.DB
	Owner    string
	Provider string
}

// ImportJSON imports NVD CVE JSON 1.0 optionally gzipped.
func (o CustomDataImporter) ImportJSON(ctx context.Context, r io.Reader) error {
	records, err := o.recordsFromJSON(r)
	if err != nil {
		return err
	}

	return o.importData(ctx, records)
}

// ImportFile imports NVD CVE JSON 1.0 optionally gzipped from file.
func (o CustomDataImporter) ImportFile(ctx context.Context, name string) error {
	f, err := os.Open(name)
	if err != nil {
		return errors.Wrap(err, "cannot open custom data file")
	}
	defer f.Close()

	return o.ImportJSON(ctx, f)
}

func (o CustomDataImporter) recordsFromJSON(r io.Reader) ([]CustomDataRecord, error) {
	feed, err := parseNVDCVEJSON(r)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse custom data records")
	}

	records := make([]CustomDataRecord, len(feed.CVEItems))

	for i, item := range feed.CVEItems {
		cve := cveItem{item}
		records[i] = CustomDataRecord{
			Owner:     o.Owner,
			Provider:  o.Provider,
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

func (o CustomDataImporter) importData(ctx context.Context, data []CustomDataRecord) error {
	r := sqlutil.NewRecords(data)
	q := sqlutil.Replace().
		Into("custom_data").
		Fields(r.Fields()...).
		Values(r...)

	query, args := q.String(), q.QueryArgs()

	if debug.V(2) {
		flog.Infof("running: %q", query)
	}

	_, err := o.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot insert custom data records")
	}

	return nil
}

// CustomDataExporter is a helper for exporting custom data.
type CustomDataExporter struct {
	DB         *sql.DB
	Provider   string
	FilterCVEs []string
}

func (o CustomDataExporter) condition() *sqlutil.QueryConditionSet {
	cond := sqlutil.Cond().Equal("provider", o.Provider)
	if len(o.FilterCVEs) > 0 {
		cond = cond.And().In("cve_id", o.FilterCVEs)
	}
	return cond
}

// CSV writes custom data records to w.
func (o CustomDataExporter) CSV(ctx context.Context, w io.Writer, header bool) error {
	fields := []string{
		"owner",
		"provider",
		"cve_id",
		"published",
		"modified",
		"base_score",
		"summary",
	}
	q := sqlutil.Select(
		fields...,
	).From(
		"custom_data",
	).Where(
		o.condition(),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := o.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot query vendor data")
	}

	defer rows.Close()

	cw := csv.NewWriter(w)
	defer cw.Flush()

	if header {
		cw.Write(fields)
	}

	for rows.Next() {
		var v CustomDataRecord
		r := sqlutil.NewRecordType(&v).Subset(fields...)
		err = rows.Scan(r.Values()...)
		if err != nil {
			return errors.Wrap(err, "cannot scan custom data")
		}

		cw.Write([]string{
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
func (o CustomDataExporter) JSON(ctx context.Context, w io.Writer, indent string) error {
	q := sqlutil.Select(
		"cve_id",
		"cve_json",
	).From(
		"custom_data",
	).Where(
		o.condition(),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := o.DB.QueryContext(ctx, query, args...)
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

// CustomDataDeleter is a helper for deleting custom data.
type CustomDataDeleter struct {
	DB         *sql.DB
	Provider   string
	FilterCVEs []string
}

// Delete deletes custom data from the database.
func (o CustomDataDeleter) Delete(ctx context.Context) error {
	cond := sqlutil.Cond().Equal("provider", o.Provider)
	if len(o.FilterCVEs) > 0 {
		cond = cond.And().In("cve_id", o.FilterCVEs)
	}

	q := sqlutil.Delete().From("custom_data").Where(cond)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	_, err := o.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot delete custom data")
	}

	return nil
}
