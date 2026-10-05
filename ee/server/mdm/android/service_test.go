package android

import (
	"context"
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
	assert.Contains(t, resp.DPCExtras, "existing-token-value")
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
	assert.Contains(t, resp.DPCExtras, "new-token-value")
	assert.Contains(t, resp.DPCExtras, "EXTRA_ENROLLMENT_TOKEN")
	assert.True(t, mockDS.CreateZeroTouchEnrollmentTokenFuncInvoked)
	assert.True(t, apiClient.EnterprisesEnrollmentTokensCreateFuncInvoked)
}

type notFoundError struct{}

func (e *notFoundError) Error() string    { return "not found" }
func (e *notFoundError) IsNotFound() bool { return true }
