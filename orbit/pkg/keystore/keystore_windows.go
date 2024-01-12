//go:build windows && cgo

package keystore

import (
	"github.com/danieljoos/wincred"
)

// Using a var instead of const so that it can be overridden in tests.
var service = "com.fleetdm.fleetd.enroll.secret"

func Exists() bool {
	return true
}

func Name() string {
	return "Credential Manager"
}

// AddSecret will add a secret to the keychain. This secret can be retrieved by this application without any user authorization.
func AddSecret(secret string) error {
	cred := wincred.NewGenericCredential(service)
	cred.CredentialBlob = []byte(secret)
	err := cred.Write()
	return err
}

// UpdateSecret will update a secret in the keychain. This secret can be retrieved by this application without any user authorization.
func UpdateSecret(secret string) error {
	return AddSecret(secret)
}

// GetSecret will retrieve a secret from the keychain. If the secret was added by user or another application,
// then this application needs to be authorized to retrieve the secret.
func GetSecret() (string, error) {
	cred, err := wincred.GetGenericCredential(service)
	if err != nil {
		return "", err
	}
	return string(cred.CredentialBlob), nil
}
