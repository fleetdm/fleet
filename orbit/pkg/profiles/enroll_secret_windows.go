//go:build windows

package profiles

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// On Windows the enroll secret is delivered by a Fleet-managed configuration profile that writes it
// to the registry, rather than as an MSI property, so it is never visible to process listing at
// install time and never comes to rest in an MSI log or in secret.txt. orbit adopts the value and
// then clears it, so the presence of the value means "a secret is waiting" and its absence means
// orbit already took it. That makes one channel serve both first enrollment and recovery: an
// administrator resends the profile and the value reappears.
//
// Fleet already owns SOFTWARE\FleetDM\Orbit (the installer records the install path there), so the
// delivered secret lives alongside it rather than in a new hive. The MSI creates the key with a DACL
// restricted to SYSTEM and Administrators, matching the ACL it puts on secret.txt, so a non-admin
// local user cannot read a secret that has not been adopted yet.
const (
	enrollSecretKeyPath   = `SOFTWARE\FleetDM\Orbit`
	enrollSecretValueName = "EnrollSecret"
)

// enrollSecretKeySDDL keeps the key readable only by SYSTEM and Administrators, with inheritance
// disabled. It is the registry counterpart of the ACL the installer puts on secret.txt
// (O:SYG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;BA) in orbit/pkg/packaging/wix/transform.go), with KA
// (KEY_ALL_ACCESS) standing in for FA (FILE_ALL_ACCESS). Fleet deliberately strips regular users
// from secret.txt while every other orbit file leaves them read access, and a delivered secret that
// has not been adopted yet deserves the same treatment.
const enrollSecretKeySDDL = "O:SYG:SYD:PAI(A;;KA;;;SY)(A;;KA;;;BA)"

// EnsureEnrollSecretKey creates the key that carries the MDM-delivered enroll secret and applies the
// DACL above. orbit runs as LocalSystem, so it can both create the key and set its permissions;
// doing it here rather than in the installer means a fleetd that upgraded in place is protected
// without waiting for a new MSI, and it closes the window where a profile that arrives before orbit
// has ever run would otherwise create the key with inherited, world-readable permissions.
//
// The DACL is reapplied even when the key already exists, because the key may have been created
// implicitly by the write that delivered a secret into it. SYSTEM keeps full control, so reapplying
// never locks out the MDM channel that writes the value.
func EnsureEnrollSecretKey() error {
	return ensureEnrollSecretKey(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

func ensureEnrollSecretKey(root registry.Key, path string) error {
	key, _, err := registry.CreateKey(root, path, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := key.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}

	securityDescriptor, err := windows.SecurityDescriptorFromString(enrollSecretKeySDDL)
	if err != nil {
		return fmt.Errorf("parse enroll secret key security descriptor: %w", err)
	}
	dacl, _, err := securityDescriptor.DACL()
	if err != nil {
		return fmt.Errorf("read enroll secret key DACL: %w", err)
	}

	// SetNamedSecurityInfo names registry objects as MACHINE\... rather than HKEY_LOCAL_MACHINE\...
	objectName := registryObjectName(root, path)
	if err := windows.SetNamedSecurityInfo(
		objectName,
		windows.SE_REGISTRY_KEY,
		// PROTECTED_DACL disables inheritance, which is what keeps a permissive parent from granting
		// access back to users we just removed.
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil,
	); err != nil {
		return fmt.Errorf("set permissions on %s: %w", objectName, err)
	}
	return nil
}

func registryObjectName(root registry.Key, path string) string {
	switch root {
	case registry.CURRENT_USER:
		return `CURRENT_USER\` + path
	default:
		return `MACHINE\` + path
	}
}

// GetEnrollSecret returns the enroll secret Fleet MDM delivered to this device, or
// ErrEnrollSecretNotFound when none is waiting. It never puts the value in an error, so a returned
// error is safe to log.
func GetEnrollSecret() (string, error) {
	return getEnrollSecret(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// ClearEnrollSecret removes the delivered secret. It is called once orbit has adopted the value, so
// the window in which an unconsumed secret sits readable on disk is as short as orbit can make it.
// Clearing an already-absent value is not an error.
func ClearEnrollSecret() error {
	return clearEnrollSecret(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// getEnrollSecret and clearEnrollSecret take the root and path explicitly so tests can exercise them
// against a key they can actually create; writing under HKLM needs elevation a test may not have.
func getEnrollSecret(root registry.Key, path string) (string, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrEnrollSecretNotFound
		}
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer key.Close()

	secret, _, err := key.GetStringValue(enrollSecretValueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrEnrollSecretNotFound
		}
		return "", fmt.Errorf("read %s from %s: %w", enrollSecretValueName, path, err)
	}

	secret = strings.TrimSpace(secret)
	// The installer seeds unset properties with "dummy"; treat that, and an empty value, as nothing
	// waiting rather than handing an unusable secret to the enroll path.
	if secret == "" || secret == constant.UnusedFlagKeyword {
		return "", ErrEnrollSecretNotFound
	}
	return secret, nil
}

// WaitForEnrollSecretChange blocks until the key that carries the enroll secret changes, or until
// ctx is done. Windows can tell us when that happens, so the caller does not poll: the profile that
// writes the value lands over its own MDM session, and RegNotifyChangeKeyValue wakes us as soon as
// it does instead of on some interval chosen in advance.
//
// A single notification is all Windows guarantees per registration, so callers must re-arm by
// calling this again. It returns nil when something changed and when ctx is done, because both mean
// "go look again"; only a failure to register is an error.
func WaitForEnrollSecretChange(ctx context.Context) error {
	return waitForEnrollSecretChange(ctx, registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

func waitForEnrollSecretChange(ctx context.Context, root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.NOTIFY)
	if err != nil {
		// The key is created by the installer, so its absence is not something waiting will fix.
		return fmt.Errorf("open %s to watch: %w", path, err)
	}
	defer key.Close()

	// Manual-reset, initially unsignalled: the wait below is the only consumer.
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return fmt.Errorf("create registry change event: %w", err)
	}
	defer windows.CloseHandle(event) //nolint:errcheck // nothing actionable on close failure

	if err := windows.RegNotifyChangeKeyValue(
		windows.Handle(key),
		false, // this key only; the secret lives directly under it
		windows.REG_NOTIFY_CHANGE_LAST_SET,
		event,
		true, // asynchronous: signal the event rather than blocking this call
	); err != nil {
		return fmt.Errorf("watch %s for changes: %w", path, err)
	}

	// Cancelling the context has to wake the wait, so signal the same event when ctx is done. The
	// watcher goroutine is stopped on return so it cannot outlive this call.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = windows.SetEvent(event)
		case <-stop:
		}
	}()

	if _, err := windows.WaitForSingleObject(event, windows.INFINITE); err != nil {
		return fmt.Errorf("wait for %s to change: %w", path, err)
	}
	return nil
}

func clearEnrollSecret(root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open %s for write: %w", path, err)
	}
	defer key.Close()

	if err := key.DeleteValue(enrollSecretValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete %s from %s: %w", enrollSecretValueName, path, err)
	}
	return nil
}
