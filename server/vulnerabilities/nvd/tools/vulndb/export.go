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
	"github.com/facebookincubator/nvdtools/vulndb/debug"
	"github.com/facebookincubator/nvdtools/vulndb/sqlutil"
)

// DataExporter is a helper for exporting vulnerability records from the db.
type DataExporter struct {
	DB              *sql.DB
	FilterProviders []string
	FilterCVEs      []string
}

func (exp DataExporter) selectLatestVersion() *sqlutil.SelectStmt {
	q := sqlutil.Select(
		"latest.version",
	).From().
		SelectGroup("latest", latestVendorVersion())

	if len(exp.FilterProviders) > 0 {
		q = q.Where(
			sqlutil.Cond().In("provider", exp.FilterProviders),
		)
	}

	return q
}

// fields must be vendor.field_name or vendor_data.field_name
func (exp DataExporter) selectVendorData(fields ...string) *sqlutil.SelectStmt {
	q := sqlutil.Select(
		fields...,
	).From(
		"vendor_data",
	).Literal(
		"LEFT JOIN vendor ON vendor.version = vendor_data.version",
	).Literal(
		"LEFT JOIN custom_data ON custom_data.cve_id = vendor_data.cve_id",
	)

	cond := sqlutil.Cond().
		InSelect("vendor.version", exp.selectLatestVersion()).
		And().
		IsNull("custom_data.cve_id")

	if len(exp.FilterCVEs) > 0 {
		cond = cond.And().In("vendor_data.cve_id", exp.FilterCVEs)
	}

	q = q.Where(cond)
	return q
}

func (exp DataExporter) selectOverrides(fields ...string) *sqlutil.SelectStmt {
	q := sqlutil.Select(
		fields...,
	).From(
		"custom_data",
	)

	var cond *sqlutil.QueryConditionSet

	if len(exp.FilterProviders) > 0 {
		cond = sqlutil.Cond().In("provider", exp.FilterProviders)
	}

	if len(exp.FilterCVEs) > 0 {
		if cond == nil {
			cond = sqlutil.Cond()
		} else {
			cond = cond.And()
		}
		cond = cond.In("cve_id", exp.FilterCVEs)
	}

	if cond != nil {
		q = q.Where(cond)
	}

	return q
}

// CSV exports data to w.
func (exp DataExporter) CSV(ctx context.Context, w io.Writer, header bool) error {
	q := sqlutil.Select(
		"d.owner",
		"d.provider",
		"d.cve_id",
		"d.published",
		"d.modified",
		"d.base_score",
		"d.summary",
	).From().SelectGroup(
		"d",
		exp.selectVendorData(
			"vendor.owner",
			"vendor.provider",
			"vendor_data.cve_id",
			"vendor_data.published",
			"vendor_data.modified",
			"vendor_data.base_score",
			"vendor_data.summary",
		).Literal("UNION ALL").
			Select(exp.selectOverrides(
				"custom_data.owner",
				"custom_data.provider",
				"custom_data.cve_id",
				"custom_data.published",
				"custom_data.modified",
				"custom_data.base_score",
				"custom_data.summary",
			)),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := exp.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot query export data")
	}

	defer rows.Close()

	record := struct {
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
			return errors.Wrap(err, "cannot scan export data")
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

	return errors.Wrap(rows.Err(), "unable to read all rows from result set")
}

// JSON exports NVD CVE JSON to w.
func (exp DataExporter) JSON(ctx context.Context, w io.Writer, indent string) error {
	q := sqlutil.Select(
		"d.cve_id",
		"d.cve_json",
	).From().SelectGroup(
		"d",
		exp.selectVendorData(
			"vendor_data.cve_id AS cve_id",
			"vendor_data.cve_json AS cve_json",
		).
			Literal("UNION ALL").
			Select(exp.selectOverrides(
				"custom_data.cve_id",
				"custom_data.cve_json",
			)),
	)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := exp.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot query export data")
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
			return errors.Wrap(err, "cannot scan export data")
		}

		err = f.Add(v.CVE, v.JSON)
		if err != nil {
			return err
		}
	}

	if rows.Err() != nil {
		return errors.Wrap(rows.Err(), "unable to read all rows from result set")
	}

	if indent == "" {
		return f.EncodeJSON(w)
	}

	const prefix = ""
	return f.EncodeIndentedJSON(w, prefix, indent)
}
