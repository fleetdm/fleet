package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/fleetdm/fleet/server/ptr"
	"github.com/fleetdm/fleet/server/test"
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
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}

	var appConfigTests = []struct {
		configPayload kolide.AppConfigPayload
	}{
		{
			configPayload: kolide.AppConfigPayload{
				OrgInfo: &kolide.OrgInfo{
					OrgLogoURL: ptr.String("acme.co/images/logo.png"),
					OrgName:    ptr.String("Acme"),
				},
				ServerSettings: &kolide.ServerSettings{
					KolideServerURL:   ptr.String("https://acme.co:8080/"),
					LiveQueryDisabled: ptr.Bool(true),
				},
			},
		},
	}

	for _, tt := range appConfigTests {
		var result *kolide.AppConfig
		ds.NewAppConfigFunc = func(config *kolide.AppConfig) (*kolide.AppConfig, error) {
			result = config
			return config, nil
		}

		var gotSecrets []*kolide.EnrollSecret
		ds.ApplyEnrollSecretsFunc = func(teamID *uint, secrets []*kolide.EnrollSecret) error {
			gotSecrets = secrets
			return nil
		}

		ctx := test.UserContext(test.UserAdmin)
		_, err := svc.NewAppConfig(ctx, tt.configPayload)
		require.Nil(t, err)

		payload := tt.configPayload
		assert.Equal(t, *payload.OrgInfo.OrgLogoURL, result.OrgLogoURL)
		assert.Equal(t, *payload.OrgInfo.OrgName, result.OrgName)
		assert.Equal(t, "https://acme.co:8080", result.KolideServerURL)
		assert.Equal(t, *payload.ServerSettings.LiveQueryDisabled, result.LiveQueryDisabled)

		// Ensure enroll secret was set
		require.NotNil(t, gotSecrets)
		require.Len(t, gotSecrets, 1)
		assert.Len(t, gotSecrets[0].Secret, 32)
	}
}

func TestEmptyEnrollSecret(t *testing.T) {
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ds.ApplyEnrollSecretsFunc = func(teamID *uint, secrets []*kolide.EnrollSecret) error {
		return nil
	}
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}

	err = svc.ApplyEnrollSecretSpec(
		context.Background(),
		&kolide.EnrollSecretSpec{
			Secrets: []*kolide.EnrollSecret{{}},
		},
	)
	require.Error(t, err)

	err = svc.ApplyEnrollSecretSpec(
		context.Background(),
		&kolide.EnrollSecretSpec{Secrets: []*kolide.EnrollSecret{{Secret: ""}}},
	)
	require.Error(t, err, "empty secret should be disallowed")

	err = svc.ApplyEnrollSecretSpec(
		context.Background(),
		&kolide.EnrollSecretSpec{
			Secrets: []*kolide.EnrollSecret{{Secret: "foo"}},
		},
	)
	require.NoError(t, err)
}
