package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// SoftwareEntry represents a software item from customer data
type SoftwareEntry struct {
	Name             string  `json:"name" csv:"name"`
	Version          string  `json:"version" csv:"version"`
	Source           string  `json:"source" csv:"source"`
	BundleIdentifier string  `json:"bundle_identifier" csv:"bundle_identifier"`
	Vendor           string  `json:"vendor" csv:"vendor"`
	Arch             string  `json:"arch" csv:"arch"`
	Release          string  `json:"release" csv:"release"`
	ExtensionID      string  `json:"extension_id" csv:"extension_id"`
	ExtensionFor     string  `json:"extension_for" csv:"extension_for"`
	ApplicationID    *string `json:"application_id" csv:"application_id"`
	UpgradeCode      *string `json:"upgrade_code" csv:"upgrade_code"`
}

// Known public software (always keep when filtering by vendor)
var knownPublicSoftware = []string{
	"chrome", "google chrome", "firefox", "mozilla firefox",
	"python", "docker", "git", "visual studio", "vscode",
	"slack", "zoom", "microsoft office", "office", "teams",
	"excel", "word", "powerpoint", "outlook", "skype",
	"java", "node", "nodejs", "rust", "go", "kubectl",
	"aws", "terraform", "ansible", "jenkins", "jira",
	"confluence", "postman",
}

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
	inputFile := flag.String("input", "", "Input CSV or JSON file (required)")
	format := flag.String("format", "", "Input format: csv or json (auto-detected if not specified)")
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

	if err := run(*inputFile, *format, *dbPath, *dryRun, *verbose, *filter, *filterVendor); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func run(inputFile, format, dbPath string, dryRun, verbose bool, filter, filterVendor string) error {
	// Auto-detect format
	if format == "" {
		switch {
		case strings.HasSuffix(inputFile, ".csv"):
			format = "csv"
		case strings.HasSuffix(inputFile, ".json"):
			format = "json"
		default:
			return errors.New("could not detect format. Please specify --format")
		}
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
	fmt.Printf("   Format: %s\n", format)
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

	// Import data
	if format == "csv" {
		err = importer.importCSV(inputFile)
	} else {
		err = importer.importJSON(inputFile)
	}

	if err != nil {
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

func (imp *Importer) importJSON(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening JSON file: %w", err)
	}
	defer file.Close()

	var entries []SoftwareEntry
	decoder := json.NewDecoder(file)

	// Try to decode as array first
	if err := decoder.Decode(&entries); err != nil {
		// Try as object with "software" key
		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Errorf("seeking file: %w", err)
		}
		var data struct {
			Software []SoftwareEntry `json:"software"`
		}
		if err := json.NewDecoder(file).Decode(&data); err != nil {
			return fmt.Errorf("decoding JSON: %w", err)
		}
		entries = data.Software
	}

	fmt.Printf("📁 Importing from JSON: %s\n", filename)
	fmt.Printf("📊 Processing %d software entries...\n", len(entries))

	imp.stats.Total = len(entries)

	for i, entry := range entries {
		if (i+1)%1000 == 0 {
			fmt.Printf("  Processed %d/%d...\n", i+1, len(entries))
		}
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

	// Filter by vendor (if configured)
	if imp.filterVendor != "" {
		filterVendorLower := strings.ToLower(imp.filterVendor)
		if strings.Contains(vendorLower, filterVendorLower) {
			// Check if it's known public software (always keep)
			for _, publicName := range knownPublicSoftware {
				if strings.Contains(nameLower, publicName) {
					return true, "known_public"
				}
			}

			// Allow public products from the filtered vendor
			// (e.g., CUDA, drivers, GeForce, Quadro are public products)
			publicKeywords := []string{"cuda", "driver", "geforce", "quadro"}
			for _, keyword := range publicKeywords {
				if strings.Contains(nameLower, keyword) {
					return true, "public_product"
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
