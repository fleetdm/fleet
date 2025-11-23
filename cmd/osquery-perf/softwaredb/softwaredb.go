// Package softwaredb provides SQLite database loading for realistic software data used in osquery-perf load testing.
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

const (
	// SoftwareMutationProb is the probability of mutating software after initial load
	SoftwareMutationProb = 0.2
	// MaxSoftwareAdd is the maximum number of software items to add during mutation
	MaxSoftwareAdd = 20
	// MaxSoftwareRemove is the maximum number of software items to remove during mutation
	MaxSoftwareRemove = 20
	// MaxSoftwarePerPlatform is the maximum number of software items to load per platform
	MaxSoftwarePerPlatform = 50000
)

// String interning pools to reduce memory usage by reusing common strings
var (
	sourcePool = map[string]string{
		"apps":              "apps",
		"homebrew_packages": "homebrew_packages",
		"firefox_addons":    "firefox_addons",
		"chrome_extensions": "chrome_extensions",
		"python_packages":   "python_packages",
		"vscode_extensions": "vscode_extensions",
		"safari_extensions": "safari_extensions",
		"programs":          "programs",
		"ie_extensions":     "ie_extensions",
		"deb_packages":      "deb_packages",
		"npm_packages":      "npm_packages",
		"rpm_packages":      "rpm_packages",
		"android_apps":      "android_apps",
		"ios_apps":          "ios_apps",
		"ipados_apps":       "ipados_apps",
		"jetbrains_plugins": "jetbrains_plugins",
	}
	vendorPool = make(map[string]string) // populated during load
)

// internString returns an interned version of s from the vendor pool, reducing memory usage
func internString(s string) string {
	if s == "" {
		return ""
	}
	if interned, ok := vendorPool[s]; ok {
		return interned
	}
	vendorPool[s] = s
	return s
}

// internSource returns an interned source string
func internSource(s string) string {
	if interned, ok := sourcePool[s]; ok {
		return interned
	}
	return s
}

// ptrString returns a pointer to s, or nil if s is empty
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// RandomSoftwareCount returns a random count between min and max for the given platform
func RandomSoftwareCount(platform string) int {
	config, ok := platformCounts[platform]
	if !ok {
		return 0
	}
	return config.min + rand.IntN(config.max-config.min+1) // nolint:gosec,G404 // load testing, not security-sensitive
}

// DarwinSoftware represents macOS/iOS software
type DarwinSoftware struct {
	Name             string
	Version          string
	Source           string  // apps, homebrew_packages, firefox_addons, chrome_extensions, python_packages, vscode_extensions, safari_extensions (interned)
	BundleIdentifier *string // optional - used by apps
	Vendor           *string // optional - used by apps, vscode_extensions (interned)
	ExtensionID      *string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
	ExtensionFor     *string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
}

// WindowsSoftware represents Windows software
type WindowsSoftware struct {
	Name         string
	Version      string
	Source       string  // firefox_addons, chrome_extensions, programs, vscode_extensions, ie_extensions, python_packages, deb_packages (interned)
	Vendor       *string // optional - used by programs, vscode_extensions (interned)
	UpgradeCode  *string // optional - used by programs
	ExtensionID  *string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
	ExtensionFor *string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
}

// UbuntuSoftware represents Ubuntu/Linux software
type UbuntuSoftware struct {
	Name         string
	Version      string
	Source       string  // firefox_addons, chrome_extensions, python_packages, deb_packages, vscode_extensions, npm_packages, rpm_packages (interned)
	Vendor       *string // optional - used by rpm_packages, vscode_extensions (interned)
	Arch         *string // optional - used by rpm_packages
	Release      *string // optional - used by rpm_packages
	ExtensionID  *string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
	ExtensionFor *string // optional - used by firefox_addons, chrome_extensions, vscode_extensions
}

// DB holds the loaded software data for each platform
type DB struct {
	Darwin  []DarwinSoftware
	Windows []WindowsSoftware
	Ubuntu  []UbuntuSoftware
}

// DarwinToMaps converts Darwin software at given indices to osquery result format
func (db *DB) DarwinToMaps(indices []uint32) []map[string]string {
	results := make([]map[string]string, 0, len(indices))
	for _, idx := range indices {
		s := db.Darwin[idx]
		m := map[string]string{
			"name":    s.Name,
			"source":  s.Source,
			"version": s.Version,
		}
		if s.BundleIdentifier != nil {
			m["bundle_identifier"] = *s.BundleIdentifier
		}
		if s.Vendor != nil {
			m["vendor"] = *s.Vendor
		}
		if s.ExtensionID != nil {
			m["extension_id"] = *s.ExtensionID
		}
		if s.ExtensionFor != nil {
			m["browser"] = *s.ExtensionFor
		}
		results = append(results, m)
	}
	return results
}

