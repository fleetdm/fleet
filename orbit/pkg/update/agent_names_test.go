package update

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestAgentNamesReceiver(t *testing.T) {
	capabilities := fleet.CapabilityMap{}
	var calls int
	applyErr := errors.New("apply failed")
	r := &AgentNamesReceiver{
		serverCapabilities: func() fleet.CapabilityMap { return capabilities },
		apply: func() error {
			calls++
			return applyErr
		},
	}

	// A server older than Fleet 5 doesn't advertise the capability: nothing changes.
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Zero(t, calls)

	// A failure doesn't fail the config fetch and is retried on the next one.
	capabilities[fleet.CapabilityFleetAgentNames] = struct{}{}
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 2, calls)

	// Once applied, it isn't applied again during this orbit run.
	applyErr = nil
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 3, calls)
}
