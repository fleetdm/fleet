package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSONotPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	var p fleet.AppConfig
	validateSSOSettings(p, &fleet.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())

}

func TestNeedFieldsPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: &fleet.SSOSettings{
			EnableSSO:   ptr.Bool(true),
			EntityID:    ptr.String("fleet"),
			IssuerURI:   ptr.String("http://issuer.idp.com"),
			MetadataURL: ptr.String("http://isser.metadata.com"),
			IDPName:     ptr.String("onelogin"),
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())
}

func TestMissingMetadata(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: &fleet.SSOSettings{
			EnableSSO: ptr.Bool(true),
			EntityID:  ptr.String("fleet"),
			IssuerURI: ptr.String("http://issuer.idp.com"),
			IDPName:   ptr.String("onelogin"),
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid)
	require.True(t, invalid.HasErrors())
	assert.Contains(t, invalid.Error(), "metadata")
	assert.Contains(t, invalid.Error(), "either metadata or metadata_url must be defined")
}
