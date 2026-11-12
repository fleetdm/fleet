package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
)

const childProcessEnvVar = "MIGRATION_RETRY_CHECK_CHILD"

func main() {
	migrations := flag.String("migrations", "", "whitespace-separated migration file paths, as produced by git diff --diff-filter=A")
	flag.Parse()

	if childUsername := os.Getenv(childProcessEnvVar); childUsername != "" {
		database, err := openDatabase(childUsername, "migration_retry_check")
		if err != nil {
			log.Fatal(err)
		}
		if err := tables.MigrationClient.UpByOne(database.DB, ""); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *migrations == "" {
		log.Fatal("no migrations given, pass -migrations with migration file paths")
	}

	var migrationVersions []int64
	for path := range strings.FieldsSeq(*migrations) {
		timestamp, _, _ := strings.Cut(filepath.Base(path), "_")
		version, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			log.Fatalf("parsing migration timestamp from %q: %s", path, err)
		}

		// An unregistered migration would run the one below it instead.
		next, err := tables.MigrationClient.Migrations.Next(version - 1)
		if err != nil || next.Version != version {
			log.Fatalf("%d is not registered, check that its init calls AddMigration", version)
		}

		migrationVersions = append(migrationVersions, version)
	}

	failed := false
	for _, migrationVersion := range migrationVersions {
		if err := checkMigrationIsRetryable(migrationVersion); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s\n", err)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func checkMigrationIsRetryable(migrationVersion int64) error {
	server, err := openDatabase("root", "")
	if err != nil {
		return err
	}
	defer server.Close()

	if _, err := server.Exec(`
DROP DATABASE IF EXISTS migration_retry_check;
CREATE DATABASE migration_retry_check`); err != nil {
		return err
	}

	if err := applyMigrationsBeforeVersion(migrationVersion); err != nil {
		return err
	}

	// Create a user that can change the schema but not INSERT, UPDATE or DELETE.
	// The grant on the version table has to run after the migrations that create it.
	grants := fmt.Sprintf(`
DROP USER IF EXISTS 'ddlonly'@'%%';
CREATE USER 'ddlonly'@'%%' IDENTIFIED BY 'toor';
GRANT ALL PRIVILEGES ON migration_retry_check.* TO 'ddlonly'@'%%';
REVOKE INSERT, UPDATE, DELETE ON migration_retry_check.* FROM 'ddlonly'@'%%';
GRANT INSERT ON migration_retry_check.%s TO 'ddlonly'@'%%'`, tables.MigrationClient.TableName)

	if _, err := server.Exec(grants); err != nil {
		return err
	}

	// goose calls log.Fatalf on failure, so only a child's exit code reveals it.
	exitCode, output := applyMigrationInSubprocess("ddlonly")
	if exitCode == 0 {
		fmt.Printf("%d: not applicable, no writes were attempted against the empty database\n", migrationVersion)
		return nil
	}

	// 1142 and 1143 are the table and column level access denied errors.
	if !strings.Contains(output, "Error 1142") && !strings.Contains(output, "Error 1143") {
		return fmt.Errorf("❌ fail: %d could not be checked, the run without write privileges failed for another reason", migrationVersion)
	}

	if exitCode, _ := applyMigrationInSubprocess("root"); exitCode != 0 {
		return fmt.Errorf("❌ fail: %d is not retryable after a mid-migration failure\n"+
			"make its CREATE and ALTER statements safe to run twice: IF NOT EXISTS, or columnExists and friends in migration.go", migrationVersion)
	}

	return nil
}

func applyMigrationsBeforeVersion(migrationVersion int64) error {
	database, err := openDatabase("root", "migration_retry_check")
	if err != nil {
		return err
	}
	defer database.Close()

	for {
		currentVersion, err := tables.MigrationClient.GetDBVersion(database.DB)
		if err != nil {
			return err
		}

		nextMigration, err := tables.MigrationClient.Migrations.Next(currentVersion)
		if err != nil {
			return err
		}

		if nextMigration.Version == migrationVersion {
			return nil
		}

		if err := tables.MigrationClient.UpByOne(database.DB, ""); err != nil {
			return err
		}
	}
}

func applyMigrationInSubprocess(username string) (int, string) {
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	var output strings.Builder
	command := exec.Command(executablePath)
	command.Env = append(os.Environ(), childProcessEnvVar+"="+username)
	command.Stdout = io.MultiWriter(os.Stdout, &output)
	command.Stderr = io.MultiWriter(os.Stderr, &output)

	err = command.Run()
	if err == nil {
		return 0, output.String()
	}

	if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitError.ExitCode(), output.String()
	}

	log.Fatal(err)
	return 1, ""
}

func openDatabase(username string, database string) (*sqlx.DB, error) {
	address := "localhost:3307"
	if port := os.Getenv("FLEET_MYSQL_TEST_PORT"); port != "" {
		address = "localhost:" + port
	}

	config := platform_mysql.MysqlConfig{
		Protocol:     "tcp",
		Address:      address,
		Username:     username,
		Password:     "toor",
		Database:     database,
		MaxOpenConns: 5,
		MaxIdleConns: 5,
	}

	return platform_mysql.NewDB(&config, &platform_mysql.DBOptions{
		MaxAttempts: 5,
		Logger:      slog.Default(),
	}, "")
}
