package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/require"
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
