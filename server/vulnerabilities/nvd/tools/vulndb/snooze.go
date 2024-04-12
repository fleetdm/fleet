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
	"time"

	"github.com/pkg/errors"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/debug"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/sqlutil"
)

// SnoozeRecord represents a database record of the `snooze` table.
type SnoozeRecord struct {
	Owner     string           `sql:"owner"`
	Collector string           `sql:"collector"`
	Provider  string           `sql:"provider"`
	CVE       string           `sql:"cve_id"`
	Deadline  sqlutil.NullTime `sql:"deadline"`
	Metadata  []byte           `sql:"metadata"`
}

// SnoozeCreator is a helper for creating snoozes.
type SnoozeCreator struct {
	DB        *sql.DB
	Owner     string
	Collector string
	Provider  string
	Deadline  time.Time
	Metadata  []byte
}

// Create creates a snooze.
func (s SnoozeCreator) Create(ctx context.Context, cve ...string) error {
	records := make([]SnoozeRecord, len(cve))
	for i := 0; i < len(records); i++ {
		records[i] = SnoozeRecord{
			Owner:     s.Owner,
			Collector: s.Collector,
			Provider:  s.Provider,
			CVE:       cve[i],
			Deadline: sqlutil.NullTime{
				Valid: !s.Deadline.IsZero(),
				Time:  s.Deadline,
			},
			Metadata: s.Metadata,
		}
	}
	r := sqlutil.NewRecords(records)
	q := sqlutil.Replace().
		Into("snooze").
		Fields(r.Fields()...).
		Values(r...)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	_, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot create snooze")
	}

	return nil
}

// SnoozeGetter gets data from the database.
type SnoozeGetter struct {
	DB         *sql.DB
	Collector  string
	Provider   string
	FilterCVEs []string
}

// CSV writes snooze records to w.
func (s SnoozeGetter) CSV(ctx context.Context, w io.Writer, header bool) error {
	r := sqlutil.NewRecordType(SnoozeRecord{})
	q := sqlutil.Select(
		r.Fields()...,
	).From(
		"snooze",
	)

	var cond *sqlutil.QueryConditionSet

	if s.Collector != "" {
		cond = sqlutil.Cond().Equal("collector", s.Collector)
	}

	if s.Provider != "" {
		if cond == nil {
			cond = sqlutil.Cond()
		} else {
			cond = cond.And()
		}
		cond = cond.Equal("provider", s.Provider)
	}

	if len(s.FilterCVEs) > 0 {
		if cond == nil {
			cond = sqlutil.Cond()
		} else {
			cond = cond.And()
		}
		cond = cond.In("cve_id", s.FilterCVEs)
	}

	if cond != nil {
		q = q.Where(cond)
	}

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot query snooze")
	}

	defer rows.Close()

	cw := csv.NewWriter(w)
	defer cw.Flush()

	if header {
		cw.Write(r.Fields())
	}

	for rows.Next() {
		var sr SnoozeRecord
		err = rows.Scan(sqlutil.NewRecordType(&sr).Values()...)
		if err != nil {
			return errors.Wrap(err, "cannot scan snooze data")
		}

		var deadline string
		if sr.Deadline.Valid {
			deadline = sr.Deadline.Time.Format(TimeLayout)
		}

		cw.Write([]string{
			sr.Owner,
			sr.Collector,
			sr.Provider,
			sr.CVE,
			deadline,
			string(sr.Metadata),
		})
	}

	return nil
}

// SnoozeDeleter deletes snoozes from the database.
type SnoozeDeleter struct {
	DB         *sql.DB
	Collector  string
	Provider   string
	FilterCVEs []string
}

// Delete deletes snooze data from the database.
func (s SnoozeDeleter) Delete(ctx context.Context) error {
	cond := sqlutil.Cond().
		Equal("collector", s.Provider).
		And().
		Equal("provider", s.Provider)

	if len(s.FilterCVEs) > 0 {
		cond = cond.And().In("cve_id", s.FilterCVEs)
	}

	q := sqlutil.Delete().From("snooze").Where(cond)

	query, args := q.String(), q.QueryArgs()

	if debug.V(1) {
		flog.Infof("running: %q / %#v", query, args)
	}

	_, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "cannot delete snooze data")
	}

	return nil
}
