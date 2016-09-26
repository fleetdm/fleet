package datastore

import (
	"os"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func TestOrgInfo(t *testing.T) {
	var ds kolide.Datastore
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		ds = setup(t)
	} else {
		ds = setupMySQLGORM(t)
		defer teardownMySQLGORM(t, ds)
	}

	testOrgInfo(t, ds)
}

func testOrgInfo(t *testing.T, db kolide.Datastore) {
	info := &kolide.OrgInfo{
		OrgName:    "Kolide",
		OrgLogoURL: "localhost:8080/logo.png",
	}

	info, err := db.NewOrgInfo(info)
	assert.Nil(t, err)
	assert.Equal(t, info.ID, uint(1))

	info2, err := db.OrgInfo()
	assert.Nil(t, err)
	assert.Equal(t, info2.ID, uint(1))
	assert.Equal(t, info2.OrgName, info.OrgName)

	info2.OrgName = "koolide"
	err = db.SaveOrgInfo(info2)
	assert.Nil(t, err)

	info3, err := db.OrgInfo()
	assert.Nil(t, err)
	assert.Equal(t, info3.OrgName, info2.OrgName)

	info4, err := db.NewOrgInfo(info3)
	assert.Nil(t, err)
	assert.Equal(t, info4.ID, uint(1))
}
