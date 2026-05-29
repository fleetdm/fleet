package google_cloud_identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeServiceAccountKey returns a minimal but VALID JSON key payload that
// google.JWTConfigFromJSON will accept (it parses, generates an RSA key, and
// constructs a JWTConfig). The token source produced from it will not work
// against real Google APIs — the goal is only to exercise the parsing/return
// path in newServiceAccountTokenSource.
func fakeServiceAccountKey(t *testing.T) []byte {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	payload := map[string]string{ //nolint:gosec // G101: synthetic SA-JSON test fixture, no real credentials
		"type":           "service_account",
		"project_id":     "test-project",
		"private_key_id": "kid-1",
		"private_key":    string(pemBytes),
		"client_email":   "test-sa@test-project.iam.gserviceaccount.com",
		"client_id":      "1234567890",
		"auth_uri":       "https://accounts.google.com/o/oauth2/auth",
		"token_uri":      "https://oauth2.googleapis.com/token",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return b
}

func TestNewTokenSource_ConfigNotSet(t *testing.T) {
	t.Parallel()
	_, err := NewTokenSource(context.Background(), config.GoogleCloudIdentityConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config not set")
}

func TestNewTokenSource_SAJSONBytes(t *testing.T) {
	t.Parallel()
	cfg := baseValidConfig(t)
	cfg.ServiceAccountJSONBytes = string(fakeServiceAccountKey(t))

	ts, err := NewTokenSource(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ts)
}

func TestNewTokenSource_SAJSONFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sa.json")
	require.NoError(t, os.WriteFile(path, fakeServiceAccountKey(t), 0o600))

	cfg := baseValidConfig(t)
	cfg.ServiceAccountJSONBytes = "" // force the file-path branch
	cfg.ServiceAccountJSON = path

	ts, err := NewTokenSource(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ts)
}

func TestNewTokenSource_SAJSONBytesWinsOverFilePath(t *testing.T) {
	t.Parallel()
	// Even if file path is also configured, bytes takes precedence.
	cfg := baseValidConfig(t)
	cfg.ServiceAccountJSONBytes = string(fakeServiceAccountKey(t))
	cfg.ServiceAccountJSON = "/this/path/does/not/exist.json" // would error if read

	ts, err := NewTokenSource(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ts)
}

func TestNewTokenSource_SAJSONFileMissing(t *testing.T) {
	t.Parallel()
	cfg := baseValidConfig(t)
	cfg.ServiceAccountJSONBytes = ""
	cfg.ServiceAccountJSON = "/nonexistent/sa.json"

	_, err := NewTokenSource(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read service account JSON")
}

func TestNewTokenSource_SAJSONMalformed(t *testing.T) {
	t.Parallel()
	cfg := baseValidConfig(t)
	cfg.ServiceAccountJSONBytes = "{ this is not valid JSON"

	_, err := NewTokenSource(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse service account JSON")
}

func TestNewTokenSource_WIFNotImplemented(t *testing.T) {
	t.Parallel()
	cfg := baseValidConfig(t)
	cfg.ServiceAccountJSON = ""
	cfg.ServiceAccountJSONBytes = "" // clear the prime so IsSet relies on WIF fields
	cfg.WorkloadIdentityAudience = "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider"
	cfg.WorkloadIdentityServiceAccountEmail = "wif@test.iam.gserviceaccount.com"

	_, err := NewTokenSource(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workload identity federation not yet implemented")
}

// baseValidConfig returns a config block where IsSet() returns true but no
// auth method is wired. Caller adds SAJSON or WIF fields to drive the
// specific code path under test.
func baseValidConfig(t *testing.T) config.GoogleCloudIdentityConfig {
	t.Helper()
	return config.GoogleCloudIdentityConfig{
		ServiceAccountJSONBytes: " ", // non-empty so IsSet() returns true; callers can overwrite.
		ImpersonatedAdmin:       "admin@example.com",
		CustomerID:              "C0xxxxxxx",
		WorkspaceDomains:        "example.com",
	}
}
