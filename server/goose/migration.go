package goose

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type MigrationRecord struct {
	VersionId int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

type Migration struct {
	Version  int64
	Next     int64               // next version, or -1 if none
	Previous int64               // previous version, -1 if none
	Source   string              // path to .sql script
	UpFn     func(*sql.Tx) error // Up go migration function
	DownFn   func(*sql.Tx) error // Down go migration function
}

const (
	migrateUp   = true
	migrateDown = !migrateUp
)

func (m *Migration) String() string {
	return fmt.Sprint(m.Source)
}

func (c *Client) runMigration(db *sql.DB, m *Migration, direction bool) error {
	switch filepath.Ext(m.Source) {
	case ".sql":
		if err := c.runSQLMigration(db, m.Source, m.Version, direction); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}

	case ".go":
		name, date := parseNameAndDate(m.Source)
		log.Printf("[%s] %s\n", date, name)

		tx, err := db.Begin()
		if err != nil {
			log.Fatal("db.Begin: ", err)
		}

		fn := m.UpFn
		if !direction {
			fn = m.DownFn
		}
		if fn != nil {
			if err := fn(tx); err != nil {
				tx.Rollback() //nolint:errcheck
				log.Fatalf("FAIL %s (%v), quitting migration.", filepath.Base(m.Source), err)
				return err
			}
		}

		if err = c.FinalizeMigration(tx, direction, m.Version); err != nil {
			log.Fatalf("error finalizing migration %s, quitting. (%v)", filepath.Base(m.Source), err)
		}
	}

	return nil
}

var (
	upperReplace         = regexp.MustCompile("([a-z])([A-Z])")       // e.g. UpdateBuiltin -> Update Builtin
	allUpperWordsReplace = regexp.MustCompile("([A-Z]+)([A-Z][a-z])") // e.g. IDIn -> ID In
)

func parseNameAndDate(source string) (name string, date string) {
	parts := strings.SplitN(strings.TrimSuffix(filepath.Base(source), ".go"), "_", 2)
	// Stripping seconds [:8] because Fleet developers add seconds when re-arranging new migrations
	// e.g.: 2022/10/10 15:43:46 fail to parse time: parsing time "20201021104586": second out of range
	datePart := parts[0][:8]
	mt, err := time.Parse("20060102", datePart)
	if err != nil {
		log.Fatalf("fail to parse time: %s", err)
	}
	name = upperReplace.ReplaceAllString(parts[1], "$1 $2")     // add spaces in the filename
	name = allUpperWordsReplace.ReplaceAllString(name, "$1 $2") // add spaces in the filename
	date = mt.Format("2006-01-02")
	return
}

// look for migration scripts with names in the form:
//
//	XXX_descriptivename.ext
//
// where XXX specifies the version number
// and ext specifies the type of migration
func NumericComponent(name string) (int64, error) {
	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("not a recognized migration file type")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no separator found")
	}

	n, e := strconv.ParseInt(base[:idx], 10, 64)
	if e == nil && n <= 0 {
		return 0, errors.New("migration IDs must be greater than zero")
	}

	return n, e
}

func CreateMigration(name, migrationType, dir string, t time.Time) ([]string, error) {
	if migrationType != "go" && migrationType != "sql" {
		return nil, errors.New("migration type must be 'go' or 'sql'")
	}

	timestamp := t.Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.%s", timestamp, name, migrationType)

	fpath := filepath.Join(dir, filename)
	tmpl := sqlMigrationTemplate
	if migrationType == "go" {
		tmpl = goSqlMigrationTemplate
	}

	var paths []string

	migrationPath, err := writeTemplateToFile(fpath, tmpl, timestamp)
	if err != nil {
		return nil, err
	}
	paths = append(paths, migrationPath)

	if migrationType == "go" {
		fpath := strings.Replace(filepath.Join(dir, filename), ".go", "_test.go", 1)
		migrationTestPath, err := writeTemplateToFile(fpath, goSqlMigrationTestTemplate, timestamp)
		if err != nil {
			return nil, err
		}
		paths = append(paths, migrationTestPath)
	}

	return paths, nil
}

// Update the version table for the given migration,
// and finalize the transaction.
func (c *Client) FinalizeMigration(tx *sql.Tx, direction bool, v int64) error {
	// XXX: drop goose_db_version table on some minimum version number?
	stmt := c.Dialect.insertVersionSql(c.TableName)
	if _, err := tx.Exec(stmt, v, direction); err != nil {
		tx.Rollback() //nolint:errcheck
		return err
	}

	return tx.Commit()
}

var sqlMigrationTemplate = template.Must(template.New("goose.sql-migration").Parse(`
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

`))

var goSqlMigrationTemplate = template.Must(template.New("goose.go-migration").Parse(`
package tables

import (
    "database/sql"
)

func init() {
    MigrationClient.AddMigration(Up_{{.}}, Down_{{.}})
}

func Up_{{.}}(tx *sql.Tx) error {
    return nil
}

func Down_{{.}}(tx *sql.Tx) error {
    return nil
}
`))

var goSqlMigrationTestTemplate = template.Must(template.New("goose.go-migration").Parse(`
package tables

import "testing"

func TestUp_{{.}}(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
}`))
