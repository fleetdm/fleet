//go:build windows && cgo

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
	assert.True(t, Exists())
}

func TestName(t *testing.T) {
	t.Parallel()
	assert.True(t, strings.Contains(Name(), "Credential Manager"))
}

func TestSecret(t *testing.T) {
	t.Parallel()
	t.Cleanup(
		func() {
			cred, err := wincred.GetGenericCredential(service)
			if err != nil {
				t.Log(err)
				return
			}
			_ = cred.Delete()
		},
	)

	service = "com.fleetdm.fleetd.enroll.secret.test"

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
