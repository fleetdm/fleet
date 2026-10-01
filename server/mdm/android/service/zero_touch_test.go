package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupZeroTouchService(t *testing.T) (*Service, *AndroidMockDS, *android_mock.Client) {
	t.Helper()

	fleetDS := InitCommonDSMocks()
	androidAPIClient := android_mock.Client{}
	androidAPIClient.InitCommonMocks()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, "test-private-key", &fleetDS.DataStore, noopNewActivity, config.AndroidAgentConfig{})
	require.NoError(t, err)

	return svc.(*Service), fleetDS, &androidAPIClient
}

func adminCtx(t *testing.T) context.Context {
	t.Helper()
	return viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
}

func TestZeroTouchStubReturnsErrMissingLicense(t *testing.T) {
	svc, _, _ := setupZeroTouchService(t)
	_, err := svc.GetZeroTouchConfiguration(adminCtx(t), nil)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)
}

func TestZeroTouchTokenDeletedOnEnterpriseDelete(t *testing.T) {
	svc, fleetDS, apiClient := setupZeroTouchService(t)

	var tokenDeleted bool
	fleetDS.Store.DeleteZeroTouchEnrollmentTokensFunc = func(_ context.Context) error {
		tokenDeleted = true
		return nil
	}

	apiClient.EnterpriseDeleteFunc = func(_ context.Context, _ string) error {
		return nil
	}

	fleetDS.Store.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{EnterpriseID: "LC00test"}, nil
	}

	err := svc.DeleteEnterprise(adminCtx(t))
	require.NoError(t, err)
	assert.True(t, tokenDeleted, "zero-touch tokens should be deleted when enterprise is deleted")
}

func TestResolveTeamFromEnrollmentData(t *testing.T) {
	svc, fleetDS, _ := setupZeroTouchService(t)

	t.Run("zero-touch with null fleet_id goes to unassigned", func(t *testing.T) {
		teamID, idpUUID, err := svc.resolveTeamFromEnrollmentData(t.Context(), `{"fleet_id": null}`)
		require.NoError(t, err)
		assert.Nil(t, teamID)
		assert.Empty(t, idpUUID)
	})

	t.Run("zero-touch with existing fleet_id goes to that team", func(t *testing.T) {
		fleetDS.Store.TeamExistsFunc = func(_ context.Context, id uint) (bool, error) {
			assert.Equal(t, uint(3), id)
			return true, nil
		}
		teamID, idpUUID, err := svc.resolveTeamFromEnrollmentData(t.Context(), `{"fleet_id": 3}`)
		require.NoError(t, err)
		require.NotNil(t, teamID)
		assert.Equal(t, uint(3), *teamID)
		assert.Empty(t, idpUUID)
	})

	t.Run("zero-touch with non-existent fleet_id falls back to unassigned", func(t *testing.T) {
		fleetDS.Store.TeamExistsFunc = func(_ context.Context, id uint) (bool, error) {
			return false, nil
		}
		teamID, idpUUID, err := svc.resolveTeamFromEnrollmentData(t.Context(), `{"fleet_id": 99}`)
		require.NoError(t, err)
		assert.Nil(t, teamID, "non-existent team should fall back to unassigned")
		assert.Empty(t, idpUUID)
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		_, _, err := svc.resolveTeamFromEnrollmentData(t.Context(), `not valid json`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshalling enrollment token data")
	})

	t.Run("QR enrollment with enroll secret falls back to existing flow", func(t *testing.T) {
		expectedTeamID := uint(5)
		fleetDS.Store.VerifyEnrollSecretFunc = func(_ context.Context, secret string) (*fleet.EnrollSecret, error) {
			assert.Equal(t, "test-secret", secret)
			return &fleet.EnrollSecret{Secret: secret, TeamID: &expectedTeamID}, nil
		}
		teamID, idpUUID, err := svc.resolveTeamFromEnrollmentData(t.Context(), `{"EnrollSecret": "test-secret", "IdpUUID": "some-uuid"}`)
		require.NoError(t, err)
		require.NotNil(t, teamID)
		assert.Equal(t, uint(5), *teamID)
		assert.Equal(t, "some-uuid", idpUUID)
	})
}
