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

func TestParseTargetTypeFleet(t *testing.T) {
	for _, s := range []string{"team", "fleet"} {
		got, err := fleet.ParseTargetType(s)
		require.NoError(t, err, s)
		assert.Equal(t, fleet.TargetTeam, got, s)
	}

	var target fleet.TargetType
	require.NoError(t, json.Unmarshal([]byte(`"fleet"`), &target))
	assert.Equal(t, fleet.TargetTeam, target)

	_, err := fleet.ParseTargetType("fleets")
	require.Error(t, err)
}
