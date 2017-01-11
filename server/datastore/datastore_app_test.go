package datastore

import (
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOrgInfo(t *testing.T, ds kolide.Datastore) {
	info := &kolide.AppConfig{
		OrgName:    "Kolide",
		OrgLogoURL: "localhost:8080/logo.png",
	}

	info, err := ds.NewAppConfig(info)
	assert.Nil(t, err)
	require.NotNil(t, info)

	info2, err := ds.AppConfig()
	require.Nil(t, err)
	assert.Equal(t, info2.OrgName, info.OrgName)
	assert.False(t, info2.SMTPConfigured)

	info2.OrgName = "koolide"
	info2.SMTPDomain = "foo"
	info2.SMTPConfigured = true
	info2.SMTPSenderAddress = "123"
	info2.SMTPServer = "server"
	info2.SMTPPort = 100
	info2.SMTPAuthenticationType = kolide.AuthTypeUserNamePassword
	info2.SMTPUserName = "username"
	info2.SMTPPassword = "password"
	info2.SMTPEnableTLS = false
	info2.SMTPAuthenticationMethod = kolide.AuthMethodCramMD5
	info2.SMTPVerifySSLCerts = true
	info2.SMTPEnableStartTLS = true
	err = ds.SaveAppConfig(info2)
	require.Nil(t, err)

	info3, err := ds.AppConfig()
	require.Nil(t, err)
	assert.Equal(t, info2, info3)

	info4, err := ds.NewAppConfig(info3)
	assert.Nil(t, err)
	assert.Equal(t, info3, info4)
}
