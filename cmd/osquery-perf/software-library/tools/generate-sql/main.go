package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := flag.String("db", "../../software.db", "Database path")
	outputPath := flag.String("output", "../../software.sql", "Output SQL file path")
	verbose := flag.Bool("verbose", false, "Verbose output")

	flag.Parse()

	if err := run(*dbPath, *outputPath, *verbose); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run(dbPath, outputPath string, verbose bool) error {
	// Resolve paths
	absDBPath, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("resolving database path: %w", err)
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	fmt.Println("🚀 Generating software.sql...")
	fmt.Printf("   Database: %s\n", absDBPath)
	fmt.Printf("   Output:   %s\n", absOutputPath)
	fmt.Println()

	// Check if database exists
	if _, err := os.Stat(absDBPath); os.IsNotExist(err) {
		return fmt.Errorf("database file not found: %s", absDBPath)
	}

	// Connect to database
	db, err := sql.Open("sqlite3", absDBPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Create output file
	output, err := os.Create(absOutputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer output.Close()

	// Write header
	writeHeader(output)

	// Write schema (from schema.sql)
	fmt.Println("📄 Writing schema...")
	if err := writeSchema(output, absDBPath); err != nil {
		return fmt.Errorf("writing schema: %w", err)
	}

	// Write software data
	fmt.Println("💾 Writing software data...")
	count, err := writeSoftwareData(db, output, verbose)
	if err != nil {
		return fmt.Errorf("writing software data: %w", err)
	}

	// Write footer
	writeFooter(output, count)

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("✅ Successfully generated software.sql\n")
	fmt.Printf("   Total software entries: %d\n", count)
	fmt.Printf("   Output file: %s\n", absOutputPath)
	fmt.Println(strings.Repeat("=", 60))

	return nil
}

func writeHeader(output *os.File) {
	header := `-- Software Library SQL Dump
-- Generated: %s
--
-- This file can be used to recreate the software database:
--   sqlite3 software.db < software.sql
--

`
	fmt.Fprintf(output, header, getCurrentTimestamp())
}

func writeSchema(output *os.File, dbPath string) error {
	if _, err := output.WriteString("-- Software Library Schema\n"); err != nil {
		return err
	}
	if _, err := output.WriteString("-- This schema defines the structure for storing software data used in osquery-perf load testing\n\n"); err != nil {
		return err
	}

	// Hardcoded schema (single source of truth)
	schema := `-- Software table
CREATE TABLE IF NOT EXISTS software (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    source TEXT NOT NULL,
    bundle_identifier TEXT DEFAULT '',
    vendor TEXT DEFAULT '',
    arch TEXT DEFAULT '',
    release TEXT DEFAULT '',
    extension_id TEXT DEFAULT '',
    extension_for TEXT DEFAULT '',
    application_id TEXT DEFAULT NULL,
    upgrade_code TEXT DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_software_source ON software(source);
CREATE INDEX IF NOT EXISTS idx_software_name ON software(name);

-- Unique constraint to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_software_unique ON software(name, version, source, bundle_identifier);

`
	if _, err := output.WriteString(schema); err != nil {
		return err
	}
	if _, err := output.WriteString("\n"); err != nil {
		return err
	}
	return nil
}

func writeSoftwareData(db *sql.DB, output *os.File, verbose bool) (int, error) {
	if _, err := output.WriteString("-- Software Data\n"); err != nil {
		return 0, err
	}
	if _, err := output.WriteString("-- Importing software entries...\n"); err != nil {
		return 0, err
	}
	if _, err := output.WriteString("BEGIN TRANSACTION;\n\n"); err != nil {
		return 0, err
	}

	query := `
		SELECT
			name, version, source, bundle_identifier, vendor, arch, release,
			extension_id, extension_for, application_id, upgrade_code
		FROM software
		WHERE NOT (source = 'deb_packages' AND name LIKE 'linux-image-%')
		ORDER BY source, name
	`

	rows, err := db.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	batchSize := 1000
	values := make([]string, 0, batchSize)

	for rows.Next() {
		var name, version, source, bundleID, vendor, arch, release string
		var extensionID, extensionFor string
		var applicationID, upgradeCode *string

		err := rows.Scan(
			&name, &version, &source, &bundleID, &vendor, &arch, &release,
			&extensionID, &extensionFor, &applicationID, &upgradeCode,
		)
		if err != nil {
			return count, err
		}

		// Build VALUES clause
		appID := "NULL"
		if applicationID != nil {
			appID = fmt.Sprintf("'%s'", escapeSQL(*applicationID))
		}

		upgCode := "NULL"
		if upgradeCode != nil {
			upgCode = fmt.Sprintf("'%s'", escapeSQL(*upgradeCode))
		}

		value := fmt.Sprintf(
			"('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %s, %s)",
			escapeSQL(name),
			escapeSQL(version),
			escapeSQL(source),
			escapeSQL(bundleID),
			escapeSQL(vendor),
			escapeSQL(arch),
			escapeSQL(release),
			escapeSQL(extensionID),
			escapeSQL(extensionFor),
			appID,
			upgCode,
		)

		values = append(values, value)
		count++

		// Write in batches
		if len(values) >= batchSize {
			if err := writeInsertBatch(output, values); err != nil {
				return count, err
			}
			values = values[:0]

			if verbose && count%10000 == 0 {
				fmt.Printf("  Processed %d entries...\n", count)
			}
		}
	}

	// Write remaining values
	if len(values) > 0 {
		if err := writeInsertBatch(output, values); err != nil {
			return count, err
		}
	}

	if _, err := output.WriteString("\nCOMMIT;\n\n"); err != nil {
		return count, err
	}

	return count, rows.Err()
}

func writeInsertBatch(output *os.File, values []string) error {
	if _, err := output.WriteString("INSERT INTO software ("); err != nil {
		return err
	}
	if _, err := output.WriteString("name, version, source, bundle_identifier, vendor, arch, release, "); err != nil {
		return err
	}
	if _, err := output.WriteString("extension_id, extension_for, application_id, upgrade_code"); err != nil {
		return err
	}
	if _, err := output.WriteString(") VALUES\n"); err != nil {
		return err
	}

	for i, value := range values {
		if _, err := output.WriteString("  " + value); err != nil {
			return err
		}
		if i < len(values)-1 {
			if _, err := output.WriteString(",\n"); err != nil {
				return err
			}
		} else {
			if _, err := output.WriteString(";\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFooter(output *os.File, count int) {
	footer := `
-- Summary
-- Total software entries: %d
-- Generated: %s
`
	fmt.Fprintf(output, footer, count, getCurrentTimestamp())
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func getCurrentTimestamp() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
}
