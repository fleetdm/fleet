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
