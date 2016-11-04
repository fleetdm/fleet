package datastore

import (
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func testOrgInfo(t *testing.T, ds kolide.Datastore) {
	info := &kolide.AppConfig{
		OrgName:    "Kolide",
		OrgLogoURL: "localhost:8080/logo.png",
	}

	info, err := ds.NewAppConfig(info)
	assert.Nil(t, err)

	info2, err := ds.AppConfig()
	assert.Nil(t, err)
	assert.Equal(t, info2.OrgName, info.OrgName)

	info2.OrgName = "koolide"
	err = ds.SaveAppConfig(info2)
	assert.Nil(t, err)

	info3, err := ds.AppConfig()
	assert.Nil(t, err)
	assert.Equal(t, info3.OrgName, info2.OrgName)

	info4, err := ds.NewAppConfig(info3)
	assert.Nil(t, err)
	assert.Equal(t, info4.OrgName, info3.OrgName)
}
