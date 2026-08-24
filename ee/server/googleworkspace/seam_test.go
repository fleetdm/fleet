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
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestDirectoryEndpointOverride exercises the QA/load-test seam end to end: with
// FLEET_TEST_GOOGLE_WORKSPACE_ENDPOINT set and a token_uri in the service-account
// JSON, the real Directory client performs its JWT token exchange and Directory
// API calls against a local fake server (over plain HTTP).
func TestDirectoryEndpointOverride(t *testing.T) {
	// Field projections as each endpoint received them, so the test can show the
	// listings ask only for what Fleet maps. Written from the server's handler
	// goroutines, so guard it rather than relying on the request/response round trip
	// to order the writes before the assertions below.
	var fieldsMu sync.Mutex
	requestedFields := map[string]string{}
	recordFields := func(endpoint string, r *http.Request) {
		fieldsMu.Lock()
		defer fieldsMu.Unlock()
		requestedFields[endpoint] = r.URL.Query().Get("fields")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "x", "token_type": "Bearer", "expires_in": 3600})
	})
	mux.HandleFunc("GET /admin/directory/v1/users", func(w http.ResponseWriter, r *http.Request) {
		recordFields("users", r)
		_ = json.NewEncoder(w).Encode(map[string]any{"users": []map[string]any{{"id": "1", "primaryEmail": "a@b.com"}}})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		recordFields("groups", r)
		_ = json.NewEncoder(w).Encode(map[string]any{"groups": []map[string]any{{"id": "g1", "name": "G"}}})
	})
	mux.HandleFunc("GET /admin/directory/v1/groups/{k}/members", func(w http.ResponseWriter, r *http.Request) {
		recordFields("members", r)
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

	dir, err := NewDirectory(t.Context(), intg, slog.New(slog.DiscardHandler), Limits{})
	require.NoError(t, err)

	users, err := dir.ListUsers(t.Context())
	require.NoError(t, err)
	require.Len(t, users, 1)

	groups, err := dir.ListGroups(t.Context())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].MemberExternalIDs, 1)

	// Every listing must project its fields, and every projection must carry
	// nextPageToken or pagination cannot advance past the first page.
	fieldsMu.Lock()
	defer fieldsMu.Unlock()
	require.Equal(t, map[string]string{
		"users":   usersFields,
		"groups":  groupsFields,
		"members": membersFields,
	}, requestedFields)
	for name, fields := range requestedFields {
		require.Contains(t, fields, "nextPageToken", "%s projection must request nextPageToken", name)
	}
}

// TestFieldProjectionsCoverMappedFields pins each projection against the fields the
// mapping code reads. A projection that drops one of these does not fail — the API
// simply omits it — so the directory would silently sync with empty values.
func TestFieldProjectionsCoverMappedFields(t *testing.T) {
	for _, tc := range []struct {
		fields string
		// needed are the response fields the mapping code depends on.
		needed []string
	}{
		{
			// mapUser reads id, primaryEmail, suspended, archived, name, organizations
			// (department, primary) and emails (address, type, primary).
			fields: usersFields,
			needed: []string{"id", "primaryEmail", "suspended", "archived", "name", "organizations", "emails"},
		},
		{
			// groupDisplayName falls back from name to email.
			fields: groupsFields,
			needed: []string{"id", "name", "email"},
		},
		{
			// Directory.ListGroups keeps members by id and filters on type.
			fields: membersFields,
			needed: []string{"id", "type"},
		},
	} {
		t.Run(tc.fields, func(t *testing.T) {
			inner := tc.fields[strings.Index(tc.fields, "(")+1 : len(tc.fields)-1]
			require.ElementsMatch(t, tc.needed, strings.Split(inner, ","))
		})
	}
}
