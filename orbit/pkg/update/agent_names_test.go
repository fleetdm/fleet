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

func TestUndoAgentNames(t *testing.T) {
	t.Cleanup(func() { agentNamesUndone = false })
	var applied, restored int
	r := &AgentNamesReceiver{
		serverCapabilities: func() fleet.CapabilityMap {
			return fleet.CapabilityMap{fleet.CapabilityFleetAgentNames: struct{}{}}
		},
		apply: func() error {
			applied++
			return errors.New("apply failed")
		},
	}
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 1, applied)

	// Once a downgrade restored the installer's name, a receiver that is still retrying doesn't rename it back.
	undoAgentNames(func() error {
		restored++
		return nil
	})
	require.Equal(t, 1, restored)
	require.NoError(t, r.Run(&fleet.OrbitConfig{}))
	require.Equal(t, 1, applied)
}
