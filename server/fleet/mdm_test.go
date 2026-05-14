package fleet_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	ctxabm "github.com/fleetdm/fleet/v4/server/contexts/apple_bm"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestDEPClient(t *testing.T) {
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

	logger := slog.New(slog.DiscardHandler)
	ds := new(mock.Store)

	appCfg := fleet.AppConfig{}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &appCfg, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		appCfg = *config
		return nil
	}

	termsExpiredByOrgName := map[string]bool{
		"org1": false,
		"org2": false,
	}
	ds.SetABMTokenTermsExpiredForOrgNameFunc = func(ctx context.Context, orgName string, expired bool) (wasSet bool, err error) {
		was, ok := termsExpiredByOrgName[orgName]
		if !ok {
			return expired, nil
		}
		termsExpiredByOrgName[orgName] = expired
		return was, nil
	}
	ds.CountABMTokensWithTermsExpiredFunc = func(ctx context.Context) (int, error) {
		count := 0
		for _, expired := range termsExpiredByOrgName {
			if expired {
				count++
			}
		}
		return count, nil
	}

	checkDSCalled := func(readInvoked, writeTokInvoked, writeAppCfgInvoked bool) {
		require.Equal(t, readInvoked, ds.AppConfigFuncInvoked)
		require.Equal(t, readInvoked, ds.CountABMTokensWithTermsExpiredFuncInvoked)
		require.Equal(t, writeTokInvoked, ds.SetABMTokenTermsExpiredForOrgNameFuncInvoked)
		require.Equal(t, writeAppCfgInvoked, ds.SaveAppConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false
		ds.CountABMTokensWithTermsExpiredFuncInvoked = false
		ds.SaveAppConfigFuncInvoked = false
		ds.SetABMTokenTermsExpiredForOrgNameFuncInvoked = false
	}

	cases := []struct {
		token               string
		orgName             string
		wantErr             bool
		readInvoked         bool
		writeTokInvoked     bool
		writeAppCfgInvoked  bool
		wantAppCfgTermsFlag bool
		wantToksTermsFlags  map[string]bool
	}{
		// use a valid token, appconfig should not be updated (already unflagged)
		{
			token: validToken, orgName: "org1", wantErr: false, readInvoked: true, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: false, wantToksTermsFlags: map[string]bool{"org1": false, "org2": false},
		},

		// use a valid token without org, nothing is checked
		{
			token: validToken, orgName: "", wantErr: false, readInvoked: false, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: false, wantToksTermsFlags: map[string]bool{"org1": false, "org2": false},
		},

		// use an invalid token without org, call fails but nothing is checked because this is an unsaved token
		{
			token: invalidToken, orgName: "", wantErr: true, readInvoked: false, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: false, wantToksTermsFlags: map[string]bool{"org1": false, "org2": false},
		},

		// use an invalid token, appconfig should not even be read (not a terms error)
		{
			token: invalidToken, orgName: "org1", wantErr: true, readInvoked: false, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: false, wantToksTermsFlags: map[string]bool{"org1": false, "org2": false},
		},

		// terms changed for org1 during the auth request
		{
			token: termsChangedToken, orgName: "org1", wantErr: true, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: true, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": true, "org2": false},
		},

		// use of an invalid token does not update the flag
		{
			token: invalidToken, orgName: "org1", wantErr: true, readInvoked: false, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": true, "org2": false},
		},

		// use of a valid token for org1 resets the flags
		{
			token: validToken, orgName: "org1", wantErr: false, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: true, wantAppCfgTermsFlag: false, wantToksTermsFlags: map[string]bool{"org1": false, "org2": false},
		},

		// use of a valid token again with org2 does not update anything
		{
			token: validToken, orgName: "org2", wantErr: false, readInvoked: true, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: false, wantToksTermsFlags: map[string]bool{"org1": false, "org2": false},
		},

		// terms changed for org2 during the actual account request, after auth
		{
			token: termsChangedAfterAuthToken, orgName: "org2", wantErr: true, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: true, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": false, "org2": true},
		},

		// again terms changed after auth for org2, doesn't update appConfig
		{
			token: termsChangedAfterAuthToken, orgName: "org2", wantErr: true, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": false, "org2": true},
		},

		// terms changed during auth for org2, doesn't update appConfig
		{
			token: termsChangedToken, orgName: "org2", wantErr: true, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": false, "org2": true},
		},

		// terms changed during auth for org1, now both tokens have the flag, doesn't update appConfig
		{
			token: termsChangedToken, orgName: "org1", wantErr: true, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": true, "org2": true},
		},

		// use a valid token without org, nothing is checked
		{
			token: validToken, orgName: "", wantErr: false, readInvoked: false, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": true, "org2": true},
		},

		// use an invalid token without org, call fails but nothing is checked because this is an unsaved token
		{
			token: invalidToken, orgName: "", wantErr: true, readInvoked: false, writeTokInvoked: false,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": true, "org2": true},
		},

		// valid token for org1, resets that token's flag but not appConfig
		{
			token: validToken, orgName: "org1", wantErr: false, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": false, "org2": true},
		},

		// valid token again for org1, still no write to appConfig
		{
			token: validToken, orgName: "org1", wantErr: false, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: false, wantAppCfgTermsFlag: true, wantToksTermsFlags: map[string]bool{"org1": false, "org2": true},
		},

		// valid token again for org2, this time resets appConfig
		{
			token: validToken, orgName: "org2", wantErr: false, readInvoked: true, writeTokInvoked: true,
			writeAppCfgInvoked: true, wantAppCfgTermsFlag: false, wantToksTermsFlags: map[string]bool{"org1": false, "org2": false},
		},
	}

	// order of calls is important, and test must not be parallelized as it would
	// be racy. For that reason, subtests are not used (it would make it possible
	// to run one subtest in isolation, which could fail).
	for i, c := range cases {
		t.Logf("case %d", i)

		ctx := context.Background()

		store := &nanodep_mock.Storage{}
		store.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
			return &nanodep_client.OAuth1Tokens{AccessToken: c.token}, nil
		}
		store.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
			return &nanodep_client.Config{BaseURL: srv.URL}, nil
		}

		dep := apple_mdm.NewDEPClient(store, ds, logger)
		orgName := c.orgName
		if orgName == "" {
			// simulate using a new token, not yet saved in the DB, so we pass the
			// token directly in the context
			ctx = ctxabm.NewContext(ctx, &nanodep_client.OAuth1Tokens{AccessToken: c.token})
			orgName = apple_mdm.UnsavedABMTokenOrgName
		}
		res, err := dep.AccountDetail(ctx, orgName)

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
		checkDSCalled(c.readInvoked, c.writeTokInvoked, c.writeAppCfgInvoked)
		require.Equal(t, c.wantAppCfgTermsFlag, appCfg.MDM.AppleBMTermsExpired)
		require.Equal(t, c.wantToksTermsFlags, termsExpiredByOrgName)
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

	t.Run("changing labels", func(t *testing.T) {
		// When updating AppConfig, we unmarshal the incoming JSON into the existing AppConfig
		// struct, see
		// https://github.com/fleetdm/fleet/blob/d1144df1318b50482cbd9eb996b863443975f138/server/service/appconfig.go#L334-L335
		//
		// But we found there were issues unmarshaling the slice of profile specs where if a key is present in an old
		// element but not in the new element (e.g. element[0] of the old slice and element[0] of the
		// new slice), both keys were preserved. This test is designed to cover that issue, which
		// was addressed in the unmarshal function, see
		// https://github.com/fleetdm/fleet/blob/1042702def54f095335d8b42ed5fdcc90468fa0d/server/fleet/mdm.go#L551-L552

		storedConfig := fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgName: "Test",
			},
			MDM: fleet.MDM{
				MacOSSettings: fleet.MacOSSettings{
					CustomSettings: []fleet.MDMProfileSpec{
						{
							Path:             "some-profile-2",
							LabelsExcludeAny: []string{"bar"},
						},
						{
							Path:             "some-profile-1",
							LabelsIncludeAll: []string{"foo"},
						},
					},
				},
			},
		}

		incomingConfig := fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgName: "Test",
			},
			MDM: fleet.MDM{
				MacOSSettings: fleet.MacOSSettings{
					CustomSettings: []fleet.MDMProfileSpec{
						{
							Path:             "some-profile-1",
							LabelsIncludeAll: []string{"foo"},
						},
						{
							Path:             "some-profile-2",
							LabelsIncludeAny: []string{"bar"},
						},
					},
				},
			},
		}
		b, err := json.Marshal(incomingConfig)
		require.NoError(t, err)

		err = json.Unmarshal(b, &storedConfig)
		require.NoError(t, err)

		require.Equal(t, storedConfig.MDM.MacOSSettings.CustomSettings, incomingConfig.MDM.MacOSSettings.CustomSettings)
		require.Nil(t, storedConfig.MDM.MacOSSettings.CustomSettings[0].LabelsExcludeAny) // old key should be removed
		require.Nil(t, storedConfig.MDM.MacOSSettings.CustomSettings[1].LabelsIncludeAll) // old key should be removed
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
		{
			// Regression for https://github.com/fleetdm/fleet/issues/45485 —
			// duplicate Paths must be compared as a multiset, not via a
			// path-keyed map that collapses duplicates.
			name: "Duplicate Path Reflexive",
			a: []fleet.MDMProfileSpec{
				{Path: "/a"},
				{Path: "/a"},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "/a"},
				{Path: "/a"},
			},
			expected: true,
		},
		{
			name: "Duplicate Path Mismatch",
			a: []fleet.MDMProfileSpec{
				{Path: "/a"},
				{Path: "/a"},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "/a"},
				{Path: "/b"},
			},
			expected: false,
		},
		{
			// Guards the encoding used for multiset keys against control-byte
			// collisions: a label list ["a","b"] must not collapse to the
			// same key as a single label "a\x01b".
			name: "Control Byte Labels Not Collapsed",
			a: []fleet.MDMProfileSpec{
				{Path: "/a", LabelsIncludeAll: []string{"a", "b"}},
			},
			b: []fleet.MDMProfileSpec{
				{Path: "/a", LabelsIncludeAll: []string{"a\x01b"}},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fleet.MDMProfileSpecsMatch(tc.a, tc.b)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestPBT_MDMProfileSpecsMatch uses property-based testing to verify
// the contract documented on MDMProfileSpecsMatch ("contain the same spec
// elements, regardless of order") via invariants that don't depend on a
// reference implementation:
//   - Reflexivity: Match(a, a) is always true.
//   - Symmetry: Match(a, b) == Match(b, a).
//   - Permutation invariance on the slice: shuffling a does not change
//     Match(a, b).
//   - Permutation invariance on labels within a spec: shuffling any of a
//     spec's labels fields does not change Match(a, b).
//   - Sensitivity to length: appending an extra element makes Match false.
//
// Generators use narrow pools so duplicate Paths and duplicate labels are
// likely, which is where the multiset semantics matter most.
func TestPBT_MDMProfileSpecsMatch(t *testing.T) {
	labelGen := rapid.SliceOfN(rapid.SampledFrom([]string{"x", "y", "z"}), 0, 3)
	pathGen := rapid.SampledFrom([]string{"/a", "/b"})
	specGen := rapid.Custom(func(t *rapid.T) fleet.MDMProfileSpec {
		return fleet.MDMProfileSpec{
			Path:             pathGen.Draw(t, "Path"),
			Labels:           labelGen.Draw(t, "Labels"),
			LabelsIncludeAll: labelGen.Draw(t, "LabelsIncludeAll"),
			LabelsIncludeAny: labelGen.Draw(t, "LabelsIncludeAny"),
			LabelsExcludeAny: labelGen.Draw(t, "LabelsExcludeAny"),
		}
	})
	sliceGen := rapid.SliceOfN(specGen, 0, 5)

	t.Run("Reflexive", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			a := sliceGen.Draw(t, "a")
			require.Truef(t, fleet.MDMProfileSpecsMatch(a, a),
				"Match(a, a) should be true; a=%+v", a)
		})
	})

	t.Run("Symmetric", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			a := sliceGen.Draw(t, "a")
			b := sliceGen.Draw(t, "b")
			require.Equalf(t, fleet.MDMProfileSpecsMatch(a, b), fleet.MDMProfileSpecsMatch(b, a),
				"Match should be symmetric; a=%+v b=%+v", a, b)
		})
	})

	t.Run("PermutationInvariantOnSlice", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			a := sliceGen.Draw(t, "a")
			b := sliceGen.Draw(t, "b")
			aShuf := rapid.Permutation(a).Draw(t, "permA")
			require.Equalf(t, fleet.MDMProfileSpecsMatch(a, b), fleet.MDMProfileSpecsMatch(aShuf, b),
				"Match changed after shuffling a; a=%+v aShuf=%+v b=%+v", a, aShuf, b)
		})
	})

	t.Run("PermutationInvariantOnLabels", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			a := sliceGen.Draw(t, "a")
			b := sliceGen.Draw(t, "b")
			aShuf := make([]fleet.MDMProfileSpec, len(a))
			for i, s := range a {
				prefix := fmt.Sprintf("a[%d]", i)
				aShuf[i] = fleet.MDMProfileSpec{
					Path:             s.Path,
					Labels:           rapid.Permutation(s.Labels).Draw(t, prefix+".Labels"),
					LabelsIncludeAll: rapid.Permutation(s.LabelsIncludeAll).Draw(t, prefix+".LabelsIncludeAll"),
					LabelsIncludeAny: rapid.Permutation(s.LabelsIncludeAny).Draw(t, prefix+".LabelsIncludeAny"),
					LabelsExcludeAny: rapid.Permutation(s.LabelsExcludeAny).Draw(t, prefix+".LabelsExcludeAny"),
				}
			}
			require.Equalf(t, fleet.MDMProfileSpecsMatch(a, b), fleet.MDMProfileSpecsMatch(aShuf, b),
				"Match changed after shuffling labels inside a; a=%+v aShuf=%+v b=%+v", a, aShuf, b)
		})
	})

	t.Run("SensitiveToLengthChange", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			a := sliceGen.Draw(t, "a")
			extra := specGen.Draw(t, "extra")
			aPlus := append(append([]fleet.MDMProfileSpec{}, a...), extra)
			require.Falsef(t, fleet.MDMProfileSpecsMatch(aPlus, a),
				"Match should reject slices of different lengths; a=%+v aPlus=%+v", a, aPlus)
		})
	})
}

