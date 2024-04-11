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

// SummaryExporter is a helper for exporting database summary.
type SummaryExporter struct {
	DB *sql.DB
}

// SummaryRecord represents a record of the `summary` query
type SummaryRecord struct {
	DataType  string    `sql:"data_type"`
	Provider  string    `sql:"provider"`
	Version   string    `sql:"version"`
	CVEs      int64     `sql:"cves"`
	Published time.Time `sql:"published"`
	Modified  time.Time `sql:"modified"`
}

// SummaryRecords retrieves the summary from the DB and returns it as a list of records
func (exp SummaryExporter) SummaryRecords(ctx context.Context) ([]SummaryRecord, error) {
	var records []SummaryRecord

	query := summaryQuery

	if debug.V(1) {
		flog.Infof("running: %q", query)
	}

	rows, err := exp.DB.QueryContext(ctx, query)
	if err != nil {
		return records, errors.Wrap(err, "cannot query summary data")
	}

	defer rows.Close()

	for rows.Next() {
		v := SummaryRecord{}
		err = rows.Scan(sqlutil.NewRecordType(&v).Values()...)
		if err != nil {
			errors.Wrap(err, "cannot scan summary data")
		}
		records = append(records, v)
	}

	return records, nil
}

// CSV writes summary records to w.
func (exp SummaryExporter) CSV(ctx context.Context, w io.Writer, header bool) error {
	records, err := exp.SummaryRecords(ctx)
	if err != nil {
		return err
	}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	if header {
		fields := sqlutil.NewRecordType(SummaryRecord{}).Fields()
		cw.Write(fields)
	}

	for _, record := range records {
		cw.Write([]string{
			record.DataType,
			record.Provider,
			record.Version,
			strconv.FormatInt(record.CVEs, 10),
			record.Published.Format(TimeLayout),
			record.Modified.Format(TimeLayout),
		})
	}
	return nil
}

const summaryQuery = `
(
	SELECT
		'snooze'      AS data_type,
		provider      AS provider,
		'current'     AS version,
		COUNT(cve_id) AS cves,
        NULL          AS published,
		NULL          AS modified
	FROM
		snooze
	GROUP BY
		provider
)

UNION ALL

(
	SELECT
		'custom_data'  AS data_type,
		provider       AS provider,
		'current'      AS version,
		COUNT(cve_id)  AS cves,
		MAX(published) AS published,
		MAX(modified)  AS modified
	FROM
		custom_data
	GROUP BY
		provider
)

UNION ALL

(
	SELECT
		'vendor_data'              AS data_type,
		vendor.provider            AS provider,
		vendor.version             AS version,
		COUNT(vendor_data.cve_id)  AS cves,
		MAX(vendor_data.published) AS published,
		MAX(vendor_data.modified)  AS modified
	FROM
		vendor_data
	LEFT JOIN
		vendor
	ON
		vendor.version = vendor_data.version
	WHERE
		vendor.ready = true
	GROUP BY
		vendor.provider,
		vendor.version
)

ORDER BY
	version DESC
`
