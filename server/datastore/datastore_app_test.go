package datastore

import (
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func testOrgInfo(t *testing.T, ds kolide.Datastore) {
	info := &kolide.OrgInfo{
		OrgName:    "Kolide",
		OrgLogoURL: "localhost:8080/logo.png",
	}

	info, err := ds.NewOrgInfo(info)
	assert.Nil(t, err)

	info2, err := ds.OrgInfo()
	assert.Nil(t, err)
	assert.Equal(t, info2.OrgName, info.OrgName)

	info2.OrgName = "koolide"
	err = ds.SaveOrgInfo(info2)
	assert.Nil(t, err)

	info3, err := ds.OrgInfo()
	assert.Nil(t, err)
	assert.Equal(t, info3.OrgName, info2.OrgName)

	info4, err := ds.NewOrgInfo(info3)
	assert.Nil(t, err)
	assert.Equal(t, info4.OrgName, info3.OrgName)
}
