package update

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestDebugLogReceiver(t *testing.T) {
	// Save and restore the global level so tests stay hermetic.
	orig := zerolog.GlobalLevel()
	t.Cleanup(func() { zerolog.SetGlobalLevel(orig) })

	r := NewDebugLogReceiver()

	trueVal := true
	falseVal := false

	cases := []struct {
		name          string
		startLevel    zerolog.Level
		debugLogging  *bool
		expectedLevel zerolog.Level
	}{
		{
			name:          "nil config field preserves current level (info)",
			startLevel:    zerolog.InfoLevel,
			debugLogging:  nil,
			expectedLevel: zerolog.InfoLevel,
		},
		{
			name:          "nil config field preserves current level (debug)",
			startLevel:    zerolog.DebugLevel,
			debugLogging:  nil,
			expectedLevel: zerolog.DebugLevel,
		},
		{
			name:          "true flips info to debug",
			startLevel:    zerolog.InfoLevel,
			debugLogging:  &trueVal,
			expectedLevel: zerolog.DebugLevel,
		},
		{
			name:          "false flips debug to info",
			startLevel:    zerolog.DebugLevel,
			debugLogging:  &falseVal,
			expectedLevel: zerolog.InfoLevel,
		},
		{
			name:          "true is idempotent when already debug",
			startLevel:    zerolog.DebugLevel,
			debugLogging:  &trueVal,
			expectedLevel: zerolog.DebugLevel,
		},
		{
			name:          "false is idempotent when already info",
			startLevel:    zerolog.InfoLevel,
			debugLogging:  &falseVal,
			expectedLevel: zerolog.InfoLevel,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			zerolog.SetGlobalLevel(tc.startLevel)
			err := r.Run(&fleet.OrbitConfig{DebugLogging: tc.debugLogging})
			require.NoError(t, err)
			require.Equal(t, tc.expectedLevel, zerolog.GlobalLevel())
		})
	}
}

func TestDebugLogReceiverNilConfig(t *testing.T) {
	// Defensive: nil OrbitConfig should not panic.
	orig := zerolog.GlobalLevel()
	t.Cleanup(func() { zerolog.SetGlobalLevel(orig) })

	require.NoError(t, NewDebugLogReceiver().Run(nil))
}
