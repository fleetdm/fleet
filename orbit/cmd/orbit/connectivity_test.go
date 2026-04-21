package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTarget(t *testing.T) {
	srcCert, err := os.ReadFile(filepath.Join("..", "..", "pkg", "cryptoinfo", "testdata", "test_crt.pem"))
	require.NoError(t, err)

	rootDir := t.TempDir()
	urlFile := filepath.Join(rootDir, constant.FleetURLFileName)
	require.NoError(t, os.WriteFile(urlFile, []byte("https://enrolled.example.com\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "certs.pem"), srcCert, 0o600))

	t.Run("fleet-url flag", func(t *testing.T) {
		url, pool, insecure, err := resolveTarget(resolveInput{fleetURL: "https://f.example.com"})
		require.NoError(t, err)
		assert.Equal(t, "https://f.example.com", url)
		assert.Nil(t, pool)
		assert.False(t, insecure)
	})

	t.Run("adds https scheme when missing", func(t *testing.T) {
		url, _, _, err := resolveTarget(resolveInput{fleetURL: "fleet.example.com"})
		require.NoError(t, err)
		assert.Equal(t, "https://fleet.example.com", url)
	})

	t.Run("missing url errors", func(t *testing.T) {
		_, _, _, err := resolveTarget(resolveInput{})
		require.Error(t, err)
	})

	t.Run("from-enrollment reads url and certs.pem", func(t *testing.T) {
		url, pool, _, err := resolveTarget(resolveInput{fromEnrollment: true, rootDir: rootDir})
		require.NoError(t, err)
		assert.Equal(t, "https://enrolled.example.com", url)
		assert.NotNil(t, pool)
	})

	t.Run("from-enrollment without root-dir errors", func(t *testing.T) {
		_, _, _, err := resolveTarget(resolveInput{fromEnrollment: true})
		require.Error(t, err)
	})

	t.Run("from-enrollment with conflicting url errors", func(t *testing.T) {
		_, _, _, err := resolveTarget(resolveInput{
			fromEnrollment: true,
			rootDir:        rootDir,
			fleetURL:       "https://other.example.com",
		})
		require.Error(t, err)
	})

	t.Run("from-enrollment with matching url is fine", func(t *testing.T) {
		_, _, _, err := resolveTarget(resolveInput{
			fromEnrollment: true,
			rootDir:        rootDir,
			fleetURL:       "https://enrolled.example.com",
		})
		require.NoError(t, err)
	})

	t.Run("insecure + cert errors", func(t *testing.T) {
		_, _, _, err := resolveTarget(resolveInput{
			fleetURL: "https://f.example.com",
			certPath: filepath.Join(rootDir, "certs.pem"),
			insecure: true,
		})
		require.Error(t, err)
	})

	t.Run("invalid cert path errors", func(t *testing.T) {
		_, _, _, err := resolveTarget(resolveInput{
			fleetURL: "https://f.example.com",
			certPath: filepath.Join(rootDir, "does-not-exist.pem"),
		})
		require.Error(t, err)
	})
}

func TestResolveTargetMissingURLFile(t *testing.T) {
	_, _, _, err := resolveTarget(resolveInput{fromEnrollment: true, rootDir: t.TempDir()})
	require.Error(t, err)
}
