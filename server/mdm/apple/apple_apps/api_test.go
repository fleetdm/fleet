package apple_apps

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetBaseURL(t *testing.T) {
	t.Run("Default URL", func(t *testing.T) {
		os.Setenv("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", "")
		require.Equal(t, "https://fleetdm.com/api/vpp/v1/metadata/us?platform=iphone&additionalPlatforms=ipad,mac&extend[apps]=latestVersionInfo", getBaseURL())
	})

	t.Run("Custom URL", func(t *testing.T) {
		customURL := "http://localhost:8000"
		os.Setenv("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", customURL)
		require.Equal(t, customURL, getBaseURL())
	})
}

func setupFakeServer(t *testing.T, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	os.Setenv("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", server.URL)
	t.Cleanup(server.Close)
}

func TestGetMetadataRetries(t *testing.T) {
	t.Run("successful on first attempt", func(t *testing.T) {
		var callCount int
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			resp := metadataResp{
				Data: []Metadata{
					{ID: "123", Attributes: Attributes{Name: "Test App"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		})

		result, err := GetMetadata([]string{"123"}, "vppToken", func(bool) (string, error) {
			return "bearer-token", nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, callCount)
		require.Len(t, result, 1)
		require.Equal(t, "Test App", result["123"].Attributes.Name)
	})

	t.Run("retries on 500 error and succeeds", func(t *testing.T) {
		var callCount int
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("server error"))
				return
			}
			w.WriteHeader(http.StatusOK)
			resp := metadataResp{
				Data: []Metadata{
					{ID: "456", Attributes: Attributes{Name: "Retry App"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		})

		result, err := GetMetadata([]string{"456"}, "vppToken", func(bool) (string, error) {
			return "bearer-token", nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, callCount)
		require.Len(t, result, 1)
		require.Equal(t, "Retry App", result["456"].Attributes.Name)
	})

	t.Run("exhausts retries on persistent 500 error", func(t *testing.T) {
		var callCount int
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("persistent server error"))
		})

		_, err := GetMetadata([]string{"789"}, "vppToken", func(bool) (string, error) {
			return "bearer-token", nil
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "retrieving asset metadata")
		// Should have retried 3 times (max attempts)
		require.Equal(t, 3, callCount)
	})

	t.Run("does not retry on auth error", func(t *testing.T) {
		var callCount int
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		})

		_, err := GetMetadata([]string{"999"}, "vppToken", func(forceRenew bool) (string, error) {
			// Always return the same token to simulate auth failure
			return "invalid-token", nil
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "auth")
		// Should have called twice: initial + one retry with forceRenew, then bail
		require.Equal(t, 2, callCount)
	})

	t.Run("returns multiple apps", func(t *testing.T) {
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			resp := metadataResp{
				Data: []Metadata{
					{ID: "111", Attributes: Attributes{Name: "App One"}},
					{ID: "222", Attributes: Attributes{Name: "App Two"}},
					{ID: "333", Attributes: Attributes{Name: "App Three"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		})

		result, err := GetMetadata([]string{"111", "222", "333"}, "vppToken", func(bool) (string, error) {
			return "bearer-token", nil
		})
		require.NoError(t, err)
		require.Len(t, result, 3)
		require.Equal(t, "App One", result["111"].Attributes.Name)
		require.Equal(t, "App Two", result["222"].Attributes.Name)
		require.Equal(t, "App Three", result["333"].Attributes.Name)
	})
}

// mockDataStore implements the DataStore interface for testing GetAuthenticator
type mockDataStore struct {
	appConfig                   *fleet.AppConfig
	appConfigErr                error
	assets                      map[fleet.MDMAssetName]fleet.MDMConfigAsset
	getAssetsErr                error
	insertedAsset               *fleet.MDMConfigAsset
	hardDeletedAsset            fleet.MDMAssetName
	insertOrReplaceCalled       bool
	hardDeleteCalled            bool
	getAssetsByNameCalled       bool
	insertMDMConfigAssetsCalled bool
}

func (m *mockDataStore) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	return m.appConfig, m.appConfigErr
}

func (m *mockDataStore) InsertMDMConfigAssets(ctx context.Context, assets []fleet.MDMConfigAsset, tx sqlx.ExtContext) error {
	m.insertMDMConfigAssetsCalled = true
	return nil
}

func (m *mockDataStore) InsertOrReplaceMDMConfigAsset(ctx context.Context, asset fleet.MDMConfigAsset) error {
	m.insertOrReplaceCalled = true
	m.insertedAsset = &asset
	return nil
}

func (m *mockDataStore) GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName, queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
	m.getAssetsByNameCalled = true
	if m.getAssetsErr != nil {
		return nil, m.getAssetsErr
	}
	return m.assets, nil
}

func (m *mockDataStore) GetAllMDMConfigAssetsHashes(ctx context.Context, assetNames []fleet.MDMAssetName) (map[fleet.MDMAssetName]string, error) {
	return nil, nil
}

func (m *mockDataStore) DeleteMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName) error {
	return nil
}

