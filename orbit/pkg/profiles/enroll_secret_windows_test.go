//go:build windows

package profiles

import (
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

// testEnrollSecretKeyPath mirrors the production key under HKCU so creating it needs no elevation.
const testEnrollSecretKeyPath = `SOFTWARE\FleetDM\OrbitTest`

// setEnrollSecretValue prepares the key and writes a delivered secret into it.
func setEnrollSecretValue(t *testing.T, value string) {
	t.Helper()
	createTestKey(t)

	key, err := registry.OpenKey(registry.CURRENT_USER, testEnrollSecretKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, key.SetStringValue(enrollSecretValueName, value))
	require.NoError(t, key.Close())
}

// createTestKey leaves the key present but empty, which is what orbit finds on a host that has not
// been handed a secret yet.
func createTestKey(t *testing.T) {
	t.Helper()

	deleteTestKey() // a run that could not clean up must not decide this one
	t.Cleanup(deleteTestKey)

	key, _, err := registry.CreateKey(registry.CURRENT_USER, testEnrollSecretKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, key.Close())
}

func deleteTestKey() {
	_ = registry.DeleteKey(registry.CURRENT_USER, testEnrollSecretKeyPath)
}

func TestGetEnrollSecretReportsNothingWaiting(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
	}{
		{
			name:  "key absent",
			setup: func(t *testing.T) { deleteTestKey() },
		},
		{
			name:  "value absent",
			setup: createTestKey,
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

func TestClearEnrollSecret(t *testing.T) {
	setEnrollSecretValue(t, "s3cret-value")

	require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))
	require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))

	_, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.ErrorIs(t, err, ErrEnrollSecretNotFound, "a cleared secret must not be readable again")

	// A key that was never created is the same no-op.
	deleteTestKey()
	require.NoError(t, clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath))
}
