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
		OrgInfo: &fleet.OrgInfo{
			OrgName:    ptr.String("Test"),
			OrgLogoURL: ptr.String("localhost:8080/logo.png"),
		},
	}

	info, err := ds.NewAppConfig(info)
	assert.Nil(t, err)
	require.NotNil(t, info)

	info2, err := ds.AppConfig()
	require.Nil(t, err)
	assert.Equal(t, *info2.OrgInfo.OrgName, *info.OrgInfo.OrgName)
	smtpConfigured := info2.GetBool("smtp_settings.configured")
	assert.False(t, smtpConfigured)

	info2.OrgInfo.OrgName = ptr.String("testss")
	info2.SMTPSettings = &fleet.SMTPSettings{}
	info2.SMTPSettings.SMTPDomain = ptr.String("foo")
	info2.SMTPSettings.SMTPConfigured = ptr.Bool(true)
	info2.SMTPSettings.SMTPSenderAddress = ptr.String("123")
	info2.SMTPSettings.SMTPServer = ptr.String("server")
	info2.SMTPSettings.SMTPPort = ptr.Uint(100)
	info2.SMTPSettings.SMTPAuthenticationType = ptr.String(fleet.AuthTypeNameUserNamePassword)
	info2.SMTPSettings.SMTPUserName = ptr.String("username")
	info2.SMTPSettings.SMTPPassword = ptr.String("password")
	info2.SMTPSettings.SMTPEnableTLS = ptr.Bool(false)
	info2.SMTPSettings.SMTPAuthenticationMethod = ptr.String(fleet.AuthMethodNameCramMD5)
	info2.SMTPSettings.SMTPVerifySSLCerts = ptr.Bool(true)
	info2.SMTPSettings.SMTPEnableStartTLS = ptr.Bool(true)
	info2.SSOSettings = &fleet.SSOSettings{}
	info2.SSOSettings.EnableSSO = ptr.Bool(true)
	info2.SSOSettings.EntityID = ptr.String("test")
	info2.SSOSettings.MetadataURL = ptr.String("https://idp.com/metadata.xml")
	info2.SSOSettings.IssuerURI = ptr.String("https://idp.issuer.com")
	info2.SSOSettings.IDPName = ptr.String("My IDP")
	info2.HostSettings = &fleet.HostSettings{}
	info2.HostSettings.EnableSoftwareInventory = ptr.Bool(true)

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

	info4.SSOSettings.EnableSSO = ptr.Bool(false)
	err = ds.SaveAppConfig(info4)
	assert.Nil(t, err)

	verify, err = ds.UserByEmail(email)
	assert.Nil(t, err)
	assert.False(t, verify.SSOEnabled)
}

func TestAdditionalQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	additional := json.RawMessage("not valid json")
	info := &fleet.AppConfig{
		OrgInfo: &fleet.OrgInfo{
			OrgName:    ptr.String("Test"),
			OrgLogoURL: ptr.String("localhost:8080/logo.png"),
		},
		HostSettings: &fleet.HostSettings{
			AdditionalQueries: &additional,
		},
	}

	_, err := ds.NewAppConfig(info)
	require.Error(t, err)

	additional = json.RawMessage(`{}`)
	info, err = ds.NewAppConfig(info)
	require.NoError(t, err)

	additional = json.RawMessage(`{"foo": "bar"}`)
	info, err = ds.NewAppConfig(info)
	require.NoError(t, err)
	rawJson := info.GetJSON("host_settings.additional_queries")
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
