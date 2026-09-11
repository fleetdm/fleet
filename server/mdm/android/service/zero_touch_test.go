package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
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

func TestZeroTouchAuth(t *testing.T) {
	svc, fleetDS, _ := setupZeroTouchService(t)

	fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	fleetDS.Store.GetZeroTouchEnrollmentTokenFunc = func(_ context.Context, _ *uint) (*android.ZeroTouchToken, error) {
		return &android.ZeroTouchToken{
			TokenValue: "existing-token",
			ExpiresAt:  time.Now().Add(100 * 365 * 24 * time.Hour),
		}, nil
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: new(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: new(fleet.RoleMaintainer)},
			true,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: new(fleet.RoleGitOps)},
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: new(fleet.RoleObserver)},
			true,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: new(fleet.RoleObserverPlus)},
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}, //nolint:modernize // mixed keyed/unkeyed not allowed
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}}, //nolint:modernize // mixed keyed/unkeyed not allowed
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}, //nolint:modernize // mixed keyed/unkeyed not allowed
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: tt.user})
			_, err := svc.GetZeroTouchConfiguration(ctx, nil)
			checkAuthErr(t, tt.shouldFail, err)
		})
	}
}

func TestZeroTouchAndroidNotConfigured(t *testing.T) {
	svc, fleetDS, _ := setupZeroTouchService(t)

	fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: false}}, nil
	}

	_, err := svc.GetZeroTouchConfiguration(adminCtx(t), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Android MDM is NOT configured")
}

func TestZeroTouchReturnsExistingToken(t *testing.T) {
	svc, fleetDS, _ := setupZeroTouchService(t)

	fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	fleetDS.Store.GetZeroTouchEnrollmentTokenFunc = func(_ context.Context, teamID *uint) (*android.ZeroTouchToken, error) {
		assert.Nil(t, teamID, "should query for unassigned team")
		return &android.ZeroTouchToken{
			TokenValue: "existing-token-value",
			ExpiresAt:  time.Now().Add(100 * 365 * 24 * time.Hour),
		}, nil
	}

	resp, err := svc.GetZeroTouchConfiguration(adminCtx(t), nil)
	require.NoError(t, err)
	assert.Contains(t, resp.DPCExtras, "existing-token-value")
	assert.True(t, fleetDS.Store.GetZeroTouchEnrollmentTokenFuncInvoked)
	assert.False(t, fleetDS.Store.CreateZeroTouchEnrollmentTokenFuncInvoked)
}

func TestZeroTouchCreatesTokenWhenNoneExists(t *testing.T) {
	svc, fleetDS, apiClient := setupZeroTouchService(t)

	fleetDS.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	fleetDS.Store.GetZeroTouchEnrollmentTokenFunc = func(_ context.Context, _ *uint) (*android.ZeroTouchToken, error) {
		return nil, &notFoundError{}
	}
	fleetDS.Store.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{EnterpriseID: "LC00test"}, nil
	}

	apiClient.EnterprisesEnrollmentTokensCreateFunc = func(_ context.Context, enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
		assert.Equal(t, "enterprises/LC00test", enterpriseName)
		assert.Equal(t, "3153600000s", token.Duration)
		assert.False(t, token.OneTimeOnly)
		assert.Equal(t, "PERSONAL_USAGE_DISALLOWED", token.AllowPersonalUsage)
		assert.Contains(t, token.AdditionalData, `"fleet_id"`)
		return &androidmanagement.EnrollmentToken{
			Name:                "enterprises/LC00test/enrollmentTokens/abc123",
			Value:               "new-token-value",
			ExpirationTimestamp: time.Now().Add(100 * 365 * 24 * time.Hour).Format(time.RFC3339),
		}, nil
	}

	fleetDS.Store.CreateZeroTouchEnrollmentTokenFunc = func(_ context.Context, token *android.ZeroTouchToken) (*android.ZeroTouchToken, error) {
		assert.Equal(t, "enterprises/LC00test/enrollmentTokens/abc123", token.TokenName)
		assert.Equal(t, "new-token-value", token.TokenValue)
		assert.Nil(t, token.TeamID)
		token.ID = 1
		return token, nil
	}

	resp, err := svc.GetZeroTouchConfiguration(adminCtx(t), nil)
	require.NoError(t, err)
	assert.Contains(t, resp.DPCExtras, "new-token-value")
	assert.Contains(t, resp.DPCExtras, "EXTRA_ENROLLMENT_TOKEN")
	assert.True(t, fleetDS.Store.CreateZeroTouchEnrollmentTokenFuncInvoked)
	assert.True(t, apiClient.EnterprisesEnrollmentTokensCreateFuncInvoked)
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

	t.Run("zero-touch with null team_id goes to unassigned", func(t *testing.T) {
		teamID, idpUUID, err := svc.resolveTeamFromEnrollmentData(t.Context(), `{"fleet_id": null}`)
		require.NoError(t, err)
		assert.Nil(t, teamID)
		assert.Empty(t, idpUUID)
	})

	t.Run("zero-touch with existing team_id goes to that team", func(t *testing.T) {
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

	t.Run("zero-touch with non-existent team_id falls back to unassigned", func(t *testing.T) {
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
