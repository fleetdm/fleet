package mysql

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfig(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"OrgInfo", testAppConfigOrgInfo},
		{"AdditionalQueries", testAppConfigAdditionalQueries},
		{"EnrollSecrets", testAppConfigEnrollSecrets},
		{"EnrollSecretsCaseSensitive", testAppConfigEnrollSecretsCaseSensitive},
		{"EnrollSecretRoundtrip", testAppConfigEnrollSecretRoundtrip},
		{"EnrollSecretUniqueness", testAppConfigEnrollSecretUniqueness},
		{"Defaults", testAppConfigDefaults},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testAppConfigOrgInfo(t *testing.T, ds *Datastore) {
	info := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName:    "Test",
			OrgLogoURL: "localhost:8080/logo.png",
		},
	}

	info, err := ds.NewAppConfig(context.Background(), info)
	assert.Nil(t, err)
	require.NotNil(t, info)

	// Checking some defaults
	require.Equal(t, 24*time.Hour, info.WebhookSettings.Interval.Duration)
	require.False(t, info.WebhookSettings.HostStatusWebhook.Enable)
	require.NotNil(t, info.AgentOptions)

	info2, err := ds.AppConfig(context.Background())
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

	err = ds.SaveAppConfig(context.Background(), info2)
	require.Nil(t, err)

	info3, err := ds.AppConfig(context.Background())
	require.Nil(t, err)
	assert.Equal(t, info2, info3)

	info4, err := ds.NewAppConfig(context.Background(), info3)
	assert.Nil(t, err)
	assert.Equal(t, info3, info4)

	email := "e@mail.com"
	u := &fleet.User{
		Password:   []byte("pass"),
		Email:      email,
		SSOEnabled: true,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	_, err = ds.NewUser(context.Background(), u)
	assert.Nil(t, err)

	verify, err := ds.UserByEmail(context.Background(), email)
	assert.Nil(t, err)
	assert.True(t, verify.SSOEnabled)

	info4.SSOSettings.EnableSSO = false
	err = ds.SaveAppConfig(context.Background(), info4)
	assert.Nil(t, err)

	verify, err = ds.UserByEmail(context.Background(), email)
	assert.Nil(t, err)
	assert.False(t, verify.SSOEnabled)
}

