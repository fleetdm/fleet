package apple_mdm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
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
		defaultProfile := depSvc.GetDefaultProfile()
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
	checkErr := func(platform string, wantErr string, gotErrs map[string]string, msg string) {
		if wantErr == "" {
			assert.Empty(t, gotErrs, msg+": expected no error for platform %s but got: %v", platform, gotErrs)
		} else {
			assert.Len(t, gotErrs, 1, msg+": expected error for platform %s but got no errors", platform)
			assert.Contains(t, gotErrs, platform, msg+": expected error for platform %s but key not found in error map: %v", platform, gotErrs)
			assert.Contains(t, gotErrs[platform], wantErr, msg+": expected error for platform %s but got: %v", platform, gotErrs[platform])
		}
	}

	t.Run("macos", func(t *testing.T) {
		t.Run("app config mdm settings", func(t *testing.T) {
			ac := mockAppConfigMDM()
			for _, v := range expectSupportedMacOSPublic {
				ac.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				got, err := ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
				require.NoError(t, err)
				checkErr("macos", "", got, "expect public macOS version to be supported when including non-public asset sets")
				got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
				require.NoError(t, err)
				checkErr("macos", "", got, "expect public macOS version to be supported when excluding non-public asset sets")
			}
			for _, v := range expectSupportedMacOSNonPublic {
				ac.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				got, err := ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
				require.NoError(t, err)
				checkErr("macos", "", got, "expect non-public macOS version to be supported when including non-public asset sets")
				got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
				require.NoError(t, err)
				checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, got, "expect non-public macOS version to return error when excluding non-public asset sets")
			}

			ac.MacOSUpdates.MinimumVersion = optjson.SetString("11.7.9") // not supported in either asset set, so we expect an error in both cases
			got, err := ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
			require.NoError(t, err)
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported macOS version to return error when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
			require.NoError(t, err)
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported macOS version to return error when excluding non-public asset sets")
		})
		t.Run("team mdm settings", func(t *testing.T) {
			tm := mockTeamMDM()
			for _, v := range expectSupportedMacOSPublic {
				tm.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				got, err := ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
				require.NoError(t, err)
				checkErr("macos", "", got, "expect public macOS version to be supported when including non-public asset sets")
				got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
				require.NoError(t, err)
				checkErr("macos", "", got, "expect public macOS version to be supported when excluding non-public asset sets")
			}
			for _, v := range expectSupportedMacOSNonPublic {
				tm.MacOSUpdates.MinimumVersion = optjson.SetString(v)
				got, err := ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
				require.NoError(t, err)
				checkErr("macos", "", got, "expect non-public macOS version to be supported when including non-public asset sets")
				got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
				require.NoError(t, err)
				checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, got, "expect non-public macOS version to return error when excluding non-public asset sets")
			}

			tm.MacOSUpdates.MinimumVersion = optjson.SetString("11.7.9") // not supported in either asset set, so we expect an error in both cases
			got, err := ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
			require.NoError(t, err)
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported macOS version to return error when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
			require.NoError(t, err)
			checkErr("macos", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported macOS version to return error when excluding non-public asset sets")
		})
	})

	t.Run("ios", func(t *testing.T) {
		t.Run("app config mdm settings", func(t *testing.T) {
			ac := mockAppConfigMDM()
			ac.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			got, err := ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
			require.NoError(t, err)
			checkErr("ios", "", got, "expect public iOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
			require.NoError(t, err)
			checkErr("ios", "", got, "expect public iOS version to be supported when excluding non-public asset sets")

			ac.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
			require.NoError(t, err)
			checkErr("ios", "", got, "expect non-public iOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
			require.NoError(t, err)
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, got, "expect non-public iOS version to return error when excluding non-public asset sets")

			ac.IOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
			require.NoError(t, err)
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iOS version to return error when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
			require.NoError(t, err)
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iOS version to return error when excluding non-public asset sets")
		})

		t.Run("team mdm settings", func(t *testing.T) {
			tm := mockTeamMDM()
			tm.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			got, err := ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
			require.NoError(t, err)
			checkErr("ios", "", got, "expect public iOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
			require.NoError(t, err)
			checkErr("ios", "", got, "expect public iOS version to be supported when excluding non-public asset sets")

			tm.IOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
			require.NoError(t, err)
			checkErr("ios", "", got, "expect non-public iOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
			require.NoError(t, err)
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, got, "expect non-public iOS version to return error when excluding non-public asset sets")

			tm.IOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
			require.NoError(t, err)
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iOS version to return error when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
			require.NoError(t, err)
			checkErr("ios", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iOS version to return error when excluding non-public asset sets")
		})
	})

	t.Run("ipados", func(t *testing.T) {
		t.Run("app config mdm settings", func(t *testing.T) {
			ac := mockAppConfigMDM()
			ac.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			got, err := ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
			require.NoError(t, err)
			checkErr("ipados", "", got, "expect public iPadOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
			require.NoError(t, err)
			checkErr("ipados", "", got, "expect public iPadOS version to be supported when excluding non-public asset sets")

			ac.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
			require.NoError(t, err)
			checkErr("ipados", "", got, "expect non-public iPadOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
			require.NoError(t, err)
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, got, "expect non-public iPadOS version to return error when excluding non-public asset sets")

			ac.IPadOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
			require.NoError(t, err)
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iPadOS version to return error when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(ac, true)
			require.NoError(t, err)
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iPadOS version to return error when excluding non-public asset sets")
		})

		t.Run("team mdm settings", func(t *testing.T) {
			tm := mockTeamMDM()
			tm.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSPublic)
			got, err := ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
			require.NoError(t, err)
			checkErr("ipados", "", got, "expect public iPadOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
			require.NoError(t, err)
			checkErr("ipados", "", got, "expect public iPadOS version to be supported when excluding non-public asset sets")

			tm.IPadOSUpdates.MinimumVersion = optjson.SetString(expectSupportedIOSNonPublic)
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
			require.NoError(t, err)
			checkErr("ipados", "", got, "expect non-public iPadOS version to be supported when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
			require.NoError(t, err)
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, got, "expect non-public iPadOS version to return error when excluding non-public asset sets")

			tm.IPadOSUpdates.MinimumVersion = optjson.SetString("5.3.9") // only supported for Apple Watch, so we expect an error
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
			require.NoError(t, err)
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iPadOS version to return error when including non-public asset sets")
			got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, true)
			require.NoError(t, err)
			checkErr("ipados", fleet.AppleOSVersionUnsupportedMessage, got, "expect unsupported iPadOS version to return error when excluding non-public asset sets")
		})
	})

	// These subtests are placed last so that the dev_mode override cleanup for the error server
	// doesn't interfere with the earlier subtests that rely on the valid mock server.
	t.Run("GetAssetMetadata error", func(t *testing.T) {
		// Use a server that returns invalid JSON so GetAssetMetadata returns an error.
		errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("not valid json"))
			require.NoError(t, err)
		}))
		t.Cleanup(errSrv.Close)
		dev_mode.SetOverride("FLEET_DEV_GDMF_URL", errSrv.URL, t)

		ac := mockAppConfigMDM()
		got, err := ValidateMDMSettingsAppleSupportedOSVersion(ac, false)
		require.Error(t, err)
		assert.Nil(t, got)

		tm := mockTeamMDM()
		got, err = ValidateMDMSettingsAppleSupportedOSVersion(tm, false)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

type notFoundError struct{}

func (e notFoundError) IsNotFound() bool { return true }

func (e notFoundError) Error() string { return "not found" }

func TestGenerateRecoveryLockPassword(t *testing.T) {
	// Pattern: 6 groups of 4 characters from the allowed charset, separated by dashes
	pattern := regexp.MustCompile(`^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{4}(-[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{4}){5}$`)

	t.Run("format", func(t *testing.T) {
		password := GenerateRecoveryLockPassword()
		assert.True(t, pattern.MatchString(password), "password %q does not match expected format", password)
		assert.Len(t, password, 29) // 24 chars + 5 dashes
	})

	t.Run("excludes confusing characters", func(t *testing.T) {
		// Generate multiple passwords and check none contain confusing chars
		confusingChars := regexp.MustCompile(`[01OIl]`)
		for range 100 {
			password := GenerateRecoveryLockPassword()
			assert.False(t, confusingChars.MatchString(password), "password %q contains confusing characters", password)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		// Generate multiple passwords and verify they're unique
		seen := make(map[string]bool)
		for range 100 {
			password := GenerateRecoveryLockPassword()
			assert.False(t, seen[password], "duplicate password generated: %s", password)
			seen[password] = true
		}
	})
}

// TestSendRecoveryLockCommands tests the cron job that sends SetRecoveryLock commands
// to hosts that need recovery lock passwords.
//
// Note: SetRecoveryLock command results are handled synchronously in the MDM results handler
// (server/service/apple_mdm.go), which is tested separately in apple_mdm_cmd_results_test.go.
func TestSendRecoveryLockCommands(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("no hosts needing recovery lock does not send commands", func(t *testing.T) {
		ds := new(mock.Store)
		// Mock restore - no hosts to restore
		ds.RestoreRecoveryLockForReenabledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		ds.SoftDeleteRecoveryLockPasswordsForUnenrolledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		ds.GetHostsForRecoveryLockActionFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}
		// Mock clear flow - no hosts need clearing
		ds.ClaimHostsForRecoveryLockClearFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}
		// Mock auto-rotation - no hosts need auto-rotation
		ds.GetHostsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostAutoRotationInfo, error) {
			return nil, nil
		}

		var commandSent bool
		mockCommander := &mockRecoveryLockCommander{
			setRecoveryLockFn: func(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
				commandSent = true
				return nil
			},
		}

		err := sendRecoveryLockCommandsWithCommander(ctx, ds, mockCommander, logger, nil)
		require.NoError(t, err)
		assert.False(t, commandSent, "SetRecoveryLock should not be called when no hosts need it")
	})

	t.Run("host needing recovery lock gets SetRecoveryLock and password stored with pending status", func(t *testing.T) {
		ds := new(mock.Store)

		// Mock restore - no hosts to restore
		ds.RestoreRecoveryLockForReenabledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		ds.SoftDeleteRecoveryLockPasswordsForUnenrolledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		hostUUID := "host-uuid-1"
		ds.GetHostsForRecoveryLockActionFunc = func(ctx context.Context) ([]string, error) {
			return []string{hostUUID}, nil
		}
		// Mock clear flow - no hosts need clearing
		ds.ClaimHostsForRecoveryLockClearFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}
		// Mock auto-rotation - no hosts need auto-rotation
		ds.GetHostsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostAutoRotationInfo, error) {
			return nil, nil
		}

		// Track call order to verify correct sequencing
		var callOrder []string
		var storedPasswords []fleet.HostRecoveryLockPasswordPayload
		ds.SetHostsRecoveryLockPasswordsFunc = func(ctx context.Context, passwords []fleet.HostRecoveryLockPasswordPayload) error {
			callOrder = append(callOrder, "SetHostsRecoveryLockPasswords")
			storedPasswords = passwords
			return nil
		}

		var sentCmdUUID string
		mockCommander := &mockRecoveryLockCommander{
			setRecoveryLockFn: func(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
				callOrder = append(callOrder, "SetRecoveryLock")
				assert.Equal(t, []string{hostUUID}, hostUUIDs)
				sentCmdUUID = cmdUUID
				return nil
			},
		}

		err := sendRecoveryLockCommandsWithCommander(ctx, ds, mockCommander, logger, nil)
		require.NoError(t, err)

		// Verify call order: password must be stored BEFORE command is sent
		require.Equal(t, []string{"SetHostsRecoveryLockPasswords", "SetRecoveryLock"}, callOrder,
			"SetHostsRecoveryLockPasswords must be called before SetRecoveryLock")

		// Password should be stored with pending status atomically
		require.Len(t, storedPasswords, 1, "password should be stored for host")
		assert.Equal(t, hostUUID, storedPasswords[0].HostUUID)
		assert.NotEmpty(t, storedPasswords[0].Password)
		assert.NotEmpty(t, sentCmdUUID, "command UUID should have been sent")
	})

	t.Run("SetRecoveryLock failure clears pending status to allow retry", func(t *testing.T) {
		ds := new(mock.Store)

		// Mock restore - no hosts to restore
		ds.RestoreRecoveryLockForReenabledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		ds.SoftDeleteRecoveryLockPasswordsForUnenrolledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		hostUUID := "host-uuid-1"
		ds.GetHostsForRecoveryLockActionFunc = func(ctx context.Context) ([]string, error) {
			return []string{hostUUID}, nil
		}
		// Mock clear flow - no hosts need clearing
		ds.ClaimHostsForRecoveryLockClearFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}
		// Mock auto-rotation - no hosts need auto-rotation
		ds.GetHostsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostAutoRotationInfo, error) {
			return nil, nil
		}

		// Track call order to verify correct sequencing
		var callOrder []string
		ds.SetHostsRecoveryLockPasswordsFunc = func(ctx context.Context, passwords []fleet.HostRecoveryLockPasswordPayload) error {
			callOrder = append(callOrder, "SetHostsRecoveryLockPasswords")
			return nil
		}

		var clearedHostUUIDs []string
		ds.ClearRecoveryLockPendingStatusFunc = func(ctx context.Context, hostUUIDs []string) error {
			callOrder = append(callOrder, "ClearRecoveryLockPendingStatus")
			clearedHostUUIDs = hostUUIDs
			return nil
		}

		mockCommander := &mockRecoveryLockCommander{
			setRecoveryLockFn: func(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
				callOrder = append(callOrder, "SetRecoveryLock")
				return errors.New("APNs push failed")
			},
		}

		err := sendRecoveryLockCommandsWithCommander(ctx, ds, mockCommander, logger, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "APNs push failed")

		// Verify call order: password stored -> command attempt -> clear pending on failure
		require.Equal(t, []string{"SetHostsRecoveryLockPasswords", "SetRecoveryLock", "ClearRecoveryLockPendingStatus"}, callOrder,
			"Operations must occur in order: store password, attempt command, clear pending on failure")

		// Status should be cleared to allow retry on next cron run
		assert.Equal(t, []string{hostUUID}, clearedHostUUIDs, "pending status should be cleared on enqueue failure")
	})

	t.Run("APNs delivery failure does not clear pending status", func(t *testing.T) {
		ds := new(mock.Store)

		// Mock restore - no hosts to restore
		ds.RestoreRecoveryLockForReenabledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		ds.SoftDeleteRecoveryLockPasswordsForUnenrolledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		hostUUID := "host-uuid-1"
		ds.GetHostsForRecoveryLockActionFunc = func(ctx context.Context) ([]string, error) {
			return []string{hostUUID}, nil
		}
		// Mock clear flow - no hosts need clearing
		ds.ClaimHostsForRecoveryLockClearFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}
		// Mock auto-rotation - no hosts need auto-rotation
		ds.GetHostsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostAutoRotationInfo, error) {
			return nil, nil
		}

		// Track call order to verify ClearRecoveryLockPendingStatus is NOT called
		var callOrder []string
		ds.SetHostsRecoveryLockPasswordsFunc = func(ctx context.Context, passwords []fleet.HostRecoveryLockPasswordPayload) error {
			callOrder = append(callOrder, "SetHostsRecoveryLockPasswords")
			return nil
		}

		ds.ClearRecoveryLockPendingStatusFunc = func(ctx context.Context, hostUUIDs []string) error {
			callOrder = append(callOrder, "ClearRecoveryLockPendingStatus")
			return nil
		}

		mockCommander := &mockRecoveryLockCommander{
			setRecoveryLockFn: func(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
				callOrder = append(callOrder, "SetRecoveryLock")
				// Return APNs delivery error - command was persisted but push failed
				return &APNSDeliveryError{errorsByUUID: map[string]error{hostUUID: errors.New("push failed")}}
			},
		}

		err := sendRecoveryLockCommandsWithCommander(ctx, ds, mockCommander, logger, nil)
		// Should NOT return error - command was persisted, just push failed
		require.NoError(t, err)

		// Verify ClearRecoveryLockPendingStatus was NOT called (status should stay pending)
		// Command is queued and will be delivered when device checks in
		require.Equal(t, []string{"SetHostsRecoveryLockPasswords", "SetRecoveryLock"}, callOrder,
			"ClearRecoveryLockPendingStatus should NOT be called when APNs push fails (command is already queued)")
	})
}

