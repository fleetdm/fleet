package apple_mdm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestDEPService(t *testing.T) {
	t.Run("EnsureDefaultSetupAssistant", func(t *testing.T) {
		ds := new(mock.Store)
		ctx := context.Background()
		logger := log.NewNopLogger()
		depStorage := new(nanodep_mock.Storage)
		depSvc := NewDEPService(ds, depStorage, logger)
		defaultProfile := depSvc.getDefaultProfile()
		serverURL := "https://example.com/"

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			switch r.URL.Path {
			case "/session":
				_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
			case "/profile":
				_, _ = w.Write([]byte(`{"profile_uuid": "abcd"}`))
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
			default:
				require.Fail(t, "unexpected path: %s", r.URL.Path)
			}
		}))
		t.Cleanup(srv.Close)

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.ServerSettings.ServerURL = serverURL
			return appCfg, nil
		}

		var savedProfile *fleet.MDMAppleEnrollmentProfile
		ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, p fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
			require.Equal(t, fleet.MDMAppleEnrollmentTypeAutomatic, p.Type)
			require.NotEmpty(t, p.Token)
			res := &fleet.MDMAppleEnrollmentProfile{
				Token:      p.Token,
				Type:       p.Type,
				DEPProfile: p.DEPProfile,
			}
			savedProfile = res
			return res, nil
		}

		ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ fleet.MDMAppleEnrollmentType) (*fleet.MDMAppleEnrollmentProfile, error) {
			require.Equal(t, fleet.MDMAppleEnrollmentTypeAutomatic, typ)
			if savedProfile == nil {
				return nil, notFoundError{}
			}
			return savedProfile, nil
		}

		var defaultProfileUUID string
		ds.GetMDMAppleDefaultSetupAssistantFunc = func(ctx context.Context, teamID *uint) (profileUUID string, updatedAt time.Time, err error) {
			if defaultProfileUUID == "" {
				return "", time.Time{}, nil
			}
			return defaultProfileUUID, time.Now(), nil
		}

		ds.SetMDMAppleDefaultSetupAssistantProfileUUIDFunc = func(ctx context.Context, teamID *uint, profileUUID string) error {
			require.Nil(t, teamID)
			defaultProfileUUID = profileUUID
			return nil
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

		profUUID, modTime, err := depSvc.EnsureDefaultSetupAssistant(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, "abcd", profUUID)
		require.NotZero(t, modTime)
		require.True(t, ds.NewMDMAppleEnrollmentProfileFuncInvoked)
		require.True(t, ds.GetMDMAppleEnrollmentProfileByTypeFuncInvoked)
		require.True(t, ds.GetMDMAppleDefaultSetupAssistantFuncInvoked)
		require.True(t, ds.SetMDMAppleDefaultSetupAssistantProfileUUIDFuncInvoked)
		require.True(t, depStorage.RetrieveConfigFuncInvoked)
		require.False(t, depStorage.StoreAssignerProfileFuncInvoked) // not used anymore
	})

	t.Run("EnrollURL", func(t *testing.T) {
		const serverURL = "https://example.com/"

		appCfg := &fleet.AppConfig{}
		appCfg.ServerSettings.ServerURL = serverURL
		url, err := EnrollURL("token", appCfg)
		require.NoError(t, err)
		require.Equal(t, url, serverURL+"api/mdm/apple/enroll?token=token")
	})
}

func TestAddEnrollmentRefToFleetURL(t *testing.T) {
	const (
		baseFleetURL = "https://example.com"
		reference    = "enroll-ref"
	)

	tests := []struct {
		name           string
		fleetURL       string
		reference      string
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "empty Reference",
			fleetURL:       baseFleetURL,
			reference:      "",
			expectedOutput: baseFleetURL,
			expectError:    false,
		},
		{
			name:           "valid URL and Reference",
			fleetURL:       baseFleetURL,
			reference:      reference,
			expectedOutput: baseFleetURL + "?" + mobileconfig.FleetEnrollReferenceKey + "=" + reference,
			expectError:    false,
		},
		{
			name:        "invalid URL",
			fleetURL:    "://invalid-url",
			reference:   reference,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := AddEnrollmentRefToFleetURL(tc.fleetURL, tc.reference)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedOutput, output)
			}
		})
	}
}

type notFoundError struct{}

func (e notFoundError) IsNotFound() bool { return true }

