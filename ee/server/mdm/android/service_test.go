package android

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	licensectx "github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	ds_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func premiumAdminCtx(t *testing.T) context.Context {
	t.Helper()
	ctx := licensectx.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	return viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
}

func setupEEService(t *testing.T) (*Service, *ds_mock.Store, *android_mock.Client) {
	t.Helper()

	mockDS := &ds_mock.Store{}
	apiClient := &android_mock.Client{}
	apiClient.InitCommonMocks()

	// Default mocks
	mockDS.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	mockDS.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		result := make(map[fleet.MDMAssetName]fleet.MDMConfigAsset, len(names))
		for _, name := range names {
			result[name] = fleet.MDMConfigAsset{Value: []byte("value")}
		}
		return result, nil
	}

	// Stub core service (returns ErrMissingLicense for GetZeroTouchConfiguration)
	stubCore := &stubCoreService{}

	svc, err := NewService(stubCore, mockDS, mockDS, apiClient, nil)
	require.NoError(t, err)

	return svc, mockDS, apiClient
}

// stubCoreService satisfies android.Service. GetZeroTouchConfiguration returns
// ErrMissingLicense (matching the real core stub). Other methods panic if called.
type stubCoreService struct {
	android.Service
}

func TestGetZeroTouchConfiguration_PremiumRequired(t *testing.T) {
	svc, _, _ := setupEEService(t)

	// No license in context
	_, err := svc.GetZeroTouchConfiguration(t.Context(), nil)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)
}

func TestGetZeroTouchConfiguration_FreeLicense(t *testing.T) {
	svc, _, _ := setupEEService(t)

	ctx := licensectx.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierFree})
	_, err := svc.GetZeroTouchConfiguration(ctx, nil)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)
}

func TestGetZeroTouchConfiguration_AndroidNotConfigured(t *testing.T) {
	svc, mockDS, _ := setupEEService(t)

	mockDS.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: false}}, nil
	}

	_, err := svc.GetZeroTouchConfiguration(premiumAdminCtx(t), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Android MDM is NOT configured")
}

func TestGetZeroTouchConfiguration_ReturnsExistingToken(t *testing.T) {
	svc, mockDS, _ := setupEEService(t)

	mockDS.GetZeroTouchEnrollmentTokenFunc = func(_ context.Context, teamID *uint) (*android.ZeroTouchToken, error) {
		assert.Nil(t, teamID)
		return &android.ZeroTouchToken{
			TokenValue: "existing-token-value",
			ExpiresAt:  time.Now().Add(100 * 365 * 24 * time.Hour),
		}, nil
	}

	resp, err := svc.GetZeroTouchConfiguration(premiumAdminCtx(t), nil)
	require.NoError(t, err)

	var extras dpcExtras
	require.NoError(t, json.Unmarshal(resp.DPCExtras, &extras))
	assert.Equal(t, "existing-token-value", extras.AdminExtrasBundle.EnrollmentToken)

	assert.True(t, mockDS.GetZeroTouchEnrollmentTokenFuncInvoked)
	assert.False(t, mockDS.CreateZeroTouchEnrollmentTokenFuncInvoked)
}

func TestGetZeroTouchConfiguration_CreatesTokenWhenNoneExists(t *testing.T) {
	svc, mockDS, apiClient := setupEEService(t)

	mockDS.GetZeroTouchEnrollmentTokenFunc = func(_ context.Context, _ *uint) (*android.ZeroTouchToken, error) {
		return nil, &notFoundError{}
	}
	mockDS.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{EnterpriseID: "LC00test"}, nil
	}

	apiClient.EnterprisesEnrollmentTokensCreateFunc = func(_ context.Context, enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
		assert.Equal(t, "enterprises/LC00test", enterpriseName)
		assert.Equal(t, zeroTouchTokenDuration, token.Duration)
		assert.False(t, token.OneTimeOnly)
		assert.Equal(t, "PERSONAL_USAGE_DISALLOWED", token.AllowPersonalUsage)
		assert.Contains(t, token.AdditionalData, `"fleet_id"`)
		return &androidmanagement.EnrollmentToken{
			Name:                "enterprises/LC00test/enrollmentTokens/abc123",
			Value:               "new-token-value",
			ExpirationTimestamp: time.Now().Add(100 * 365 * 24 * time.Hour).Format(time.RFC3339),
		}, nil
	}

	mockDS.CreateZeroTouchEnrollmentTokenFunc = func(_ context.Context, token *android.ZeroTouchToken) (*android.ZeroTouchToken, error) {
		assert.Equal(t, "enterprises/LC00test/enrollmentTokens/abc123", token.TokenName)
		assert.Equal(t, "new-token-value", token.TokenValue)
		assert.Nil(t, token.TeamID)
		token.ID = 1
		return token, nil
	}

	resp, err := svc.GetZeroTouchConfiguration(premiumAdminCtx(t), nil)
	require.NoError(t, err)

	var extras dpcExtras
	require.NoError(t, json.Unmarshal(resp.DPCExtras, &extras))
	assert.Equal(t, "new-token-value", extras.AdminExtrasBundle.EnrollmentToken)

	assert.True(t, mockDS.CreateZeroTouchEnrollmentTokenFuncInvoked)
	assert.True(t, apiClient.EnterprisesEnrollmentTokensCreateFuncInvoked)
}

