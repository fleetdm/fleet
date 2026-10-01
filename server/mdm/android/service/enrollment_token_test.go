package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// An unauthenticated caller with an invalid enroll secret must get the same
// error whether or not Android MDM is configured, so the endpoint can't be
// used to probe configuration state.
func TestCreateEnrollmentTokenUniformInvalidSecretError(t *testing.T) {
	androidAPIClient := android_mock.Client{}
	androidAPIClient.InitCommonMocks()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	fleetDS := InitCommonDSMocks()
	fleetDS.Store.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return nil, &notFoundError{}
	}
	svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, "test-private-key", &fleetDS.DataStore, noopNewActivity, config.AndroidAgentConfig{})
	require.NoError(t, err)

	badSecretErr := func(configured bool, idpSessionID string) error {
		fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.MDM.AndroidEnabledAndConfigured = configured
			return appCfg, nil
		}
		_, err := svc.CreateEnrollmentToken(t.Context(), "bogus-secret", idpSessionID, false)
		require.Error(t, err)
		return err
	}
	errNotConfigured := badSecretErr(false, "")
	errConfigured := badSecretErr(true, "")
	// The service has no idP session store, so this only returns the uniform
	// auth error if the secret is verified before idP session validation.
	errIdPSession := badSecretErr(true, "bogus-session")

	var authFailed *fleet.AuthFailedError
	require.ErrorAs(t, errNotConfigured, &authFailed)
	require.ErrorAs(t, errConfigured, &authFailed)
	require.ErrorAs(t, errIdPSession, &authFailed)
	require.Equal(t, errConfigured.Error(), errNotConfigured.Error())
	require.Equal(t, errConfigured.Error(), errIdPSession.Error())
}

// The token carries the fleet resolved from the secret, never the secret itself, so rotating the
// secret before the device finishes enrolling doesn't change where it lands.
func TestCreateEnrollmentTokenEmbedsFleetID(t *testing.T) {
	teamID := uint(7)
	cases := []struct {
		name       string
		secret     *fleet.EnrollSecret
		wantTeamID *uint
	}{
		{name: "global secret", secret: &fleet.EnrollSecret{Secret: "global-secret"}},
		{name: "team secret", secret: &fleet.EnrollSecret{Secret: "team-secret", TeamID: &teamID}, wantTeamID: &teamID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			androidAPIClient := android_mock.Client{}
			androidAPIClient.InitCommonMocks()
			var createdToken *androidmanagement.EnrollmentToken
			androidAPIClient.EnterprisesEnrollmentTokensCreateFunc = func(_ context.Context, _ string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
				createdToken = token
				return &androidmanagement.EnrollmentToken{Value: "token-value"}, nil
			}

			fleetDS := InitCommonDSMocks()
			fleetDS.Store.VerifyEnrollSecretFunc = func(_ context.Context, secret string) (*fleet.EnrollSecret, error) {
				require.Equal(t, tc.secret.Secret, secret)
				return tc.secret, nil
			}
			fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
				appCfg := &fleet.AppConfig{}
				appCfg.MDM.AndroidEnabledAndConfigured = true
				return appCfg, nil
			}
			fleetDS.Store.TeamLiteFunc = func(_ context.Context, id uint) (*fleet.TeamLite, error) {
				return &fleet.TeamLite{ID: id}, nil
			}
			fleetDS.Store.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
				return &android.Enterprise{EnterpriseID: "LC00test"}, nil
			}
			svc, err := NewServiceWithClient(slog.New(slog.DiscardHandler), fleetDS, &androidAPIClient, "test-private-key", &fleetDS.DataStore, noopNewActivity, config.AndroidAgentConfig{})
			require.NoError(t, err)

			_, err = svc.CreateEnrollmentToken(t.Context(), tc.secret.Secret, "", false)
			require.NoError(t, err)
			require.NotNil(t, createdToken)

			assert.NotContains(t, createdToken.AdditionalData, tc.secret.Secret)
			var data teamEnrollmentRequest
			require.NoError(t, json.Unmarshal([]byte(createdToken.AdditionalData), &data))
			assert.Equal(t, tc.wantTeamID, data.TeamID)
			assert.Empty(t, data.IdpUUID)
		})
	}
}
