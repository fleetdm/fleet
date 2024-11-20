package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
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
		{"AggregateEnrollSecretPerTeam", testAggregateEnrollSecretPerTeam},
		{"Defaults", testAppConfigDefaults},
		{"Backwards Compatibility", testAppConfigBackwardsCompatibility},
		{"GetConfigEnableDiskEncryption", testGetConfigEnableDiskEncryption},
		{"IsEnrollSecretAvailable", testIsEnrollSecretAvailable},
		{"NDESSCEPProxyPassword", testNDESSCEPProxyPassword},
		{"YaraRulesRoundtrip", testYaraRulesRoundtrip},
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
	info2.Features.EnableSoftwareInventory = true

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
	assert.True(t, verify.SSOEnabled) // SSO stays enabled for user even when globally disabled
}

func testAppConfigAdditionalQueries(t *testing.T, ds *Datastore) {
	additional := ptr.RawMessage(json.RawMessage("not valid json"))
	info := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName:    "Test",
			OrgLogoURL: "localhost:8080/logo.png",
		},
		Features: fleet.Features{
			AdditionalQueries: additional,
		},
	}

	_, err := ds.NewAppConfig(context.Background(), info)
	require.Error(t, err)

	info.Features.AdditionalQueries = ptr.RawMessage(json.RawMessage(`{}`))
	info, err = ds.NewAppConfig(context.Background(), info)
	require.NoError(t, err)

	info.Features.AdditionalQueries = ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`))
	info, err = ds.NewAppConfig(context.Background(), info)
	require.NoError(t, err)
	rawJson := *info.Features.AdditionalQueries
	assert.JSONEq(t, `{"foo":"bar"}`, string(rawJson))
}

func testAppConfigEnrollSecrets(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	defer TruncateTables(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	secret, err := ds.VerifyEnrollSecret(ctx, "missing")
	assert.Error(t, err)
	assert.Nil(t, secret)

	err = ds.ApplyEnrollSecrets(ctx, &team1.ID,
		[]*fleet.EnrollSecret{
			{Secret: "one_secret", TeamID: &team1.ID},
		},
	)
	assert.NoError(t, err)

	// keep the created-at timestamp of the team's secret
	t1Secrets, err := ds.GetEnrollSecrets(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, t1Secrets, 1)
	t1CreatedAt := t1Secrets[0].CreatedAt

	secret, err = ds.VerifyEnrollSecret(ctx, "one")
	assert.Error(t, err, "secret should not match")
	assert.Nil(t, secret, "secret should be nil")
	secret, err = ds.VerifyEnrollSecret(ctx, "one_secret")
	assert.NoError(t, err)
	assert.Equal(t, &team1.ID, secret.TeamID)
	secret, err = ds.VerifyEnrollSecret(ctx, "two_secret")
	assert.Error(t, err)
	assert.Nil(t, secret)

	// Add global secret
	err = ds.ApplyEnrollSecrets(ctx, nil,
		[]*fleet.EnrollSecret{
			{Secret: "two_secret"},
		},
	)
	assert.NoError(t, err)

	// keep the created-at timestamp of the global secret
	globalSecrets, err := ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, globalSecrets, 1)
	globalCreatedAt := globalSecrets[0].CreatedAt

	secret, err = ds.VerifyEnrollSecret(ctx, "one_secret")
	assert.NoError(t, err)
	assert.Equal(t, &team1.ID, secret.TeamID)
	secret, err = ds.VerifyEnrollSecret(ctx, "two_secret")
	assert.NoError(t, err)
	assert.Equal(t, (*uint)(nil), secret.TeamID)

	// ensure mysql returns a distinct timestamp
	time.Sleep(time.Second)

	// apply a new team secret, keeping the old one
	err = ds.ApplyEnrollSecrets(ctx, &team1.ID,
		[]*fleet.EnrollSecret{
			{Secret: "one_secret", TeamID: &team1.ID},
			{Secret: "three_secret", TeamID: &team1.ID},
		},
	)
	require.NoError(t, err)

	// apply a new global secret, keeping the old one
	err = ds.ApplyEnrollSecrets(ctx, nil,
		[]*fleet.EnrollSecret{
			{Secret: "two_secret"},
			{Secret: "four_secret"},
		},
	)
	require.NoError(t, err)

	// check that old secrets kept their original created-at timestamp
	t1Secrets, err = ds.GetEnrollSecrets(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, t1Secrets, 2)
	sort.Slice(t1Secrets, func(i, j int) bool {
		l, r := t1Secrets[i], t1Secrets[j]
		return l.CreatedAt.Before(r.CreatedAt)
	})
	assert.Equal(t, t1CreatedAt, t1Secrets[0].CreatedAt)
	assert.True(t, t1Secrets[1].CreatedAt.After(t1CreatedAt))
	assert.Equal(t, "one_secret", t1Secrets[0].Secret)
	assert.Equal(t, "three_secret", t1Secrets[1].Secret)

	globalSecrets, err = ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, globalSecrets, 2)
	sort.Slice(globalSecrets, func(i, j int) bool {
		l, r := globalSecrets[i], globalSecrets[j]
		return l.CreatedAt.Before(r.CreatedAt)
	})
	assert.Equal(t, globalCreatedAt, globalSecrets[0].CreatedAt)
	assert.True(t, globalSecrets[1].CreatedAt.After(globalCreatedAt))
	assert.Equal(t, "two_secret", globalSecrets[0].Secret)
	assert.Equal(t, "four_secret", globalSecrets[1].Secret)

	// Remove team secrets
	err = ds.ApplyEnrollSecrets(ctx, &team1.ID, []*fleet.EnrollSecret{})
	assert.NoError(t, err)
	secret, err = ds.VerifyEnrollSecret(ctx, "one_secret")
	assert.Error(t, err)
	assert.Nil(t, secret)
	secret, err = ds.VerifyEnrollSecret(ctx, "two_secret")
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

	const secret = "one_secret"
	expectedSecrets := []*fleet.EnrollSecret{
		{Secret: secret},
	}
	err = ds.ApplyEnrollSecrets(context.Background(), &team1.ID, expectedSecrets)
	require.NoError(t, err)

	// Same secret at global level should not be allowed
	err = ds.ApplyEnrollSecrets(context.Background(), nil, expectedSecrets)
	require.Error(t, err)
	assert.False(t, strings.Contains(err.Error(), secret), fmt.Sprintf("error should not contain secret in plaintext: %s", err.Error()))
}

func testAppConfigDefaults(t *testing.T, ds *Datastore) {
	insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
	_, err := ds.writer(context.Background()).Exec(insertAppConfigQuery, `{}`)
	require.NoError(t, err)

	ac, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, 24*time.Hour, ac.WebhookSettings.Interval.Duration)
	require.False(t, ac.WebhookSettings.HostStatusWebhook.Enable)
	require.True(t, ac.Features.EnableHostUsers)
	require.False(t, ac.Features.EnableSoftwareInventory)

	_, err = ds.writer(context.Background()).Exec(
		insertAppConfigQuery,
		`{"webhook_settings": {"interval": "12h"}, "features": {"enable_host_users": false}}`,
	)
	require.NoError(t, err)

	ac, err = ds.AppConfig(context.Background())
	require.NoError(t, err)

	require.Equal(t, 12*time.Hour, ac.WebhookSettings.Interval.Duration)
	require.False(t, ac.Features.EnableHostUsers)
	require.False(t, ac.Features.EnableSoftwareInventory)
}

func testAppConfigBackwardsCompatibility(t *testing.T, ds *Datastore) {
	insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
	_, err := ds.writer(context.Background()).Exec(insertAppConfigQuery, `
{
  "host_settings": {
    "enable_host_users": false,
    "enable_software_inventory": true,
    "additional_queries": { "foo": "bar" }
  }
}`)

	require.NoError(t, err)

	ac, err := ds.AppConfig(context.Background())
	require.NoError(t, err)

	require.False(t, ac.Features.EnableHostUsers)
	require.True(t, ac.Features.EnableSoftwareInventory)
	require.NotNil(t, ac.Features.AdditionalQueries)
}

func testAggregateEnrollSecretPerTeam(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	defer TruncateTables(t, ds)

	// Add global secret
	err := ds.ApplyEnrollSecrets(ctx, nil,
		[]*fleet.EnrollSecret{
			{Secret: "global_secret"},
		},
	)
	assert.NoError(t, err)

	// a team with two enroll secrets
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = ds.ApplyEnrollSecrets(context.Background(), &team1.ID, []*fleet.EnrollSecret{
		{Secret: "team_1_secret_1"},
		{Secret: "team_1_secret_2"},
	})
	require.NoError(t, err)

	// a team with no enroll secrets
	_, err = ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// a team with a single enroll secret
	team3, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team3"})
	require.NoError(t, err)
	err = ds.ApplyEnrollSecrets(context.Background(), &team3.ID, []*fleet.EnrollSecret{
		{Secret: "team_3_secret_1"},
	})
	require.NoError(t, err)

	agg, err := ds.AggregateEnrollSecretPerTeam(ctx)
	require.NoError(t, err)

	require.Len(t, agg, 4)
	sort.Slice(agg, func(i, j int) bool {
		if agg[i].TeamID == nil {
			return true
		}

		if agg[j].TeamID == nil {
			return false
		}
		return *agg[i].TeamID < *agg[j].TeamID
	})

	require.ElementsMatch(t, []*fleet.EnrollSecret{
		{TeamID: nil, Secret: "global_secret"},
		{TeamID: ptr.Uint(1), Secret: "team_1_secret_1"},
		{TeamID: ptr.Uint(2), Secret: ""},
		{TeamID: ptr.Uint(3), Secret: "team_3_secret_1"},
	}, agg)
}

func testGetConfigEnableDiskEncryption(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	defer TruncateTables(t, ds)

	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	require.False(t, ac.MDM.EnableDiskEncryption.Value)

	enabled, err := ds.getConfigEnableDiskEncryption(ctx, nil)
	require.NoError(t, err)
	require.False(t, enabled)

	// Enable disk encryption for no team
	ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)
	ac, err = ds.AppConfig(ctx)
	require.NoError(t, err)
	require.True(t, ac.MDM.EnableDiskEncryption.Value)

	enabled, err = ds.getConfigEnableDiskEncryption(ctx, nil)
	require.NoError(t, err)
	require.True(t, enabled)

	// Create team
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	tm, err := ds.Team(ctx, team1.ID)
	require.NoError(t, err)
	require.NotNil(t, tm)
	require.False(t, tm.Config.MDM.EnableDiskEncryption)

	enabled, err = ds.getConfigEnableDiskEncryption(ctx, &team1.ID)
	require.NoError(t, err)
	require.False(t, enabled)

	// Enable disk encryption for the team
	tm.Config.MDM.EnableDiskEncryption = true
	tm, err = ds.SaveTeam(ctx, tm)
	require.NoError(t, err)
	require.NotNil(t, tm)
	require.True(t, tm.Config.MDM.EnableDiskEncryption)
}

func testIsEnrollSecretAvailable(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	defer TruncateTables(t, ds)

	// Create teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// Populate enroll secrets
	require.NoError(t, ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "globalSecret"}}))
	require.NoError(t, ds.ApplyEnrollSecrets(ctx, &team1.ID, []*fleet.EnrollSecret{{Secret: "teamSecret"}}))

	tests := []struct {
		secret          string
		newResult       bool
		globalResult    bool
		teamResult      bool
		otherTeamResult bool
	}{
		{"mySecret", true, true, true, true},
		{"globalSecret", false, true, false, false},
		{"teamSecret", false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(
			tt.secret, func(t *testing.T) {
				// Check if enroll secret is available
				available, err := ds.IsEnrollSecretAvailable(ctx, tt.secret, true, nil)
				require.NoError(t, err)
				require.Equal(t, tt.newResult, available)
				available, err = ds.IsEnrollSecretAvailable(ctx, tt.secret, true, &team2.ID)
				require.NoError(t, err)
				require.Equal(t, tt.newResult, available)
				available, err = ds.IsEnrollSecretAvailable(ctx, tt.secret, false, nil)
				require.NoError(t, err)
				require.Equal(t, tt.globalResult, available)
				available, err = ds.IsEnrollSecretAvailable(ctx, tt.secret, false, &team1.ID)
				require.NoError(t, err)
				require.Equal(t, tt.teamResult, available)
				available, err = ds.IsEnrollSecretAvailable(ctx, tt.secret, false, &team2.ID)
				require.NoError(t, err)
				require.Equal(t, tt.otherTeamResult, available)
			},
		)
	}
}

func testNDESSCEPProxyPassword(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ctx = ctxdb.BypassCachedMysql(ctx, true)
	defer TruncateTables(t, ds)

	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)

	adminURL := "https://localhost:8080/mscep_admin/"
	username := "admin"
	url := "https://localhost:8080/mscep/mscep.dll"
	password := "password"

	ac.Integrations.NDESSCEPProxy = optjson.Any[fleet.NDESSCEPProxyIntegration]{
		Valid: true,
		Set:   true,
		Value: fleet.NDESSCEPProxyIntegration{
			AdminURL: adminURL,
			Username: username,
			Password: password,
			URL:      url,
		},
	}

	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)

	checkProxyConfig := func() {
		result, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		require.NotNil(t, result.Integrations.NDESSCEPProxy)
		assert.Equal(t, url, result.Integrations.NDESSCEPProxy.Value.URL)
		assert.Equal(t, adminURL, result.Integrations.NDESSCEPProxy.Value.AdminURL)
		assert.Equal(t, username, result.Integrations.NDESSCEPProxy.Value.Username)
		assert.Equal(t, fleet.MaskedPassword, result.Integrations.NDESSCEPProxy.Value.Password)
	}

	checkProxyConfig()

	checkPassword := func() {
		assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetNDESPassword}, nil)
		require.NoError(t, err)
		require.Len(t, assets, 1)
		assert.Equal(t, password, string(assets[fleet.MDMAssetNDESPassword].Value))
	}
	checkPassword()

	// Set password to masked password -- should not update
	ac.Integrations.NDESSCEPProxy.Value.Password = fleet.MaskedPassword
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)
	checkProxyConfig()
	checkPassword()

	// Set password to empty -- password should not update
	url = "https://newurl.com"
	ac.Integrations.NDESSCEPProxy.Value.Password = ""
	ac.Integrations.NDESSCEPProxy.Value.URL = url
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)
	checkProxyConfig()
	checkPassword()

	// Set password to a new value
	password = "newpassword"
	ac.Integrations.NDESSCEPProxy.Value.Password = password
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)
	checkProxyConfig()
	checkPassword()
}

func testYaraRulesRoundtrip(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	defer TruncateTables(t, ds)

	// Empty insert
	expectedRules := []fleet.YaraRule{}
	err := ds.ApplyYaraRules(ctx, expectedRules)
	require.NoError(t, err)
	rules, err := ds.GetYaraRules(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedRules, rules)

	// Insert values
	expectedRules = []fleet.YaraRule{
		{
			Name: "wildcard.yar",
			Contents: `rule WildcardExample
{
    strings:
        $hex_string = { E2 34 ?? C8 A? FB }

    condition:
        $hex_string
}`,
		},
		{
			Name: "jump.yar",
			Contents: `rule JumpExample
{
    strings:
        $hex_string = { F4 23 [4-6] 62 B4 }

    condition:
        $hex_string
}`,
		},
	}
	err = ds.ApplyYaraRules(ctx, expectedRules)
	require.NoError(t, err)
	rules, err = ds.GetYaraRules(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedRules, rules)

	rule, err := ds.YaraRuleByName(ctx, expectedRules[0].Name)
	require.NoError(t, err)
	assert.Equal(t, &expectedRules[0], rule)
	rule, err = ds.YaraRuleByName(ctx, expectedRules[1].Name)
	require.NoError(t, err)
	assert.Equal(t, &expectedRules[1], rule)

	// Update rules
	expectedRules = []fleet.YaraRule{
		{
			Name: "wildcard.yar",
			Contents: `rule WildcardExample
{
    strings:
        $hex_string = { E2 34 ?? C8 A? FB }

    condition:
        $hex_string
}`,
		},
		{
			Name: "jump-modified.yar",
			Contents: `rule JumpExample
{
    strings:
        $hex_string = true

    condition:
        $hex_string
}`,
		},
	}
	err = ds.ApplyYaraRules(ctx, expectedRules)
	require.NoError(t, err)
	rules, err = ds.GetYaraRules(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedRules, rules)

	rule, err = ds.YaraRuleByName(ctx, expectedRules[0].Name)
	require.NoError(t, err)
	assert.Equal(t, &expectedRules[0], rule)
	rule, err = ds.YaraRuleByName(ctx, expectedRules[1].Name)
	require.NoError(t, err)
	assert.Equal(t, &expectedRules[1], rule)

	// Clear rules
	expectedRules = []fleet.YaraRule{}
	err = ds.ApplyYaraRules(ctx, expectedRules)
	require.NoError(t, err)
	rules, err = ds.GetYaraRules(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedRules, rules)

	// Get rule that doesn't exist
	_, err = ds.YaraRuleByName(ctx, "wildcard.yar")
	require.Error(t, err)
}
