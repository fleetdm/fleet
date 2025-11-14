// Package softwaredb provides SQLite database loading for realistic software data
// used in osquery-perf load testing.
package softwaredb

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DarwinSoftware represents macOS/iOS software
type DarwinSoftware struct {
	Name             string
	Version          string
	Source           string // apps, homebrew_packages, firefox_addons, chrome_extensions, python_packages, vscode_extensions, safari_extensions
	BundleIdentifier string // optional - used by apps
	Vendor           string // optional - used by apps, vscode_extensions
	ExtensionID      string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
	ExtensionFor     string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
}

// WindowsSoftware represents Windows software
type WindowsSoftware struct {
	Name         string
	Version      string
	Source       string // firefox_addons, chrome_extensions, programs, vscode_extensions, ie_extensions, python_packages, deb_packages
	Vendor       string // optional - used by programs, vscode_extensions
	UpgradeCode  string // optional - used by programs
	ExtensionID  string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
	ExtensionFor string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
}

// UbuntuSoftware represents Ubuntu/Linux software
type UbuntuSoftware struct {
	Name         string
	Version      string
	Source       string // firefox_addons, chrome_extensions, python_packages, deb_packages, vscode_extensions, npm_packages, rpm_packages
	Vendor       string // optional - used by rpm_packages, vscode_extensions
	Arch         string // optional - used by rpm_packages
	Release      string // optional - used by rpm_packages
	ExtensionID  string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
	ExtensionFor string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
}

// DB holds the loaded software data for each platform
type DB struct {
	Darwin  []DarwinSoftware
	Windows []WindowsSoftware
	Ubuntu  []UbuntuSoftware
}

// DarwinToMaps converts Darwin software to osquery result format
func (db *DB) DarwinToMaps() []map[string]string {
	results := make([]map[string]string, 0, len(db.Darwin))
	for _, s := range db.Darwin {
		m := map[string]string{
			"name":    s.Name,
			"source":  s.Source,
			"version": s.Version,
		}
		// Add optional fields if present
		if s.BundleIdentifier != "" {
			m["bundle_identifier"] = s.BundleIdentifier
		}
		if s.Vendor != "" {
			m["vendor"] = s.Vendor
		}
		if s.ExtensionID != "" {
			m["extension_id"] = s.ExtensionID
		}
		if s.ExtensionFor != "" {
			m["browser"] = s.ExtensionFor
		}
		results = append(results, m)
	}
	return results
}

// WindowsToMaps converts Windows software to osquery result format
func (db *DB) WindowsToMaps() []map[string]string {
	results := make([]map[string]string, 0, len(db.Windows))
	for _, s := range db.Windows {
		m := map[string]string{
			"name":    s.Name,
			"source":  s.Source,
			"version": s.Version,
		}
		// Add optional fields if present
		if s.Vendor != "" {
			m["vendor"] = s.Vendor
		}
		if s.UpgradeCode != "" {
			m["upgrade_code"] = s.UpgradeCode
		}
		if s.ExtensionID != "" {
			m["extension_id"] = s.ExtensionID
		}
		if s.ExtensionFor != "" {
			m["browser"] = s.ExtensionFor
		}
		results = append(results, m)
	}
	return results
}

// UbuntuToMaps converts Ubuntu software to osquery result format
func (db *DB) UbuntuToMaps() []map[string]string {
	results := make([]map[string]string, 0, len(db.Ubuntu))
	for _, s := range db.Ubuntu {
		m := map[string]string{
			"name":    s.Name,
			"source":  s.Source,
			"version": s.Version,
		}
		// Add optional fields if present
		if s.Vendor != "" {
			m["vendor"] = s.Vendor
		}
		if s.Arch != "" {
			m["arch"] = s.Arch
		}
		if s.Release != "" {
			m["release"] = s.Release
		}
		if s.ExtensionID != "" {
			m["extension_id"] = s.ExtensionID
		}
		if s.ExtensionFor != "" {
			m["browser"] = s.ExtensionFor
		}
		results = append(results, m)
	}
	return results
}

