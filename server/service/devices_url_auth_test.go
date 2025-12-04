package service

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestCheckURLAuthQueryParam(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		req := &http.Request{URL: &url.URL{RawQuery: "udid=true"}}
		ctx := checkURLAuthQueryParam(context.Background(), req)
		require.True(t, urlAuthFromContext(ctx))
	})

	t.Run("absent", func(t *testing.T) {
		req := &http.Request{URL: &url.URL{RawQuery: ""}}
		ctx := checkURLAuthQueryParam(context.Background(), req)
		require.False(t, urlAuthFromContext(ctx))
	})

	t.Run("wrong value", func(t *testing.T) {
		req := &http.Request{URL: &url.URL{RawQuery: "udid=false"}}
		ctx := checkURLAuthQueryParam(context.Background(), req)
		require.False(t, urlAuthFromContext(ctx))
	})
}

func TestAuthenticatedDeviceURLAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)

	// Mock HostByIdentifier for URL auth
	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		if identifier == "valid-uuid" {
			return &fleet.Host{
				ID:       1,
				UUID:     "valid-uuid",
				Platform: "ios",
			}, nil
		}
		return nil, newNotFoundError()
	}

	// Mock AppConfig to avoid panic in debugEnabledForHost
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	middleware := authenticatedDevice(svc, log.NewNopLogger(), func(ctx context.Context, request interface{}) (interface{}, error) {
		return "success", nil
	})

	t.Run("success_with_query_param", func(t *testing.T) {
		// Simulate context with URL auth flag set (as if middleware ran)
		ctx := newURLAuthContext(context.Background())
		req := mockDeviceAuthRequest{Token: "valid-uuid"}
		_, err := middleware(ctx, req)
		require.NoError(t, err)
	})

	t.Run("failure_without_query_param", func(t *testing.T) {
		// Without the flag, it should try token auth and fail (since we didn't mock LoadHostByDeviceAuthToken)
		ds.LoadHostByDeviceAuthTokenFunc = func(ctx context.Context, authToken string, ttl time.Duration) (*fleet.Host, error) {
			return nil, newNotFoundError()
		}

		req := mockDeviceAuthRequest{Token: "valid-uuid"}
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

func TestAuthenticateDeviceByURL(t *testing.T) {
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

		host, debug, err := svc.AuthenticateDeviceByURL(context.Background(), "valid-uuid")
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

		host, debug, err := svc.AuthenticateDeviceByURL(context.Background(), "valid-uuid")
		require.NoError(t, err)
		require.False(t, debug)
		require.NotNil(t, host)
		require.Equal(t, uint(1), host.ID)
	})

	t.Run("error - missing host UUID", func(t *testing.T) {
		host, debug, err := svc.AuthenticateDeviceByURL(context.Background(), "")
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

		host, debug, err := svc.AuthenticateDeviceByURL(context.Background(), "invalid-uuid")
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

		host, debug, err := svc.AuthenticateDeviceByURL(context.Background(), "valid-uuid")
		require.Error(t, err)
		var authReqErr *fleet.AuthRequiredError
		require.ErrorAs(t, err, &authReqErr)
		require.Equal(t, "authentication error: URL authentication only supported for iOS and iPadOS devices", authReqErr.Internal())
		require.Nil(t, host)
		require.False(t, debug)
	})
}