type notFoundError struct{}

func (e *notFoundError) Error() string    { return "not found" }
func (e *notFoundError) IsNotFound() bool { return true }

func TestGetZeroTouchConfiguration_FleetSelection(t *testing.T) {
	const existingFleetID uint = 3

	cases := []struct {
		name               string
		fleetID            *uint
		hasExistingToken   bool
		wantTeamID         *uint
		wantAdditionalData string
		wantNotFound       bool
		wantTeamExistsCall bool
		wantAMAPICall      bool
	}{
		{
			name:               "no fleet creates an Unassigned token",
			fleetID:            nil,
			wantTeamID:         nil,
			wantAdditionalData: `{"fleet_id":null}`,
			wantAMAPICall:      true,
		},
		{
			name:               "fleet 0 is treated as Unassigned",
			fleetID:            new(uint(0)),
			wantTeamID:         nil,
			wantAdditionalData: `{"fleet_id":null}`,
			wantAMAPICall:      true,
		},
		{
			name:               "existing fleet creates a token for that fleet",
			fleetID:            new(existingFleetID),
			wantTeamID:         new(existingFleetID),
			wantAdditionalData: `{"fleet_id":3}`,
			wantTeamExistsCall: true,
			wantAMAPICall:      true,
		},
		{
			name:               "existing fleet with a token reuses it",
			fleetID:            new(existingFleetID),
			hasExistingToken:   true,
			wantTeamID:         new(existingFleetID),
			wantTeamExistsCall: true,
		},
		{
			name:               "missing fleet returns not found",
			fleetID:            new(uint(999)),
			wantNotFound:       true,
			wantTeamExistsCall: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, mockDS, apiClient := setupEEService(t)

			mockDS.TeamExistsFunc = func(_ context.Context, id uint) (bool, error) {
				return id == existingFleetID, nil
			}
			mockDS.GetZeroTouchEnrollmentTokenFunc = func(_ context.Context, teamID *uint) (*android.ZeroTouchToken, error) {
				assert.Equal(t, c.wantTeamID, teamID)
				if c.hasExistingToken {
					return &android.ZeroTouchToken{TeamID: teamID, TokenValue: "existing-token-value"}, nil
				}
				return nil, &notFoundError{}
			}
			mockDS.GetEnterpriseFunc = func(_ context.Context) (*android.Enterprise, error) {
				return &android.Enterprise{EnterpriseID: "LC00test"}, nil
			}
			apiClient.EnterprisesEnrollmentTokensCreateFunc = func(_ context.Context, _ string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
				assert.JSONEq(t, c.wantAdditionalData, token.AdditionalData)
				return &androidmanagement.EnrollmentToken{
					Name:                "enterprises/LC00test/enrollmentTokens/abc123",
					Value:               "new-token-value",
					ExpirationTimestamp: time.Now().Add(100 * 365 * 24 * time.Hour).Format(time.RFC3339),
				}, nil
			}
			mockDS.CreateZeroTouchEnrollmentTokenFunc = func(_ context.Context, token *android.ZeroTouchToken) (*android.ZeroTouchToken, error) {
				assert.Equal(t, c.wantTeamID, token.TeamID)
				token.ID = 1
				return token, nil
			}

			resp, err := svc.GetZeroTouchConfiguration(premiumAdminCtx(t), c.fleetID)
			assert.Equal(t, c.wantTeamExistsCall, mockDS.TeamExistsFuncInvoked)
			assert.Equal(t, c.wantAMAPICall, apiClient.EnterprisesEnrollmentTokensCreateFuncInvoked)

			if c.wantNotFound {
				require.Error(t, err)
				var statusErr interface{ Status() int }
				require.ErrorAs(t, err, &statusErr)
				assert.Equal(t, http.StatusNotFound, statusErr.Status())
				assert.False(t, mockDS.GetZeroTouchEnrollmentTokenFuncInvoked)
				assert.False(t, mockDS.CreateZeroTouchEnrollmentTokenFuncInvoked)
				return
			}

			require.NoError(t, err)
			var extras dpcExtras
			require.NoError(t, json.Unmarshal(resp.DPCExtras, &extras))
			wantToken := "new-token-value"
			if c.hasExistingToken {
				wantToken = "existing-token-value"
			}
			assert.Equal(t, wantToken, extras.AdminExtrasBundle.EnrollmentToken)
			assert.Equal(t, !c.hasExistingToken, mockDS.CreateZeroTouchEnrollmentTokenFuncInvoked)
		})
	}
}

func TestGetZeroTouchConfiguration_FleetCheckedAfterLicense(t *testing.T) {
	svc, mockDS, _ := setupEEService(t)

	ctx := licensectx.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierFree})
	_, err := svc.GetZeroTouchConfiguration(ctx, new(uint(1)))
	require.ErrorIs(t, err, fleet.ErrMissingLicense)
	assert.False(t, mockDS.TeamExistsFuncInvoked)
}
