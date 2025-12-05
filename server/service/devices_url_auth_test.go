package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestAuthenticatedDeviceFallbackAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)

	// Mock AppConfig to avoid panic in debugEnabledForHost
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	middleware := authenticatedDevice(svc, log.NewNopLogger(), func(ctx context.Context, request interface{}) (interface{}, error) {
		return "success", nil
	})

	t.Run("success_token_auth_for_macos", func(t *testing.T) {
		// macOS device with valid token - token auth succeeds first (hot path)
		ds.LoadHostByDeviceAuthTokenFunc = func(ctx context.Context, authToken string, ttl time.Duration) (*fleet.Host, error) {
			if authToken == "valid-device-token" {
				return &fleet.Host{
					ID:       1,
					UUID:     "macos-device-uuid",
					Platform: "darwin",
				}, nil
			}
			return nil, newNotFoundError()
		}

		req := mockDeviceAuthRequest{Token: "valid-device-token"}
		_, err := middleware(context.Background(), req)
		require.NoError(t, err)
	})

	t.Run("fallback_to_uuid_auth_for_ios", func(t *testing.T) {
		// iOS device with UUID in URL - token auth fails, falls back to UUID auth
		ds.LoadHostByDeviceAuthTokenFunc = func(ctx context.Context, authToken string, ttl time.Duration) (*fleet.Host, error) {
			return nil, newNotFoundError()
		}

		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			if identifier == "ios-device-uuid" {
				return &fleet.Host{
					ID:       1,
					UUID:     "ios-device-uuid",
					Platform: "ios",
				}, nil
			}
			return nil, newNotFoundError()
		}

		req := mockDeviceAuthRequest{Token: "ios-device-uuid"}
		_, err := middleware(context.Background(), req)
		require.NoError(t, err)
	})

	t.Run("fallback_to_uuid_auth_for_ipados", func(t *testing.T) {
		// iPadOS device with UUID in URL - token auth fails, falls back to UUID auth
		ds.LoadHostByDeviceAuthTokenFunc = func(ctx context.Context, authToken string, ttl time.Duration) (*fleet.Host, error) {
			return nil, newNotFoundError()
		}

		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			if identifier == "ipados-device-uuid" {
				return &fleet.Host{
					ID:       2,
					UUID:     "ipados-device-uuid",
					Platform: "ipados",
				}, nil
			}
			return nil, newNotFoundError()
		}

		req := mockDeviceAuthRequest{Token: "ipados-device-uuid"}
		_, err := middleware(context.Background(), req)
		require.NoError(t, err)
	})

	t.Run("failure_when_both_auth_methods_fail", func(t *testing.T) {
		// Neither token nor UUID auth succeeds
		ds.LoadHostByDeviceAuthTokenFunc = func(ctx context.Context, authToken string, ttl time.Duration) (*fleet.Host, error) {
			return nil, newNotFoundError()
		}

		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return nil, newNotFoundError()
		}

		req := mockDeviceAuthRequest{Token: "invalid-token"}
		_, err := middleware(context.Background(), req)
		require.Error(t, err)
	})
}

type mockDeviceAuthRequest struct {
	Token string
}

func (m mockDeviceAuthRequest) deviceAuthToken() string {
	return m.Token
}

func TestAuthenticateIDeviceByURL(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)

	// Mock AppConfig to avoid panic in debugEnabledForHost
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	t.Run("success - valid UUID for iOS device", func(t *testing.T) {
		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{
				ID:       1,
				UUID:     "valid-uuid",
				Platform: "ios",
			}, nil
		}

		host, debug, err := svc.AuthenticateIDeviceByURL(context.Background(), "valid-uuid")
		require.NoError(t, err)
		require.False(t, debug)
		require.NotNil(t, host)
		require.Equal(t, uint(1), host.ID)
	})

	t.Run("success - valid UUID for iPadOS device", func(t *testing.T) {
		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{
				ID:       1,
				UUID:     "valid-uuid",
				Platform: "ipados",
			}, nil
		}

		host, debug, err := svc.AuthenticateIDeviceByURL(context.Background(), "valid-uuid")
		require.NoError(t, err)
		require.False(t, debug)
		require.NotNil(t, host)
		require.Equal(t, uint(1), host.ID)
	})

	t.Run("error - missing host UUID", func(t *testing.T) {
		host, debug, err := svc.AuthenticateIDeviceByURL(context.Background(), "")
		require.Error(t, err)
		var authReqErr *fleet.AuthRequiredError
		require.ErrorAs(t, err, &authReqErr)
		require.Equal(t, "authentication error: missing host UUID", authReqErr.Internal())
		require.Nil(t, host)
		require.False(t, debug)
	})

	t.Run("error - host not found", func(t *testing.T) {
		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return nil, newNotFoundError()
		}

		host, debug, err := svc.AuthenticateIDeviceByURL(context.Background(), "invalid-uuid")
		require.Error(t, err)
		var authReqErr *fleet.AuthRequiredError
		require.ErrorAs(t, err, &authReqErr)
		require.Contains(t, authReqErr.Internal(), "host not found")
		require.Nil(t, host)
		require.False(t, debug)
	})

	t.Run("error - host platform is not iOS or iPadOS (macOS)", func(t *testing.T) {
		ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
			return &fleet.Host{
				ID:       1,
				UUID:     "valid-uuid",
				Platform: "darwin",
			}, nil
		}

		host, debug, err := svc.AuthenticateIDeviceByURL(context.Background(), "valid-uuid")
		require.Error(t, err)
		var authReqErr *fleet.AuthRequiredError
		require.ErrorAs(t, err, &authReqErr)
		require.Equal(t, "authentication error: URL authentication only supported for iOS and iPadOS devices", authReqErr.Internal())
		require.Nil(t, host)
		require.False(t, debug)
	})
}
