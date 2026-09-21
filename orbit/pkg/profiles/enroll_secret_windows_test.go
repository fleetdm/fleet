//go:build windows

package profiles

import (
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

// testEnrollSecretKeyPath mirrors the production key under HKCU so the test needs no elevation.
const testEnrollSecretKeyPath = `SOFTWARE\FleetDM\OrbitTest`

func setEnrollSecretValue(t *testing.T, value string) {
	t.Helper()

	key, _, err := registry.CreateKey(registry.CURRENT_USER, testEnrollSecretKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	t.Cleanup(func() { _ = registry.DeleteKey(registry.CURRENT_USER, testEnrollSecretKeyPath) })
	require.NoError(t, key.SetStringValue(enrollSecretValueName, value))
	require.NoError(t, key.Close())
}

func TestGetEnrollSecretReportsNothingWaiting(t *testing.T) {
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
				setEnrollSecretValue(t, "")
				require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))
			},
		},
		{
			name:  "value empty",
			setup: func(t *testing.T) { setEnrollSecretValue(t, "") },
		},
		{
			name:  "value whitespace only",
			setup: func(t *testing.T) { setEnrollSecretValue(t, "   ") },
		},
		{
			// The installer seeds unset properties with this sentinel; it must not reach the enroll path.
			name:  "value is the installer placeholder",
			setup: func(t *testing.T) { setEnrollSecretValue(t, constant.UnusedFlagKeyword) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			secret, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
			require.ErrorIs(t, err, ErrEnrollSecretNotFound)
			require.Empty(t, secret)
		})
	}
}

func TestGetEnrollSecretReturnsDeliveredSecret(t *testing.T) {
	setEnrollSecretValue(t, "  s3cret-value  ")

	secret, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	require.Equal(t, "s3cret-value", secret, "surrounding whitespace should be trimmed")
}

func TestClearEnrollSecretRemovesTheSecret(t *testing.T) {
	setEnrollSecretValue(t, "s3cret-value")

	require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))

	_, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.ErrorIs(t, err, ErrEnrollSecretNotFound, "a cleared secret must not be readable again")
}

func TestClearEnrollSecretIsIdempotent(t *testing.T) {
	// Clearing a key that was never created, and one whose value is already gone, are both the
	// ordinary state on a host that has already adopted its secret.
	require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))

	setEnrollSecretValue(t, "s3cret-value")
	require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))
	require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))
}

func TestGetEnrollSecretDoesNotLeakTheSecretIntoErrors(t *testing.T) {
	// A returned error is logged by the caller, so it must never carry the value.
	setEnrollSecretValue(t, "s3cret-value")

	_, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath+`\missing`)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cret-value")
}
