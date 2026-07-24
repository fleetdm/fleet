package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func newTestApp(writer io.Writer) *cli.App {
	return &cli.App{
		Writer:    writer,
		ErrWriter: io.Discard,
		Flags:     []cli.Flag{&cli.StringFlag{Name: "root-dir"}},
		Commands:  []*cli.Command{connectivityCommand},
		// cli.ExitErrHandler is a no-op so os.Exit isn't called in tests.
		ExitErrHandler: func(*cli.Context, error) {},
	}
}

// fleetishServer returns an httptest server that sets the Fleet capabilities
// header on every response and echoes 200 — enough to pass the default
// unauthenticated fingerprint checks for most of the catalogue.
func fleetishServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(fleet.CapabilitiesHeader, "orbit_endpoints")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok","errors":[]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	if ec, ok := err.(cli.ExitCoder); ok {
		return ec.ExitCode()
	}
	return -1
}

func TestConnectivityCommand_Reachable(t *testing.T) {
	srv := fleetishServer(t)
	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{
		"orbit", "connectivity-check",
		"--fleet-url", srv.URL,
		"--features", "osquery,fleet-desktop",
		"--timeout", "2s",
	})
	assert.Equal(t, 0, exitCodeOf(err))
	assert.Contains(t, out.String(), "Summary:")
	assert.Contains(t, out.String(), srv.URL)
}

func TestConnectivityCommand_Blocked(t *testing.T) {
	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{
		"orbit", "connectivity-check",
		"--fleet-url", "http://127.0.0.1:1",
		"--features", "osquery",
		"--timeout", "1s",
	})
	assert.Equal(t, 1, exitCodeOf(err))
	assert.Contains(t, out.String(), "blocked")
}

func TestConnectivityCommand_NotFleet(t *testing.T) {
	// Returns 200 with no Fleet markers — fingerprinted endpoints will flag
	// this as not-fleet (exit 3).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<html>not fleet</html>`))
	}))
	t.Cleanup(srv.Close)

	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{
		"orbit", "connectivity-check",
		"--fleet-url", srv.URL,
		"--features", "fleetctl",
		"--timeout", "2s",
	})
	assert.Equal(t, 3, exitCodeOf(err))
	assert.Contains(t, out.String(), "not-fleet")
}

func TestConnectivityCommand_List(t *testing.T) {
	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{"orbit", "connectivity-check", "--list"})
	assert.Equal(t, 0, exitCodeOf(err))
	assert.Contains(t, out.String(), "/api/osquery/enroll")
	assert.Contains(t, out.String(), "/mdm/apple/scep")
}

func TestConnectivityCommand_JSON(t *testing.T) {
	srv := fleetishServer(t)
	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{
		"orbit", "connectivity-check",
		"--fleet-url", srv.URL,
		"--features", "osquery",
		"--timeout", "2s",
		"--json",
	})
	assert.Equal(t, 0, exitCodeOf(err))

	var parsed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out.Bytes(), &parsed), "output must be valid JSON")
	assert.Contains(t, parsed, "fleet_url")
	assert.Contains(t, parsed, "results")
	assert.Contains(t, parsed, "summary")
}

func TestConnectivityCommand_BadFeaturesIsUsageError(t *testing.T) {
	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{
		"orbit", "connectivity-check",
		"--features", "bogus",
	})
	assert.Equal(t, 2, exitCodeOf(err))
}

func TestConnectivityCommand_NegativeTimeoutIsUsageError(t *testing.T) {
	// Go's http.Client silently treats negative Timeout as no timeout. Reject
	// before probing so users don't inadvertently disable timeout enforcement.
	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{
		"orbit", "connectivity-check",
		"--fleet-url", "http://127.0.0.1:1",
		"--timeout", "-1s",
	})
	assert.Equal(t, 2, exitCodeOf(err))
}

func TestConnectivityCommand_MissingEnrollmentStateIsUsageError(t *testing.T) {
	// No --fleet-url and no --root-dir set → resolveTarget errors as usage.
	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{"orbit", "connectivity-check"})
	assert.Equal(t, 2, exitCodeOf(err))
}

func TestConnectivityCommand_FromEnrollmentState(t *testing.T) {
	srv := fleetishServer(t)
	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, constant.FleetURLFileName),
		[]byte(srv.URL+"\n"), 0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, constant.OrbitNodeKeyFileName),
		[]byte("enrolled-key\n"), 0o600,
	))

	var out bytes.Buffer
	app := newTestApp(&out)
	err := app.Run([]string{
		"orbit", "--root-dir", rootDir, "connectivity-check",
		"--features", "osquery",
		"--timeout", "2s",
	})
	assert.Equal(t, 0, exitCodeOf(err))
	assert.Contains(t, out.String(), srv.URL)
}

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

func TestResolveTargetCertStatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		// os.Symlink on Windows requires SeCreateSymbolicLinkPrivilege
		// (admin) or Developer Mode, which CI runners don't have.
		t.Skip("os.Symlink requires elevated privileges on Windows")
	}
	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, constant.FleetURLFileName),
		[]byte("https://fleet.example.com\n"), 0o600,
	))
	// Place a symlink at certs.pem that points to itself so os.Stat fails
	// with something other than ENOENT (ELOOP on macOS/Linux). Confirms we
	// surface stat failures instead of silently falling back to system roots.
	certPath := filepath.Join(rootDir, "certs.pem")
	require.NoError(t, os.Symlink(certPath, certPath))

	_, err := resolveTarget(resolveInput{rootDir: rootDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat fleet certificate")
}

func TestResolveTargetNodeKeyReadError(t *testing.T) {
	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, constant.FleetURLFileName),
		[]byte("https://fleet.example.com\n"), 0o600,
	))
	// Place a directory where the node-key file is expected so os.ReadFile
	// returns a non-ENOENT error (EISDIR). Confirms we surface such failures
	// instead of silently downgrading to unauthenticated probing.
	require.NoError(t, os.Mkdir(filepath.Join(rootDir, constant.OrbitNodeKeyFileName), 0o700))

	_, err := resolveTarget(resolveInput{rootDir: rootDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read orbit node key")
}
