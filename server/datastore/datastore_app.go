package datastore

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
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
	info2.EnableSSO = true
	info2.EntityID = "kolide"
	info2.MetadataURL = "https://idp.com/metadata.xml"
	info2.IssuerURI = "https://idp.issuer.com"
	info2.IDPName = "My IDP"

	err = ds.SaveAppConfig(info2)
	require.Nil(t, err)

	info3, err := ds.AppConfig()
	require.Nil(t, err)
	assert.Equal(t, info2, info3)

	info4, err := ds.NewAppConfig(info3)
	assert.Nil(t, err)
	assert.Equal(t, info3, info4)
}

func testAdditionalQueries(t *testing.T, ds kolide.Datastore) {
	additional := json.RawMessage("not valid json")
	info := &kolide.AppConfig{
		OrgName:           "Kolide",
		OrgLogoURL:        "localhost:8080/logo.png",
		AdditionalQueries: &additional,
	}

	_, err := ds.NewAppConfig(info)
	assert.NotNil(t, err)

	additional = json.RawMessage(`{}`)
	info, err = ds.NewAppConfig(info)
	assert.Nil(t, err)

	additional = json.RawMessage(`{"foo": "bar"}`)
	info, err = ds.NewAppConfig(info)
	assert.Nil(t, err)
	assert.JSONEq(t, `{"foo":"bar"}`, string(*info.AdditionalQueries))
}

func testEnrollSecrets(t *testing.T, ds kolide.Datastore) {
	name, err := ds.VerifyEnrollSecret("missing")
	assert.Error(t, err)
	assert.Empty(t, name)

	err = ds.ApplyEnrollSecretSpec(
		&kolide.EnrollSecretSpec{
			Secrets: []kolide.EnrollSecret{
				kolide.EnrollSecret{Name: "one", Secret: "one_secret", Active: true},
				kolide.EnrollSecret{Name: "two", Secret: "two_secret", Active: false},
			},
		},
	)
	assert.NoError(t, err)

	name, err = ds.VerifyEnrollSecret("one")
	assert.Error(t, err, "secret should not match")
	assert.Empty(t, name, "secret name should be empty")
	name, err = ds.VerifyEnrollSecret("one_secret")
	assert.NoError(t, err)
	assert.Equal(t, "one", name)
	name, err = ds.VerifyEnrollSecret("two_secret")
	assert.Error(t, err)
	assert.Equal(t, "", name)

	err = ds.ApplyEnrollSecretSpec(
		&kolide.EnrollSecretSpec{
			Secrets: []kolide.EnrollSecret{
				kolide.EnrollSecret{Name: "one", Secret: "one_secret", Active: false},
				kolide.EnrollSecret{Name: "two", Secret: "two_secret", Active: true},
			},
		},
	)
	assert.NoError(t, err)

	name, err = ds.VerifyEnrollSecret("one_secret")
	assert.Error(t, err)
	assert.Equal(t, "", name)
	name, err = ds.VerifyEnrollSecret("two_secret")
	assert.NoError(t, err)
	assert.Equal(t, "two", name)
}

func testEnrollSecretsCaseSensitive(t *testing.T, ds kolide.Datastore) {
	err := ds.ApplyEnrollSecretSpec(
		&kolide.EnrollSecretSpec{
			Secrets: []kolide.EnrollSecret{
				kolide.EnrollSecret{Name: "one", Secret: "one_secret", Active: true},
				kolide.EnrollSecret{Name: "two", Secret: "two_secret", Active: false},
			},
		},
	)
	require.NoError(t, err)

	_, err = ds.VerifyEnrollSecret("one_secret")
	assert.NoError(t, err, "enroll secret should match with matching case")
	_, err = ds.VerifyEnrollSecret("One_Secret")
	assert.Error(t, err, "enroll secret with different case should not verify")
}

func testEnrollSecretRoundtrip(t *testing.T, ds kolide.Datastore) {
	spec, err := ds.GetEnrollSecretSpec()
	require.NoError(t, err)
	assert.Len(t, spec.Secrets, 0)

	expectedSpec := kolide.EnrollSecretSpec{
		Secrets: []kolide.EnrollSecret{
			kolide.EnrollSecret{Name: "one", Secret: "one_secret", Active: false},
			kolide.EnrollSecret{Name: "two", Secret: "two_secret", Active: true},
		},
	}
	err = ds.ApplyEnrollSecretSpec(&expectedSpec)
	require.NoError(t, err)

	spec, err = ds.GetEnrollSecretSpec()
	require.NoError(t, err)
	require.Len(t, spec.Secrets, 2)
	// sort secrets before equality checks to ensure proper order
	sort.Slice(spec.Secrets, func(i, j int) bool { return spec.Secrets[i].Name < spec.Secrets[j].Name })

	assert.Equal(t, "one", spec.Secrets[0].Name)
	assert.Equal(t, "one_secret", spec.Secrets[0].Secret)
	assert.Equal(t, false, spec.Secrets[0].Active)

	assert.Equal(t, "two", spec.Secrets[1].Name)
	assert.Equal(t, "two_secret", spec.Secrets[1].Secret)
	assert.Equal(t, true, spec.Secrets[1].Active)
}
