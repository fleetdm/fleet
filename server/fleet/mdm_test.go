package fleet_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestDEPClient(t *testing.T) {
	ctx := context.Background()

	rxToken := regexp.MustCompile(`oauth_token="(\w+)"`)
	const (
		validToken                 = "OK"
		invalidToken               = "FAIL"
		termsChangedToken          = "TERMS"
		termsChangedAfterAuthToken = "TERMS_AFTER"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session" {
			matches := rxToken.FindStringSubmatch(r.Header.Get("Authorization"))
			require.NotNil(t, matches)
			token := matches[1]

			switch token {
			case validToken:
				_, _ = w.Write([]byte(`{"auth_session_token": "ok"}`))
			case termsChangedAfterAuthToken:
				_, _ = w.Write([]byte(`{"auth_session_token": "fail"}`))
			case termsChangedToken:
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"code": "T_C_NOT_SIGNED"}`))
			case invalidToken:
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"code": "ACCESS_DENIED"}`))
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}
			return
		}

		require.Equal(t, "/account", r.URL.Path)
		authSsn := r.Header.Get("X-Adm-Auth-Session")
		if authSsn == "fail" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code": "T_C_NOT_SIGNED"}`))
			return
		}

		// otherwise, return account information, details not important for this
		// test.
		_, _ = w.Write([]byte(`{"admin_id": "test"}`))
	}))
	defer srv.Close()

	logger := log.NewNopLogger()
	ds := new(mock.Store)

	appCfg := fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &appCfg, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		appCfg = *config
		return nil
	}

	checkDSCalled := func(readInvoked, writeInvoked bool) {
		require.Equal(t, readInvoked, ds.AppConfigFuncInvoked)
		require.Equal(t, writeInvoked, ds.SaveAppConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false
		ds.SaveAppConfigFuncInvoked = false
	}

	cases := []struct {
		token        string
		wantErr      bool
		readInvoked  bool
		writeInvoked bool
		termsFlag    bool
	}{
		// use a valid token, appconfig should not be updated (already unflagged)
		{token: validToken, wantErr: false, readInvoked: true, writeInvoked: false, termsFlag: false},

		// use an invalid token, appconfig should not even be read (not a terms error)
		{token: invalidToken, wantErr: true, readInvoked: false, writeInvoked: false, termsFlag: false},

		// terms changed during the auth request
		{token: termsChangedToken, wantErr: true, readInvoked: true, writeInvoked: true, termsFlag: true},

		// use of an invalid token does not update the flag
		{token: invalidToken, wantErr: true, readInvoked: false, writeInvoked: false, termsFlag: true},

		// use of a valid token resets the flag
		{token: validToken, wantErr: false, readInvoked: true, writeInvoked: true, termsFlag: false},

		// use of a valid token again does not update the appConfig
		{token: validToken, wantErr: false, readInvoked: true, writeInvoked: false, termsFlag: false},

		// terms changed during the actual account request, after auth
		{token: termsChangedAfterAuthToken, wantErr: true, readInvoked: true, writeInvoked: true, termsFlag: true},

		// again terms changed after auth, doesn't update appConfig
		{token: termsChangedAfterAuthToken, wantErr: true, readInvoked: true, writeInvoked: false, termsFlag: true},

		// terms changed during auth, doesn't update appConfig
		{token: termsChangedToken, wantErr: true, readInvoked: true, writeInvoked: false, termsFlag: true},

		// valid token, resets the flag
		{token: validToken, wantErr: false, readInvoked: true, writeInvoked: true, termsFlag: false},
	}

	// TODO(mna): update test to check with different tokens the effect on the terms_expired flags.

	// order of calls is important, and test must not be parallelized as it would
	// be racy. For that reason, subtests are not used (it would make it possible
	// to run one subtest in isolation, which could fail).
	for i, c := range cases {
		t.Logf("case %d", i)

		store := &nanodep_mock.Storage{}
		store.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
			return &nanodep_client.OAuth1Tokens{AccessToken: c.token}, nil
		}
		store.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
			return &nanodep_client.Config{BaseURL: srv.URL}, nil
		}

		dep := apple_mdm.NewDEPClient(store, ds, logger)
		res, err := dep.AccountDetail(ctx, apple_mdm.DEPName)

		if c.wantErr {
			var httpErr *godep.HTTPError
			require.Error(t, err)
			if errors.As(err, &httpErr) {
				require.Equal(t, http.StatusForbidden, httpErr.StatusCode)
			} else {
				var authErr *nanodep_client.AuthError
				require.ErrorAs(t, err, &authErr)
				require.Equal(t, http.StatusForbidden, authErr.StatusCode)
			}
			if c.token == termsChangedToken || c.token == termsChangedAfterAuthToken {
				require.True(t, godep.IsTermsNotSigned(err))
			} else {
				require.False(t, godep.IsTermsNotSigned(err))
			}
		} else {
			require.NoError(t, err)
			require.Equal(t, "test", res.AdminID)
			require.True(t, store.RetrieveAuthTokensFuncInvoked)
			require.True(t, store.RetrieveConfigFuncInvoked)
		}
		checkDSCalled(c.readInvoked, c.writeInvoked)
		require.Equal(t, c.termsFlag, appCfg.MDM.AppleBMTermsExpired)
	}
}

