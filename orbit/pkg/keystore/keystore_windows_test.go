//go:build windows

package keystore

import (
	"github.com/danieljoos/wincred"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestExists(t *testing.T) {
	t.Parallel()
	assert.True(t, Supported())
}

func TestName(t *testing.T) {
	t.Parallel()
	assert.True(t, strings.Contains(Name(), "Credential Manager"))
}

func TestSecret(t *testing.T) {
	t.Parallel()

	// Use a different service name for testing
	origService := service
	service = "com.fleetdm.fleetd.enroll.secret.test"

	deleteSecret := func() {
		mu.Lock()
		defer mu.Unlock()
		cred, err := wincred.GetGenericCredential(service)
		if err != nil {
			return
		}
		_ = cred.Delete()
	}

	t.Cleanup(
		func() {
			deleteSecret()
			service = origService
		},
	)

	// Make sure the secret doesn't exist
	deleteSecret()

	// Get secret -- should be empty
	result, err := GetSecret()
	require.NoError(t, err)
	assert.Equal(t, "", result)

	// Add empty secret
	assert.Error(t, AddSecret(""))

	// Add secret
	secret := "testSecret"
	require.NoError(t, AddSecret(secret))
	result, err = GetSecret()
	require.NoError(t, err)
	assert.Equal(t, secret, result)

	// Update empty secret
	assert.Error(t, UpdateSecret(""))

	// Update secret
	secret = "updatedSecret"
	require.NoError(t, UpdateSecret(secret))
	result, err = GetSecret()
	require.NoError(t, err)
	assert.Equal(t, secret, result)
}
