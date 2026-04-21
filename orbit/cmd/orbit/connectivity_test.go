package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTargetOverride(t *testing.T) {
	srcCert, err := os.ReadFile(filepath.Join("..", "..", "pkg", "cryptoinfo", "testdata", "test_crt.pem"))
	require.NoError(t, err)
	certDir := t.TempDir()
	certPath := filepath.Join(certDir, "certs.pem")
	require.NoError(t, os.WriteFile(certPath, srcCert, 0o600))

	t.Run("bare URL gets https prefix", func(t *testing.T) {
		tgt, err := resolveTarget(resolveInput{fleetURLOverride: "fleet.example.com"})
		require.NoError(t, err)
		assert.Equal(t, "https://fleet.example.com", tgt.baseURL)
		assert.Empty(t, tgt.orbitNodeKey, "override mode must never send orbit node key")
	})

	t.Run("URL with scheme preserved", func(t *testing.T) {
		tgt, err := resolveTarget(resolveInput{fleetURLOverride: "http://fleet.local:8080"})
		require.NoError(t, err)
		assert.Equal(t, "http://fleet.local:8080", tgt.baseURL)
	})

	t.Run("cert override loads", func(t *testing.T) {
		tgt, err := resolveTarget(resolveInput{fleetURLOverride: "https://f", certOverride: certPath})
		require.NoError(t, err)
		assert.NotNil(t, tgt.rootCAs)
	})

	t.Run("insecure + cert errors", func(t *testing.T) {
		_, err := resolveTarget(resolveInput{
			fleetURLOverride: "https://f", certOverride: certPath, insecure: true,
		})
		require.Error(t, err)
	})

	t.Run("bad cert errors", func(t *testing.T) {
		_, err := resolveTarget(resolveInput{
			fleetURLOverride: "https://f", certOverride: filepath.Join(certDir, "does-not-exist.pem"),
		})
		require.Error(t, err)
	})
}

func TestResolveTargetFromEnrollment(t *testing.T) {
	srcCert, err := os.ReadFile(filepath.Join("..", "..", "pkg", "cryptoinfo", "testdata", "test_crt.pem"))
	require.NoError(t, err)

	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, constant.FleetURLFileName),
		[]byte("https://enrolled.example.com\n"), 0o600,
	))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "certs.pem"), srcCert, 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, constant.OrbitNodeKeyFileName),
		[]byte("enrolled-orbit-key\n"), 0o600,
	))

	tgt, err := resolveTarget(resolveInput{rootDir: rootDir})
	require.NoError(t, err)
	assert.Equal(t, "https://enrolled.example.com", tgt.baseURL)
	assert.NotNil(t, tgt.rootCAs, "certs.pem in root-dir must be loaded")
	assert.Equal(t, "enrolled-orbit-key", tgt.orbitNodeKey)
}

func TestResolveTargetFromEnrollmentMissingState(t *testing.T) {
	t.Run("no root-dir", func(t *testing.T) {
		_, err := resolveTarget(resolveInput{})
		require.Error(t, err)
	})
	t.Run("root-dir without fleet_url", func(t *testing.T) {
		_, err := resolveTarget(resolveInput{rootDir: t.TempDir()})
		require.Error(t, err)
	})
	t.Run("empty fleet_url file", func(t *testing.T) {
		rootDir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(rootDir, constant.FleetURLFileName), []byte("\n"), 0o600,
		))
		_, err := resolveTarget(resolveInput{rootDir: rootDir})
		require.Error(t, err)
	})
}

func TestResolveTargetOptionalOrbitKey(t *testing.T) {
	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, constant.FleetURLFileName),
		[]byte("https://fleet.example.com\n"), 0o600,
	))

	tgt, err := resolveTarget(resolveInput{rootDir: rootDir})
	require.NoError(t, err)
	assert.Empty(t, tgt.orbitNodeKey, "missing node-key file must not error")
}