func (e notFoundError) Error() string { return "not found" }

type mockAssigner struct {
	AssignProfileFuncInvoked bool
	AssignProfileFunc        func(ctx context.Context, name, uuid string, serials ...string) (*godep.ProfileResponse, error)
	DefineProfileFuncInvoked bool
	DefineProfileFunc        func(ctx context.Context, name string, profile *godep.Profile) (*godep.ProfileResponse, error)
}

func (m *mockAssigner) AssignProfile(ctx context.Context, name, uuid string, serials ...string) (*godep.ProfileResponse, error) {
	m.AssignProfileFuncInvoked = true
	return m.AssignProfileFunc(ctx, name, uuid, serials...)
}

func (m *mockAssigner) DefineProfile(ctx context.Context, name string, profile *godep.Profile) (*godep.ProfileResponse, error) {

	m.DefineProfileFuncInvoked = true
	return m.DefineProfileFunc(ctx, name, profile)
}

func TestProcessDeviceResponse(t *testing.T) {
	type testCase struct {
		name                    string
		deviceResponse          *godep.DeviceResponse
		expectedError           error
		expectedAddedDevices    []godep.Device
		expectedDeletedSerials  []string
		expectedModifiedSerials []string
	}

	testCases := []testCase{
		{
			name:                    "No devices in response",
			deviceResponse:          &godep.DeviceResponse{},
			expectedError:           nil,
			expectedAddedDevices:    nil,
			expectedDeletedSerials:  nil,
			expectedModifiedSerials: nil,
		},
		{
			name: "Added device",
			deviceResponse: &godep.DeviceResponse{
				Devices: []godep.Device{{OpType: "added", SerialNumber: "123"}},
			},
			expectedError:           nil,
			expectedAddedDevices:    []godep.Device{{OpType: "added", SerialNumber: "123"}},
			expectedDeletedSerials:  nil,
			expectedModifiedSerials: nil,
		},
		{
			name: "Modified device with existing serial",
			deviceResponse: &godep.DeviceResponse{
				Devices: []godep.Device{{OpType: "modified", SerialNumber: "456"}},
			},
			expectedError:           nil,
			expectedAddedDevices:    nil,
			expectedDeletedSerials:  nil,
			expectedModifiedSerials: []string{"456"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger := log.NewNopLogger()
			ds := new(mock.Store)
			depSvc := DEPService{logger: logger, ds: ds, depClient: &mockAssigner{}}

			// Mock necessary datastore and logger interactions
			// Example for modified devices:
			ds.GetMatchingHostSerialsFunc = func(ctx context.Context, serials []string) (map[string]*fleet.Host, error) {
				if len(serials) == 0 {
					return nil, nil
				}
				hostMap := make(map[string]*fleet.Host)
				for _, serial := range serials {
					hostMap[serial] = &fleet.Host{HardwareSerial: serial}
				}
				return hostMap, nil
			}

			ds.DeleteHostDEPAssignmentsFunc = func(ctx context.Context, serials []string) error {
				return nil
			}

			ds.IngestMDMAppleDevicesFromDEPSyncFunc = func(ctx context.Context, devices []godep.Device) (int64, *uint, error) {
				return 1, ptr.Uint(1), nil
			}

			ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
				return &fleet.Team{}, nil
			}

			ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
				return &fleet.MDMAppleSetupAssistant{Profile: json.RawMessage("{}")}, nil
			}

			ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ fleet.MDMAppleEnrollmentType) (*fleet.MDMAppleEnrollmentProfile, error) {
				return &fleet.MDMAppleEnrollmentProfile{Token: "tok-1", DEPProfile: ptr.RawMessage(json.RawMessage("{}"))}, nil
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{}, nil
			}

			ds.SetMDMAppleSetupAssistantProfileUUIDFunc = func(ctx context.Context, teamID *uint, profileUUID string) error {
				return nil
			}

			ds.ScreenDEPAssignProfileSerialsForCooldownFunc = func(ctx context.Context, serials []string) (skipSerials []string, assignSerials []string, err error) {
				return []string{}, []string{}, nil
			}

			ds.UpsertMDMAppleHostDEPAssignmentsFunc = func(ctx context.Context, hosts []fleet.Host) error {
				return nil
			}

			err := depSvc.processDeviceResponse(ctx, tc.deviceResponse)
			require.Equal(t, tc.expectedError, err)
		})
	}
}