// mockRecoveryLockCommander implements RecoveryLockCommander for testing.
type mockRecoveryLockCommander struct {
	setRecoveryLockFn    func(ctx context.Context, hostUUIDs []string, cmdUUID string) error
	clearRecoveryLockFn  func(ctx context.Context, hostUUIDs []string, cmdUUID string) error
	rotateRecoveryLockFn func(ctx context.Context, hostUUID string, cmdUUID string) error
}

func (m *mockRecoveryLockCommander) SetRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	if m.setRecoveryLockFn != nil {
		return m.setRecoveryLockFn(ctx, hostUUIDs, cmdUUID)
	}
	return nil
}

func (m *mockRecoveryLockCommander) ClearRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	if m.clearRecoveryLockFn != nil {
		return m.clearRecoveryLockFn(ctx, hostUUIDs, cmdUUID)
	}
	return nil
}

func (m *mockRecoveryLockCommander) RotateRecoveryLock(ctx context.Context, hostUUID string, cmdUUID string) error {
	if m.rotateRecoveryLockFn != nil {
		return m.rotateRecoveryLockFn(ctx, hostUUID, cmdUUID)
	}
	return nil
}

func TestSendClearRecoveryLockCommands(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)

	t.Run("hosts needing clear get ClearRecoveryLock command", func(t *testing.T) {
		ds := new(mock.Store)

		// Mock restore - no hosts to restore
		ds.RestoreRecoveryLockForReenabledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		ds.SoftDeleteRecoveryLockPasswordsForUnenrolledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		// No hosts need SET
		ds.GetHostsForRecoveryLockActionFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}

		hostUUID := "host-uuid-1"
		// ClaimHostsForRecoveryLockClear queries verified hosts where config is disabled and marks them pending
		ds.ClaimHostsForRecoveryLockClearFunc = func(ctx context.Context) ([]string, error) {
			return []string{hostUUID}, nil
		}
		// Mock auto-rotation - no hosts need auto-rotation
		ds.GetHostsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostAutoRotationInfo, error) {
			return nil, nil
		}

		var clearCalled bool
		mockCommander := &mockRecoveryLockCommander{
			clearRecoveryLockFn: func(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
				clearCalled = true
				require.Equal(t, []string{hostUUID}, hostUUIDs)
				return nil
			},
		}

		err := sendRecoveryLockCommandsWithCommander(ctx, ds, mockCommander, logger, nil)
		require.NoError(t, err)
		require.True(t, clearCalled, "ClearRecoveryLock should have been called")
	})

	t.Run("no hosts needing clear does not send commands", func(t *testing.T) {
		ds := new(mock.Store)

		// Mock restore - no hosts to restore
		ds.RestoreRecoveryLockForReenabledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		ds.SoftDeleteRecoveryLockPasswordsForUnenrolledHostsFunc = func(ctx context.Context) (int64, error) {
			return 0, nil
		}
		// No hosts need SET
		ds.GetHostsForRecoveryLockActionFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}

		ds.ClaimHostsForRecoveryLockClearFunc = func(ctx context.Context) ([]string, error) {
			return nil, nil
		}
		// Mock auto-rotation - no hosts need auto-rotation
		ds.GetHostsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostAutoRotationInfo, error) {
			return nil, nil
		}

		mockCommander := &mockRecoveryLockCommander{
			clearRecoveryLockFn: func(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
				t.Fatal("ClearRecoveryLock should not be called when no hosts need clearing")
				return nil
			},
		}

		err := sendRecoveryLockCommandsWithCommander(ctx, ds, mockCommander, logger, nil)
		require.NoError(t, err)
	})
}

