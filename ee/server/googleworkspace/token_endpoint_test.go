package googleworkspace

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
)

func testIntegration(t *testing.T, tokenURI string) *fleet.GoogleWorkspaceIntegration {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	values := map[string]string{
		fleet.GoogleCalendarEmail:      "sa@b.com",
		fleet.GoogleCalendarPrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
	}
	if tokenURI != "" {
		values[tokenURIKey] = tokenURI
	}
	return &fleet.GoogleWorkspaceIntegration{
		Domain:                "b.com",
		ImpersonatedUserEmail: "admin@b.com",
		ApiKey:                fleet.GoogleCalendarApiKey{Values: values},
	}
}

// The service-account JSON is operator-supplied, so its token_uri only chooses
// the endpoint under the QA override. Everything else, including Google's own
// legacy endpoint found in older keys, resolves to the current Google endpoint.
func TestResolveTokenURL(t *testing.T) {
	cases := []struct {
		name     string
		tokenURI string
		override string
		want     string
	}{
		{"absent", "", "", google.JWTTokenURL},
		{"google current", google.JWTTokenURL, "", google.JWTTokenURL},
		{"google legacy", "https://accounts.google.com/o/oauth2/token", "", google.JWTTokenURL},
		{"private address ignored", "http://127.0.0.1:9931/internal/token", "", google.JWTTokenURL},
		{"honored under override", "http://127.0.0.1:8091/token", "http://127.0.0.1:8091", "http://127.0.0.1:8091/token"},
		{"absent under override", "", "http://127.0.0.1:8091", google.JWTTokenURL},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			values := map[string]string{}
			if c.tokenURI != "" {
				values[tokenURIKey] = c.tokenURI
			}
			require.Equal(t, c.want, resolveTokenURL(t.Context(), values, c.override, slog.New(slog.DiscardHandler)))
		})
	}
}

// Outside the QA seam a non-Google token_uri is ignored rather than contacted,
// and the client still builds so a key with a legacy endpoint keeps working.
func TestTokenURIIgnoredWithoutEndpointOverride(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv(endpointOverrideEnv, "")

	_, err := NewDirectory(t.Context(), testIntegration(t, srv.URL+"/internal/token"), slog.New(slog.DiscardHandler), Limits{})
	require.NoError(t, err)
	require.Zero(t, hits.Load(), "a non-Google token_uri must never be contacted")
}

// The token exchange must go through Fleet's outbound client so the private
// network guard applies to it like every other integration request.
func TestTokenExchangeUsesOutboundNetworkGuard(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// The override seam is the only way token_uri is honored, which is also the
	// path an operator would need for this to matter.
	t.Setenv(endpointOverrideEnv, srv.URL)
	fleethttp.SetNetworkBlockingMode(fleethttp.BlockingFull)
	t.Cleanup(func() { fleethttp.SetNetworkBlockingMode(fleethttp.BlockingDisabled) })

	dir, err := NewDirectory(t.Context(), testIntegration(t, srv.URL+"/token"), slog.New(slog.DiscardHandler), Limits{})
	require.NoError(t, err)

	_, err = dir.ListUsers(t.Context())
	// oauth2 flattens the transport error into its own message, so match on text.
	require.ErrorContains(t, err, fleethttp.ErrPrivateNetworkBlocked.Error())
	require.Zero(t, hits.Load(), "loopback token endpoint must be blocked before any request is sent")
}
