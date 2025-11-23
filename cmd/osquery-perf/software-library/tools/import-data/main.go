package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// SoftwareEntry represents a software item from server data
type SoftwareEntry struct {
	Name             string
	Version          string
	Source           string
	BundleIdentifier string
	Vendor           string
	Arch             string
	Release          string
	ExtensionID      string
	ExtensionFor     string
	ApplicationID    *string
	UpgradeCode      *string
}

// Known public software (always keep when filtering by vendor)
var knownPublicSoftware = []string{
	"chrome", "google chrome", "firefox", "mozilla firefox",
	"python", "docker", "git", "visual studio", "vscode",
	"slack", "zoom", "microsoft office", "office", "teams",
	"excel", "word", "powerpoint", "outlook", "skype",
	"java", "node", "nodejs", "rust", "go", "kubectl",
	"aws", "terraform", "ansible", "jenkins", "jira",
	"confluence", "postman", "cuda", "geforce", "quadro",
}

// privateIPRegex matches private IP address ranges:
// - 10.0.0.0/8 (10.x.x.x)
// - 172.16.0.0/12 (172.16.x.x - 172.31.x.x)
// - 192.168.0.0/16 (192.168.x.x)
// - 127.0.0.0/8 (127.x.x.x - loopback)
var privateIPRegex = regexp.MustCompile(`^(10(\.\d{1,3}){3}|127(\.\d{1,3}){3}|192\.168(\.\d{1,3}){2}|172\.(1[6-9]|2[0-9]|3[0-1])(\.\d{1,3}){2})`)

type ImportStats struct {
	Total             int
	Imported          int
	FilteredInternal  int
	FilteredVendor    int
	FilteredAmbiguous int
	Duplicates        int
}

type Importer struct {
	db             *sql.DB
	dryRun         bool
	verbose        bool
	stats          ImportStats
	filterPatterns []string // Patterns to filter out (e.g., "internal", "corp-")
	filterVendor   string   // Vendor to filter out
}

func main() {
	inputFile := flag.String("input", "", "Input CSV file (required)")
	dbPath := flag.String("db", "../../software.db", "Database path")
	dryRun := flag.Bool("dry-run", false, "Validate data without importing")
	verbose := flag.Bool("verbose", false, "Verbose output")
	filter := flag.String("filter", "", "Comma-separated patterns to filter out (e.g., 'internal,corp-')")
	filterVendor := flag.String("filter-vendor", "", "Vendor to filter out")

	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Error: --input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if err := run(*inputFile, *dbPath, *dryRun, *verbose, *filter, *filterVendor); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func run(inputFile, dbPath string, dryRun, verbose bool, filter, filterVendor string) error {
	// Verify input file is CSV
	if !strings.HasSuffix(inputFile, ".csv") {
		return errors.New("input file must be a CSV file")
	}

	// Resolve database path
	absDBPath, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("resolving database path: %w", err)
	}

	// Parse filter patterns
	var filterPatterns []string
	if filter != "" {
		filterPatterns = strings.Split(filter, ",")
		for i := range filterPatterns {
			filterPatterns[i] = strings.TrimSpace(filterPatterns[i])
		}
	}

	fmt.Println("🚀 Starting import...")
	fmt.Printf("   Input:  %s\n", inputFile)
	fmt.Printf("   Database: %s\n", absDBPath)
	if dryRun {
		fmt.Println("   Mode: DRY RUN")
	}
	if len(filterPatterns) > 0 || filterVendor != "" {
		fmt.Println("   Filtering: ENABLED")
		if len(filterPatterns) > 0 {
			fmt.Printf("      Patterns: %s\n", strings.Join(filterPatterns, ", "))
		}
		if filterVendor != "" {
			fmt.Printf("      Vendor: %s\n", filterVendor)
		}
	} else {
		fmt.Println("   Filtering: DISABLED (all entries will be imported)")
	}
	fmt.Println()

	// Check if database exists
	if _, err := os.Stat(absDBPath); os.IsNotExist(err) {
		return fmt.Errorf("database not found: %s\n\nPlease create the database first:\n  cd %s\n  sqlite3 software.db < software.sql",
			absDBPath, filepath.Dir(absDBPath))
	}

	// Connect to database
	db, err := sql.Open("sqlite3", absDBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Verify database has required tables
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='software'").Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("database exists but 'software' table not found\n\nPlease initialize the database:\n  cd %s\n  sqlite3 software.db < software.sql",
			filepath.Dir(absDBPath))
	}

	// Create importer
	importer := &Importer{
		db:             db,
		dryRun:         dryRun,
		verbose:        verbose,
		filterPatterns: filterPatterns,
		filterVendor:   filterVendor,
	}

	// Import CSV data
	if err := importer.importCSV(inputFile); err != nil {
		return err
	}

	// Print statistics
	importer.printStats()
	return nil
}