func (m *mockDataStore) HardDeleteMDMConfigAsset(ctx context.Context, assetName fleet.MDMAssetName) error {
	m.hardDeleteCalled = true
	m.hardDeletedAsset = assetName
	return nil
}

func (m *mockDataStore) ReplaceMDMConfigAssets(ctx context.Context, assets []fleet.MDMConfigAsset, tx sqlx.ExtContext) error {
	return nil
}

func (m *mockDataStore) GetAllCAConfigAssetsByType(ctx context.Context, assetType fleet.CAConfigAssetType) (map[string]fleet.CAConfigAsset, error) {
	return nil, nil
}

func TestAuthentication(t *testing.T) {
	// Clear any dev env vars that might interfere
	originalDevToken := os.Getenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN")
	originalAuthURL := os.Getenv("FLEET_DEV_VPP_PROXY_AUTH_URL")
	t.Cleanup(func() {
		os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", originalDevToken)
		os.Setenv("FLEET_DEV_VPP_PROXY_AUTH_URL", originalAuthURL)
	})

	t.Run("uses bearer token env var when set", func(t *testing.T) {
		os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", "dev-test-token")
		defer os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", "")

		ds := &mockDataStore{}
		auth := GetAuthenticator(context.Background(), ds, "license-key")

		// Should return bearer token regardless of forceRenew
		token, err := auth(false)
		require.NoError(t, err)
		require.Equal(t, "dev-test-token", token)

		token, err = auth(true)
		require.NoError(t, err)
		require.Equal(t, "dev-test-token", token)

		// Should not have accessed the datastore
		require.False(t, ds.getAssetsByNameCalled)
	})

	t.Run("returns cached token from database when not forced renewal", func(t *testing.T) {
		os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", "")

		ds := &mockDataStore{
			assets: map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetVPPProxyBearerToken: {
					Name:  fleet.MDMAssetVPPProxyBearerToken,
					Value: []byte("cached-token-from-db"),
				},
			},
		}

		auth := GetAuthenticator(context.Background(), ds, "license-key")
		token, err := auth(false)
		require.NoError(t, err)
		require.Equal(t, "cached-token-from-db", token)
		require.True(t, ds.getAssetsByNameCalled)
		require.False(t, ds.insertOrReplaceCalled)
	})

	t.Run("requests new token when forced renewal even if cached exists", func(t *testing.T) {
		os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", "")

		// Set up a mock auth server
		authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify the license key is in the Authorization header
			require.Equal(t, "Bearer test-license-key", r.Header.Get("Authorization"))

			// Verify the URL is set
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.JSONEq(t, `{"serverUrl": "https://fleet.example.com"}`, string(body))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"fleetServerSecret": "new-token-from-auth"}`))
		}))
		defer authServer.Close()
		os.Setenv("FLEET_DEV_VPP_PROXY_AUTH_URL", authServer.URL)

		ds := &mockDataStore{
			assets: map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetVPPProxyBearerToken: {
					Name:  fleet.MDMAssetVPPProxyBearerToken,
					Value: []byte("cached-token-from-db"),
				},
			},
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://fleet.example.com",
				},
			},
		}

		auth := GetAuthenticator(context.Background(), ds, "test-license-key")
		token, err := auth(true) // Force renewal
		require.NoError(t, err)
		require.Equal(t, "new-token-from-auth", token)
		// Should not have checked DB since forceRenew=true
		require.False(t, ds.getAssetsByNameCalled)
		// Should have stored the new token
		require.True(t, ds.insertOrReplaceCalled)
		require.Equal(t, fleet.MDMAssetVPPProxyBearerToken, ds.insertedAsset.Name)
		require.Equal(t, []byte("new-token-from-auth"), ds.insertedAsset.Value)
		// Should have deleted the old token
		require.True(t, ds.hardDeleteCalled)
		require.Equal(t, fleet.MDMAssetVPPProxyBearerToken, ds.hardDeletedAsset)
	})

	t.Run("requests new token when nothing in database and no forced renewal", func(t *testing.T) {
		os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", "")

		// Set up a mock auth server
		authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "Bearer my-license-key", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"fleetServerSecret": "fresh-token"}`))
		}))
		defer authServer.Close()
		os.Setenv("FLEET_DEV_VPP_PROXY_AUTH_URL", authServer.URL)

		ds := &mockDataStore{
			assets: map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, // Empty - no cached token
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://fleet.example.com",
				},
			},
		}

		auth := GetAuthenticator(context.Background(), ds, "my-license-key")
		token, err := auth(false) // Not forced renewal, but no token in DB
		require.NoError(t, err)
		require.Equal(t, "fresh-token", token)
		// Should have checked DB first
		require.True(t, ds.getAssetsByNameCalled)
		// Should have stored the new token
		require.True(t, ds.insertOrReplaceCalled)
		require.Equal(t, fleet.MDMAssetVPPProxyBearerToken, ds.insertedAsset.Name)
		require.Equal(t, []byte("fresh-token"), ds.insertedAsset.Value)
	})

	t.Run("returns error when auth server fails", func(t *testing.T) {
		os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", "")

		// Set up a mock auth server that fails
		authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "invalid license"}`))
		}))
		defer authServer.Close()
		os.Setenv("FLEET_DEV_VPP_PROXY_AUTH_URL", authServer.URL)

		ds := &mockDataStore{
			assets: map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, // Empty
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://fleet.example.com",
				},
			},
		}

		auth := GetAuthenticator(context.Background(), ds, "bad-license-key")
		_, err := auth(false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authenticating to VPP metadata service")
	})

	t.Run("returns error when auth response has empty token", func(t *testing.T) {
		os.Setenv("FLEET_DEV_VPP_METADATA_BEARER_TOKEN", "")

		// Set up a mock auth server that returns empty token
		authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"fleetServerSecret": ""}`))
		}))
		defer authServer.Close()
		os.Setenv("FLEET_DEV_VPP_PROXY_AUTH_URL", authServer.URL)

		ds := &mockDataStore{
			assets: map[fleet.MDMAssetName]fleet.MDMConfigAsset{},
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://fleet.example.com",
				},
			},
		}

		auth := GetAuthenticator(context.Background(), ds, "license-key")
		_, err := auth(false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no access token received")
	})
}

