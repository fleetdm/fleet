package apple_mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDEPService(t *testing.T) {
	t.Run("EnsureDefaultSetupAssistant", func(t *testing.T) {
		ds := new(mock.Store)
		ctx := context.Background()
		logger := slog.New(slog.DiscardHandler)
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
				assert.Empty(t, got.ConfigurationWebURL)
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

func TestValidateMDMSettingsAppleSupportedOSVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// load the test data from the file
		b, err := os.ReadFile("./gdmf/testdata/gdmf.json")
		require.NoError(t, err)
		_, err = w.Write(b)
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)
	dev_mode.SetOverride("FLEET_DEV_GDMF_URL", srv.URL, t)

	// selected versions of from testdata/gdmf.json that we'll use in out tests
	expectSupportedMacOSPublic := []string{"14.6.1", "13.6.9", "12.7.6", "11.7.10"}
	expectSupportedMacOSNonPublic := []string{"14.5", "14.6", "13.6.8", "13.6.7", "12.7.5"}
	expectSupportedIOSPublic := "17.6.1"
	expectSupportedIOSNonPublic := "17.5.1"
	// lastIPodSupportedVersion     : = "15.8.3"

	// helper function to initialize app config MDM settings with known good versions (tests will modify as needed)
	mockAppConfigMDM := func() fleet.MDM {
		return fleet.MDM{
			MacOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(expectSupportedMacOSPublic[0]),
			},
			IOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(expectSupportedIOSPublic),
			},
			IPadOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(expectSupportedIOSPublic),
			},
		}
	}
	// helper function to initialize team MDM settings with known good versions (tests will modify as needed)
	mockTeamMDM := func() fleet.TeamMDM {
		return fleet.TeamMDM{
			MacOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(expectSupportedMacOSPublic[0]),
			},
			IOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(expectSupportedIOSPublic),
			},
			IPadOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(expectSupportedIOSPublic),
			},
		}
	}

	// helper function to check if the error matches expectations for a given platform and log appropriately
	checkErr := func(platform string, wantErr string, gotErrs map[string]error, msg string) {
		key := fmt.Sprintf("mdm.%s_updates.minimum_version", platform)
		if wantErr == "" {
			assert.Empty(t, gotErrs, msg+": expected no error for platform %s but got: %v", platform, gotErrs)
		} else {
			assert.Len(t, gotErrs, 1, msg+": expected error for platform %s but got no errors", platform)
			assert.Contains(t, gotErrs, key, msg+": expected error for platform %s but got no error", platform)
			assert.ErrorContains(t, gotErrs[key], wantErr, msg+": expected error for platform %s but got: %v", platform, gotErrs[key])
		}
	}

	t.Run("macos", func(t *testing.T) {
		t.Run("app config mdm settings", func(t *testing.T) {
			ac := mockAppConfigMDM()
			for _, v := range expectSupportedMacOSPublic {
				ac.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				checkErr("macos", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect public macOS version to be supported when including non-public asset sets")
				checkErr("macos", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect public macOS version to be supported when excluding non-public asset sets")
			}
			for _, v := range expectSupportedMacOSNonPublic {
				ac.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				checkErr("macos", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect non-public macOS version to be supported when including non-public asset sets")
				checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect non-public macOS version to return error when excluding non-public asset sets")
			}

			ac.MacOSUpdates.MinimumVersion = optjson.SetString("11.7.9") // not supported in either asset set, so we expect an error in both cases
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect unsupported macOS version to return error when including non-public asset sets")
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect unsupported macOS version to return error when excluding non-public asset sets")
		})
		t.Run("team mdm settings", func(t *testing.T) {
			tm := mockTeamMDM()
			for _, v := range expectSupportedMacOSPublic {
				tm.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				checkErr("macos", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect public macOS version to be supported when including non-public asset sets")
				checkErr("macos", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect public macOS version to be supported when excluding non-public asset sets")
			}
			for _, v := range expectSupportedMacOSNonPublic {
				tm.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				checkErr("macos", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect non-public macOS version to be supported when including non-public asset sets")
				checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect non-public macOS version to return error when excluding non-public asset sets")
			}

			tm.MacOSUpdates.MinimumVersion = optjson.SetString("11.7.9") // not supported in either asset set, so we expect an error in both cases
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect unsupported macOS version to return error when including non-public asset sets")
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect unsupported macOS version to return error when excluding non-public asset sets")
		})
	})

	t.Run("ios", func(t *testing.T) {
		t.Run("app config mdm settings", func(t *testing.T) {
			ac := mockAppConfigMDM()
			ac.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			checkErr("ios", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect public iOS version to be supported when including non-public asset sets")
			checkErr("ios", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect public iOS version to be supported when excluding non-public asset sets")

			ac.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			checkErr("ios", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect non-public iOS version to be supported when including non-public asset sets")
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect non-public iOS version to return error when excluding non-public asset sets")

			ac.IOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect unsupported iOS version to return error when including non-public asset sets")
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect unsupported iOS version to return error when excluding non-public asset sets")
		})

		t.Run("team mdm settings", func(t *testing.T) {
			tm := mockTeamMDM()
			tm.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			checkErr("ios", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect public iOS version to be supported when including non-public asset sets")
			checkErr("ios", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect public iOS version to be supported when excluding non-public asset sets")

			tm.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			checkErr("ios", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect non-public iOS version to be supported when including non-public asset sets")
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect non-public iOS version to return error when excluding non-public asset sets")

			tm.IOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect unsupported iOS version to return error when including non-public asset sets")
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect unsupported iOS version to return error when excluding non-public asset sets")
		})
	})

	t.Run("ipados", func(t *testing.T) {
		t.Run("app config mdm settings", func(t *testing.T) {
			ac := mockAppConfigMDM()
			ac.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			checkErr("ipados", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect public iPadOS version to be supported when including non-public asset sets")
			checkErr("ipados", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect public iPadOS version to be supported when excluding non-public asset sets")

			ac.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			checkErr("ipados", "", ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect non-public iPadOS version to be supported when including non-public asset sets")
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect non-public iPadOS version to return error when excluding non-public asset sets")

			ac.IPadOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, false), "expect unsupported iPadOS version to return error when including non-public asset sets")
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(ac, true), "expect unsupported iPadOS version to return error when excluding non-public asset sets")
		})

		t.Run("team mdm settings", func(t *testing.T) {
			tm := mockTeamMDM()
			tm.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			checkErr("ipados", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect public iPadOS version to be supported when including non-public asset sets")
			checkErr("ipados", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect public iPadOS version to be supported when excluding non-public asset sets")

			tm.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			checkErr("ipados", "", ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect non-public iPadOS version to be supported when including non-public asset sets")
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect non-public iPadOS version to return error when excluding non-public asset sets")

			tm.IPadOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, false), "expect unsupported iPadOS version to return error when including non-public asset sets")
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, ValidateMDMSettingsAppleSupportedOSVersion(tm, true), "expect unsupported iPadOS version to return error when excluding non-public asset sets")
		})
	})
}

type notFoundError struct{}

func (e notFoundError) IsNotFound() bool { return true }

func (e notFoundError) Error() string { return "not found" }