func (imp *Importer) importCSV(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV headers: %w", err)
	}

	// Map headers to indices
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[header] = i
	}

	fmt.Printf("📁 Importing from CSV: %s\n", filename)

	// Read all rows
	rowNum := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV row %d: %w", rowNum, err)
		}

		imp.stats.Total++
		rowNum++

		if rowNum%1000 == 0 {
			fmt.Printf("  Processed %d rows...\n", rowNum)
		}

		// Parse row into SoftwareEntry
		entry := parseSoftwareFromCSV(record, headerMap)
		imp.importEntry(entry)
	}

	return nil
}

func (imp *Importer) importEntry(entry SoftwareEntry) {
	// Validate required fields
	if entry.Name == "" || entry.Version == "" || entry.Source == "" {
		if imp.verbose {
			fmt.Printf("  ⚠️  Skipping entry with missing required fields\n")
		}
		return
	}

	// Check if software should be imported (filtering is optional)
	shouldImport, reason := imp.shouldImport(entry.Name, entry.Vendor)
	if !shouldImport {
		switch {
		case strings.HasPrefix(reason, "internal_pattern"):
			imp.stats.FilteredInternal++
		case strings.Contains(reason, "vendor"):
			imp.stats.FilteredVendor++
		default:
			imp.stats.FilteredAmbiguous++
		}

		if imp.verbose {
			fmt.Printf("  ❌ Filtered: %s (%s)\n", entry.Name, reason)
		}
		return
	}

	// Insert into database
	if !imp.dryRun {
		err := imp.insertSoftware(entry)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				imp.stats.Duplicates++
				if imp.verbose {
					fmt.Printf("  ⏭️  Duplicate: %s v%s\n", entry.Name, entry.Version)
				}
			} else {
				fmt.Printf("  ❌ Error inserting %s: %v\n", entry.Name, err)
			}
			return
		}
	}

	imp.stats.Imported++
	if imp.verbose {
		fmt.Printf("  ✅ Imported: %s v%s (%s)\n", entry.Name, entry.Version, entry.Source)
	}
}

// isInternalDomain checks if a vendor string looks like an internal domain
// for the given filter vendor (e.g., "confluence.numa.com", "gitlab.acme.com")
func isInternalDomain(vendor, filterVendor string) bool {
	vendorLower := strings.ToLower(vendor)
	filterVendorLower := strings.ToLower(filterVendor)

	// Check if vendor contains a domain pattern with the filter vendor
	// e.g., "confluence.numa.com", "gitlab.acme.com", "*.company.com"
	if strings.Contains(vendorLower, "."+filterVendorLower+".com") ||
		strings.Contains(vendorLower, filterVendorLower+".com") ||
		strings.HasSuffix(vendorLower, "."+filterVendorLower+".net") ||
		strings.Contains(vendorLower, "."+filterVendorLower+".") {
		return true
	}

	return false
}

