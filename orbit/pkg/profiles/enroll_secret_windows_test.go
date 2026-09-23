//go:build windows

package profiles

import (
	"context"
	"testing"
	"time"

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
	writeEnrollSecretValue(t, value)
}

// writeEnrollSecretValue writes into a key the test already prepared, leaving the key itself alone
// so a watch armed beforehand stays registered.
func writeEnrollSecretValue(t *testing.T, value string) {
	t.Helper()

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

	require.NoError(t, ensureEnrollSecretKeyExists(registry.CURRENT_USER, testEnrollSecretKeyPath))
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

func TestEnsureEnrollSecretKeyExists(t *testing.T) {
	deleteTestKey()
	t.Cleanup(deleteTestKey)

	require.NoError(t, ensureEnrollSecretKeyExists(registry.CURRENT_USER, testEnrollSecretKeyPath))

	// The point of creating the key is that a watch can register on it; that is the only reason orbit does this.
	watch, err := armEnrollSecretWatch(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	require.NoError(t, watch.Close())

	// Called on every orbit start, so it has to be safe to repeat, and must not disturb a secret already delivered.
	writeEnrollSecretValue(t, "s3cret-value")
	require.NoError(t, ensureEnrollSecretKeyExists(registry.CURRENT_USER, testEnrollSecretKeyPath))
	secret, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	require.Equal(t, "s3cret-value", secret)
}

// armTestWatch arms a watch on the test key and closes it when the test ends.
func armTestWatch(t *testing.T) *EnrollSecretWatch {
	t.Helper()

	watch, err := armEnrollSecretWatch(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, watch.Close()) })
	return watch
}

// waitWithDeadline runs Wait on its own goroutine. Wait blocks in a Win32 call that nothing in Go can interrupt.
func waitWithDeadline(t *testing.T, ctx context.Context, watch *EnrollSecretWatch) error {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- watch.Wait(ctx) }()

	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("Wait did not return")
		return nil
	}
}

func TestEnrollSecretWatchFiresWhenTheSecretIsDelivered(t *testing.T) {
	createTestKey(t)
	// Arming before the write is the ordering orbit depends on: it registers, then blocks.
	watch := armTestWatch(t)

	writeEnrollSecretValue(t, "s3cret-value")

	require.NoError(t, waitWithDeadline(t, t.Context(), watch))

	// Waking is only useful if the value is readable by the time Wait returns.
	secret, err := getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	require.Equal(t, "s3cret-value", secret)
}

func TestEnrollSecretWatchBlocksUntilTheContextExpires(t *testing.T) {
	createTestKey(t)
	watch := armTestWatch(t)

	const blockFor = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(t.Context(), blockFor)
	defer cancel()

	start := time.Now()
	require.NoError(t, waitWithDeadline(t, ctx, watch))
	require.Greater(t, time.Since(start), blockFor/2, "Wait returned before the context expired")
}

func TestEnrollSecretWatchUnblocksWhenTheContextIsAlreadyDone(t *testing.T) {
	createTestKey(t)
	watch := armTestWatch(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, waitWithDeadline(t, ctx, watch))
}

func TestArmEnrollSecretWatchFailsWhenTheKeyIsMissing(t *testing.T) {
	deleteTestKey()

	watch, err := armEnrollSecretWatch(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.Error(t, err)

	require.NoError(t, watch.Close(), "closing a watch that was never armed must be a no-op")
}
