package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/go-kit/log"
)

const (
	testUsername = "root"
	testPassword = "toor"
	testAddress  = "localhost:3307"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

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
	ds, err := mysql.New(config, clock.NewMockClock(), mysql.Logger(log.NewNopLogger()), mysql.LimitAttempts(1))
	panicif(err)
	defer ds.Close()
	androidDs := mysql.NewAndroidDS(ds)
	panicif(androidDs.MigrateTables(context.Background()))

	// Set created_at/updated_at for migrations and app_config_json to prevent the schema from being changed every time
	// This schema is to test anyway
	fixedDate := time.Date(2020, 01, 01, 01, 01, 01, 01, time.UTC)
	_, err = db.Exec(`USE schemadb`)
	panicif(err)
	_, err = db.Exec(`UPDATE android_migration_status SET tstamp = ?`, fixedDate)
	panicif(err)

	// Dump schema to dumpfile
	cmd := exec.Command(
		"docker", "compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysqldump", "-u"+testUsername, "-p"+testPassword, "schemadb", "--compact", "--skip-comments",
	)
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	panicif(cmd.Run())

	panicif(os.WriteFile(os.Args[1], stdoutBuf.Bytes(), 0o644))
}
