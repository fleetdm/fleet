package fleethttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestWithOAuth2Client(t *testing.T) {
	ctx := WithOAuth2Client(t.Context())

	client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
	require.True(t, ok, "context must carry an *http.Client under oauth2.HTTPClient")
	require.NotNil(t, client.Transport)
	require.NotSame(t, http.DefaultTransport, client.Transport, "oauth2 must not fall back to the unguarded default transport")
}
