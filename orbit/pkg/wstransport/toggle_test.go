package wstransport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToggleReceiver(t *testing.T) {
	rootDir := t.TempDir()
	var restarts []string
	receiver := NewToggleReceiver(rootDir, func(reason string) { restarts = append(restarts, reason) })

	// Default state: disabled, no state file.
	assert.False(t, Enabled(rootDir))

	// Config without the directive: no change, no restart.
	require.NoError(t, receiver.Run(&fleet.OrbitConfig{}))
	assert.False(t, Enabled(rootDir))
	assert.Empty(t, restarts)

	// Directive turns the transport on: persisted + restart.
	cfgOn := &fleet.OrbitConfig{WebSocketTransport: &fleet.OrbitWebSocketTransportConfig{Enabled: true}}
	require.NoError(t, receiver.Run(cfgOn))
	assert.True(t, Enabled(rootDir))
	assert.Len(t, restarts, 1)

	// Same directive again: no additional restart.
	require.NoError(t, receiver.Run(cfgOn))
	assert.Len(t, restarts, 1)

	// Directive disappears (server flag turned off): persisted + restart.
	require.NoError(t, receiver.Run(&fleet.OrbitConfig{}))
	assert.False(t, Enabled(rootDir))
	assert.Len(t, restarts, 2)

	// Explicit Enabled=false behaves like an absent directive.
	require.NoError(t, receiver.Run(&fleet.OrbitConfig{WebSocketTransport: &fleet.OrbitWebSocketTransportConfig{Enabled: false}}))
	assert.Len(t, restarts, 2)
}

func TestEnabledCorruptStateFile(t *testing.T) {
	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, stateFileName), []byte("not json"), 0o600))
	assert.False(t, Enabled(rootDir))
}
