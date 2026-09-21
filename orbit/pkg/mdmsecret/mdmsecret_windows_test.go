//go:build windows

package mdmsecret

import (
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

// testKeyPath mirrors the production key under HKCU so the test does not need elevation.
const testKeyPath = `SOFTWARE\FleetDM\OrbitTest`

func setValue(t *testing.T, value string) {
	t.Helper()

	key, _, err := registry.CreateKey(registry.CURRENT_USER, testKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	t.Cleanup(func() { _ = registry.DeleteKey(registry.CURRENT_USER, testKeyPath) })
	require.NoError(t, key.SetStringValue(valueName, value))
	require.NoError(t, key.Close())
}

func TestReadReportsNotFound(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
	}{
		{
			name:  "key absent",
			setup: func(t *testing.T) {},
		},
		{
			name: "value absent",
			setup: func(t *testing.T) {
				setValue(t, "")
				require.NoError(t, clearSecret(registry.CURRENT_USER, testKeyPath))
			},
		},
		{
			name:  "value empty",
			setup: func(t *testing.T) { setValue(t, "") },
		},
		{
			name:  "value whitespace only",
			setup: func(t *testing.T) { setValue(t, "   ") },
		},
		{
			// The installer seeds unset properties with this sentinel; it must not reach the enroll path.
			name:  "value is the installer placeholder",
			setup: func(t *testing.T) { setValue(t, constant.UnusedFlagKeyword) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			secret, err := read(registry.CURRENT_USER, testKeyPath)
			require.ErrorIs(t, err, ErrNotFound)
			require.Empty(t, secret)
		})
	}
}

func TestReadReturnsDeliveredSecret(t *testing.T) {
	setValue(t, "  s3cret-value  ")

	secret, err := read(registry.CURRENT_USER, testKeyPath)
	require.NoError(t, err)
	require.Equal(t, "s3cret-value", secret, "surrounding whitespace should be trimmed")
}

func TestClearRemovesTheSecret(t *testing.T) {
	setValue(t, "s3cret-value")

	require.NoError(t, clearSecret(registry.CURRENT_USER, testKeyPath))

	_, err := read(registry.CURRENT_USER, testKeyPath)
	require.ErrorIs(t, err, ErrNotFound, "a cleared secret must not be readable again")
}

func TestClearIsIdempotent(t *testing.T) {
	// Clearing a key that was never created, and one whose value is already gone, are both the
	// ordinary state on a host that has already enrolled.
	require.NoError(t, clearSecret(registry.CURRENT_USER, testKeyPath))

	setValue(t, "s3cret-value")
	require.NoError(t, clearSecret(registry.CURRENT_USER, testKeyPath))
	require.NoError(t, clearSecret(registry.CURRENT_USER, testKeyPath))
}

func TestReadDoesNotLeakTheSecretIntoErrors(t *testing.T) {
	// A returned error is logged by the caller, so it must never carry the value.
	setValue(t, "s3cret-value")

	_, err := read(registry.CURRENT_USER, testKeyPath+`\missing`)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cret-value")
}
