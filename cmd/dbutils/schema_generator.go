package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/go-kit/kit/log"
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
	_, err = db.Exec("DROP DATABASE IF EXISTS TestMigrations; CREATE DATABASE TestMigrations;")
	panicif(err)

	// Create a datastore client in order to run migrations as usual
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Address:  testAddress,
		Database: "TestMigrations",
	}
	ds, err := mysql.New(config, clock.NewMockClock(), mysql.Logger(log.NewNopLogger()), mysql.LimitAttempts(1))
	panicif(err)
	defer ds.Close()
	panicif(ds.MigrateTables())

	// Dump schema to dumpfile
	cmd := exec.Command(
		"docker-compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysqldump", "-u"+testUsername, "-p"+testPassword, "TestMigrations", "--compact", "--skip-comments",
	)
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	panicif(cmd.Run())

	panicif(ioutil.WriteFile(os.Args[1], stdoutBuf.Bytes(), 0o655))
}