// WindowsToMaps converts Windows software at given indices to osquery result format
func (db *DB) WindowsToMaps(indices []uint32) []map[string]string {
	results := make([]map[string]string, 0, len(indices))
	for _, idx := range indices {
		s := db.Windows[idx]
		m := map[string]string{
			"name":    s.Name,
			"source":  s.Source,
			"version": s.Version,
		}
		if s.Vendor != nil {
			m["vendor"] = *s.Vendor
		}
		if s.UpgradeCode != nil {
			m["upgrade_code"] = *s.UpgradeCode
		}
		if s.ExtensionID != nil {
			m["extension_id"] = *s.ExtensionID
		}
		if s.ExtensionFor != nil {
			m["browser"] = *s.ExtensionFor
		}
		results = append(results, m)
	}
	return results
}

// UbuntuToMaps converts Ubuntu software at given indices to osquery result format
func (db *DB) UbuntuToMaps(indices []uint32) []map[string]string {
	results := make([]map[string]string, 0, len(indices))
	for _, idx := range indices {
		s := db.Ubuntu[idx]
		m := map[string]string{
			"name":    s.Name,
			"source":  s.Source,
			"version": s.Version,
		}
		if s.Vendor != nil {
			m["vendor"] = *s.Vendor
		}
		if s.Arch != nil {
			m["arch"] = *s.Arch
		}
		if s.Release != nil {
			m["release"] = *s.Release
		}
		if s.ExtensionID != nil {
			m["extension_id"] = *s.ExtensionID
		}
		if s.ExtensionFor != nil {
			m["browser"] = *s.ExtensionFor
		}
		results = append(results, m)
	}
	return results
}

