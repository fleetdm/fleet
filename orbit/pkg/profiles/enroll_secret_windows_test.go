//go:build windows

package profiles

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
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

// writeEnrollSecretValue writes into a key the test has already prepared. It leaves the key itself
// alone, so a watch armed beforehand stays registered on the handle it was given.
func writeEnrollSecretValue(t *testing.T, value string) {
	t.Helper()

	key, err := registry.OpenKey(registry.CURRENT_USER, testEnrollSecretKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, key.SetStringValue(enrollSecretValueName, value))
	require.NoError(t, key.Close())
}

// createTestKey leaves the key present but empty, which is what orbit finds on a host that has not
// been handed a secret yet. Every test that touches the key starts here, directly or through
// setEnrollSecretValue, so that each one gets the key in a known state.
func createTestKey(t *testing.T) {
	t.Helper()

	// A run that could not clean up must not decide this one.
	deleteTestKey()
	t.Cleanup(deleteTestKey)

	key, _, err := registry.CreateKey(registry.CURRENT_USER, testEnrollSecretKeyPath, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, key.Close())
}

// deleteTestKey restores inheritance before deleting. A test that protected the key left the
// running user without DELETE on it, and only the owner's implicit WRITE_DAC gets it back. Without
// this, a non-elevated run would strand an unreadable key and fail every run after it.
func deleteTestKey() {
	//nolint:errcheck // best effort: the key may not exist, or may never have been protected
	windows.SetNamedSecurityInfo(
		registryObjectName(registry.CURRENT_USER, testEnrollSecretKeyPath),
		windows.SE_REGISTRY_KEY,
		windows.DACL_SECURITY_INFORMATION|windows.UNPROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, nil, nil,
	)
	_ = registry.DeleteKey(registry.CURRENT_USER, testEnrollSecretKeyPath)
}

// The functions under test take the root and path so that tests can point them at HKCU. Every test
// uses the same pair, so bind it once here rather than at each call.
func getTestSecret() (string, error) {
	return getEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
}

func clearTestSecret() error {
	return clearEnrollSecret(registry.CURRENT_USER, testEnrollSecretKeyPath)
}

func protectTestKey() error {
	return ensureEnrollSecretKeyIsProtected(registry.CURRENT_USER, testEnrollSecretKeyPath)
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
			// The installer seeds unset properties with this sentinel; it must not reach the enroll path.
			name:  "value is the installer placeholder",
			setup: func(t *testing.T) { setEnrollSecretValue(t, constant.UnusedFlagKeyword) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			secret, err := getTestSecret()
			require.ErrorIs(t, err, ErrEnrollSecretNotFound)
			require.Empty(t, secret)
		})
	}
}

func TestGetEnrollSecretReturnsDeliveredSecret(t *testing.T) {
	setEnrollSecretValue(t, "  s3cret-value  ")

	secret, err := getTestSecret()
	require.NoError(t, err)
	require.Equal(t, "s3cret-value", secret, "surrounding whitespace should be trimmed")
}

func TestClearEnrollSecret(t *testing.T) {
	// A key that was never created and a key whose value is already gone are both the ordinary state
	// on a host that has already adopted its secret, so all three cases have to end the same way.
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
			name:  "value present",
			setup: func(t *testing.T) { setEnrollSecretValue(t, "s3cret-value") },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			require.NoError(t, clearTestSecret())
			require.NoError(t, clearTestSecret(), "clearing again must stay a no-op")

			_, err := getTestSecret()
			require.ErrorIs(t, err, ErrEnrollSecretNotFound, "a cleared secret must not be readable again")
		})
	}
}

