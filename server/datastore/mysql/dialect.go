package mysql

import (
	"database/sql"
	"strings"
	"sync"
)

type DBDialect int

const (
	DialectUnknown DBDialect = iota
	DialectMySQL
	DialectMariaDB
)

func (d DBDialect) String() string {
	switch d {
	case DialectMySQL:
		return "MySQL"
	case DialectMariaDB:
		return "MariaDB"
	default:
		return "Unknown"
	}
}

type dialectDetector struct {
	mu      sync.RWMutex
	dialect DBDialect
	version string
}

var globalDialect = &dialectDetector{
	dialect: DialectUnknown,
}

// DetectDialect detects whether the database is MySQL or MariaDB
func DetectDialect(db *sql.DB) (DBDialect, string, error) {
	globalDialect.mu.RLock()
	if globalDialect.dialect != DialectUnknown {
		defer globalDialect.mu.RUnlock()
		return globalDialect.dialect, globalDialect.version, nil
	}
	globalDialect.mu.RUnlock()

	globalDialect.mu.Lock()
	defer globalDialect.mu.Unlock()

	if globalDialect.dialect != DialectUnknown {
		return globalDialect.dialect, globalDialect.version, nil
	}

	var version string
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return DialectUnknown, "", err
	}

	// MariaDB version strings contain "MariaDB"
	// Example: "11.6.2-MariaDB-1:11.6.2+maria~ubu2404"
	// MySQL version strings look like: "8.0.36"
	dialect := DialectMySQL
	if strings.Contains(version, "MariaDB") {
		dialect = DialectMariaDB
	}

	globalDialect.dialect = dialect
	globalDialect.version = version
	return dialect, version, nil
}

func IsMariaDB(db *sql.DB) bool {
	dialect, _, _ := DetectDialect(db)
	return dialect == DialectMariaDB
}

func IsMySQL(db *sql.DB) bool {
	dialect, _, _ := DetectDialect(db)
	return dialect == DialectMySQL
}

type SQLDialect struct {
	db *sql.DB
}

func NewSQLDialect(db *sql.DB) *SQLDialect {
	return &SQLDialect{db: db}
}

// JSONBoolTrue returns the SQL for a true boolean value in JSON context
// MySQL: CAST(TRUE AS JSON)
// MariaDB: true
func (s *SQLDialect) JSONBoolTrue() string {
	if IsMariaDB(s.db) {
		return "true"
	}
	return "CAST(TRUE AS JSON)"
}

// JSONBoolFalse returns the SQL for a false boolean value in JSON context
// MySQL: CAST(FALSE AS JSON)
// MariaDB: false
func (s *SQLDialect) JSONBoolFalse() string {
	if IsMariaDB(s.db) {
		return "false"
	}
	return "CAST(FALSE AS JSON)"
}

// GeneratedColumnStoredSuffix returns the suffix for generated stored columns
// MySQL: STORED or STORED NOT NULL
// MariaDB: STORED only (doesn't support NOT NULL after STORED)
func (s *SQLDialect) GeneratedColumnStoredSuffix(nullable bool) string {
	if IsMariaDB(s.db) {
		return "STORED"
	}
	if nullable {
		return "STORED"
	}
	return "STORED NOT NULL"
}

// JSONEquals returns SQL comparing two JSON expressions for semantic equality,
// ignoring key order and insignificant whitespace.
//
// MySQL stores JSON in a native binary type, so comparing the column against
// CAST(expr AS JSON) compares parsed values. MariaDB has no JSON type -- JSON is
// an alias for LONGTEXT -- so CAST(... AS JSON) is a syntax error and a bare
// comparison would be a raw string compare. JSON_NORMALIZE puts both sides into
// a canonical form so the comparison stays semantic.
//
// Note that on MariaDB this wraps the column, so it cannot use an index on it.
func (s *SQLDialect) JSONEquals(left, right string) string {
	if IsMariaDB(s.db) {
		return "JSON_NORMALIZE(" + left + ") = JSON_NORMALIZE(" + right + ")"
	}
	return left + " = CAST(" + right + " AS JSON)"
}

// JSONNotEquals is the negation of JSONEquals.
func (s *SQLDialect) JSONNotEquals(left, right string) string {
	if IsMariaDB(s.db) {
		return "JSON_NORMALIZE(" + left + ") != JSON_NORMALIZE(" + right + ")"
	}
	return left + " != CAST(" + right + " AS JSON)"
}
