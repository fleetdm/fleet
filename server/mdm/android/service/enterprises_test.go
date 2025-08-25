package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	ds_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

func TestEnterprisesAuth(t *testing.T) {
	androidAPIClient := android_mock.Client{}
	androidAPIClient.InitCommonMocks()
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	fleetDS := InitCommonDSMocks()
	fleetSvc := mockService{}
	svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
	require.NoError(t, err)

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			true,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			true,
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, err := svc.GetEnterprise(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

			err = svc.DeleteEnterprise(ctx)
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.EnterpriseSignup(ctx)
			checkAuthErr(t, tt.shouldFailWrite, err)

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			_, err = svc.EnterpriseSignupSSE(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}

	t.Run("unauthorized", func(t *testing.T) {
		err := svc.EnterpriseSignupCallback(context.Background(), "signup_token", "token")
		checkAuthErr(t, false, err)
		err = svc.EnterpriseSignupCallback(context.Background(), "bad_token", "token")
		checkAuthErr(t, true, err)
	})
}

func TestEnterpriseSignupMissingPrivateKey(t *testing.T) {
	androidAPIClient := android_mock.Client{}
	androidAPIClient.InitCommonMocks()
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	fleetDS := InitCommonDSMocks()
	fleetSvc := mockService{}

	svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
	require.NoError(t, err)

	user := &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})

	testSetEmptyPrivateKey = true
	t.Cleanup(func() { testSetEmptyPrivateKey = false })

	_, err = svc.EnterpriseSignup(ctx)
	require.Error(t, err)

	require.Contains(t, err.Error(), "missing required private key")
	require.Contains(t, err.Error(), "https://fleetdm.com/learn-more-about/fleet-server-private-key")
}

func checkAuthErr(t *testing.T, shouldFail bool, err error) {
	t.Helper()
	if shouldFail {
		require.Error(t, err)
		var forbiddenError *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenError)
	} else {
		require.NoError(t, err)
	}
}

func InitCommonDSMocks() *AndroidMockDS {
	ds := AndroidMockDS{}
	// Set up basic Android datastore mocks directly on the Fleet datastore mock
	ds.Store.CreateEnterpriseFunc = func(ctx context.Context, _ uint) (uint, error) {
		return 1, nil
	}
	ds.Store.UpdateEnterpriseFunc = func(ctx context.Context, enterprise *android.EnterpriseDetails) error {
		return nil
	}
	ds.Store.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{}, nil
	}
	ds.Store.GetEnterpriseByIDFunc = func(ctx context.Context, ID uint) (*android.EnterpriseDetails, error) {
		return &android.EnterpriseDetails{}, nil
	}
	ds.Store.GetEnterpriseBySignupTokenFunc = func(ctx context.Context, signupToken string) (*android.EnterpriseDetails, error) {
		if signupToken == "signup_token" {
			return &android.EnterpriseDetails{}, nil
		}
		return nil, &notFoundError{}
	}
	ds.Store.DeleteAllEnterprisesFunc = func(ctx context.Context) error {
		return nil
	}
	ds.Store.DeleteOtherEnterprisesFunc = func(ctx context.Context, ID uint) error {
		return nil
	}

	ds.Store.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.Store.SetAndroidEnabledAndConfiguredFunc = func(_ context.Context, configured bool) error {
		return nil
	}
	ds.Store.UserOrDeletedUserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id}, nil
	}
	ds.Store.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		queryerContext sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		result := make(map[fleet.MDMAssetName]fleet.MDMConfigAsset, len(assetNames))
		for _, name := range assetNames {
			result[name] = fleet.MDMConfigAsset{Value: []byte("value")}
		}
		return result, nil
	}
	ds.Store.InsertOrReplaceMDMConfigAssetFunc = func(ctx context.Context, asset fleet.MDMConfigAsset) error {
		return nil
	}
	ds.Store.DeleteMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName) error {
		return nil
	}
	ds.Store.BulkSetAndroidHostsUnenrolledFunc = func(ctx context.Context) error {
		return nil
	}
	return &ds
}

type AndroidMockDS struct {
	ds_mock.Store
}

type notFoundError struct{}

func (e *notFoundError) Error() string    { return "not found" }
func (e *notFoundError) IsNotFound() bool { return true }

type mockService struct {
	mock.Mock
	fleet.Service
}

// NewActivity mocks the fleet.Service method.
func (m *mockService) NewActivity(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
	return nil
}

