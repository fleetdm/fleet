//go:build windows

package autopilot

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

// testPolicyCacheKeyPath mirrors the production key under HKCU so creating it needs no elevation.
const testPolicyCacheKeyPath = `SOFTWARE\FleetDM\AutopilotPolicyCacheTest`

func createTestKey(t *testing.T) registry.Key {
	t.Helper()
	deleteTestKey() // a run that could not clean up must not decide this one
	t.Cleanup(deleteTestKey)

	key, _, err := registry.CreateKey(registry.CURRENT_USER, testPolicyCacheKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	t.Cleanup(func() { _ = key.Close() })
	return key
}

func deleteTestKey() {
	_ = registry.DeleteKey(registry.CURRENT_USER, testPolicyCacheKeyPath)
}

func TestRegistrationIDFromRegistry(t *testing.T) {
	t.Run("key absent", func(t *testing.T) {
		deleteTestKey()
		got, err := registrationID(registry.CURRENT_USER, testPolicyCacheKeyPath)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("value absent", func(t *testing.T) {
		createTestKey(t)
		got, err := registrationID(registry.CURRENT_USER, testPolicyCacheKeyPath)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("assigned profile", func(t *testing.T) {
		key := createTestKey(t)
		require.NoError(t, key.SetStringValue(policyCacheValueName, cachedProfile))
		got, err := registrationID(registry.CURRENT_USER, testPolicyCacheKeyPath)
		require.NoError(t, err)
		require.Equal(t, "efdb13f9-44d6-4f99-a93f-08833fccef82", got)
	})
}
