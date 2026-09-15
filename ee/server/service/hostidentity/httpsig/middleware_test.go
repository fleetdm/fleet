package httpsig

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

type notFoundErr struct{}

func (notFoundErr) Error() string    { return "not found" }
func (notFoundErr) IsNotFound() bool { return true }

// The middleware must not let an unauthenticated caller distinguish failure
// modes (missing signature, unknown key ID, malformed signature): every
// rejection is the same status and body, with no internal detail.
func TestMiddlewareUniformErrorResponses(t *testing.T) {
	ds := new(mock.Store)
	ds.GetHostIdentityCertBySerialNumberFunc = func(ctx context.Context, serialNumber uint64) (*types.HostIdentityCertificate, error) {
		return nil, notFoundErr{}
	}

	mw, err := Middleware(ds, true, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	do := func(setup func(*http.Request)) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "https://fleet.example.com/api/fleet/orbit/config", strings.NewReader("{}"))
		setup(req)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr
	}

	cases := map[string]*httptest.ResponseRecorder{
		"missing signature": do(func(_ *http.Request) {}),
		"unknown key id": do(func(r *http.Request) {
			r.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1700000000;keyid="ffff";alg="ecdsa-p256-sha256"`)
			r.Header.Set("Signature", "sig1=:MEQCIAa1sm9sBBVh/nBXPn3z2011K11GkTPB2gO9BgHfyLdSAiAM9uYJ+t/1n0acOZBW1AF9PdTFTva7n2323w4gVUXKrQ==:")
		}),
		"malformed signature headers": do(func(r *http.Request) {
			r.Header.Set("Signature-Input", "not a valid structured field")
			r.Header.Set("Signature", "garbage")
		}),
	}

	bodies := make(map[string]struct{})
	for name, rr := range cases {
		require.Equal(t, http.StatusUnauthorized, rr.Code, name)
		body := rr.Body.String()
		for _, leak := range []string{"keyID", "keyid", "path=", "certificate", "not found", "host_uuid", "orbit/config"} {
			require.NotContains(t, body, leak, name)
		}
		bodies[body] = struct{}{}
	}
	require.Len(t, bodies, 1, "all rejection bodies must be identical")
}

// Internal faults (e.g. the KeySpecer type assertion failing) must not be
// reported as 401: agents interpret persistent 401s as a revoked identity and
// discard their certificate. The body must still carry no internal detail.
func TestHandleInternalErrorStatusAndGenericBody(t *testing.T) {
	ctx := context.Background()

	rr := httptest.NewRecorder()
	handleInternalError(ctx, rr, errors.New("could not extract host identity certificate key: path=/api/fleet/orbit/config"))
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.NotContains(t, rr.Body.String(), "path=")
	require.NotContains(t, rr.Body.String(), "certificate")

	rr = httptest.NewRecorder()
	handleError(ctx, rr, errors.New("request not verified: path=/api/fleet/orbit/config"))
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.NotContains(t, rr.Body.String(), "path=")
}
