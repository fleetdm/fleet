package service

import (
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSONotPresent(t *testing.T) {
	invalid := &invalidArgumentError{}
	var p kolide.AppConfigPayload
	validateSSOSettings(p, &kolide.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())

}

func TestNeedFieldsPresent(t *testing.T) {
	invalid := &invalidArgumentError{}
	config := kolide.AppConfig{
		EnableSSO:   true,
		EntityID:    "kolide",
		IssuerURI:   "http://issuer.idp.com",
		MetadataURL: "http://isser.metadata.com",
		IDPName:     "onelogin",
	}
	p := appConfigPayloadFromAppConfig(&config)
	validateSSOSettings(*p, &kolide.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())
}

func TestMissingMetadata(t *testing.T) {
	invalid := invalidArgumentError{}
	config := kolide.AppConfig{
		EnableSSO: true,
		EntityID:  "kolide",
		IssuerURI: "http://issuer.idp.com",
		IDPName:   "onelogin",
	}
	p := appConfigPayloadFromAppConfig(&config)
	validateSSOSettings(*p, &kolide.AppConfig{}, &invalid)
	require.True(t, invalid.HasErrors())
	require.Len(t, invalid, 1)
	assert.Equal(t, "metadata", invalid[0].name)
	assert.Equal(t, "either metadata or metadata_url must be defined", invalid[0].reason)
}