func TestMDMAppleBootstrapPackage(t *testing.T) {
	bp := &fleet.MDMAppleBootstrapPackage{
		Token: "abc-def",
	}

	url, err := bp.URL("http://example.com")
	require.NoError(t, err)
	require.Equal(t, "http://example.com/api/latest/fleet/mdm/bootstrap?token=abc-def", url)

	url, err = bp.URL(" http://example.com")
	require.Empty(t, url)
	require.Error(t, err)
}

func TestMDMProfileSpecUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		expectPath   string
		expectLabels []string
		expectError  bool
	}{
		{
			name:         "empty input",
			input:        []byte(""),
			expectPath:   "",
			expectLabels: nil,
			expectError:  false,
		},
		{
			name:         "new format",
			input:        []byte(`{"path": "testpath", "labels": ["label1", "label2"]}`),
			expectPath:   "testpath",
			expectLabels: []string{"label1", "label2"},
			expectError:  false,
		},
		{
			name:         "old format",
			input:        []byte(`"oldpath"`),
			expectPath:   "oldpath",
			expectLabels: nil,
			expectError:  false,
		},
		{
			name:         "invalid JSON",
			input:        []byte(`{invalid json}`),
			expectPath:   "",
			expectLabels: nil,
			expectError:  true,
		},
		{
			name:         "valid JSON with extra fields",
			input:        []byte(`{"path": "testpath", "labels": ["label1"], "extra": "field"}`),
			expectPath:   "testpath",
			expectLabels: []string{"label1"},
			expectError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p fleet.MDMProfileSpec
			err := p.UnmarshalJSON(tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectPath, p.Path)
				require.Equal(t, tc.expectLabels, p.Labels)
			}
		})
	}

	t.Run("complex scenario", func(t *testing.T) {
		var p fleet.MDMProfileSpec
		// test new format
		data := []byte(`{"path": "newpath", "labels": ["label1", "label2"]}`)
		err := p.UnmarshalJSON(data)
		require.NoError(t, err)
		require.Equal(t, "newpath", p.Path)
		require.Equal(t, []string{"label1", "label2"}, p.Labels)

		// test old format
		p = fleet.MDMProfileSpec{}
		data = []byte(`"oldpath"`)
		err = p.UnmarshalJSON(data)
		require.NoError(t, err)
		require.Equal(t, "oldpath", p.Path)
		require.Empty(t, p.Labels)
	})
}

