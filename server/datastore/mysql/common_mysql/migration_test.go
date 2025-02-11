package common_mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	for _, tc := range []struct {
		name string

		v1            []int64
		v2            []int64
		knownUnknowns map[int64]struct{}

		expMissing []int64
		expUnknown []int64
		expEqual   bool
	}{
		{
			name:     "both-empty",
			v1:       nil,
			v2:       nil,
			expEqual: true,
		},
		{
			name:     "equal",
			v1:       []int64{1, 2, 3},
			v2:       []int64{1, 2, 3},
			expEqual: true,
		},
		{
			name:     "equal-out-of-order",
			v1:       []int64{1, 2, 3},
			v2:       []int64{1, 3, 2},
			expEqual: true,
		},
		{
			name:       "empty-with-unknown",
			v1:         nil,
			v2:         []int64{1},
			expEqual:   false,
			expUnknown: []int64{1},
		},
		{
			name:       "empty-with-missing",
			v1:         []int64{1},
			v2:         nil,
			expEqual:   false,
			expMissing: []int64{1},
		},
		{
			name:       "missing",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 3},
			expMissing: []int64{2},
			expEqual:   false,
		},
		{
			name:       "unknown",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 2, 3, 4},
			expUnknown: []int64{4},
			expEqual:   false,
		},
		{
			name: "known-unknown",
			v1:   []int64{1, 2, 3},
			v2:   []int64{1, 2, 3, 4},
			knownUnknowns: map[int64]struct{}{
				4: {},
			},
			expEqual: true,
		},
		{
			name:       "unknowns",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 2, 3, 4, 5},
			expUnknown: []int64{5},
			knownUnknowns: map[int64]struct{}{
				4: {},
			},
			expEqual: false,
		},
		{
			name:       "missing-and-unknown",
			v1:         []int64{1, 2, 3},
			v2:         []int64{1, 2, 4},
			expMissing: []int64{3},
			expUnknown: []int64{4},
			expEqual:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			missing, unknown, equal := CompareVersions(tc.v1, tc.v2, tc.knownUnknowns)
			require.Equal(t, tc.expMissing, missing)
			require.Equal(t, tc.expUnknown, unknown)
			require.Equal(t, tc.expEqual, equal)
		})
	}
}
