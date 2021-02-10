package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockValidationItem struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}
type mockValidationError struct {
	Message string               `json:"message"`
	Errors  []mockValidationItem `json:"errors"`
}

func testGetAppConfig(t *testing.T, r *testResource) {
	req, err := http.NewRequest("GET", r.server.URL+"/api/v1/fleet/config", nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var configInfo appConfigResponse
	err = json.NewDecoder(resp.Body).Decode(&configInfo)
	require.Nil(t, err)
	require.NotNil(t, configInfo.SMTPSettings)
	config := configInfo.SMTPSettings
	assert.Equal(t, uint(465), *config.SMTPPort)
	require.NotNil(t, *configInfo.OrgInfo)
	assert.Equal(t, "Kolide", *configInfo.OrgInfo.OrgName)
	assert.Equal(t, "http://foo.bar/image.png", *configInfo.OrgInfo.OrgLogoURL)
	assert.False(t, *configInfo.HostExpirySettings.HostExpiryEnabled)
	assert.Equal(t, 0, *configInfo.HostExpirySettings.HostExpiryWindow)
	assert.False(t, *configInfo.ServerSettings.LiveQueryDisabled)

}

func testModifyAppConfig(t *testing.T, r *testResource) {
	config := &kolide.AppConfig{
		KolideServerURL:        "https://foo.com",
		OrgName:                "Zip",
		OrgLogoURL:             "http://foo.bar/image.png",
		SMTPPort:               567,
		SMTPAuthenticationType: kolide.AuthTypeNone,
		SMTPServer:             "foo.com",
		SMTPEnableTLS:          true,
		SMTPVerifySSLCerts:     true,
		SMTPEnableStartTLS:     true,
		EnableSSO:              true,
		IDPName:                "idpname",
		Metadata:               "metadataxxxxxx",
		IssuerURI:              "http://issuer.idp.com",
		EntityID:               "kolide",
		HostExpiryEnabled:      true,
		HostExpiryWindow:       42,
		LiveQueryDisabled:      true,
	}
	payload := appConfigPayloadFromAppConfig(config)
	payload.SMTPTest = new(bool)
	*payload.SMTPSettings.SMTPEnabled = true

	var buffer bytes.Buffer
	err := json.NewEncoder(&buffer).Encode(payload)
	require.Nil(t, err)
	req, err := http.NewRequest("PATCH", r.server.URL+"/api/v1/fleet/config", &buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var respBody appConfigResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.Nil(t, err)
	require.NotNil(t, respBody.OrgInfo)
	assert.Equal(t, config.OrgName, *respBody.OrgInfo.OrgName)
	saved, err := r.ds.AppConfig()
	require.Nil(t, err)
	// verify email configuration succeeded
	assert.True(t, saved.SMTPConfigured)
	// verify that SSO settings were saved
	assert.True(t, saved.EnableSSO)
	assert.Equal(t, "idpname", saved.IDPName)
	assert.Equal(t, "metadataxxxxxx", saved.Metadata)
	assert.Equal(t, "http://issuer.idp.com", saved.IssuerURI)
	assert.Equal(t, "kolide", saved.EntityID)
	// verify that host expiry settings were saved
	assert.True(t, saved.HostExpiryEnabled)
	assert.Equal(t, 42, saved.HostExpiryWindow)
	//verify that live query disabled setting was saved
	assert.True(t, saved.LiveQueryDisabled)

}

func testModifyAppConfigWithValidationFail(t *testing.T, r *testResource) {
	config := &kolide.AppConfig{
		SMTPEnableStartTLS: false,
	}
	payload := appConfigPayloadFromAppConfig(config)
	payload.SMTPTest = new(bool)

	var buffer bytes.Buffer
	err := json.NewEncoder(&buffer).Encode(payload)
	require.Nil(t, err)
	req, err := http.NewRequest("PATCH", r.server.URL+"/api/v1/fleet/config", &buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var validationErrors mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&validationErrors)
	require.Nil(t, err)
	require.Equal(t, 0, len(validationErrors.Errors))
	existing, err := r.ds.AppConfig()
	assert.Nil(t, err)
	assert.Equal(t, config.SMTPEnableStartTLS, existing.SMTPEnableStartTLS)
}

func appConfigPayloadFromAppConfig(config *kolide.AppConfig) *kolide.AppConfigPayload {
	return &kolide.AppConfigPayload{
		OrgInfo: &kolide.OrgInfo{
			OrgLogoURL: &config.OrgLogoURL,
			OrgName:    &config.OrgName,
		},
		ServerSettings: &kolide.ServerSettings{
			KolideServerURL:   &config.KolideServerURL,
			LiveQueryDisabled: &config.LiveQueryDisabled,
		},
		SMTPSettings: smtpSettingsFromAppConfig(config),
		SSOSettings: &kolide.SSOSettingsPayload{
			EnableSSO:   &config.EnableSSO,
			IDPName:     &config.IDPName,
			Metadata:    &config.Metadata,
			MetadataURL: &config.MetadataURL,
			IssuerURI:   &config.IssuerURI,
			EntityID:    &config.EntityID,
		},
		HostExpirySettings: &kolide.HostExpirySettings{
			HostExpiryEnabled: &config.HostExpiryEnabled,
			HostExpiryWindow:  &config.HostExpiryWindow,
		},
	}
}