// Platform-specific counts based on production averages (±20%):
// - Ubuntu: 2,460 ± 20% = 1,968 to 2,952
// - Darwin: 453 ± 20% = 362 to 544
// - Windows: 251 ± 20% = 201 to 301
var platformCounts = map[string]struct {
	sources []string
	min     int
	max     int
}{
	"darwin": {
		sources: []string{"apps", "homebrew_packages", "firefox_addons", "chrome_extensions", "python_packages", "vscode_extensions", "safari_extensions"},
		min:     362,
		max:     544,
	},
	"windows": {
		sources: []string{"firefox_addons", "chrome_extensions", "programs", "vscode_extensions", "ie_extensions", "python_packages", "deb_packages"},
		min:     201,
		max:     301,
	},
	"ubuntu": {
		sources: []string{"firefox_addons", "chrome_extensions", "python_packages", "deb_packages", "vscode_extensions", "npm_packages", "rpm_packages"},
		min:     1968,
		max:     2952,
	},
}

// LoadFromDatabase loads software from the SQLite database with platform-specific counts.
// If the database doesn't exist, it attempts to auto-generate it from software.sql.
func LoadFromDatabase(dbPath string) (*DB, error) {
	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Database doesn't exist, try to generate it from software.sql
		log.Printf("Database not found at %s, attempting to generate from software.sql...", dbPath)

		// Look for software.sql in the same directory
		sqlPath := strings.TrimSuffix(dbPath, ".db") + ".sql"
		if _, err := os.Stat(sqlPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("software database not found: %s\nAlso could not find SQL file: %s\n\nPlease ensure software.sql exists, or create the database manually:\n  cd cmd/osquery-perf/software-library\n  sqlite3 software.db < software.sql", dbPath, sqlPath)
		}

		if err := generateDatabaseFromSQL(dbPath, sqlPath); err != nil {
			return nil, err
		}
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening software database: %w", err)
	}
	defer db.Close()

	// Verify table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='software'").Scan(&count)
	if err != nil || count == 0 {
		return nil, errors.New("database exists but 'software' table not found\n\nPlease initialize the database:\n  cd cmd/osquery-perf/software-library\n  sqlite3 software.db < software.sql")
	}

	// Load software for each platform
	softwareDB := &DB{}

	// Load Darwin software
	darwinConfig := platformCounts["darwin"]
	darwinCount := darwinConfig.min + rand.IntN(darwinConfig.max-darwinConfig.min+1) // nolint:gosec,G404
	darwinSoftware, err := loadDarwinSoftware(db, darwinConfig.sources, darwinCount)
	if err != nil {
		return nil, err
	}
	softwareDB.Darwin = darwinSoftware
	log.Printf("Loaded %d darwin software items from database", len(darwinSoftware))

	// Load Windows software
	windowsConfig := platformCounts["windows"]
	windowsCount := windowsConfig.min + rand.IntN(windowsConfig.max-windowsConfig.min+1) // nolint:gosec,G404
	windowsSoftware, err := loadWindowsSoftware(db, windowsConfig.sources, windowsCount)
	if err != nil {
		return nil, err
	}
	softwareDB.Windows = windowsSoftware
	log.Printf("Loaded %d windows software items from database", len(windowsSoftware))

	// Load Ubuntu software
	ubuntuConfig := platformCounts["ubuntu"]
	ubuntuCount := ubuntuConfig.min + rand.IntN(ubuntuConfig.max-ubuntuConfig.min+1) // nolint:gosec,G404
	ubuntuSoftware, err := loadUbuntuSoftware(db, ubuntuConfig.sources, ubuntuCount)
	if err != nil {
		return nil, err
	}
	softwareDB.Ubuntu = ubuntuSoftware
	log.Printf("Loaded %d ubuntu software items from database", len(ubuntuSoftware))

	return softwareDB, nil
}

// generateDatabaseFromSQL creates a SQLite database from a SQL file
func generateDatabaseFromSQL(dbPath, sqlPath string) error {
	// Read SQL file
	sqlContent, err := os.ReadFile(sqlPath)
	if err != nil {
		return fmt.Errorf("reading SQL file %s: %w", sqlPath, err)
	}

	log.Printf("Found %s (%d bytes), creating database...", sqlPath, len(sqlContent))

	// Create database and execute SQL
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}
	defer db.Close()

	// Execute the SQL file
	if _, err := db.Exec(string(sqlContent)); err != nil {
		os.Remove(dbPath) // Clean up partial database
		return fmt.Errorf("executing SQL file: %w", err)
	}

	log.Printf("✅ Successfully created database from %s", sqlPath)
	return nil
}