func TestHasCAVariables(t *testing.T) {
	tests := []struct {
		name     string
		vars     []string
		expected bool
	}{
		{"empty", nil, false},
		{"no CA vars", []string{string(fleet.FleetVarHostUUID), string(fleet.FleetVarHostHardwareSerial)}, false},
		{"NDES challenge", []string{string(fleet.FleetVarHostUUID), string(fleet.FleetVarNDESSCEPChallenge)}, true},
		{"NDES proxy URL", []string{string(fleet.FleetVarNDESSCEPProxyURL)}, true},
		{"SCEP renewal", []string{string(fleet.FleetVarSCEPRenewalID)}, true},
		{"DigiCert data", []string{string(fleet.FleetVarDigiCertDataPrefix) + "my_ca"}, true},
		{"DigiCert password", []string{string(fleet.FleetVarDigiCertPasswordPrefix) + "my_ca"}, true},
		{"Custom SCEP challenge", []string{string(fleet.FleetVarCustomSCEPChallengePrefix) + "my_ca"}, true},
		{"Custom SCEP proxy URL", []string{string(fleet.FleetVarCustomSCEPProxyURLPrefix) + "my_ca"}, true},
		{"Smallstep challenge", []string{string(fleet.FleetVarSmallstepSCEPChallengePrefix) + "my_ca"}, true},
		{"Smallstep proxy URL", []string{string(fleet.FleetVarSmallstepSCEPProxyURLPrefix) + "my_ca"}, true},
		{"Windows SCEP cert ID", []string{string(fleet.FleetVarSCEPWindowsCertificateID)}, true},
		{"mixed with CA", []string{string(fleet.FleetVarHostUUID), string(fleet.FleetVarHostHardwareSerial), string(fleet.FleetVarNDESSCEPChallenge)}, true},
		{"unknown var", []string{"UNKNOWN_VAR"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fleet.HasCAVariables(tc.vars)
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

func TestFilterOutUserScopedProfiles(t *testing.T) {
	systemScopedProfile := fleet.MDMAppleProfilePayload{
		ProfileName: fleetmdm.FleetFileVaultProfileName,
		Scope:       fleet.PayloadScopeSystem,
	}
	userScopedProfile := fleet.MDMAppleProfilePayload{
		ProfileName: fleetmdm.FleetCAConfigProfileName,
		Scope:       fleet.PayloadScopeUser,
	}

	profilesToFilter := []*fleet.MDMAppleProfilePayload{&systemScopedProfile, &userScopedProfile}

	filteredProfiles := fleet.FilterOutUserScopedProfiles(profilesToFilter)

	require.ElementsMatch(t, filteredProfiles, []*fleet.MDMAppleProfilePayload{&systemScopedProfile})
}
