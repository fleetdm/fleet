package googleworkspace

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestDirectoryEndpointOverride exercises the QA/load-test seam end to end: with
// FLEET_TEST_GOOGLE_WORKSPACE_ENDPOINT set and a token_uri in the service-account
// JSON, the real Directory client performs its JWT token exchange and Directory
// API calls against a local fake server (over plain HTTP).
func TestDirectoryEndpointOverride(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "x", "token_type": "Bearer", "expires_in": 3600})
	})
	mux.HandleFunc("GET /admin/directory/v1/users", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"users": []map[string]any{{"id": "1", "primaryEmail": "a@b.com"}}})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"groups": []map[string]any{{"id": "g1", "name": "G"}}})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups/{k}/members", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"members": []map[string]any{{"id": "1", "type": "USER"}}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	t.Setenv(endpointOverrideEnv, srv.URL)
	intg := &fleet.GoogleWorkspaceIntegration{
		Domain:                "b.com",
		ImpersonatedUserEmail: "admin@b.com",
		ApiKey: fleet.GoogleCalendarApiKey{Values: map[string]string{
			fleet.GoogleCalendarEmail:      "sa@b.com",
			fleet.GoogleCalendarPrivateKey: string(pemKey),
			tokenURIKey:                    srv.URL + "/token",
		}},
	}

	dir, err := NewDirectory(t.Context(), intg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	users, err := dir.ListUsers(t.Context())
	require.NoError(t, err)
	require.Len(t, users, 1)

	groups, err := dir.ListGroups(t.Context())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].MemberExternalIDs, 1)
}
