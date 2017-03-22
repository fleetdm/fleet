package service

import (
	"context"
	"testing"

	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupURL(t *testing.T) {
	tests := []struct {
		in       string
		expected string
		name     string
	}{
		{"  http://foo.bar.com  ", "http://foo.bar.com", "leading and trailing whitespace"},
		{"\n http://foo.com \t", "http://foo.com", "whitespace"},
		{"http://foo.com", "http://foo.com", "noop"},
		{"http://foo.com/", "http://foo.com", "trailing slash"},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actual := cleanupURL(test.in)
			assert.Equal(tt, test.expected, actual)
		})
	}

}

func TestCreateAppConfig(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	require.Nil(t, ds.MigrateData())

	svc, err := newTestService(ds, nil)
	require.Nil(t, err)
	var appConfigTests = []struct {
		configPayload kolide.AppConfigPayload
	}{
		{
			configPayload: kolide.AppConfigPayload{
				OrgInfo: &kolide.OrgInfo{
					OrgLogoURL: stringPtr("acme.co/images/logo.png"),
					OrgName:    stringPtr("Acme"),
				},
				ServerSettings: &kolide.ServerSettings{
					KolideServerURL: stringPtr("https://acme.co:8080/"),
				},
			},
		},
	}

	for _, tt := range appConfigTests {
		result, err := svc.NewAppConfig(context.Background(), tt.configPayload)
		require.Nil(t, err)

		payload := tt.configPayload
		assert.NotEmpty(t, result.ID)
		assert.Equal(t, *payload.OrgInfo.OrgLogoURL, result.OrgLogoURL)
		assert.Equal(t, *payload.OrgInfo.OrgName, result.OrgName)
		assert.Equal(t, "https://acme.co:8080", result.KolideServerURL)
	}
}
