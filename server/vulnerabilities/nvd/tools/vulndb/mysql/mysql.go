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

// Package mysql provides a connector to vulndb via MySQL.
package mysql

import (
	"database/sql"
	"net/url"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/debug"
)

// OpenRead opens a connection to MySQL for reading.
//
// See https://github.com/go-sql-driver/mysql#parameters for dsn details.
func OpenRead(dsn string) (*sql.DB, error) {
	return openRead(dsn)
}

// OpenWrite opens a connection to MySQL for writing.
//
// See https://github.com/go-sql-driver/mysql#parameters for dsn details.
func OpenWrite(dsn string) (*sql.DB, error) {
	return openWrite(dsn)
}

func openDB(dsn string) (*sql.DB, error) {
	dsn = SetParams(dsn, url.Values{
		"parseTime": []string{"true"},
		"charset":   []string{"utf8mb4"},
	})
	if debug.V(1) {
		flog.Infof("connecting to %q", dsn)
	}
	db, err := sql.Open(mysqlDriver, dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(10)
	return db, nil
}

// SetParams takes a DSN as input and sets the desired parameters
// for the MySQL connection, e.g. parseTime=true.
//
// See https://github.com/go-sql-driver/mysql#parameters for details.
func SetParams(dsn string, params url.Values) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}

	q := u.Query()
	for k, v := range params {
		q[k] = v
	}

	u.RawQuery = q.Encode()
	return u.String()
}
