// Command bump-migration bumps the timestamp of a migration file and updates
// the code accordingly. If there is a test file for the migration, it is also
// renamed and updated. It can optionally regenerate the database schema file.
//
// This operation is required when a PR has a database migration that is older
// than an existing migration in the main branch, e.g. because the PR has been
// pending merge for a while and another PR got merged with a more recent
// DB migration.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	const timeFormat = "20060102150405"

	var (
		sourceMigration = flag.String("source-migration", "", "Name of the source migration file to bump (required).")
		regenSchema     = flag.Bool("regen-schema", false, "Regenerate the database schema file after bumping the migration (optional).")
	)

	flag.Parse()
	if *sourceMigration == "" {
		log.Println("The --source-migration flag is required.")
		flag.Usage()
		return
	}

	sourceFilename := filepath.Base(*sourceMigration)
	migrationsDir := filepath.Join("server", "datastore", "mysql", "migrations", "tables")
	fullPath := filepath.Join(migrationsDir, sourceFilename)
	switch _, err := os.Stat(fullPath); {
	case errors.Is(err, os.ErrNotExist):
		log.Fatalf("The migration file '%s' does not exist in the expected path, make sure you run this command from the root of the repository: %s", sourceFilename, fullPath)
	case err != nil:
		log.Fatalf("Error checking the migration file '%s': %v", sourceFilename, err)
	default:
		if strings.HasSuffix(sourceFilename, "_test.go") {
			log.Fatalf("The migration file '%s' is a test file, please provide the original migration file instead.", sourceFilename)
		}
	}

	oldTimestamp, _, ok := strings.Cut(sourceFilename, "_")
	if !ok {
		log.Fatalf("Bad filename pattern, expected to find the migration's current timestamp before '_' in '%s'", sourceFilename)
	}
	if _, err := time.Parse(timeFormat, oldTimestamp); err != nil {
		log.Fatalf("Bad filename pattern, '%s' is not a valid timestamp in '%s'", oldTimestamp, sourceFilename)
	}
	newTimestamp := time.Now().Format(timeFormat)

	newMig, newTest, err := renameMigrationFiles(migrationsDir, sourceFilename, oldTimestamp, newTimestamp)
	if err != nil {
		log.Fatal(err)
	}
	if err := updateMigrationCode(migrationsDir, newMig, newTest, oldTimestamp, newTimestamp); err != nil {
		log.Fatal(err)
	}

	if *regenSchema {
		cmd := exec.Command("make", "dump-test-schema")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Fatalf("Error regenerating the schema: %v", err)
		}
	}
}

func updateMigrationCode(migrationsDir, migrationFilename, testFilename, oldTimestamp, newTimestamp string) error {
	migrationReplacer := strings.NewReplacer(
		fmt.Sprintf("MigrationClient.AddMigration(Up_%s, Down_%s)", oldTimestamp, oldTimestamp),
		fmt.Sprintf("MigrationClient.AddMigration(Up_%s, Down_%s)", newTimestamp, newTimestamp),
		fmt.Sprintf("func Up_%s(tx *sql.Tx)", oldTimestamp),
		fmt.Sprintf("func Up_%s(tx *sql.Tx)", newTimestamp),
		fmt.Sprintf("func Down_%s(tx *sql.Tx)", oldTimestamp),
		fmt.Sprintf("func Down_%s(tx *sql.Tx)", newTimestamp),
	)

	oldData, err := os.ReadFile(filepath.Join(migrationsDir, migrationFilename))
	if err != nil {
		return fmt.Errorf("Error reading migration file '%s': %w", migrationFilename, err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, migrationFilename), []byte(migrationReplacer.Replace(string(oldData))), 0o644); err != nil {
		return fmt.Errorf("Error writing migration file '%s': %w", migrationFilename, err)
	}

	if testFilename != "" {
		testReplacer := strings.NewReplacer(
			// test files can have multiple tests with pattern
			// TestUp_<timestamp>_Blah (or sub-tests, but those should not have the
			// old timestamp in the name)
			fmt.Sprintf("func TestUp_%s", oldTimestamp),
			fmt.Sprintf("func TestUp_%s", newTimestamp),
		)

		oldData, err := os.ReadFile(filepath.Join(migrationsDir, testFilename))
		if err != nil {
			return fmt.Errorf("Error reading migration test file '%s': %w", testFilename, err)
		}
		if err := os.WriteFile(filepath.Join(migrationsDir, testFilename), []byte(testReplacer.Replace(string(oldData))), 0o644); err != nil {
			return fmt.Errorf("Error writing migration test file '%s': %w", testFilename, err)
		}
	}
	return nil
}

func renameMigrationFiles(migrationsDir, migrationFilename, oldTimestamp, newTimestamp string) (newMig, newTest string, err error) {
	oldPath := filepath.Join(migrationsDir, migrationFilename)
	newMigFilename := strings.Replace(migrationFilename, oldTimestamp, newTimestamp, 1)
	newPath := filepath.Join(migrationsDir, newMigFilename)

	// rename the migration file itself
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", "", fmt.Errorf("Rename migration file failed: %w", err)
	}

	// check if a test file exists
	testFilename := strings.TrimSuffix(migrationFilename, ".go") + "_test.go"
	oldPath = filepath.Join(migrationsDir, testFilename)
	newTestFilename := strings.Replace(testFilename, oldTimestamp, newTimestamp, 1)
	newPath = filepath.Join(migrationsDir, newTestFilename)
	switch _, err := os.Stat(oldPath); {
	case errors.Is(err, os.ErrNotExist):
		// nothing to do, test file does not exist
		newTestFilename = ""
	case err != nil:
		return "", "", fmt.Errorf("Error checking the migration test file '%s': %w", oldPath, err)
	default:
		// test file exists, rename it
		if err := os.Rename(oldPath, newPath); err != nil {
			return "", "", fmt.Errorf("Rename migration test file failed: %w", err)
		}
	}

	return newMigFilename, newTestFilename, nil
}
