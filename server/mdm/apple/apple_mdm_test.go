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
	"github.com/go-kit/log"
	"github.com/groob/plist"
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
				defaultProfile.AwaitDeviceConfigured = true // this is now always set to true
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
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
				},
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
		ds.GetMDMAppleDefaultSetupAssistantFunc = func(ctx context.Context, teamID *uint, orgName string) (profileUUID string, updatedAt time.Time, err error) {
			if defaultProfileUUID == "" {
				return "", time.Time{}, nil
			}
			return defaultProfileUUID, time.Now(), nil
		}

		ds.SetMDMAppleDefaultSetupAssistantProfileUUIDFunc = func(ctx context.Context, teamID *uint, profileUUID, orgName string) error {
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
			require.NotEmpty(t, profileUUID)
			return nil
		}

		ds.GetABMTokenOrgNamesAssociatedWithTeamFunc = func(ctx context.Context, teamID *uint) ([]string, error) {
			return []string{"org1"}, nil
		}

		ds.CountABMTokensWithTermsExpiredFunc = func(ctx context.Context) (int, error) {
			return 0, nil
		}

		profUUID, modTime, err := depSvc.EnsureDefaultSetupAssistant(ctx, nil, "org1")
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

func TestGenerateEnrollmentProfileMobileconfig(t *testing.T) {
	type scepPayload struct {
		Challenge string
		URL       string
	}

	type enrollmentPayload struct {
		PayloadType    string
		ServerURL      string      // used by the enrollment payload
		PayloadContent scepPayload // scep contains a nested payload content dict
	}

	type enrollmentProfile struct {
		PayloadIdentifier string
		PayloadContent    []enrollmentPayload
	}

	tests := []struct {
		name          string
		orgName       string
		fleetURL      string
		scepChallenge string
		expectError   bool
	}{
		{
			name:          "valid input with simple values",
			orgName:       "Fleet",
			fleetURL:      "https://example.com",
			scepChallenge: "testChallenge",
			expectError:   false,
		},
		{
			name:          "organization name and enroll secret with special characters",
			orgName:       `Fleet & Co. "Special" <Org>`,
			fleetURL:      "https://example.com",
			scepChallenge: "test/&Challenge",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateEnrollmentProfileMobileconfig(tt.orgName, tt.fleetURL, tt.scepChallenge, "com.foo.bar")
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				var profile enrollmentProfile

				require.NoError(t, plist.Unmarshal(result, &profile))

				for _, p := range profile.PayloadContent {
					switch p.PayloadType {
					case "com.apple.security.scep":
						scepURL, err := ResolveAppleSCEPURL(tt.fleetURL)
						require.NoError(t, err)
						require.Equal(t, scepURL, p.PayloadContent.URL)
						require.Equal(t, tt.scepChallenge, p.PayloadContent.Challenge)
					case "com.apple.mdm":
						mdmURL, err := ResolveAppleMDMURL(tt.fleetURL)
						require.NoError(t, err)
						require.Contains(t, mdmURL, p.ServerURL)
					default:
						require.Failf(t, "unrecognized payload type in enrollment profile: %s", p.PayloadType)
					}
				}
			}
		})
	}
}

type notFoundError struct{}

func (e notFoundError) IsNotFound() bool { return true }

func (e notFoundError) Error() string { return "not found" }
