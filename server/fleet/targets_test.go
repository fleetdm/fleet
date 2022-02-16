package fleet_test

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetTypeJSON(t *testing.T) {
	testCases := []struct {
		expected  fleet.TargetType
		shouldErr bool
	}{
		{fleet.TargetLabel, false},
		{fleet.TargetHost, false},
		{fleet.TargetTeam, false},
		{fleet.TargetType(37), true},
	}
	for _, tt := range testCases {
		t.Run(tt.expected.String(), func(t *testing.T) {
			b, err := json.Marshal(tt.expected)
			require.NoError(t, err)
			var target fleet.TargetType
			err = json.Unmarshal(b, &target)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, target)
			}
		})
	}
}
