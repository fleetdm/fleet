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

// On Windows MDM the enroll secret is delivered by a Fleet-managed configuration profile that writes it to the registry. orbit
// loads the value and then clears it, so the presence of the value means "a secret is waiting" and its absence means orbit
// already took it. That makes one channel serve both first enrollment and recovery: an administrator resends the profile (with a
// new secret) and the value reappears.
//
// Fleet already owns SOFTWARE\FleetDM\Orbit (the installer records the install path there), so the delivered secret lives
// alongside it rather than in a new hive. A non-admin local user cannot read a secret that has not been loaded yet.
const (
	enrollSecretKeyPath   = `SOFTWARE\FleetDM\Orbit`
	enrollSecretValueName = "EnrollSecret"
)

// enrollSecretKeySDDL keeps the key readable only by SYSTEM and Administrators, with inheritance disabled.
// Reference: https://learn.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-string-format
//
// DACL (Discretionary Access Control List) only:
//
//	P   protected, inheritance blocked
//	AI  auto-inherited
//	A   access allowed
//	KA  KEY_ALL_ACCESS
//	SY  Local System
//	BA  Built-in Administrators
const enrollSecretKeySDDL = "D:PAI(A;;KA;;;SY)(A;;KA;;;BA)"

// EnsureEnrollSecretKeyIsProtected applies the DACL above to the key that carries the MDM-delivered enroll secret, creating the
// key first when it is missing. The key has to exist before a secret is wanted.
func EnsureEnrollSecretKeyIsProtected() error {
	return ensureEnrollSecretKeyIsProtected(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

func ensureEnrollSecretKeyIsProtected(root registry.Key, path string) error {
	// SetNamedSecurityInfo names registry objects as MACHINE\... rather than HKEY_LOCAL_MACHINE\...
	objectName := registryObjectName(root, path)

	if enrollSecretKeyWasProtected(objectName) {
		return nil
	}

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

	if err := windows.SetNamedSecurityInfo(
		objectName,
		windows.SE_REGISTRY_KEY,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil,
	); err != nil {
		return fmt.Errorf("set permissions on %s: %w", objectName, err)
	}

	return nil
}

// enrollSecretKeyWasProtected reports whether the key already carried the DACL we apply. A key we cannot read the security of is
// reported as unprotected, because the point of the check is to prove protection rather than to assume it.
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

// GetEnrollSecret returns the enroll secret Fleet MDM delivered to this device, or ErrEnrollSecretNotFound when none is waiting.
func GetEnrollSecret() (string, error) {
	return getEnrollSecret(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// ClearEnrollSecret removes the delivered secret. It is called once orbit has loaded the value.
func ClearEnrollSecret() error {
	return clearEnrollSecret(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// getEnrollSecret and clearEnrollSecret take the root and path explicitly so tests can exercise them.
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
	// The installer seeds unset properties with "dummy"; treat that, and an empty value, as nothing waiting.
	if secret == "" || secret == constant.UnusedFlagKeyword {
		return "", ErrEnrollSecretNotFound
	}
	return secret, nil
}

// ArmEnrollSecretWatch registers for the next change to the key that carries the enroll secret and returns immediately. Windows
// tells us when that happens, so the caller does not poll: the profile that writes the value lands over its own MDM session, and
// RegNotifyChangeKeyValue reports it as soon as it does.
func ArmEnrollSecretWatch() (*EnrollSecretWatch, error) {
	return armEnrollSecretWatch(registry.LOCAL_MACHINE, enrollSecretKeyPath)
}

// EnrollSecretWatch is an armed, one-shot registration for changes to the key that carries the enroll secret.
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

// Wait blocks until the watched key changes or ctx is done, whichever comes first. Both mean "go look again". The registration is
// spent afterwards either way, so a caller that wants to keep watching must Close this one and arm another.
func (w *EnrollSecretWatch) Wait(ctx context.Context) error {
	// stop signals that registry fired first and we are exiting this function.
	stop := make(chan struct{})
	// joined signals that the goroutine has exited and it is safe to close the event handle.
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

	// This is a blocking Win32 call.
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