// MaybeMutateSoftware randomly mutates software indices (adds/removes items) 20% of the time.
// This simulates software being installed/uninstalled on a host over time.
// maxPoolSize is the total number of available software items in the database.
func MaybeMutateSoftware(indices []uint32, maxPoolSize int) []uint32 {
	// Only mutate 20% of the time
	if rand.Float64() >= SoftwareMutationProb { // nolint:gosec,G404 // load testing, not security-sensitive
		return indices
	}

	// Copy indices to avoid mutating the original slice
	result := make([]uint32, len(indices))
	copy(result, indices)

	// Randomly remove 0-20 items
	numToRemove := rand.IntN(MaxSoftwareRemove + 1) // nolint:gosec,G404 // load testing, not security-sensitive
	if numToRemove > len(result) {
		numToRemove = len(result)
	}
	if numToRemove > 0 {
		// Remove random items
		rand.Shuffle(len(result), func(i, j int) { // nolint:gosec,G404 // load testing, not security-sensitive
			result[i], result[j] = result[j], result[i]
		})
		result = result[:len(result)-numToRemove]
	}

	// Randomly add 0-20 items
	numToAdd := rand.IntN(MaxSoftwareAdd + 1) // nolint:gosec,G404 // load testing, not security-sensitive
	if numToAdd > 0 {
		// Create a map of existing indices for quick lookup
		existing := make(map[uint32]bool, len(result))
		for _, idx := range result {
			existing[idx] = true
		}

		// Add new random indices that don't already exist
		added := 0
		attempts := 0
		maxAttempts := numToAdd * 10 // Avoid infinite loop
		for added < numToAdd && attempts < maxAttempts {
			newIdx := uint32(rand.IntN(maxPoolSize)) // nolint:gosec,G404 // load testing, not security-sensitive
			if !existing[newIdx] {
				result = append(result, newIdx)
				existing[newIdx] = true
				added++
			}
			attempts++
		}
	}

	return result
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

	// Load ALL software for each platform (agents will select random subsets)
	softwareDB := &DB{}

	// Load Darwin software
	darwinConfig := platformCounts["darwin"]
	darwinSoftware, err := loadDarwinSoftware(db, darwinConfig.sources)
	if err != nil {
		return nil, err
	}
	softwareDB.Darwin = darwinSoftware
	log.Printf("Loaded %d darwin software items from database", len(darwinSoftware))

	// Load Windows software
	windowsConfig := platformCounts["windows"]
	windowsSoftware, err := loadWindowsSoftware(db, windowsConfig.sources)
	if err != nil {
		return nil, err
	}
	softwareDB.Windows = windowsSoftware
	log.Printf("Loaded %d windows software items from database", len(windowsSoftware))

	// Load Ubuntu software
	ubuntuConfig := platformCounts["ubuntu"]
	ubuntuSoftware, err := loadUbuntuSoftware(db, ubuntuConfig.sources)
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

// loadDarwinSoftware loads all macOS/iOS software from the database for the given sources
func loadDarwinSoftware(db *sql.DB, sources []string) ([]DarwinSoftware, error) {
	sourceList := "'" + strings.Join(sources, "', '") + "'"
	// nolint:gosec // sources are hardcoded, not user input
	query := fmt.Sprintf(`
		SELECT name, version, source, bundle_identifier, vendor, extension_id, extension_for
		FROM software
		WHERE source IN (%s)
		ORDER BY RANDOM()
		LIMIT %d
	`, sourceList, MaxSoftwarePerPlatform)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying darwin software: %w", err)
	}
	defer rows.Close()

	software := make([]DarwinSoftware, 0, MaxSoftwarePerPlatform)
	for rows.Next() {
		var sw DarwinSoftware
		var source string
		var bundleID, vendor, extensionID, extensionFor sql.NullString

		err := rows.Scan(&sw.Name, &sw.Version, &source, &bundleID, &vendor, &extensionID, &extensionFor)
		if err != nil {
			return nil, fmt.Errorf("scanning darwin software row: %w", err)
		}

		// Use interned source string
		sw.Source = internSource(source)

		// Use pointers for optional fields
		if bundleID.Valid {
			sw.BundleIdentifier = ptrString(bundleID.String)
		}
		if vendor.Valid {
			sw.Vendor = ptrString(internString(vendor.String))
		}
		if extensionID.Valid {
			sw.ExtensionID = ptrString(extensionID.String)
		}
		if extensionFor.Valid {
			sw.ExtensionFor = ptrString(extensionFor.String)
		}

		software = append(software, sw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating darwin software rows: %w", err)
	}

	return software, nil
}

// loadWindowsSoftware loads all Windows software from the database for the given sources
func loadWindowsSoftware(db *sql.DB, sources []string) ([]WindowsSoftware, error) {
	sourceList := "'" + strings.Join(sources, "', '") + "'"
	// nolint:gosec // sources are hardcoded, not user input
	query := fmt.Sprintf(`
		SELECT name, version, source, vendor, upgrade_code, extension_id, extension_for
		FROM software
		WHERE source IN (%s)
		ORDER BY RANDOM()
		LIMIT %d
	`, sourceList, MaxSoftwarePerPlatform)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying windows software: %w", err)
	}
	defer rows.Close()

	software := make([]WindowsSoftware, 0, MaxSoftwarePerPlatform)
	for rows.Next() {
		var sw WindowsSoftware
		var source string
		var vendor, upgradeCode, extensionID, extensionFor sql.NullString

		err := rows.Scan(&sw.Name, &sw.Version, &source, &vendor, &upgradeCode, &extensionID, &extensionFor)
		if err != nil {
			return nil, fmt.Errorf("scanning windows software row: %w", err)
		}

		// Use interned source string
		sw.Source = internSource(source)

		// Use pointers for optional fields
		if vendor.Valid {
			sw.Vendor = ptrString(internString(vendor.String))
		}
		if upgradeCode.Valid {
			sw.UpgradeCode = ptrString(upgradeCode.String)
		}
		if extensionID.Valid {
			sw.ExtensionID = ptrString(extensionID.String)
		}
		if extensionFor.Valid {
			sw.ExtensionFor = ptrString(extensionFor.String)
		}

		software = append(software, sw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating windows software rows: %w", err)
	}

	return software, nil
}

// loadUbuntuSoftware loads all Ubuntu/Linux software from the database for the given sources
func loadUbuntuSoftware(db *sql.DB, sources []string) ([]UbuntuSoftware, error) {
	sourceList := "'" + strings.Join(sources, "', '") + "'"
	// nolint:gosec // sources are hardcoded, not user input
	query := fmt.Sprintf(`
		SELECT name, version, source, vendor, arch, release, extension_id, extension_for
		FROM software
		WHERE source IN (%s)
		ORDER BY RANDOM()
		LIMIT %d
	`, sourceList, MaxSoftwarePerPlatform)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying ubuntu software: %w", err)
	}
	defer rows.Close()

	software := make([]UbuntuSoftware, 0, MaxSoftwarePerPlatform)
	for rows.Next() {
		var sw UbuntuSoftware
		var source string
		var vendor, arch, release, extensionID, extensionFor sql.NullString

		err := rows.Scan(&sw.Name, &sw.Version, &source, &vendor, &arch, &release, &extensionID, &extensionFor)
		if err != nil {
			return nil, fmt.Errorf("scanning ubuntu software row: %w", err)
		}

		// Use interned source string
		sw.Source = internSource(source)

		// Use pointers for optional fields
		if vendor.Valid {
			sw.Vendor = ptrString(internString(vendor.String))
		}
		if arch.Valid {
			sw.Arch = ptrString(arch.String)
		}
		if release.Valid {
			sw.Release = ptrString(release.String)
		}
		if extensionID.Valid {
			sw.ExtensionID = ptrString(extensionID.String)
		}
		if extensionFor.Valid {
			sw.ExtensionFor = ptrString(extensionFor.String)
		}

		software = append(software, sw)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating ubuntu software rows: %w", err)
	}

	return software, nil
}