// shouldImport determines if software should be imported based on optional filters
func (imp *Importer) shouldImport(name, vendor string) (bool, string) {
	// If no filters are configured, import everything
	if len(imp.filterPatterns) == 0 && imp.filterVendor == "" {
		return true, "no_filter"
	}

	nameLower := strings.ToLower(name)
	vendorLower := strings.ToLower(vendor)

	// Check for internal patterns (if configured)
	if len(imp.filterPatterns) > 0 {
		for _, pattern := range imp.filterPatterns {
			patternLower := strings.ToLower(pattern)
			if strings.Contains(nameLower, patternLower) {
				return false, fmt.Sprintf("internal_pattern:%s", pattern)
			}
		}
	}

	// Always filter private IP addresses when filtering is enabled
	if privateIPRegex.MatchString(vendor) {
		return false, "vendor_private_ip"
	}

	// Filter by vendor (if configured)
	if imp.filterVendor != "" {
		filterVendorLower := strings.ToLower(imp.filterVendor)
		if strings.Contains(vendorLower, filterVendorLower) {
			// First check if vendor is an internal domain or private IP
			// If it is, always filter it out regardless of software name
			if isInternalDomain(vendor, imp.filterVendor) {
				return false, fmt.Sprintf("vendor_internal_domain:%s", imp.filterVendor)
			}

			// Check if it's known public software (always keep)
			// Only applies if vendor is NOT an internal domain
			for _, publicName := range knownPublicSoftware {
				if strings.Contains(nameLower, publicName) {
					return true, "known_public"
				}
			}

			return false, fmt.Sprintf("vendor:%s", imp.filterVendor)
		}
	}

	// Default: allow
	return true, "allowed"
}

func (imp *Importer) insertSoftware(entry SoftwareEntry) error {
	query := `
		INSERT INTO software (
			name, version, source, bundle_identifier, vendor, arch, release,
			extension_id, extension_for, application_id, upgrade_code
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := imp.db.Exec(query,
		entry.Name,
		entry.Version,
		entry.Source,
		entry.BundleIdentifier,
		entry.Vendor,
		entry.Arch,
		entry.Release,
		entry.ExtensionID,
		entry.ExtensionFor,
		entry.ApplicationID,
		entry.UpgradeCode,
	)

	return err
}

func (imp *Importer) printStats() {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("📊 Import Statistics")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total entries processed:     %d\n", imp.stats.Total)
	fmt.Printf("Successfully imported:       %d\n", imp.stats.Imported)

	// Only show filter stats if filtering was enabled
	if len(imp.filterPatterns) > 0 || imp.filterVendor != "" {
		fmt.Printf("Filtered (internal pattern): %d\n", imp.stats.FilteredInternal)
		fmt.Printf("Filtered (vendor):           %d\n", imp.stats.FilteredVendor)
		fmt.Printf("Filtered (ambiguous):        %d\n", imp.stats.FilteredAmbiguous)
	}

	fmt.Printf("Duplicates skipped:          %d\n", imp.stats.Duplicates)
	fmt.Println(strings.Repeat("=", 60))

	totalFiltered := imp.stats.FilteredInternal + imp.stats.FilteredVendor + imp.stats.FilteredAmbiguous
	if totalFiltered > 0 {
		fmt.Printf("\n⚠️  %d entries were filtered out\n", totalFiltered)
	} else if len(imp.filterPatterns) == 0 && imp.filterVendor == "" {
		fmt.Println("\nℹ️  No filtering was applied - all valid entries were imported")
	}

	if imp.dryRun {
		fmt.Println("\n🔍 DRY RUN - No changes were made to the database")
	} else {
		fmt.Printf("\n✅ Data successfully imported\n")
	}
}

func parseSoftwareFromCSV(record []string, headerMap map[string]int) SoftwareEntry {
	get := func(field string) string {
		if idx, ok := headerMap[field]; ok && idx < len(record) {
			return record[idx]
		}
		return ""
	}

	getPtr := func(field string) *string {
		val := get(field)
		if val == "" {
			return nil
		}
		return &val
	}

	return SoftwareEntry{
		Name:             get("name"),
		Version:          get("version"),
		Source:           get("source"),
		BundleIdentifier: get("bundle_identifier"),
		Vendor:           get("vendor"),
		Arch:             get("arch"),
		Release:          get("release"),
		ExtensionID:      get("extension_id"),
		ExtensionFor:     get("extension_for"),
		ApplicationID:    getPtr("application_id"),
		UpgradeCode:      getPtr("upgrade_code"),
	}
}