// loadDarwinSoftware loads macOS/iOS software from the database
func loadDarwinSoftware(db *sql.DB, sources []string, count int) ([]DarwinSoftware, error) {
	sourceList := "'" + strings.Join(sources, "', '") + "'"
	// nolint:gosec // sources are hardcoded, not user input
	query := fmt.Sprintf(`
		SELECT name, version, source, bundle_identifier, vendor, extension_id, extension_for
		FROM software
		WHERE source IN (%s)
		ORDER BY RANDOM()
		LIMIT ?
	`, sourceList)

	rows, err := db.Query(query, count)
	if err != nil {
		return nil, fmt.Errorf("querying darwin software: %w", err)
	}
	defer rows.Close()

	software := make([]DarwinSoftware, 0, count)
	for rows.Next() {
		var sw DarwinSoftware
		var bundleID, vendor, extensionID, extensionFor sql.NullString

		err := rows.Scan(&sw.Name, &sw.Version, &sw.Source, &bundleID, &vendor, &extensionID, &extensionFor)
		if err != nil {
			return nil, fmt.Errorf("scanning darwin software row: %w", err)
		}

		if bundleID.Valid {
			sw.BundleIdentifier = bundleID.String
		}
		if vendor.Valid {
			sw.Vendor = vendor.String
		}
		if extensionID.Valid {
			sw.ExtensionID = extensionID.String
		}
		if extensionFor.Valid {
			sw.ExtensionFor = extensionFor.String
		}

		software = append(software, sw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating darwin software rows: %w", err)
	}

	return software, nil
}

// loadWindowsSoftware loads Windows software from the database
func loadWindowsSoftware(db *sql.DB, sources []string, count int) ([]WindowsSoftware, error) {
	sourceList := "'" + strings.Join(sources, "', '") + "'"
	// nolint:gosec // sources are hardcoded, not user input
	query := fmt.Sprintf(`
		SELECT name, version, source, vendor, upgrade_code, extension_id, extension_for
		FROM software
		WHERE source IN (%s)
		ORDER BY RANDOM()
		LIMIT ?
	`, sourceList)

	rows, err := db.Query(query, count)
	if err != nil {
		return nil, fmt.Errorf("querying windows software: %w", err)
	}
	defer rows.Close()

	software := make([]WindowsSoftware, 0, count)
	for rows.Next() {
		var sw WindowsSoftware
		var vendor, upgradeCode, extensionID, extensionFor sql.NullString

		err := rows.Scan(&sw.Name, &sw.Version, &sw.Source, &vendor, &upgradeCode, &extensionID, &extensionFor)
		if err != nil {
			return nil, fmt.Errorf("scanning windows software row: %w", err)
		}

		if vendor.Valid {
			sw.Vendor = vendor.String
		}
		if upgradeCode.Valid {
			sw.UpgradeCode = upgradeCode.String
		}
		if extensionID.Valid {
			sw.ExtensionID = extensionID.String
		}
		if extensionFor.Valid {
			sw.ExtensionFor = extensionFor.String
		}

		software = append(software, sw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating windows software rows: %w", err)
	}

	return software, nil
}

// loadUbuntuSoftware loads Ubuntu/Linux software from the database
func loadUbuntuSoftware(db *sql.DB, sources []string, count int) ([]UbuntuSoftware, error) {
	sourceList := "'" + strings.Join(sources, "', '") + "'"
	// nolint:gosec // sources are hardcoded, not user input
	query := fmt.Sprintf(`
		SELECT name, version, source, vendor, arch, release, extension_id, extension_for
		FROM software
		WHERE source IN (%s)
		ORDER BY RANDOM()
		LIMIT ?
	`, sourceList)

	rows, err := db.Query(query, count)
	if err != nil {
		return nil, fmt.Errorf("querying ubuntu software: %w", err)
	}
	defer rows.Close()

	software := make([]UbuntuSoftware, 0, count)
	for rows.Next() {
		var sw UbuntuSoftware
		var vendor, arch, release, extensionID, extensionFor sql.NullString

		err := rows.Scan(&sw.Name, &sw.Version, &sw.Source, &vendor, &arch, &release, &extensionID, &extensionFor)
		if err != nil {
			return nil, fmt.Errorf("scanning ubuntu software row: %w", err)
		}

		if vendor.Valid {
			sw.Vendor = vendor.String
		}
		if arch.Valid {
			sw.Arch = arch.String
		}
		if release.Valid {
			sw.Release = release.String
		}
		if extensionID.Valid {
			sw.ExtensionID = extensionID.String
		}
		if extensionFor.Valid {
			sw.ExtensionFor = extensionFor.String
		}

		software = append(software, sw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating ubuntu software rows: %w", err)
	}

	return software, nil
}
