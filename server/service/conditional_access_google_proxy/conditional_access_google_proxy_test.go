package conditional_access_google_proxy

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

const origin = "https://fleet.example.com"

func newProxy(t *testing.T, handler http.HandlerFunc) *Proxy {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p, err := New(srv.URL, func() (string, error) { return origin, nil })
	require.NoError(t, err)
	return p
}

func TestCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var (
			gotReq  *http.Request
			gotBody map[string]string
		)
		p := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
			gotReq = r
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = w.Write([]byte(`{"fleet_server_secret":"s3cret","oauth_client_id":"client.apps.googleusercontent.com"}`))
		})
		resp, err := p.Create(t.Context(), "C01234abc")
		require.NoError(t, err)
		require.Equal(t, http.MethodPost, gotReq.Method)
		require.Equal(t, "/api/v1/google-conditional-access", gotReq.URL.Path)
		require.Equal(t, origin, gotReq.Header.Get("Origin"))
		require.Equal(t, map[string]string{"googleCustomerId": "C01234abc"}, gotBody)
		require.Equal(t, &CreateResponse{FleetServerSecret: "s3cret", OAuthClientID: "client.apps.googleusercontent.com"}, resp)
	})

	t.Run("incomplete response", func(t *testing.T) {
		p := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"fleet_server_secret":"s3cret"}`))
		})
		_, err := p.Create(t.Context(), "C01234abc")
		require.ErrorContains(t, err, "missing")
	})

	t.Run("conflict carries the body", func(t *testing.T) {
		p := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"already registered"}`))
		})
		_, err := p.Create(t.Context(), "C01234abc")
		se, ok := errors.AsType[*StatusError](err)
		require.True(t, ok)
		require.Equal(t, http.StatusConflict, se.StatusCode())
		require.Contains(t, se.Body(), "already registered")
	})

	t.Run("missing origin", func(t *testing.T) {
		var called bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
		t.Cleanup(srv.Close)
		p, err := New(srv.URL, func() (string, error) { return "", nil })
		require.NoError(t, err)
		_, err = p.Create(t.Context(), "C01234abc")
		require.ErrorContains(t, err, "missing origin")
		require.False(t, called)
	})
}

func TestDelete(t *testing.T) {
	const (
		customerID = "C01234abc&fleetServerSecret=smuggled"
		secret     = "secret=a&b"
	)

	t.Run("success with encoded query", func(t *testing.T) {
		var gotReq *http.Request
		p := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
			gotReq = r
		})
		require.NoError(t, p.Delete(t.Context(), customerID, secret))
		require.Equal(t, http.MethodDelete, gotReq.Method)
		require.Equal(t, origin, gotReq.Header.Get("Origin"))
		require.Equal(t, url.Values{"googleCustomerId": {customerID}, "fleetServerSecret": {secret}}, gotReq.URL.Query())
	})

	t.Run("not found", func(t *testing.T) {
		p := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		err := p.Delete(t.Context(), customerID, secret)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("server error", func(t *testing.T) {
		p := newProxy(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		err := p.Delete(t.Context(), customerID, secret)
		require.Error(t, err)
		require.False(t, fleet.IsNotFound(err))
	})
}
