package update

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestDebugLogReceiver(t *testing.T) {
	orig := zerolog.GlobalLevel()
	t.Cleanup(func() { zerolog.SetGlobalLevel(orig) })

	r := NewDebugLogReceiver(false)

	trueVal := true
	falseVal := false

	cases := []struct {
		name          string
		startLevel    zerolog.Level
		debugLogging  *bool
		expectedLevel zerolog.Level
	}{
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
		{
			name:          "nil config field treated as false, idempotent when already info",
			startLevel:    zerolog.InfoLevel,
			debugLogging:  nil,
			expectedLevel: zerolog.InfoLevel,
		},
		{
			name:          "nil config field treated as false, flips debug to info",
			startLevel:    zerolog.DebugLevel,
			debugLogging:  nil,
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

func TestDebugLogReceiverStartupFlagIsFloor(t *testing.T) {
	orig := zerolog.GlobalLevel()
	t.Cleanup(func() { zerolog.SetGlobalLevel(orig) })

	r := NewDebugLogReceiver(true)

	trueVal := true
	falseVal := false

	// Server off + already debug: floor keeps it on.
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	require.NoError(t, r.Run(&fleet.OrbitConfig{DebugLogging: &falseVal}))
	require.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())

	// Server nil + already debug: floor keeps it on.
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	require.NoError(t, r.Run(&fleet.OrbitConfig{DebugLogging: nil}))
	require.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())

	// Server on: always honored.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	require.NoError(t, r.Run(&fleet.OrbitConfig{DebugLogging: &trueVal}))
	require.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}