func testAppConfigAdditionalQueries(t *testing.T, ds *Datastore) {
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

	_, err := ds.NewAppConfig(context.Background(), info)
	require.Error(t, err)

	info.HostSettings.AdditionalQueries = ptr.RawMessage(json.RawMessage(`{}`))
	info, err = ds.NewAppConfig(context.Background(), info)
	require.NoError(t, err)

	info.HostSettings.AdditionalQueries = ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`))
	info, err = ds.NewAppConfig(context.Background(), info)
	require.NoError(t, err)
	rawJson := *info.HostSettings.AdditionalQueries
	assert.JSONEq(t, `{"foo":"bar"}`, string(rawJson))
}

func testAppConfigEnrollSecrets(t *testing.T, ds *Datastore) {
	defer TruncateTables(t, ds)

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	secret, err := ds.VerifyEnrollSecret(context.Background(), "missing")
	assert.Error(t, err)
	assert.Nil(t, secret)

	err = ds.ApplyEnrollSecrets(context.Background(), &team1.ID,
		[]*fleet.EnrollSecret{
			{Secret: "one_secret", TeamID: &team1.ID},
		},
	)
	assert.NoError(t, err)

	secret, err = ds.VerifyEnrollSecret(context.Background(), "one")
	assert.Error(t, err, "secret should not match")
	assert.Nil(t, secret, "secret should be nil")
	secret, err = ds.VerifyEnrollSecret(context.Background(), "one_secret")
	assert.NoError(t, err)
	assert.Equal(t, &team1.ID, secret.TeamID)
	secret, err = ds.VerifyEnrollSecret(context.Background(), "two_secret")
	assert.Error(t, err)
	assert.Nil(t, secret)

	// Add global secret
	err = ds.ApplyEnrollSecrets(
		context.Background(),
		nil,
		[]*fleet.EnrollSecret{
			{Secret: "two_secret"},
		},
	)
	assert.NoError(t, err)

	secret, err = ds.VerifyEnrollSecret(context.Background(), "one_secret")
	assert.NoError(t, err)
	assert.Equal(t, &team1.ID, secret.TeamID)
	secret, err = ds.VerifyEnrollSecret(context.Background(), "two_secret")
	assert.NoError(t, err)
	assert.Equal(t, (*uint)(nil), secret.TeamID)

	// Remove team secret
	err = ds.ApplyEnrollSecrets(context.Background(), &team1.ID, []*fleet.EnrollSecret{})
	assert.NoError(t, err)
	secret, err = ds.VerifyEnrollSecret(context.Background(), "one_secret")
	assert.Error(t, err)
	assert.Nil(t, secret)
	secret, err = ds.VerifyEnrollSecret(context.Background(), "two_secret")
	assert.NoError(t, err)
	assert.Equal(t, (*uint)(nil), secret.TeamID)
}

func testAppConfigEnrollSecretsCaseSensitive(t *testing.T, ds *Datastore) {
	defer TruncateTables(t, ds)

	err := ds.ApplyEnrollSecrets(
		context.Background(),
		nil,
		[]*fleet.EnrollSecret{
			{Secret: "one_secret"},
		},
	)
	require.NoError(t, err)

	_, err = ds.VerifyEnrollSecret(context.Background(), "one_secret")
	assert.NoError(t, err, "enroll secret should match with matching case")
	_, err = ds.VerifyEnrollSecret(context.Background(), "One_Secret")
	assert.Error(t, err, "enroll secret with different case should not verify")
}

func testAppConfigEnrollSecretRoundtrip(t *testing.T, ds *Datastore) {
	defer TruncateTables(t, ds)

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	secrets, err := ds.GetEnrollSecrets(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 0)

	secrets, err = ds.GetEnrollSecrets(context.Background(), &team1.ID)
	require.NoError(t, err)
	assert.Len(t, secrets, 0)

	expectedSecrets := []*fleet.EnrollSecret{
		{Secret: "one_secret"},
		{Secret: "two_secret"},
	}
	err = ds.ApplyEnrollSecrets(context.Background(), &team1.ID, expectedSecrets)
	require.NoError(t, err)

	secrets, err = ds.GetEnrollSecrets(context.Background(), &team1.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 2)
	// sort secrets before equality checks to ensure proper order
	sort.Slice(secrets, func(i, j int) bool { return secrets[i].Secret < secrets[j].Secret })

	assert.Equal(t, "one_secret", secrets[0].Secret)
	assert.Equal(t, "two_secret", secrets[1].Secret)

	expectedSecrets[0].Secret += "_global"
	expectedSecrets[1].Secret += "_global"
	err = ds.ApplyEnrollSecrets(context.Background(), nil, expectedSecrets)
	require.NoError(t, err)

	secrets, err = ds.GetEnrollSecrets(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, secrets, 2)

}

func testAppConfigEnrollSecretUniqueness(t *testing.T, ds *Datastore) {
	defer TruncateTables(t, ds)

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	expectedSecrets := []*fleet.EnrollSecret{
		{Secret: "one_secret"},
	}
	err = ds.ApplyEnrollSecrets(context.Background(), &team1.ID, expectedSecrets)
	require.NoError(t, err)

	// Same secret at global level should not be allowed
	err = ds.ApplyEnrollSecrets(context.Background(), nil, expectedSecrets)
	require.Error(t, err)
}

func testAppConfigDefaults(t *testing.T, ds *Datastore) {
	insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
	_, err := ds.writer.Exec(insertAppConfigQuery, `{}`)
	require.NoError(t, err)

	ac, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, 24*time.Hour, ac.WebhookSettings.Interval.Duration)
	require.False(t, ac.WebhookSettings.HostStatusWebhook.Enable)
	require.True(t, ac.HostSettings.EnableHostUsers)
	require.False(t, ac.HostSettings.EnableSoftwareInventory)

	_, err = ds.writer.Exec(
		insertAppConfigQuery,
		`{"webhook_settings": {"interval": "12h"}, "host_settings": {"enable_host_users": false}}`,
	)
	require.NoError(t, err)

	ac, err = ds.AppConfig(context.Background())
	require.NoError(t, err)

	require.Equal(t, 12*time.Hour, ac.WebhookSettings.Interval.Duration)
	require.False(t, ac.HostSettings.EnableHostUsers)
	require.False(t, ac.HostSettings.EnableSoftwareInventory)
}
