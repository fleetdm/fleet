package datastore

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func setupMySQLGORM(t *testing.T) kolide.Datastore {
	user := "kolide"
	password := "kolide"
	dbName := "kolide"

	// try container first
	host := os.Getenv("MYSQL_PORT_3306_TCP_ADDR")
	if host == "" {
		host = "127.0.0.1"
	}
	host = fmt.Sprintf("%s:3306", host)

	conn := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, dbName)
	db, err := New("gorm-mysql", conn, LimitAttempts(1))
	assert.Nil(t, err)

	backend := db.(gormDB)
	err = backend.Migrate()
	assert.Nil(t, err)

	return db
}

func teardownMySQLGORM(t *testing.T, db kolide.Datastore) {
	err := db.Drop()
	assert.Nil(t, err)
}

func TestEnrollHostMySQLGORM(t *testing.T) {
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		t.SkipNow()
	}
	db := setupMySQLGORM(t)
	defer teardownMySQLGORM(t, db)

	testEnrollHost(t, db)
}

func TestAuthenticateHostMySQLGORM(t *testing.T) {
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		t.SkipNow()
	}
	db := setup(t)
	defer teardown(t, db)

	testAuthenticateHost(t, db)
}

func TestUserByIDMySQLGORM(t *testing.T) {
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		t.SkipNow()
	}
	db := setup(t)
	defer teardown(t, db)

	testUserByID(t, db)
}

// TestCreateUser tests the UserStore interface
// this test uses the MySQL GORM backend
func TestCreateUserMySQLGORM(t *testing.T) {
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		t.SkipNow()
	}

	db := setupMySQLGORM(t)
	defer teardownMySQLGORM(t, db)

	testCreateUser(t, db)
}

func TestSaveUserMySQLGORM(t *testing.T) {
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		t.SkipNow()
	}

	db := setupMySQLGORM(t)
	defer teardownMySQLGORM(t, db)

	testSaveUser(t, db)
}

func TestSaveQuery(t *testing.T) {
	ds := setup(t)
	testSaveQuery(t, ds)
}

func TestDeleteQuery(t *testing.T) {
	ds := setup(t)
	testDeleteQuery(t, ds)
}

func TestDeletePack(t *testing.T) {
	ds := setup(t)
	testDeletePack(t, ds)
}

func TestAddAndRemoveQueryFromPack(t *testing.T) {
	ds := setup(t)
	testAddAndRemoveQueryFromPack(t, ds)
}

func TestManagingLabelsOnPacks(t *testing.T) {
	ds := setup(t)
	testManagingLabelsOnPacks(t, ds)
}
