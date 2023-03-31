package apple_mdm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/go-kit/log"
	"github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	"github.com/stretchr/testify/require"
)

func TestDEPService(t *testing.T) {
	t.Run("CreateDefaultProfile", func(t *testing.T) {
		ds := new(mock.Store)
		ctx := context.Background()
		logger := log.NewNopLogger()
		depStorage := new(nanodep_mock.Storage)
		depSvc := NewDEPService(ds, depStorage, logger, true)
		defaultProfile := depSvc.GetDefaultProfile()
		serverURL := "https://example.com/"

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			switch r.URL.Path {
			case "/session":
				_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
			case "/profile":
				_, _ = w.Write([]byte(`{"profile_uuid": "xyz"}`))
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var got godep.Profile
				err = json.Unmarshal(body, &got)
				require.NoError(t, err)
				require.Contains(t, got.URL, serverURL+"api/mdm/apple/enroll?token=")
				require.Contains(t, got.ConfigurationWebURL, serverURL+"api/mdm/apple/enroll?token=")
				got.URL = ""
				got.ConfigurationWebURL = ""
				require.Equal(t, defaultProfile, &got)
			}
		}))
		t.Cleanup(srv.Close)

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.ServerSettings.ServerURL = serverURL
			return appCfg, nil
		}

		ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, p fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
			require.Equal(t, fleet.MDMAppleEnrollmentType("automatic"), p.Type)
			require.NotEmpty(t, p.Token)
			//require.JSONEq
			return &fleet.MDMAppleEnrollmentProfile{}, nil

		}

		ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
			return nil
		}

		depStorage.RetrieveConfigFunc = func(ctx context.Context, name string) (*client.Config, error) {
			return &client.Config{BaseURL: srv.URL}, nil
		}

		depStorage.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*client.OAuth1Tokens, error) {
			return &client.OAuth1Tokens{}, nil
		}

		depStorage.StoreAssignerProfileFunc = func(ctx context.Context, name string, profileUUID string) error {
			require.Equal(t, name, DEPName)
			require.NotEmpty(t, profileUUID)
			return nil
		}

		err := depSvc.CreateDefaultProfile(ctx)
		require.NoError(t, err)
		require.True(t, ds.NewMDMAppleEnrollmentProfileFuncInvoked)
		require.True(t, depStorage.RetrieveConfigFuncInvoked)
		require.True(t, depStorage.StoreAssignerProfileFuncInvoked)
	})

	t.Run("EnrollURL", func(t *testing.T) {
		ds := new(mock.Store)
		logger := log.NewNopLogger()
		depStorage := new(nanodep_mock.Storage)
		depSvc := NewDEPService(ds, depStorage, logger, true)
		testErr := errors.New("test")
		serverURL := "https://example.com/"

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return nil, testErr
		}

		url, err := depSvc.EnrollURL("token")
		require.ErrorIs(t, err, testErr)
		require.Empty(t, url)
		require.True(t, ds.AppConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.ServerSettings.ServerURL = " http://foo.com"
			return appCfg, nil
		}

		url, err = depSvc.EnrollURL("token")
		require.Error(t, err)
		require.Empty(t, url)
		require.True(t, ds.AppConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.ServerSettings.ServerURL = serverURL
			return appCfg, nil
		}

		url, err = depSvc.EnrollURL("token")
		require.NoError(t, err)
		require.Equal(t, url, serverURL+"api/mdm/apple/enroll?token=token")
		require.True(t, ds.AppConfigFuncInvoked)

	})
}
