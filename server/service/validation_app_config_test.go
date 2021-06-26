package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSONotPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	var p fleet.AppConfigPayload
	validateSSOSettings(p, &fleet.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())

}

func TestNeedFieldsPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		EnableSSO:   true,
		EntityID:    "fleet",
		IssuerURI:   "http://issuer.idp.com",
		MetadataURL: "http://isser.metadata.com",
		IDPName:     "onelogin",
	}
	p := appConfigPayloadFromAppConfig(&config)
	validateSSOSettings(*p, &fleet.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())
}

func TestMissingMetadata(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		EnableSSO: true,
		EntityID:  "fleet",
		IssuerURI: "http://issuer.idp.com",
		IDPName:   "onelogin",
	}
	p := appConfigPayloadFromAppConfig(&config)
	validateSSOSettings(*p, &fleet.AppConfig{}, invalid)
	require.True(t, invalid.HasErrors())
	assert.Contains(t, invalid.Error(), "metadata")
	assert.Contains(t, invalid.Error(), "either metadata or metadata_url must be defined")
}

func appConfigPayloadFromAppConfig(config *fleet.AppConfig) *fleet.AppConfigPayload {
	return &fleet.AppConfigPayload{
		OrgInfo: &fleet.OrgInfo{
			OrgLogoURL: &config.OrgLogoURL,
			OrgName:    &config.OrgName,
		},
		ServerSettings: &fleet.ServerSettings{
			ServerURL:         &config.ServerURL,
			LiveQueryDisabled: &config.LiveQueryDisabled,
		},
		SMTPSettings: smtpSettingsFromAppConfig(config),
		SSOSettings: &fleet.SSOSettingsPayload{
			EnableSSO:   &config.EnableSSO,
			IDPName:     &config.IDPName,
			Metadata:    &config.Metadata,
			MetadataURL: &config.MetadataURL,
			IssuerURI:   &config.IssuerURI,
			EntityID:    &config.EntityID,
		},
		HostExpirySettings: &fleet.HostExpirySettings{
			HostExpiryEnabled: &config.HostExpiryEnabled,
			HostExpiryWindow:  &config.HostExpiryWindow,
		},
	}
}
