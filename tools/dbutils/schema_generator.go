package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const (
	testUsername = "root"
	testPassword = "toor"
)

var (
	testAddress = getTestAddress()
)

func getTestAddress() string {
	if port := os.Getenv("FLEET_MYSQL_TEST_PORT"); port != "" {
		return "localhost:" + port
	}
	return "localhost:3307"
}

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

// main requires 2 arguments:
// 1. Path to the schema dumpfile
// 2. Path to the migration status dumpfile
func main() {
	if len(os.Args) != 3 {
		panic("usage: dbutils <schema.sql> <schema_migrations.sql>")
	}
	fmt.Println("dumping schema to", os.Args[1], "and", os.Args[2])

	// Create the database (must use raw MySQL client to do this)
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true", testUsername, testPassword, testAddress),
	)
	panicif(err)
	defer db.Close()
	_, err = db.Exec("DROP DATABASE IF EXISTS schemadb; CREATE DATABASE schemadb;")
	panicif(err)

	// Create a datastore client in order to run migrations as usual
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Address:  testAddress,
		Database: "schemadb",
	}
	ds, err := mysql.New(config, clock.NewMockClock(), mysql.Logger(slog.New(slog.DiscardHandler)), mysql.LimitAttempts(1))
	panicif(err)
	defer ds.Close()
	panicif(ds.MigrateTables(context.Background()))

	// Set created_at/updated_at for app_config_json to prevent the schema from being changed every time
	// This schema is to test anyway
	fixedDate := time.Date(2020, 01, 01, 01, 01, 01, 01, time.UTC)
	_, err = db.Exec(`USE schemadb`)
	panicif(err)
	_, err = db.Exec(`UPDATE app_config_json SET created_at = ?, updated_at = ?`, fixedDate, fixedDate)
	panicif(err)

	// Tests create these built-in labels themselves via test.AddBuiltinLabels.
	testBuiltinLabels := []string{
		fleet.BuiltinLabelNameAllHosts,
		fleet.BuiltinLabelNameMacOS,
		fleet.BuiltinLabelNameUbuntuLinux,
		fleet.BuiltinLabelNameCentOSLinux,
		fleet.BuiltinLabelNameWindows,
		fleet.BuiltinLabelNameRedHatLinux,
		fleet.BuiltinLabelNameAllLinux,
		fleet.BuiltinLabelNameChrome,
	}
	stmt, args, err := sqlx.In(`DELETE FROM labels WHERE name IN (?)`, testBuiltinLabels)
	panicif(err)
	_, err = db.Exec(stmt, args...)
	panicif(err)
	// Resets AUTO_INCREMENT to MAX(id)+1, so label IDs in tests don't depend on the deleted rows.
	_, err = db.Exec(`ALTER TABLE labels AUTO_INCREMENT = 1`)
	panicif(err)

	// Dump schema to dumpfile
	// --set-gtid-purged=OFF omits replication transaction IDs from the dump, making it
	// portable across MySQL servers. This was the default behavior before MySQL 9.5.
	cmd := exec.Command(
		"docker", "compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysqldump", "-u"+testUsername, "-p"+testPassword, "schemadb", "--compact", "--skip-comments", "--set-gtid-purged=OFF",
	)
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	panicif(cmd.Run())

	panicif(os.WriteFile(os.Args[1], stripMigrationStatus(stdoutBuf.Bytes()), 0o655))

	// Migration rows are written one self-contained statement per line, without
	// the auto-increment id, so that concurrent PRs adding migrations only add
	// lines and the file can be merged with git's union driver (see .gitattributes).
	var versions []struct {
		VersionID int64 `db:"version_id"`
		IsApplied bool  `db:"is_applied"`
	}
	panicif(sqlx.Select(sqlx.NewDb(db, "mysql"), &versions,
		`SELECT version_id, is_applied FROM schemadb.migration_status_tables ORDER BY version_id, is_applied`))
	pending := make(map[string]bool, len(versions))
	var ordered []string
	for _, v := range versions {
		isApplied := 0
		if v.IsApplied {
			isApplied = 1
		}
		line := fmt.Sprintf("INSERT INTO `migration_status_tables` (`version_id`, `is_applied`) VALUES (%d, %d);", v.VersionID, isApplied)
		pending[line] = true
		ordered = append(ordered, line)
	}
	// Keep the existing line order: a union merge appends lines in arbitrary
	// order, and re-sorting would make the CI up-to-date check fail after it.
	var sb strings.Builder
	if existing, err := os.ReadFile(os.Args[2]); err == nil {
		for line := range strings.Lines(string(existing)) {
			line = strings.TrimRight(line, "\n")
			if pending[line] {
				sb.WriteString(line + "\n")
				delete(pending, line)
			}
		}
	}
	for _, line := range ordered {
		if pending[line] {
			sb.WriteString(line + "\n")
		}
	}
	panicif(os.WriteFile(os.Args[2], []byte(sb.String()), 0o655))
}

var (
	migrationStatusInsertRe  = regexp.MustCompile("(?m)^INSERT INTO `migration_status_tables` VALUES .*;\n")
	migrationStatusCreateRe  = regexp.MustCompile("(?s)CREATE TABLE `migration_status_tables` \\(.*?\\) ENGINE=[^;]*;")
	migrationStatusAutoIncRe = regexp.MustCompile(` AUTO_INCREMENT=\d+`)
)

// stripMigrationStatus removes the migration_status_tables rows and its
// AUTO_INCREMENT counter from the dump, as they change with every migration.
func stripMigrationStatus(dump []byte) []byte {
	if !migrationStatusInsertRe.Match(dump) || !migrationStatusCreateRe.Match(dump) {
		panic("migration_status_tables not found in dump")
	}
	dump = migrationStatusInsertRe.ReplaceAll(dump, nil)
	return migrationStatusCreateRe.ReplaceAllFunc(dump, func(b []byte) []byte {
		return migrationStatusAutoIncRe.ReplaceAll(b, nil)
	})
}