func TestGetEnterprise(t *testing.T) {
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	user := &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})

	t.Run("normal enterprise retrieval works", func(t *testing.T) {
		androidAPIClient := android_mock.Client{}
		androidAPIClient.InitCommonMocks()

		fleetDS := InitCommonDSMocks()
		fleetSvc := mockService{}
		svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
		require.NoError(t, err)

		enterprise, err := svc.GetEnterprise(ctx)
		require.NoError(t, err)
		require.NotNil(t, enterprise)
		assert.True(t, androidAPIClient.EnterpriseGetFuncInvoked)
	})

	t.Run("deleted enterprise returns 404", func(t *testing.T) {
		androidAPIClient := android_mock.Client{}
		androidAPIClient.InitCommonMocks()

		// Override EnterpriseGetFunc to simulate deleted enterprise (403 from Google)
		androidAPIClient.EnterpriseGetFunc = func(_ context.Context, enterpriseName string) (*androidmanagement.Enterprise, error) {
			return nil, &googleapi.Error{
				Code:    http.StatusForbidden,
				Message: "Caller is not authorized to manage enterprise.",
			}
		}

		fleetDS := InitCommonDSMocks()
		fleetSvc := mockService{}
		svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
		require.NoError(t, err)

		enterprise, err := svc.GetEnterprise(ctx)
		require.Error(t, err)
		require.Nil(t, enterprise)
		assert.True(t, androidAPIClient.EnterpriseGetFuncInvoked)

		// Check that it's a Fleet error with proper message and verify status code if available
		assert.Contains(t, err.Error(), "Android Enterprise has been deleted")

		// Check if the error implements the StatusCode interface
		if statusErr, ok := err.(interface{ StatusCode() int }); ok {
			assert.Equal(t, http.StatusNotFound, statusErr.StatusCode())
		} else if statusErr, ok := err.(interface{ Status() int }); ok {
			assert.Equal(t, http.StatusNotFound, statusErr.Status())
		}
	})

	t.Run("Google API authentication failures", func(t *testing.T) {
		androidAPIClient := android_mock.Client{}
		androidAPIClient.InitCommonMocks()

		// Override EnterpriseGetFunc to simulate authentication failure
		androidAPIClient.EnterpriseGetFunc = func(_ context.Context, enterpriseName string) (*androidmanagement.Enterprise, error) {
			return nil, &googleapi.Error{
				Code:    http.StatusUnauthorized,
				Message: "Request had invalid authentication credentials.",
			}
		}

		fleetDS := InitCommonDSMocks()
		fleetSvc := mockService{}
		svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
		require.NoError(t, err)

		enterprise, err := svc.GetEnterprise(ctx)
		require.Error(t, err)
		require.Nil(t, enterprise)
		assert.True(t, androidAPIClient.EnterpriseGetFuncInvoked)

		// Should NOT be treated as enterprise deleted - should be wrapped error
		assert.Contains(t, err.Error(), "verifying enterprise with Google")
		assert.Contains(t, err.Error(), "Request had invalid authentication credentials")
	})

	t.Run("unexpected Google API errors", func(t *testing.T) {
		androidAPIClient := android_mock.Client{}
		androidAPIClient.InitCommonMocks()

		// Override EnterpriseGetFunc to simulate unexpected error
		androidAPIClient.EnterpriseGetFunc = func(_ context.Context, enterpriseName string) (*androidmanagement.Enterprise, error) {
			return nil, &googleapi.Error{
				Code:    http.StatusInternalServerError,
				Message: "Internal server error occurred.",
			}
		}

		fleetDS := InitCommonDSMocks()
		fleetSvc := mockService{}
		svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
		require.NoError(t, err)

		enterprise, err := svc.GetEnterprise(ctx)
		require.Error(t, err)
		require.Nil(t, enterprise)
		assert.True(t, androidAPIClient.EnterpriseGetFuncInvoked)

		// Should be wrapped as verification error
		assert.Contains(t, err.Error(), "verifying enterprise with Google")
		assert.Contains(t, err.Error(), "Internal server error occurred")
	})

	t.Run("non-googleapi errors", func(t *testing.T) {
		androidAPIClient := android_mock.Client{}
		androidAPIClient.InitCommonMocks()

		// Override EnterpriseGetFunc to simulate non-googleapi error
		androidAPIClient.EnterpriseGetFunc = func(_ context.Context, enterpriseName string) (*androidmanagement.Enterprise, error) {
			return nil, errors.New("network timeout")
		}

		fleetDS := InitCommonDSMocks()
		fleetSvc := mockService{}
		svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
		require.NoError(t, err)

		enterprise, err := svc.GetEnterprise(ctx)
		require.Error(t, err)
		require.Nil(t, enterprise)
		assert.True(t, androidAPIClient.EnterpriseGetFuncInvoked)

		// Should be wrapped as verification error
		assert.Contains(t, err.Error(), "verifying enterprise with Google")
		assert.Contains(t, err.Error(), "network timeout")
	})

	t.Run("enterprise not found in database", func(t *testing.T) {
		androidAPIClient := android_mock.Client{}
		androidAPIClient.InitCommonMocks()

		// Override GetEnterpriseFunc to return not found
		fleetDS := InitCommonDSMocks()
		fleetDS.Store.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
			return nil, &notFoundError{}
		}

		fleetSvc := mockService{}
		svc, err := NewServiceWithClient(logger, fleetDS, &androidAPIClient, &fleetSvc, "test-private-key")
		require.NoError(t, err)

		enterprise, err := svc.GetEnterprise(ctx)
		require.Error(t, err)
		require.Nil(t, enterprise)

		// Should not call Google API if enterprise not found in database
		assert.False(t, androidAPIClient.EnterpriseGetFuncInvoked)
		assert.Contains(t, err.Error(), "No enterprise found")
	})
}
