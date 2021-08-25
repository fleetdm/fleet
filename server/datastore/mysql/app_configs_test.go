package mysql

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgInfo(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	info := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName:    "Test",
			OrgLogoURL: "localhost:8080/logo.png",
		},
	}

	info, err := ds.NewAppConfig(info)
	assert.Nil(t, err)
	require.NotNil(t, info)

	info2, err := ds.AppConfig()
	require.Nil(t, err)
	assert.Equal(t, info2.OrgInfo.OrgName, info.OrgInfo.OrgName)
	smtpConfigured := info2.SMTPSettings.SMTPConfigured
	assert.False(t, smtpConfigured)

	info2.OrgInfo.OrgName = "testss"
	info2.SMTPSettings.SMTPDomain = "foo"
	info2.SMTPSettings.SMTPConfigured = true
	info2.SMTPSettings.SMTPSenderAddress = "123"
	info2.SMTPSettings.SMTPServer = "server"
	info2.SMTPSettings.SMTPPort = 100
	info2.SMTPSettings.SMTPAuthenticationType = fleet.AuthTypeNameUserNamePassword
	info2.SMTPSettings.SMTPUserName = "username"
	info2.SMTPSettings.SMTPPassword = "password"
	info2.SMTPSettings.SMTPEnableTLS = false
	info2.SMTPSettings.SMTPAuthenticationMethod = fleet.AuthMethodNameCramMD5
	info2.SMTPSettings.SMTPVerifySSLCerts = true
	info2.SMTPSettings.SMTPEnableStartTLS = true
	info2.SSOSettings.EnableSSO = true
	info2.SSOSettings.EntityID = "test"
	info2.SSOSettings.MetadataURL = "https://idp.com/metadata.xml"
	info2.SSOSettings.IssuerURI = "https://idp.issuer.com"
	info2.SSOSettings.IDPName = "My IDP"
	info2.HostSettings.EnableSoftwareInventory = true

	err = ds.SaveAppConfig(info2)
	require.Nil(t, err)

	info3, err := ds.AppConfig()
	require.Nil(t, err)
	assert.Equal(t, info2, info3)

	info4, err := ds.NewAppConfig(info3)
	assert.Nil(t, err)
	assert.Equal(t, info3, info4)

	email := "e@mail.com"
	u := &fleet.User{
		Password:   []byte("pass"),
		Email:      email,
		SSOEnabled: true,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	_, err = ds.NewUser(u)
	assert.Nil(t, err)

	verify, err := ds.UserByEmail(email)
	assert.Nil(t, err)
	assert.True(t, verify.SSOEnabled)

	info4.SSOSettings.EnableSSO = false
	err = ds.SaveAppConfig(info4)
	assert.Nil(t, err)

	verify, err = ds.UserByEmail(email)
	assert.Nil(t, err)
	assert.False(t, verify.SSOEnabled)
}

func TestAdditionalQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	additional := ptr.RawMessage(json.RawMessage("not valid json"))
	info := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName:    "Test",
			OrgLogoURL: "localhost:8080/logo.png",
		},
		HostSettings: fleet.HostSettings{
			AdditionalQueries: additional,
		},
	}

	_, err := ds.NewAppConfig(info)
	require.Error(t, err)

	info.HostSettings.AdditionalQueries = ptr.RawMessage(json.RawMessage(`{}`))
	info, err = ds.NewAppConfig(info)
	require.NoError(t, err)

	info.HostSettings.AdditionalQueries = ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`))
	info, err = ds.NewAppConfig(info)
	require.NoError(t, err)
	rawJson := *info.HostSettings.AdditionalQueries
	assert.JSONEq(t, `{"foo":"bar"}`, string(rawJson))
}

func TestEnrollSecrets(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	team1, err := ds.NewTeam(&fleet.Team{Name: "team1"})
	require.NoError(t, err)

	secret, err := ds.VerifyEnrollSecret("missing")
	assert.Error(t, err)
	assert.Nil(t, secret)

	err = ds.ApplyEnrollSecrets(&team1.ID,
		[]*fleet.EnrollSecret{
			{Secret: "one_secret", TeamID: &team1.ID},
		},
	)
	assert.NoError(t, err)

	secret, err = ds.VerifyEnrollSecret("one")
	assert.Error(t, err, "secret should not match")
	assert.Nil(t, secret, "secret should be nil")
	secret, err = ds.VerifyEnrollSecret("one_secret")
	assert.NoError(t, err)
	assert.Equal(t, &team1.ID, secret.TeamID)
	secret, err = ds.VerifyEnrollSecret("two_secret")
	assert.Error(t, err)
	assert.Nil(t, secret)

	// Add global secret
	err = ds.ApplyEnrollSecrets(
		nil,
		[]*fleet.EnrollSecret{
			{Secret: "two_secret"},
		},
	)
	assert.NoError(t, err)

	secret, err = ds.VerifyEnrollSecret("one_secret")
	assert.NoError(t, err)
	assert.Equal(t, &team1.ID, secret.TeamID)
	secret, err = ds.VerifyEnrollSecret("two_secret")
	assert.NoError(t, err)
	assert.Equal(t, (*uint)(nil), secret.TeamID)

	// Remove team secret
	err = ds.ApplyEnrollSecrets(&team1.ID, []*fleet.EnrollSecret{})
	assert.NoError(t, err)
	secret, err = ds.VerifyEnrollSecret("one_secret")
	assert.Error(t, err)
	assert.Nil(t, secret)
	secret, err = ds.VerifyEnrollSecret("two_secret")
	assert.NoError(t, err)
	assert.Equal(t, (*uint)(nil), secret.TeamID)
}

func TestEnrollSecretsCaseSensitive(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	err := ds.ApplyEnrollSecrets(
		nil,
		[]*fleet.EnrollSecret{
			{Secret: "one_secret"},
		},
	)
	require.NoError(t, err)

	_, err = ds.VerifyEnrollSecret("one_secret")
	assert.NoError(t, err, "enroll secret should match with matching case")
	_, err = ds.VerifyEnrollSecret("One_Secret")
	assert.Error(t, err, "enroll secret with different case should not verify")
}

func TestEnrollSecretRoundtrip(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	team1, err := ds.NewTeam(&fleet.Team{Name: "team1"})
	require.NoError(t, err)

	secrets, err := ds.GetEnrollSecrets(nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 0)

	secrets, err = ds.GetEnrollSecrets(&team1.ID)
	require.NoError(t, err)
	assert.Len(t, secrets, 0)

	expectedSecrets := []*fleet.EnrollSecret{
		{Secret: "one_secret"},
		{Secret: "two_secret"},
	}
	err = ds.ApplyEnrollSecrets(&team1.ID, expectedSecrets)
	require.NoError(t, err)

	secrets, err = ds.GetEnrollSecrets(&team1.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 2)
	// sort secrets before equality checks to ensure proper order
	sort.Slice(secrets, func(i, j int) bool { return secrets[i].Secret < secrets[j].Secret })

	assert.Equal(t, "one_secret", secrets[0].Secret)
	assert.Equal(t, "two_secret", secrets[1].Secret)

	expectedSecrets[0].Secret += "_global"
	expectedSecrets[1].Secret += "_global"
	err = ds.ApplyEnrollSecrets(nil, expectedSecrets)
	require.NoError(t, err)

	secrets, err = ds.GetEnrollSecrets(nil)
	require.NoError(t, err)
	require.Len(t, secrets, 2)

}

func TestEnrollSecretUniqueness(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	team1, err := ds.NewTeam(&fleet.Team{Name: "team1"})
	require.NoError(t, err)

	expectedSecrets := []*fleet.EnrollSecret{
		{Secret: "one_secret"},
	}
	err = ds.ApplyEnrollSecrets(&team1.ID, expectedSecrets)
	require.NoError(t, err)

	// Same secret at global level should not be allowed
	err = ds.ApplyEnrollSecrets(nil, expectedSecrets)
	require.Error(t, err)
}