// mockManagedLocalAccountCommander implements ManagedLocalAccountRotationCommander for testing.
type mockManagedLocalAccountCommander struct {
	setAutoAdminPasswordFn func(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error
}

func (m *mockManagedLocalAccountCommander) SetAutoAdminPassword(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error {
	if m.setAutoAdminPasswordFn != nil {
		return m.setAutoAdminPasswordFn(ctx, hostUUID, guid, passwordHashPlist, cmdUUID)
	}
	return nil
}

func TestSendManagedLocalAccountRotationCommands(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("no hosts due", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetManagedLocalAccountsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostManagedLocalAccountAutoRotationInfo, error) {
			return nil, nil
		}
		mockCommander := &mockManagedLocalAccountCommander{
			setAutoAdminPasswordFn: func(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error {
				t.Fatal("SetAutoAdminPassword must not be called when no hosts are due")
				return nil
			},
		}
		require.NoError(t, sendManagedLocalAccountRotationCommandsWithCommander(ctx, ds, mockCommander, logger, nil))
	})

	t.Run("rotates and logs activity for view-driven row", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetManagedLocalAccountsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostManagedLocalAccountAutoRotationInfo, error) {
			return []fleet.HostManagedLocalAccountAutoRotationInfo{
				{HostUUID: "uuid-1", HostID: 1, DisplayName: "host-1", AccountUUID: "acct-1", InitiatedByFleet: true},
			}, nil
		}
		var initiateCalled, setCalled bool
		ds.InitiateManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID, plaintextPassword, cmdUUID string) error {
			initiateCalled = true
			require.Equal(t, "uuid-1", hostUUID)
			require.NotEmpty(t, plaintextPassword)
			require.NotEmpty(t, cmdUUID)
			return nil
		}
		mockCommander := &mockManagedLocalAccountCommander{
			setAutoAdminPasswordFn: func(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error {
				setCalled = true
				require.Equal(t, "uuid-1", hostUUID)
				require.Equal(t, "acct-1", guid)
				require.NotEmpty(t, passwordHashPlist)
				return nil
			},
		}
		var loggedActivities []fleet.ActivityDetails
		newActivityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			loggedActivities = append(loggedActivities, activity)
			require.Nil(t, user, "auto-rotation must log with nil user (Fleet actor)")
			return nil
		}
		require.NoError(t, sendManagedLocalAccountRotationCommandsWithCommander(ctx, ds, mockCommander, logger, newActivityFn))
		require.True(t, initiateCalled)
		require.True(t, setCalled)
		require.Len(t, loggedActivities, 1)
		rotated, ok := loggedActivities[0].(fleet.ActivityTypeRotatedManagedLocalAccountPassword)
		require.True(t, ok)
		require.True(t, rotated.FleetInitiated)
	})

	t.Run("does NOT log activity for deferred manual rotation", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetManagedLocalAccountsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostManagedLocalAccountAutoRotationInfo, error) {
			return []fleet.HostManagedLocalAccountAutoRotationInfo{
				{HostUUID: "uuid-2", HostID: 2, DisplayName: "host-2", AccountUUID: "acct-2", InitiatedByFleet: false},
			}, nil
		}
		ds.InitiateManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID, plaintextPassword, cmdUUID string) error {
			return nil
		}
		mockCommander := &mockManagedLocalAccountCommander{}
		var loggedActivities []fleet.ActivityDetails
		newActivityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			loggedActivities = append(loggedActivities, activity)
			return nil
		}
		require.NoError(t, sendManagedLocalAccountRotationCommandsWithCommander(ctx, ds, mockCommander, logger, newActivityFn))
		require.Empty(t, loggedActivities, "deferred manual rotations were already logged at click time and must not be re-logged by the cron")
	})

	t.Run("benign race - rotation pending - skipped silently", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetManagedLocalAccountsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostManagedLocalAccountAutoRotationInfo, error) {
			return []fleet.HostManagedLocalAccountAutoRotationInfo{
				{HostUUID: "uuid-3", HostID: 3, DisplayName: "host-3", AccountUUID: "acct-3", InitiatedByFleet: true},
			}, nil
		}
		ds.InitiateManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID, plaintextPassword, cmdUUID string) error {
			return fleet.ErrManagedLocalAccountRotationPending
		}
		mockCommander := &mockManagedLocalAccountCommander{
			setAutoAdminPasswordFn: func(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error {
				t.Fatal("SetAutoAdminPassword must not be called when initiate failed with benign race")
				return nil
			},
		}
		require.NoError(t, sendManagedLocalAccountRotationCommandsWithCommander(ctx, ds, mockCommander, logger, nil))
	})

	t.Run("APNs error keeps pending state and still logs activity", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetManagedLocalAccountsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostManagedLocalAccountAutoRotationInfo, error) {
			return []fleet.HostManagedLocalAccountAutoRotationInfo{
				{HostUUID: "uuid-4", HostID: 4, DisplayName: "host-4", AccountUUID: "acct-4", InitiatedByFleet: true},
			}, nil
		}
		ds.InitiateManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID, plaintextPassword, cmdUUID string) error {
			return nil
		}
		var clearCalled bool
		ds.ClearManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID string) error {
			clearCalled = true
			return nil
		}
		mockCommander := &mockManagedLocalAccountCommander{
			setAutoAdminPasswordFn: func(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error {
				return &APNSDeliveryError{errorsByUUID: map[string]error{hostUUID: errors.New("push failed")}}
			},
		}
		var loggedActivities []fleet.ActivityDetails
		newActivityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			loggedActivities = append(loggedActivities, activity)
			return nil
		}
		require.NoError(t, sendManagedLocalAccountRotationCommandsWithCommander(ctx, ds, mockCommander, logger, newActivityFn))
		require.False(t, clearCalled, "APNs failures must not roll back pending rotation — the command is persisted")
		require.Len(t, loggedActivities, 1)
	})

	t.Run("persistence error rolls back pending rotation and skips activity", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetManagedLocalAccountsForAutoRotationFunc = func(ctx context.Context) ([]fleet.HostManagedLocalAccountAutoRotationInfo, error) {
			return []fleet.HostManagedLocalAccountAutoRotationInfo{
				{HostUUID: "uuid-5", HostID: 5, DisplayName: "host-5", AccountUUID: "acct-5", InitiatedByFleet: true},
			}, nil
		}
		ds.InitiateManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID, plaintextPassword, cmdUUID string) error {
			return nil
		}
		var clearCalled bool
		ds.ClearManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID string) error {
			clearCalled = true
			return nil
		}
		mockCommander := &mockManagedLocalAccountCommander{
			setAutoAdminPasswordFn: func(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error {
				return errors.New("storage write failed")
			},
		}
		var loggedActivities []fleet.ActivityDetails
		newActivityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			loggedActivities = append(loggedActivities, activity)
			return nil
		}
		err := sendManagedLocalAccountRotationCommandsWithCommander(ctx, ds, mockCommander, logger, newActivityFn)
		require.Error(t, err)
		require.True(t, clearCalled)
		require.Empty(t, loggedActivities)
	})
}
