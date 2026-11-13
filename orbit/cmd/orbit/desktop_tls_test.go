package main

import (
	"os"
	osuser "os/user"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareDesktopTLSFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Chown not supported on Windows")
	}

	rootDir := t.TempDir()

	// Create source cert/key files.
	certContent := []byte("-----BEGIN CERTIFICATE-----\ntest cert\n-----END CERTIFICATE-----\n")
	keyContent := []byte("-----BEGIN PRIVATE KEY-----\ntest key\n-----END PRIVATE KEY-----\n")
	certPath := filepath.Join(rootDir, "fleet_client.crt")
	keyPath := filepath.Join(rootDir, "fleet_client.key")
	require.NoError(t, os.WriteFile(certPath, certContent, 0o600))
	require.NoError(t, os.WriteFile(keyPath, keyContent, 0o600))

	currentUser, err := osuser.Current()
	require.NoError(t, err)

	d := &desktopRunner{
		rootDir:             rootDir,
		fleetClientCertPath: certPath,
		fleetClientKeyPath:  keyPath,
	}

	t.Run("creates user-readable copies", func(t *testing.T) {
		certCopy, keyCopy, err := d.prepareDesktopTLSFiles(currentUser.Username)
		require.NoError(t, err)

		assert.Equal(t, filepath.Join(rootDir, ".desktop_fleet_client.crt"), certCopy)
		assert.Equal(t, filepath.Join(rootDir, ".desktop_fleet_client.key"), keyCopy)

		// Verify content matches.
		gotCert, err := os.ReadFile(certCopy)
		require.NoError(t, err)
		assert.Equal(t, certContent, gotCert)

		gotKey, err := os.ReadFile(keyCopy)
		require.NoError(t, err)
		assert.Equal(t, keyContent, gotKey)

		// Verify key has restricted permissions.
		keyInfo, err := os.Stat(keyCopy)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o400), keyInfo.Mode().Perm())
	})

	t.Run("rejects empty username", func(t *testing.T) {
		_, _, err := d.prepareDesktopTLSFiles("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot prepare TLS files without a target username")
	})

	t.Run("rejects nonexistent user", func(t *testing.T) {
		_, _, err := d.prepareDesktopTLSFiles("nonexistent_user_abc123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup uid")
	})

	t.Run("fails when cert source missing", func(t *testing.T) {
		d2 := &desktopRunner{
			rootDir:             rootDir,
			fleetClientCertPath: filepath.Join(rootDir, "nonexistent.crt"),
			fleetClientKeyPath:  keyPath,
		}
		_, _, err := d2.prepareDesktopTLSFiles(currentUser.Username)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read cert")
	})

	t.Run("fails when key source missing", func(t *testing.T) {
		d2 := &desktopRunner{
			rootDir:             rootDir,
			fleetClientCertPath: certPath,
			fleetClientKeyPath:  filepath.Join(rootDir, "nonexistent.key"),
		}
		_, _, err := d2.prepareDesktopTLSFiles(currentUser.Username)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read key")
	})
}
