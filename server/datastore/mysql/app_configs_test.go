package mysql

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"

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

func TestEnrollSecrets(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestEnrollSecretsCaseSensitive(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestEnrollSecretRoundtrip(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestEnrollSecretUniqueness(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestAppConfigDefaults(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestAppConfigError(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	spec := `---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
      options:
        disable_distributed: false
        distributed_interval: 10
        distributed_plugin: tls
        distributed_tls_max_attempts: 3
        logger_plugin: tls
        logger_tls_endpoint: /api/v1/osquery/log
        logger_tls_period: 10
        pack_delimiter: /
    overrides: {}
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  host_settings:
    enable_host_users: true
    enable_software_inventory: false
  org_info:
    org_logo_url: ""
    org_name: FleetDM
  server_settings:
    enable_analytics: true
    live_query_disabled: false
    server_url: https://dogfood.fleetctl.com
  smtp_settings:
    authentication_method: authmethod_plain
    authentication_type: authtype_username_password
    configured: false
    domain: ""
    enable_smtp: false
    enable_ssl_tls: true
    enable_start_tls: true
    password: ""
    port: 587
    sender_address: ""
    server: ""
    user_name: ""
    verify_ssl_certs: true
  sso_settings:
    enable_sso: true
    enable_sso_idp_login: false
    entity_id: dogfood.fleetdm.com
    idp_image_url: ""
    idp_name: Google
    issuer_uri: https://accounts.google.com/o/saml2/idp?idpid=C01o3oocn
    metadata: |
      <?xml version="1.0" encoding="UTF-8" standalone="no"?>
      <md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://accounts.google.com/o/saml2?idpid=C01o3oocn" validUntil="2026-09-14T23:17:31.000Z">
        <md:IDPSSODescriptor WantAuthnRequestsSigned="false" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
          <md:KeyDescriptor use="signing">
            <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
              <ds:X509Data>
                <ds:X509Certificate>MIIDdDCCAlygAwIBAgIGAXvrwEkDMA0GCSqGSIb3DQEBCwUAMHsxFDASBgNVBAoTC0dvb2dsZSBJ
      bmMuMRYwFAYDVQQHEw1Nb3VudGFpbiBWaWV3MQ8wDQYDVQQDEwZHb29nbGUxGDAWBgNVBAsTD0dv
      b2dsZSBGb3IgV29yazELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWEwHhcNMjEwOTE1
      MjMxNzMxWhcNMjYwOTE0MjMxNzMxWjB7MRQwEgYDVQQKEwtHb29nbGUgSW5jLjEWMBQGA1UEBxMN
      TW91bnRhaW4gVmlldzEPMA0GA1UEAxMGR29vZ2xlMRgwFgYDVQQLEw9Hb29nbGUgRm9yIFdvcmsx
      CzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
      MIIBCgKCAQEA3MEEA4PqlMEZlz8UmmjNbbUehDQcf9TPUnURUxnUURw+KJOEuDv8E2xCsmjGpY9p
      vmGNIhIhVzohTpGjY9IQOychwbOsBHAzjgALJIo0UtUMu75dey4MG1vcQlWYpUFhzhZN+8b9hMkg
      fuLfzyl58r/w8W0iVqy7BwghWiFtM1wthtlzXEEFG9vCKaQt88iz2f1D//AX4MyxBPIRHB+IGBmb
      WTHrxAf/nnPUSnJaGmpKX5nC+rEMY+gxf5LfqREmohakvOU0QdKKyEFUcj3B/jIGQp5JTMuuH6fO
      WBh39iCFq6ruUj7ykUjnqzN8UA9IS4roxtczSh/bkwfBx3RYvQIDAQABMA0GCSqGSIb3DQEBCwUA
      A4IBAQC6Dkb6lYlCsdfapWjox2LJtp2PPaSyFvSaF0i2vJ2pTj3TLOijRUc5VKZYRuOgf228YeyQ
      GouvcuiKQvRA8tA/pv4bh4Rdbd/O7YrHZXCPQMOMUSL1SHc7v0F9AbxIg7BPYh19rFVznD5J4cbb
      ycrxQ1bwt1cGModQnLyCROoH23ZHAvyH/haaFLNOir7Uzc4vU/FGfie3QjEGqoBjnC6w/F/w0GAD
      Ho2h+JSJIlhydHolrIzc07lF0nfT+ShKA/l/waTD174JSkWW6rQ6MpCX5fGR7DLEYfRBby/Da5fA
      XkbUazMGv0D/m+FIvWmb60J+yuVHybMGRkFKWGOi4hTX</ds:X509Certificate>
              </ds:X509Data>
            </ds:KeyInfo>
          </md:KeyDescriptor>
          <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
          <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://accounts.google.com/o/saml2/idp?idpid=C01o3oocn"/>
          <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://accounts.google.com/o/saml2/idp?idpid=C01o3oocn"/>
        </md:IDPSSODescriptor>
      </md:EntityDescriptor>
    metadata_url: ""
  vulnerability_settings:
    databases_path: ""
  webhook_settings:
    host_status_webhook:
      days_count: 0
      destination_url: ""
      enable_host_status_webhook: false
      host_percentage: 0
    interval: 24h0m0s
`

	var appConfigSpec map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(spec), &appConfigSpec))

	bytes, err := json.Marshal(appConfigSpec["spec"])
	require.NoError(t, err)
	insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
	_, err = ds.writer.Exec(insertAppConfigQuery, bytes)
	require.NoError(t, err)

	ac, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	_ = ac
}
