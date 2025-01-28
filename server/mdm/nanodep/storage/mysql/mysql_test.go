package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage/storagetest"
	_ "github.com/go-sql-driver/mysql"
)

func TestMySQLStorage(t *testing.T) {
	testDSN := os.Getenv("NANODEP_MYSQL_STORAGE_TEST")
	if testDSN == "" {
		t.Skip("NANODEP_MYSQL_STORAGE_TEST not set")
	}

	storagetest.Run(t, func(t *testing.T) storage.AllDEPStorage {
		dbName := initTestDB(t)
		testDSN := fmt.Sprintf("nanodep:insecure@tcp(localhost:4242)/%s?charset=utf8mb4&loc=UTC&parseTime=true", dbName)
		s, err := New(WithDSN(testDSN))
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}

// initTestDB clears any existing data from the database.
func initTestDB(t *testing.T) string {
	rootDSN := "root:toor@tcp(localhost:4242)/?charset=utf8mb4&loc=UTC"
	db, err := sql.Open("mysql", rootDSN)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	for {
		err := db.PingContext(ctx)
		if err == nil {
			break
		}
		t.Logf("failed to connect: %s, retrying connection", err)
		select {
		case <-time.After(1 * time.Second):
			// OK, continue.
		case <-ctx.Done():
			t.Fatal("timeout connecting to MySQL")
		}
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	dbName := dbName(t)
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(fmt.Sprintf("USE %s;", dbName))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(Schema)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO 'nanodep';", dbName))
	if err != nil {
		t.Fatal(err)
	}
	return dbName
}

func dbName(t *testing.T) string {
	return strings.ReplaceAll(strings.ReplaceAll(t.Name(), "/", "_"), "-", "_")
}
