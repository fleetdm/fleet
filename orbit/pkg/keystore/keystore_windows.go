//go:build windows

package keystore

import (
	"errors"
	"github.com/danieljoos/wincred"
	"strings"
	"sync"
	"syscall"
)

// Using a var instead of const so that it can be overridden in tests.
var service = "com.fleetdm.fleetd.enroll.secret"
var mu sync.Mutex

func Supported() bool {
	return true
}

func Name() string {
	return "Credential Manager"
}

// AddSecret will add a secret to the Credential Manager. This secret can be retrieved by this user without additional authorization.
func AddSecret(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("secret cannot be empty")
	}

	mu.Lock()
	defer mu.Unlock()

	cred := wincred.NewGenericCredential(service)
	cred.CredentialBlob = []byte(secret)
	err := cred.Write()
	return err
}

// UpdateSecret will update a secret in the Credential Manager.
func UpdateSecret(secret string) error {
	return AddSecret(secret)
}

// GetSecret will retrieve a secret from the Credential Manager. If secret doesn't exist, it will return "", nil.
func GetSecret() (string, error) {
	mu.Lock()
	defer mu.Unlock()

	cred, err := wincred.GetGenericCredential(service)
	if err != nil {
		var errno syscall.Errno
		ok := errors.As(err, &errno)
		if ok && errors.Is(errno, syscall.ERROR_NOT_FOUND) {
			return "", nil
		}
		return "", err
	}
	return string(cred.CredentialBlob), nil
}