// The resulting DACL was also verified by hand against a real Windows 11 host: the key comes back
// as O:SYG:SYD:PAI(A;;KA;;;SY)(A;;KA;;;BA), with SYSTEM and Administrators holding FullControl,
// inheritance disabled, and no entry for Users.
//
// HKCU stands in for HKLM so that creating the key needs no elevation, but the DACL itself names
// only SYSTEM and Administrators, so applying it drops the running user's own access. That makes
// the assertions below hold only for an elevated run.
func TestEnsureEnrollSecretKeyIsProtectedIsIdempotent(t *testing.T) {
	deleteTestKey()
	t.Cleanup(deleteTestKey)
	objectName := registryObjectName(registry.CURRENT_USER, testEnrollSecretKeyPath)

	require.False(t, enrollSecretKeyWasProtected(objectName), "a key that does not exist is not protected")

	require.NoError(t, protectTestKey())
	require.True(t, enrollSecretKeyWasProtected(objectName), "the key must carry the DACL we set")

	// Called on every orbit start, so it has to be safe to repeat. The repeat takes the early return,
	// which is only correct if the check above recognizes the DACL the first call applied.
	require.NoError(t, protectTestKey())
	require.True(t, enrollSecretKeyWasProtected(objectName))
}

// A secret found in an unprotected key is kept, not discarded. The MDM write creates the key itself
// whenever it lands before orbit has ever run, so discarding would fail the enrollment it was sent
// for. The secret is single use and is spent the moment orbit enrolls with it, which is the control;
// the DACL is hardening on top of that.
func TestEnsureEnrollSecretKeyIsProtectedKeepsADeliveredSecret(t *testing.T) {
	// Stand in for the MDM write arriving first and creating the key with inherited permissions.
	setEnrollSecretValue(t, "delivered-before-orbit-ran")

	require.NoError(t, protectTestKey())

	secret, err := getTestSecret()
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

// armTestWatch arms a watch on the test key and closes it when the test ends.
func armTestWatch(t *testing.T) *EnrollSecretWatch {
	t.Helper()

	watch, err := armEnrollSecretWatch(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, watch.Close()) })
	return watch
}

// waitWithDeadline runs Wait on its own goroutine. Wait blocks in a Win32 call that nothing in Go
// can interrupt, so a registration that never fires would hang the whole package instead of failing
// the test that broke it.
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
	secret, err := getTestSecret()
	require.NoError(t, err)
	require.Equal(t, "s3cret-value", secret)
}

func TestEnrollSecretWatchFiresWhenTheSecretIsCleared(t *testing.T) {
	// Recovery resends the profile, so the value changing in either direction has to wake the caller.
	setEnrollSecretValue(t, "s3cret-value")
	watch := armTestWatch(t)

	require.NoError(t, clearTestSecret())

	require.NoError(t, waitWithDeadline(t, t.Context(), watch))
}

func TestEnrollSecretWatchDoesNotFireUntilTheKeyChanges(t *testing.T) {
	// Without this, a Wait that returned unconditionally would satisfy every other watch test here.
	// It also covers cancellation arriving while Wait is already blocked, which is the real sequence:
	// orbit blocks on an armed watch and the service is asked to stop.
	createTestKey(t)
	watch := armTestWatch(t)

	const blockFor = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(t.Context(), blockFor)
	defer cancel()

	start := time.Now()
	require.NoError(t, waitWithDeadline(t, ctx, watch))
	require.Greater(t, time.Since(start), blockFor/2, "Wait returned before the key changed or ctx expired")
}

func TestEnrollSecretWatchUnblocksWhenTheContextIsAlreadyDone(t *testing.T) {
	createTestKey(t)

	watch, err := armEnrollSecretWatch(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// Wait signals its own event to break the Win32 block, and here that signal can land before the
	// wait even starts. Only a manual-reset event stays signalled long enough for that to work.
	require.NoError(t, waitWithDeadline(t, ctx, watch))

	// Close releases the event handle and Windows recycles handle values, so Wait must not leave a
	// goroutine behind that could signal the recycled handle afterwards.
	require.NoError(t, watch.Close())
}

func TestArmEnrollSecretWatchFailsWhenTheKeyIsMissing(t *testing.T) {
	deleteTestKey()

	_, err := armEnrollSecretWatch(registry.CURRENT_USER, testEnrollSecretKeyPath)
	require.Error(t, err)
}
