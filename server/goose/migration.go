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
	"unicode"
	"unicode/utf8"
)

type MigrationRecord struct {
	VersionId int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

type Migration struct {
	Version  int64
	Title    string              // sentence-case description, logged when a go migration runs
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
		log.Printf("[%s] %s\n", migrationDate(m.Source), m.Title)

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
				log.Fatalf("FAIL %s: %v", filepath.Base(m.Source), err)
				return err
			}
		}

		if err = c.FinalizeMigration(tx, direction, m.Version); err != nil {
			log.Fatalf("error finalizing migration %s, quitting. (%v)", filepath.Base(m.Source), err)
		}
	}

	return nil
}

func migrationDate(source string) string {
	// Only the date is parsed because Fleet developers bump the seconds when re-arranging new migrations
	// e.g.: 2022/10/10 15:43:46 fail to parse time: parsing time "20201021104586": second out of range
	datePart, _, _ := strings.Cut(filepath.Base(source), "_")
	if len(datePart) < 8 {
		log.Fatalf("fail to parse time from migration %s", filepath.Base(source))
	}
	mt, err := time.Parse("20060102", datePart[:8])
	if err != nil {
		log.Fatalf("fail to parse time: %s", err)
	}
	return mt.Format("2006-01-02")
}

var capitalizedWord = regexp.MustCompile(`^\p{Lu}\p{Ll}`)

// ValidateTitle checks that title is a migration title in sentence case, e.g. "Add an index on hosts platform".
// Acronyms and proper nouns may be capitalized, but a title with every word capitalized is rejected as title case.
func ValidateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("migration title is required")
	}
	if strings.TrimSpace(title) != title {
		return errors.New("migration title must not start or end with whitespace")
	}
	first, _ := utf8.DecodeRuneInString(title)
	if !unicode.IsUpper(first) {
		return fmt.Errorf("migration title must be in sentence case, starting with an uppercase letter: %q", title)
	}
	if strings.HasSuffix(title, ".") {
		return fmt.Errorf("migration title must not end with a period: %q", title)
	}
	var words, capitalized int
	for _, w := range strings.Fields(title)[1:] {
		first, _ := utf8.DecodeRuneInString(w)
		if !unicode.IsLetter(first) {
			continue
		}
		words++
		if capitalizedWord.MatchString(w) {
			capitalized++
		}
	}
	if words >= 2 && capitalized == words {
		return fmt.Errorf("migration title must be in sentence case, not title case: %q", title)
	}
	return nil
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

func CreateMigration(name, title, migrationType, dir string, t time.Time) ([]string, error) {
	if migrationType != "go" && migrationType != "sql" {
		return nil, errors.New("migration type must be 'go' or 'sql'")
	}
	if err := ValidateTitle(title); err != nil {
		return nil, err
	}

	timestamp := t.Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.%s", timestamp, name, migrationType)

	fpath := filepath.Join(dir, filename)
	tmpl := sqlMigrationTemplate
	if migrationType == "go" {
		tmpl = goSqlMigrationTemplate
	}

	var paths []string

	data := migrationTemplateData{Timestamp: timestamp, Title: title}
	migrationPath, err := writeTemplateToFile(fpath, tmpl, data)
	if err != nil {
		return nil, err
	}
	paths = append(paths, migrationPath)

	if migrationType == "go" {
		fpath := strings.Replace(filepath.Join(dir, filename), ".go", "_test.go", 1)
		migrationTestPath, err := writeTemplateToFile(fpath, goSqlMigrationTestTemplate, data)
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

type migrationTemplateData struct {
	Timestamp string
	Title     string
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
    MigrationClient.AddMigration({{printf "%q" .Title}}, Up_{{.Timestamp}}, Down_{{.Timestamp}})
}

func Up_{{.Timestamp}}(tx *sql.Tx) error {
    return nil
}

func Down_{{.Timestamp}}(tx *sql.Tx) error {
    return nil
}
`))

var goSqlMigrationTestTemplate = template.Must(template.New("goose.go-migration").Parse(`
package tables

import "testing"

func TestUp_{{.Timestamp}}(t *testing.T) {
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
