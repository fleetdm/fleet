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

// The resulting DACL was verified against a real Windows 11 host: the key comes back as
// O:SYG:SYD:PAI(A;;KA;;;SY)(A;;KA;;;BA), with SYSTEM and Administrators holding FullControl,
// inheritance disabled, and no entry for Users. This test covers the code path and its idempotency;
// HKCU stands in for HKLM so it runs without elevation.
func TestEnsureEnrollSecretKeyIsProtectedIsIdempotent(t *testing.T) {
	t.Cleanup(func() { _ = registry.DeleteKey(registry.CURRENT_USER, testEnrollSecretKeyPath) })

	require.NoError(t, ensureEnrollSecretKeyIsProtected(registry.CURRENT_USER, testEnrollSecretKeyPath))
	// Called on every orbit start, so it has to be safe to repeat against a key that already exists.
	require.NoError(t, ensureEnrollSecretKeyIsProtected(registry.CURRENT_USER, testEnrollSecretKeyPath))

	// A freshly prepared key holds no secret, and reading it must not look like a failure.
	_, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.ErrorIs(t, err, ErrEnrollSecretNotFound)

	// The key must be writable by the MDM channel afterwards, and readable by us.
	key, err := registry.OpenKey(registry.CURRENT_USER, testEnrollSecretKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, key.SetStringValue(enrollSecretValueName, "s3cret-value"))
	require.NoError(t, key.Close())

	secret, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	require.Equal(t, "s3cret-value", secret)
}

// A secret found in an unprotected key is kept, not discarded. The MDM write creates the key itself
// whenever it lands before orbit has ever run, so discarding would fail the enrollment it was sent
// for. The secret is single use and is spent the moment orbit enrolls with it, which is the control;
// the DACL is hardening on top of that.
func TestEnsureEnrollSecretKeyIsProtectedKeepsADeliveredSecret(t *testing.T) {
	t.Cleanup(func() { _ = registry.DeleteKey(registry.CURRENT_USER, testEnrollSecretKeyPath) })

	// Stand in for the MDM write arriving first and creating the key with inherited permissions.
	setEnrollSecretValue(t, "delivered-before-orbit-ran")

	require.NoError(t, ensureEnrollSecretKeyIsProtected(registry.CURRENT_USER, testEnrollSecretKeyPath))

	secret, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	require.Equal(t, "delivered-before-orbit-ran", secret, "a delivered secret must survive being secured")
}

func TestGetEnrollSecretDoesNotLeakTheSecretIntoErrors(t *testing.T) {
	// A returned error is logged by the caller, so it must never carry the value.
	setEnrollSecretValue(t, "s3cret-value")

	_, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath+`\missing`)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cret-value")
}
