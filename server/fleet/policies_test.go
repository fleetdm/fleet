package fleet

import (
	"errors"
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
		name       string
		includeAny []string
		includeAll []string
		excludeAny []string
		excludeAll []string
		wantErr    error // sentinel to match with errors.Is, or nil; use errAny for a dynamic error
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
		{name: "overlap include_any/exclude_any", includeAny: []string{"a"}, excludeAny: []string{"a"}, wantErr: errAny},
		{name: "overlap include_all/exclude_all", includeAll: []string{"a"}, excludeAll: []string{"a"}, wantErr: errAny},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyPolicyLabelScopes(tc.includeAny, tc.includeAll, tc.excludeAny, tc.excludeAll)
			switch {
			case tc.wantErr == nil:
				require.NoError(t, err)
			case tc.wantErr == errAny:
				require.Error(t, err)
			default:
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

// errAny is a sentinel used by tests to mean "expect some (dynamic) error".
var errAny = errors.New("any error")

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
