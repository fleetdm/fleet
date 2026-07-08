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
			name: "dynamic policy with slug is rejected",
			spec: PolicySpec{Name: "Chrome installed", Team: "Workstations", Query: "SELECT 1;", Type: PolicyTypeDynamic, FleetMaintainedAppSlug: "google-chrome/darwin"},
			wantErr: errPolicyFMASlugRequiresPatch,
		},
		{
			name: "policy without type but with slug is rejected",
			spec: PolicySpec{Name: "Chrome installed", Team: "Workstations", Query: "SELECT 1;", FleetMaintainedAppSlug: "google-chrome/darwin"},
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