func TestMDMProfileSpecsMatch(t *testing.T) {
	tests := []struct {
		name     string
		a        []fleet.MDMProfileSpec
		b        []fleet.MDMProfileSpec
		expected bool
	}{
		{
			name:     "Empty Slices",
			a:        []fleet.MDMProfileSpec{},
			b:        []fleet.MDMProfileSpec{},
			expected: true,
		},
		{
			name: "Single Element Match",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", Labels: []string{"label1"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path1", Labels: []string{"label1"}},
			},
			expected: true,
		},
		{
			name: "Single Element Mismatch",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", Labels: []string{"label1"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path2", Labels: []string{"label1"}},
			},
			expected: false,
		},
		{
			name: "Multiple Elements Match",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", Labels: []string{"label1", "label2"}},
				{Path: "path2", Labels: []string{"label3"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path2", Labels: []string{"label3"}},
				{Path: "path1", Labels: []string{"label1", "label2"}},
			},
			expected: true,
		},
		{
			name: "Multiple Elements Mismatch",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", Labels: []string{"label1"}},
				{Path: "path2", Labels: []string{"label3"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path1", Labels: []string{"label2"}},
				{Path: "path2", Labels: []string{"label3"}},
			},
			expected: false,
		},
		{
			name: "Include Labels Match",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsIncludeAll: []string{"label1", "label2"}},
				{Path: "path2", LabelsIncludeAll: []string{"label3"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsIncludeAll: []string{"label2", "label1"}},
				{Path: "path2", LabelsIncludeAll: []string{"label3"}},
			},
			expected: true,
		},
		{
			name: "Exclude Labels Match",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsExcludeAny: []string{"label1", "label2"}},
				{Path: "path2", LabelsExcludeAny: []string{"label3"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsExcludeAny: []string{"label2", "label1"}},
				{Path: "path2", LabelsExcludeAny: []string{"label3"}},
			},
			expected: true,
		},
		{
			name: "Include Labels Mismatch",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsIncludeAll: []string{"label1", "label2"}},
				{Path: "path2", LabelsIncludeAll: []string{"label3"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsIncludeAll: []string{"label2", "label1"}},
				{Path: "path2", LabelsIncludeAll: []string{"label4"}},
			},
			expected: false,
		},
		{
			name: "Exclude Labels Mismatch",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsExcludeAny: []string{"label1", "label2"}},
				{Path: "path2", LabelsExcludeAny: []string{"label3"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsExcludeAny: []string{"label2", "label1"}},
				{Path: "path3", LabelsExcludeAny: []string{"label3"}},
			},
			expected: false,
		},
		{
			name: "Deprecated Labels Match IncludeAll",
			a: []fleet.MDMProfileSpec{
				{Path: "path1", Labels: []string{"label1", "label2"}},
				{Path: "path2", LabelsExcludeAny: []string{"label3"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "path1", LabelsIncludeAll: []string{"label2", "label1"}},
				{Path: "path2", LabelsExcludeAny: []string{"label3"}},
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fleet.MDMProfileSpecsMatch(tc.a, tc.b)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestFilterMacOSOnlyProfilesFromIOSIPadOS(t *testing.T) {
	for _, tc := range []struct {
		profiles         []*fleet.MDMAppleProfilePayload
		expectedProfiles []*fleet.MDMAppleProfilePayload
	}{
		{
			profiles:         []*fleet.MDMAppleProfilePayload{},
			expectedProfiles: []*fleet.MDMAppleProfilePayload{},
		},
		{
			profiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "darwin",
				},
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ios",
				},
				{
					ProfileName:  "SomeProfile",
					HostPlatform: "darwin",
				},
				{
					ProfileName:  fleetmdm.FleetdConfigProfileName,
					HostPlatform: "ipados",
				},
				{
					ProfileName:  fleetmdm.FleetdConfigProfileName,
					HostPlatform: "ios",
				},
				{
					ProfileName:  "SomeProfile2",
					HostPlatform: "ios",
				},
				{
					ProfileName:  "SomeProfile3",
					HostPlatform: "ipados",
				},
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ipados",
				},
			},
			expectedProfiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "darwin",
				},
				{
					ProfileName:  "SomeProfile",
					HostPlatform: "darwin",
				},
				{
					ProfileName:  "SomeProfile2",
					HostPlatform: "ios",
				},
				{
					ProfileName:  "SomeProfile3",
					HostPlatform: "ipados",
				},
			},
		},
		{
			profiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "darwin",
				},
				{
					ProfileName:  "SomeProfile",
					HostPlatform: "ios",
				},
			},
			expectedProfiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "darwin",
				},
				{
					ProfileName:  "SomeProfile",
					HostPlatform: "ios",
				},
			},
		},
		{
			profiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ios",
				},
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ipados",
				},
			},
			expectedProfiles: []*fleet.MDMAppleProfilePayload{},
		},
		{
			profiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ios",
				},
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ipados",
				},
			},
			expectedProfiles: []*fleet.MDMAppleProfilePayload{},
		},
		{
			profiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ios",
				},
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "darwin",
				},
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "ipados",
				},
			},
			expectedProfiles: []*fleet.MDMAppleProfilePayload{
				{
					ProfileName:  fleetmdm.FleetFileVaultProfileName,
					HostPlatform: "darwin",
				},
			},
		},
	} {
		actualProfiles := fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(tc.profiles)
		require.Equal(t, len(actualProfiles), len(tc.expectedProfiles))
		for i := 0; i < len(actualProfiles); i++ {
			require.Equal(t, *actualProfiles[i], *tc.expectedProfiles[i])
		}

	}
}