func TestDoRetries(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantCalls   int
		wantErr     bool
		wantMinTime time.Duration
	}{
		{
			name: "success status code",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "bad requests",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "500 requests does not retry (handled upstream)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "auth fail makes another attempt",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer foo" {
					w.WriteHeader(http.StatusUnauthorized)
				} else {
					w.WriteHeader(http.StatusOK)
				}

				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 2,
			wantErr:   false,
		},
		{
			name: "429 with retry-after header waits and retries",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls:   2,
			wantErr:     false,
			wantMinTime: 1 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls < tt.wantCalls {
					tt.handler(w, r)
					return
				}
			})

			start := time.Now()
			req, err := http.NewRequest(http.MethodGet, os.Getenv("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL"), nil)
			require.NoError(t, err)
			err = do(req, "vppToken", func(forceRenew bool) (string, error) {
				if forceRenew {
					return "foo", nil
				}
				return "", nil
			}, false, nil)
			require.NoError(t, err)
			require.Equal(t, tt.wantCalls, calls)
			elapsed := time.Since(start)
			require.WithinRange(t, time.Now(), start, start.Add(time.Duration(tt.wantCalls)*time.Second+tt.wantMinTime))
			if tt.wantMinTime > 0 {
				require.GreaterOrEqual(t, elapsed, tt.wantMinTime, "expected to wait at least %v for retry-after", tt.wantMinTime)
			}
		})
	}
}
