//go:build darwin && cgo

package keystore

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestExists(t *testing.T) {
	t.Parallel()
	assert.True(t, Exists())
}

func TestName(t *testing.T) {
	t.Parallel()
	assert.True(t, strings.Contains(Name(), "keychain"))
}

func TestSecret(t *testing.T) {
	t.Parallel()
	t.Cleanup(
		func() {
			_ = deleteSecret()
		},
	)

	serviceStringRef = stringToCFString("com.fleetdm.fleetd.enroll.secret.test")

	// Add secret
	secret := "testSecret"
	require.NoError(t, AddSecret(secret))
	result, err := GetSecret()
	require.NoError(t, err)
	assert.Equal(t, secret, result)

	// Update secret
	secret = "updatedSecret"
	require.NoError(t, UpdateSecret(secret))
	result, err = GetSecret()
	require.NoError(t, err)
	assert.Equal(t, secret, result)
}
