package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSoftwareIterQueryOptionsIsValid(t *testing.T) {
	testCases := []struct {
		excluded   []string
		included   []string
		isNotValid bool
	}{
		{
			excluded: nil,
			included: nil,
		},
		{
			excluded: []string{"a", "b"},
			included: nil,
		},
		{
			excluded: nil,
			included: []string{"a", "b"},
		},
		{
			excluded:   []string{"a", "b"},
			included:   []string{"a"},
			isNotValid: true,
		},
		{
			excluded:   []string{"a"},
			included:   []string{"a", "b"},
			isNotValid: true,
		},
		{
			excluded:   []string{"c"},
			included:   []string{"a", "b"},
			isNotValid: true,
		},
	}

	for _, tC := range testCases {
		sut := SoftwareIterQueryOptions{
			ExcludedSources: tC.excluded,
			IncludedSources: tC.included,
		}

		if tC.isNotValid {
			require.False(t, sut.IsValid())
		} else {
			require.True(t, sut.IsValid())
		}
	}
}

func TestParseSoftwareLastOpenedAtRowValue(t *testing.T) {
	// Some macOS apps return last_opened_at=-1.0 on apps
	// that were never opened.
	lastOpenedAt, err := ParseSoftwareLastOpenedAtRowValue("-1.0")
	require.NoError(t, err)
	require.Zero(t, lastOpenedAt)

	// Our software queries hardcode to 0 if such info is not available.
	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("0")
	require.NoError(t, err)
	require.Zero(t, lastOpenedAt)

	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("foobar")
	require.Error(t, err)
	require.Zero(t, lastOpenedAt)

	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("1694026958")
	require.NoError(t, err)
	require.NotZero(t, lastOpenedAt)
}
