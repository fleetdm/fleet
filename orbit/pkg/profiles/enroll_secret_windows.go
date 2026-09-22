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
//
// DACL only: EnsureEnrollSecretKey sets DACL_SECURITY_INFORMATION and passes no owner or group, so
// an owner in this string would be silently ignored. The key ends up owned by whoever created it.
const enrollSecretKeySDDL = "D:PAI(A;;KA;;;SY)(A;;KA;;;BA)"

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
	key, openedExisting, err := registry.CreateKey(root, path, registry.SET_VALUE)
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

	// Read the existing protection before overwriting it, so a secret that sat in an unprotected key
	// can be told apart from one that was protected all along.
	wasProtected := enrollSecretKeyWasProtected(objectName)

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

	// Any value that was in the key while it was not protected has to be treated as disclosed, so it is
	// discarded rather than adopted and Fleet delivers a fresh one into the now-protected key. Two ways
	// that happens:
	//
	//   - We created the key just now. Creating and securing it are separate operations, so it was
	//     briefly readable through whatever it inherited, and a value present already can only have
	//     been written in that window.
	//   - The key already existed without our DACL. The likely author is the MDM write itself landing
	//     before orbit ever ran, which creates the key with inherited, user-readable permissions.
	//
	// A key that already carried our DACL is left alone: its value was protected the whole time.
	if !openedExisting || !wasProtected {
		if err := clearEnrollSecret(root, path); err != nil {
			return fmt.Errorf("discard a secret held in %s before it was protected: %w", path, err)
		}
	}
	return nil
}

// enrollSecretKeyWasProtected reports whether the key already carried the DACL we apply. A key we
// cannot read the security of is reported as unprotected, because the point of the check is to prove
// protection rather than to assume it.
func enrollSecretKeyWasProtected(objectName string) bool {
	securityDescriptor, err := windows.GetNamedSecurityInfo(
		objectName, windows.SE_REGISTRY_KEY, windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return false
	}
	// The owner and group vary by whoever created the key, so compare only the DACL we set.
	return strings.Contains(securityDescriptor.String(), enrollSecretKeySDDL)
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

// ArmEnrollSecretWatch registers for the next change to the key that carries the enroll secret and
// returns immediately. Windows tells us when that happens, so the caller does not poll: the profile
// that writes the value lands over its own MDM session, and RegNotifyChangeKeyValue reports it as
// soon as it does rather than on an interval chosen in advance.
//
// The caller waits with Wait and releases the registration with Close. Windows guarantees a single
// notification per registration, so a caller that keeps watching arms a new one each time.
func ArmEnrollSecretWatch() (*EnrollSecretWatch, error) {
	return armEnrollSecretWatch(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// EnrollSecretWatch is an armed, one-shot registration for changes to the key that carries the
// enroll secret. Arming is separate from waiting on purpose: a caller must arm, *then* read, then
// wait. Reading first leaves a gap in which a write is seen by neither the read nor the not-yet-armed
// registration, and the change would not be noticed until something else woke the caller.
type EnrollSecretWatch struct {
	key   registry.Key
	event windows.Handle
	path  string
}

func armEnrollSecretWatch(root registry.Key, path string) (*EnrollSecretWatch, error) {
	key, err := registry.OpenKey(root, path, registry.NOTIFY)
	if err != nil {
		return nil, fmt.Errorf("open %s to watch: %w", path, err)
	}

	// Manual-reset, initially unsignalled: Wait is the only consumer.
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		key.Close()
		return nil, fmt.Errorf("create registry change event: %w", err)
	}

	if err := windows.RegNotifyChangeKeyValue(
		windows.Handle(key),
		false, // this key only; the secret lives directly under it
		windows.REG_NOTIFY_CHANGE_LAST_SET,
		event,
		true, // asynchronous: signal the event rather than blocking this call
	); err != nil {
		windows.CloseHandle(event) //nolint:errcheck // nothing actionable on close failure
		key.Close()
		return nil, fmt.Errorf("watch %s for changes: %w", path, err)
	}
	return &EnrollSecretWatch{key: key, event: event, path: path}, nil
}

// Wait blocks until the watched key changes or ctx is done, whichever comes first. Both mean "go look
// again", so only a failed wait is an error. The registration is spent afterwards either way, so a
// caller that wants to keep watching must Close this one and arm another.
func (w *EnrollSecretWatch) Wait(ctx context.Context) error {
	// Cancelling the context has to wake the wait, so signal the same event when ctx is done. The
	// goroutine is joined before returning, because Close releases the event handle and signalling a
	// released handle could reach whatever Windows recycled it for.
	stop := make(chan struct{})
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		select {
		case <-ctx.Done():
			_ = windows.SetEvent(w.event)
		case <-stop:
		}
	}()
	defer func() {
		close(stop)
		<-joined
	}()

	if _, err := windows.WaitForSingleObject(w.event, windows.INFINITE); err != nil {
		return fmt.Errorf("wait for %s to change: %w", w.path, err)
	}
	return nil
}

// Close releases the registration. It is safe to call after Wait and must be called exactly once.
func (w *EnrollSecretWatch) Close() error {
	err := windows.CloseHandle(w.event)
	if closeErr := w.key.Close(); err == nil {
		err = closeErr
	}
	return err
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
