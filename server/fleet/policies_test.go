package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyPolicyPlatforms(t *testing.T) {
	testCases := []struct {
		platforms string
		isValid   bool
	}{
		{"windows,chrome", true},
		{"chrome", true},
		{"bados", false},
	}

	for _, tc := range testCases {
		actual := verifyPolicyPlatforms(tc.platforms)

		if tc.isValid {
			require.NoError(t, actual)
			continue
		}
		require.Error(t, actual)
	}
}

func TestVerifyPolicyLabelScopes(t *testing.T) {
	testCases := []struct {
		name            string
		includeAny      []string
		includeAll      []string
		excludeAny      []string
		excludeAll      []string
		wantErr         error  // sentinel to match with errors.Is (nil means no error)
		wantErrContains string // substring to match for the dynamic overlap error
	}{
		{name: "no labels"},
		{name: "include_any only", includeAny: []string{"a"}},
		{name: "include_all only", includeAll: []string{"a"}},
		{name: "exclude_any only", excludeAny: []string{"a"}},
		{name: "exclude_all only", excludeAll: []string{"a"}},
		{name: "include_any + exclude_any combined", includeAny: []string{"a"}, excludeAny: []string{"b"}},
		{name: "include_all + exclude_any combined", includeAll: []string{"a"}, excludeAny: []string{"b"}},
		{name: "include_any + exclude_all combined", includeAny: []string{"a"}, excludeAll: []string{"b"}},
		{name: "include_all + exclude_all combined", includeAll: []string{"a"}, excludeAll: []string{"b"}},
		{name: "include_any + include_all conflict", includeAny: []string{"a"}, includeAll: []string{"b"}, wantErr: ErrPolicyConflictingIncludeLabels},
		{name: "exclude_any + exclude_all conflict", excludeAny: []string{"a"}, excludeAll: []string{"b"}, wantErr: ErrPolicyConflictingExcludeLabels},
		{name: "overlap include_any/exclude_any", includeAny: []string{"a"}, excludeAny: []string{"a"}, wantErrContains: `label "a" cannot appear in both an include and an exclude list`},
		{name: "overlap include_all/exclude_all", includeAll: []string{"a"}, excludeAll: []string{"a"}, wantErrContains: `label "a" cannot appear in both an include and an exclude list`},
		{name: "overlap include_any/exclude_all", includeAny: []string{"a"}, excludeAll: []string{"a"}, wantErrContains: `label "a" cannot appear in both an include and an exclude list`},
		{name: "overlap include_all/exclude_any", includeAll: []string{"a"}, excludeAny: []string{"a"}, wantErrContains: `label "a" cannot appear in both an include and an exclude list`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyPolicyLabelScopes(tc.includeAny, tc.includeAll, tc.excludeAny, tc.excludeAll)
			switch {
			case tc.wantErrContains != "":
				require.ErrorContains(t, err, tc.wantErrContains)
			case tc.wantErr != nil:
				require.ErrorIs(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestFirstFuplicatePolicySpecName(t *testing.T) {
	testCases := []struct {
		name     string
		result   string
		policies []*PolicySpec
	}{
		{"no specs", "", []*PolicySpec{}},
		{"no duplicate names", "", []*PolicySpec{{Name: "foo"}}},
		{"duplicate names", "foo", []*PolicySpec{{Name: "foo"}, {Name: "bar"}, {Name: "foo"}}},
	}

	for _, tc := range testCases {
		name := FirstDuplicatePolicySpecName(tc.policies)
		require.Equal(t, tc.result, name)
	}
}

func TestPolicySpecVerifyFleetMaintainedAppSlug(t *testing.T) {
	testCases := []struct {
		name    string
		spec    PolicySpec
		wantErr error
	}{
		{
			name: "patch policy with slug is allowed",
			spec: PolicySpec{Name: "Chrome up to date", Team: "Workstations", Type: PolicyTypePatch, FleetMaintainedAppSlug: "google-chrome/darwin"},
		},
		{
			name:    "dynamic policy with slug is rejected",
			spec:    PolicySpec{Name: "Chrome installed", Team: "Workstations", Query: "SELECT 1;", Type: PolicyTypeDynamic, FleetMaintainedAppSlug: "google-chrome/darwin"},
			wantErr: errPolicyFMASlugRequiresPatch,
		},
		{
			name:    "policy without type but with slug is rejected",
			spec:    PolicySpec{Name: "Chrome installed", Team: "Workstations", Query: "SELECT 1;", FleetMaintainedAppSlug: "google-chrome/darwin"},
			wantErr: errPolicyFMASlugRequiresPatch,
		},
		{
			name: "dynamic policy without slug is allowed",
			spec: PolicySpec{Name: "Chrome installed", Team: "Workstations", Query: "SELECT 1;", Type: PolicyTypeDynamic},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.spec.Verify()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestResolvePolicyResendProfile(t *testing.T) {
	testCases := []struct {
		name        string
		profileUUID *string
		want        resendProfile
		wantErr     bool
	}{
		{
			name:        "nil UUID",
			profileUUID: nil,
			want:        resendProfile{},
		},
		{
			name:        "empty UUID",
			profileUUID: new(""),
			want:        resendProfile{},
		},
		{
			name:        "Apple profile UUID",
			profileUUID: new(MDMAppleProfileUUIDPrefix + "1234"),
			want: resendProfile{
				AppleUUID: new(MDMAppleProfileUUIDPrefix + "1234"),
				Table:     "mdm_apple_configuration_profiles",
			},
		},
		{
			name:        "Windows profile UUID",
			profileUUID: new(MDMWindowsProfileUUIDPrefix + "5678"),
			want: resendProfile{
				WindowsUUID: new(MDMWindowsProfileUUIDPrefix + "5678"),
				Table:       "mdm_windows_configuration_profiles",
			},
		},
		{
			name:        "Apple declaration UUID",
			profileUUID: new(MDMAppleDeclarationUUIDPrefix + "abcd"),
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolvePolicyResendProfile(tc.profileUUID)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestPolicyVerifyResendProfile(t *testing.T) {
	appleUUID := MDMAppleProfileUUIDPrefix + "apple"
	winUUID := MDMWindowsProfileUUIDPrefix + "windows"

	cases := []struct {
		name        string
		profileUUID *string
		platform    string
		wantErr     bool
	}{
		{name: "no profile, any platform", profileUUID: nil, platform: "linux"},
		{name: "empty profile string is treated as unset", profileUUID: new(""), platform: "linux"},
		{name: "apple profile on darwin", profileUUID: &appleUUID, platform: "darwin"},
		{name: "windows profile on windows", profileUUID: &winUUID, platform: "windows"},
		// Cross-platform pairings are allowed on purpose: the automation skips hosts that can't
		// receive the profile rather than the API rejecting the policy.
		{name: "apple profile on a windows-only policy", profileUUID: &appleUUID, platform: "windows"},
		{name: "windows profile on a darwin-only policy", profileUUID: &winUUID, platform: "darwin"},
		{name: "apple profile on darwin and windows", profileUUID: &appleUUID, platform: "darwin,windows"},
		{name: "profile on a list including darwin", profileUUID: &appleUUID, platform: "linux,darwin,chrome"},
		{name: "profile on a list including windows", profileUUID: &winUUID, platform: "chrome,windows"},
		{name: "platform list with spaces", profileUUID: &appleUUID, platform: "linux, darwin"},
		// An empty platform means every platform, which the automation can be scoped to.
		{name: "profile with no platform set", profileUUID: &appleUUID, platform: ""},
		// Neither darwin nor windows: nothing the profile could be delivered to.
		{name: "profile on a linux-only policy", profileUUID: &appleUUID, platform: "linux", wantErr: true},
		{name: "profile on a chrome-only policy", profileUUID: &winUUID, platform: "chrome", wantErr: true},
		{name: "profile on linux and chrome", profileUUID: &appleUUID, platform: "linux,chrome", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := PolicyVerifyResendProfile(c.profileUUID, c.platform)
			if c.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, errPolicyResendProfileInvalidPlatform)
				return
			}
			require.NoError(t, err)
		})
	}

	// The same gate applies through both payload verifiers, which is what the API layer calls.
	t.Run("PolicyPayload.Verify enforces it", func(t *testing.T) {
		payload := PolicyPayload{Name: "p", Query: "SELECT 1;", Platform: "linux", ProfileUUID: &appleUUID}
		require.ErrorIs(t, payload.Verify(), errPolicyResendProfileInvalidPlatform)

		payload.Platform = "darwin"
		require.NoError(t, payload.Verify())
	})

	t.Run("PolicySpec.Verify enforces it", func(t *testing.T) {
		spec := PolicySpec{Name: "p", Query: "SELECT 1;", Team: "team1", Platform: "linux", ProfileUUID: &appleUUID}
		require.ErrorIs(t, spec.Verify(), errPolicyResendProfileInvalidPlatform)

		spec.Platform = "windows"
		require.NoError(t, spec.Verify())
	})
}
