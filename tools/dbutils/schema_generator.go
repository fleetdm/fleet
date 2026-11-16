package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
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

// main requires 1 argument:
// 1. Path to dumpfile
func main() {
	if len(os.Args) != 2 {
		panic("not enough arguments")
	}
	fmt.Println("dumping schema to", os.Args[1])

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
	var migrationStart string
	panicif(db.QueryRow(`SELECT NOW(6)`).Scan(&migrationStart))
	panicif(ds.MigrateTables(context.Background()))

	// Set created_at/updated_at for app_config_json to prevent the schema from being changed every time
	// This schema is to test anyway
	fixedDate := time.Date(2020, 01, 01, 01, 01, 01, 01, time.UTC)
	_, err = db.Exec(`USE schemadb`)
	panicif(err)
	_, err = db.Exec(`UPDATE app_config_json SET created_at = ?, updated_at = ?`, fixedDate, fixedDate)
	panicif(err)
	pinMigrationTimestamps(db, migrationStart, fixedDate)

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

	// Tests derive migration_status_tables rows from the migration filenames
	// (see testing_utils.LoadDefaultSchema), so make sure they match what ran.
	var applied []int64
	panicif(sqlx.Select(sqlx.NewDb(db, "mysql"), &applied,
		`SELECT version_id FROM schemadb.migration_status_tables WHERE version_id > 0 AND is_applied = 1 ORDER BY version_id`))
	fromFiles, err := testing_utils.MigrationVersions(filepath.Join(filepath.Dir(os.Args[1]), "migrations", "tables"))
	panicif(err)
	if !slices.Equal(applied, fromFiles) {
		panic(fmt.Sprintf("applied migrations %v do not match migration filenames %v", applied, fromFiles))
	}
}

// pinMigrationTimestamps replaces every timestamp/datetime value written while the migrations ran
// (DEFAULT CURRENT_TIMESTAMP, ON UPDATE CURRENT_TIMESTAMP, NOW()) with fixedDate, so the dump
// doesn't change on every run. Literal dates seeded by migrations are older and left alone.
func pinMigrationTimestamps(db *sql.DB, since string, fixedDate time.Time) {
	var cols []struct {
		Table  string `db:"table_name"`
		Column string `db:"column_name"`
	}
	panicif(sqlx.Select(sqlx.NewDb(db, "mysql"), &cols, `
		SELECT c.TABLE_NAME AS table_name, c.COLUMN_NAME AS column_name
		FROM information_schema.COLUMNS c
		JOIN information_schema.TABLES t ON t.TABLE_SCHEMA = c.TABLE_SCHEMA AND t.TABLE_NAME = c.TABLE_NAME
		WHERE c.TABLE_SCHEMA = 'schemadb' AND t.TABLE_TYPE = 'BASE TABLE'
			AND c.TABLE_NAME != 'migration_status_tables' AND c.DATA_TYPE IN ('timestamp', 'datetime')
		ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION`))

	byTable := map[string][]string{}
	var tables []string
	for _, c := range cols {
		if _, ok := byTable[c.Table]; !ok {
			tables = append(tables, c.Table)
		}
		byTable[c.Table] = append(byTable[c.Table], c.Column)
	}
	for _, table := range tables {
		// Set all columns in one statement: explicitly assigning a column stops
		// ON UPDATE CURRENT_TIMESTAMP from bumping it again.
		var sets, conds []string
		var args []any
		for _, col := range byTable[table] {
			sets = append(sets, fmt.Sprintf("`%[1]s` = IF(`%[1]s` >= ?, ?, `%[1]s`)", col))
			args = append(args, since, fixedDate)
			conds = append(conds, fmt.Sprintf("`%s` >= ?", col))
		}
		for range byTable[table] {
			args = append(args, since)
		}
		//nolint:gosec // identifiers come from information_schema
		stmt := fmt.Sprintf("UPDATE `%s` SET %s WHERE %s", table, strings.Join(sets, ", "), strings.Join(conds, " OR "))
		_, err := db.Exec(stmt, args...)
		panicif(err)
	}
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
