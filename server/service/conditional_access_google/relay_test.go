package conditional_access_google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

const origin = "https://fleet.example.com"

type recordedRequest struct {
	method, path, origin string
	body                 map[string]string
}

func newTestRelay(t *testing.T, status int, response string) (Relay, *[]recordedRequest) {
	uri, requests := newRecordingServer(t, status, response)
	p, err := NewRelay(uri, func() (string, error) { return origin, nil }, AllowHTTP())
	require.NoError(t, err)
	return p, requests
}

func newRecordingServer(t *testing.T, status int, response string) (string, *[]recordedRequest) {
	var requests []recordedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests = append(requests, recordedRequest{method: r.Method, path: r.URL.Path, origin: r.Header.Get("Origin"), body: body})
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &requests
}

func TestRegister(t *testing.T) {
	p, requests := newTestRelay(t, http.StatusOK, `{"fleetServerSecret":"s3cret","oauthClientId":"client.apps.googleusercontent.com"}`)

	resp, err := p.Register(t.Context(), "C02cajzvg")
	require.NoError(t, err)
	require.Equal(t, &Registration{FleetServerSecret: "s3cret", OAuthClientID: "client.apps.googleusercontent.com"}, resp)
	require.Equal(t, []recordedRequest{{
		method: http.MethodPost,
		path:   "/api/v1/google-conditional-access",
		origin: origin,
		body:   map[string]string{"googleCustomerId": "C02cajzvg"},
	}}, *requests)
}

func TestRegisterWithoutSecretFails(t *testing.T) {
	p, _ := newTestRelay(t, http.StatusOK, `{}`)
	_, err := p.Register(t.Context(), "C02cajzvg")
	require.ErrorContains(t, err, "fleetServerSecret")
}

func TestDeregister(t *testing.T) {
	p, requests := newTestRelay(t, http.StatusOK, ``)

	require.NoError(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"))
	require.Equal(t, []recordedRequest{{
		method: http.MethodDelete,
		path:   "/api/v1/google-conditional-access",
		origin: origin,
		body:   map[string]string{"googleCustomerId": "C02cajzvg", "fleetServerSecret": "s3cret"},
	}}, *requests)
}

func TestDeregisterNotFound(t *testing.T) {
	p, _ := newTestRelay(t, http.StatusNotFound, `{"error":"not found"}`)
	err := p.Deregister(t.Context(), "C02cajzvg", "wrong")
	require.True(t, fleet.IsNotFound(err))
}

func TestRelayStatusError(t *testing.T) {
	p, _ := newTestRelay(t, http.StatusConflict, `{"error":"registered from another origin"}`)
	_, err := p.Register(t.Context(), "C02cajzvg")
	se, ok := errors.AsType[*RelayStatusError](err)
	require.True(t, ok)
	require.Equal(t, http.StatusConflict, se.StatusCode())
	require.Contains(t, se.Body(), "registered from another origin")
	require.False(t, fleet.IsNotFound(err))

	// Only Deregister treats a 404 as "no such registration".
	p, _ = newTestRelay(t, http.StatusNotFound, `{"error":"not found"}`)
	_, err = p.Register(t.Context(), "C02cajzvg")
	require.False(t, fleet.IsNotFound(err))
}

func TestRelayResponseBodies(t *testing.T) {
	// An empty 2xx body is fine when no response is expected.
	p, _ := newTestRelay(t, http.StatusNoContent, ``)
	require.NoError(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"))

	p, _ = newTestRelay(t, http.StatusOK, ``)
	_, err := p.Register(t.Context(), "C02cajzvg")
	require.ErrorContains(t, err, "empty response body")

	p, _ = newTestRelay(t, http.StatusOK, `{"fleetServerSecret":"`+strings.Repeat("x", maxRelayResponseBytes)+`"}`)
	_, err = p.Register(t.Context(), "C02cajzvg")
	require.ErrorContains(t, err, "exceeds")
}

func TestRelayRequiresOriginAndValidURI(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	t.Cleanup(srv.Close)

	p, err := NewRelay(srv.URL, func() (string, error) { return "", nil }, AllowHTTP())
	require.NoError(t, err)
	_, err = p.Register(t.Context(), "C02cajzvg")
	require.ErrorContains(t, err, "missing origin")

	// An unusable URI fails the request, not NewRelay.
	for _, uri := range []string{"fleetdm.com", srv.URL + "?x=1", srv.URL + "#frag"} {
		p, err := NewRelay(uri, func() (string, error) { return origin, nil }, AllowHTTP())
		require.NoError(t, err, uri)
		require.ErrorContains(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"), "invalid relay uri", uri)
	}
	require.False(t, called)
}

func TestRelayTrimsTrailingSlashes(t *testing.T) {
	uri, requests := newRecordingServer(t, http.StatusOK, ``)
	p, err := NewRelay(uri+"/", func() (string, error) { return origin + "/", nil }, AllowHTTP())
	require.NoError(t, err)

	require.NoError(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"))
	require.Len(t, *requests, 1)
	require.Equal(t, "/api/v1/google-conditional-access", (*requests)[0].path)
	require.Equal(t, origin, (*requests)[0].origin)
}

func TestRelayDoesNotFollowRedirects(t *testing.T) {
	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalled = true }))
	t.Cleanup(target.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+registrationPath, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(srv.Close)

	p, err := NewRelay(srv.URL, func() (string, error) { return origin, nil }, AllowHTTP())
	require.NoError(t, err)
	err = p.Deregister(t.Context(), "C02cajzvg", "s3cret")
	se, ok := errors.AsType[*RelayStatusError](err)
	require.True(t, ok, err)
	require.Equal(t, http.StatusTemporaryRedirect, se.StatusCode())
	require.False(t, targetCalled)
}

func TestDeregisterIgnoresLargeBody(t *testing.T) {
	p, _ := newTestRelay(t, http.StatusOK, strings.Repeat("x", maxRelayResponseBytes+1))
	require.NoError(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"))
}

func TestRelayStopsOnCancel(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	p, err := NewRelay(srv.URL, func() (string, error) { return origin, nil }, AllowHTTP())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = p.Register(ctx, "C02cajzvg")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestRelayRequiresHTTPS(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	t.Cleanup(srv.Close)

	// Plain HTTP needs AllowHTTP.
	for _, uri := range []string{srv.URL, "ftp://fleetdm.example.com"} {
		p, err := NewRelay(uri, func() (string, error) { return origin, nil })
		require.NoError(t, err, uri)
		require.ErrorContains(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"), "must use https", uri)
	}
	require.False(t, called)

	// The allowance is for HTTP only.
	p, err := NewRelay("ftp://fleetdm.example.com", func() (string, error) { return origin, nil }, AllowHTTP())
	require.NoError(t, err)
	require.ErrorContains(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"), "must use https")

	// HTTPS needs no allowance.
	p, err = NewRelay("https://127.0.0.1:1", func() (string, error) { return origin, nil })
	require.NoError(t, err)
	// Nothing listens there, so the send fails.
	require.ErrorContains(t, p.Deregister(t.Context(), "C02cajzvg", "s3cret"), "send request")
}
