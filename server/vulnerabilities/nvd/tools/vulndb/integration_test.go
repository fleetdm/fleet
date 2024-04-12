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
	"bytes"
	"context"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/debug"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/vulndb/mysql"
)

func init() {
	flag.Var(&debug.Level, "debug", "set debug level")
}

var mysqlTestDSN = flag.String("mysql_test_dsn", os.Getenv("MYSQL_TEST_DSN"), "set mysql test dsn")

func TestIntegration(t *testing.T) {
	dsn := *mysqlTestDSN
	if dsn == "" {
		t.Skip("set $MYSQL_TEST_DSN to enable integration tests")
	}

	f, err := createSampleCVE()
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	db, err := mysql.OpenWrite(dsn)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err = InitSchemaSQL(ctx, db); err != nil {
		t.Fatal(err)
	}

	t.Run("vendor/import", func(t *testing.T) {
		imp := &VendorDataImporter{
			DB:       db,
			Owner:    "test",
			Provider: "test",
		}

		_, err := imp.ImportFiles(ctx, f.Name())
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("vendor/export", func(t *testing.T) {
		exp := VendorDataExporter{
			DB:         db,
			Provider:   "test",
			FilterCVEs: []string{"CVE-0000-0000"},
		}

		t.Run("csv", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.CSV(ctx, &b, false)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "CVE-0000-0000") {
				t.Fatal("missing test CVE")
			}
		})

		t.Run("json", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.JSON(ctx, &b, "\t")
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "CVE-0000-0000") {
				t.Fatal("missing test CVE", b.String())
			}
		})
	})

	t.Run("custom/import", func(t *testing.T) {
		imp := CustomDataImporter{
			DB:       db,
			Owner:    "test",
			Provider: "test",
		}

		err := imp.ImportFile(ctx, f.Name())
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("custom/export", func(t *testing.T) {
		exp := CustomDataExporter{
			DB:         db,
			Provider:   "test",
			FilterCVEs: []string{"CVE-0000-0001"},
		}

		t.Run("csv", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.CSV(ctx, &b, false)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "CVE-0000-0001") {
				t.Fatal("missing test CVE")
			}
		})

		t.Run("json", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.JSON(ctx, &b, "\t")
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "CVE-0000-0001") {
				t.Fatal("missing test CVE", b.String())
			}
		})
	})

	t.Run("snooze/create", func(t *testing.T) {
		s := SnoozeCreator{
			DB:       db,
			Owner:    "test",
			Provider: "test",
			Deadline: time.Now().Add(24 * time.Hour),
			Metadata: []byte("hello world"),
		}

		err := s.Create(ctx, "CVE-0000-0000")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("snooze/get", func(t *testing.T) {
		exp := SnoozeGetter{
			DB:         db,
			Provider:   "test",
			FilterCVEs: []string{"CVE-0000-0000"},
		}

		t.Run("csv", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.CSV(ctx, &b, false)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "CVE-0000-0000") {
				t.Fatal("missing test CVE")
			}
		})
	})

	t.Run("export", func(t *testing.T) {
		exp := DataExporter{
			DB:              db,
			FilterProviders: []string{"test"},
			FilterCVEs:      []string{"CVE-0000-0001"},
		}

		t.Run("csv", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.CSV(ctx, &b, false)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "CVE-0000-0001") {
				t.Fatal("missing test CVE")
			}
		})

		t.Run("json", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.JSON(ctx, &b, "\t")
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "CVE-0000-0001") {
				t.Fatal("missing test CVE", b.String())
			}
		})
	})

	t.Run("summary", func(t *testing.T) {
		exp := SummaryExporter{
			DB: db,
		}

		t.Run("csv", func(t *testing.T) {
			var b bytes.Buffer
			err := exp.CSV(ctx, &b, false)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(b.String(), "custom_data") {
				t.Fatal("missing data type")
			}
		})
	})

	t.Run("vendor/trim", func(t *testing.T) {
		del := VendorDataTrimmer{
			DB:                  db,
			FilterProviders:     []string{"test"},
			DeleteLatestVersion: true,
		}

		err := del.Trim(ctx)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("custom/delete", func(t *testing.T) {
		del := CustomDataDeleter{
			DB:         db,
			Provider:   "test",
			FilterCVEs: []string{"CVE-0000-0000", "CVE-0000-0001"},
		}

		err := del.Delete(ctx)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("snooze/delete", func(t *testing.T) {
		del := SnoozeDeleter{
			DB:         db,
			Provider:   "test",
			FilterCVEs: []string{"CVE-0000-0000"},
		}

		err := del.Delete(ctx)
		if err != nil {
			t.Fatal(err)
		}
	})
}
