package conditional_access_microsoft_proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestProxyDeleteNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	p, err := New(srv.URL, func() (string, error) { return "https://fleet.example.com", nil })
	require.NoError(t, err)

	_, err = p.Delete(t.Context(), "tenant", "secret")
	require.True(t, fleet.IsNotFound(err))
	// Elsewhere a 404 stays a status error carrying the code.
	_, err = p.Get(t.Context(), "tenant", "secret")
	require.False(t, fleet.IsNotFound(err))
	se, ok := errors.AsType[interface {
		error
		StatusCode() int
	}](err)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, se.StatusCode())
}

func TestProxySetComplianceStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream boom"}`))
	}))
	defer srv.Close()
	p, err := New(srv.URL, func() (string, error) { return "https://fleet.example.com", nil })
	require.NoError(t, err)

	_, err = p.SetComplianceStatus(t.Context(), "tenant", "secret", "device", "upn", true, "name", "macOS", "14.0", false, time.Now())
	// The failure activity reads these through the wrapped error.
	se, ok := errors.AsType[interface {
		error
		StatusCode() int
		Body() string
	}](err)
	require.True(t, ok)
	require.Equal(t, http.StatusBadGateway, se.StatusCode())
	require.JSONEq(t, `{"error":"upstream boom"}`, se.Body())
}

func TestProxyQueryParametersAreEncoded(t *testing.T) {
	const (
		tenantID  = "tenant&fleetServerSecret=smuggled+x"
		secret    = "secret=a&b"
		messageID = "msg&entraTenantId=other"
	)

	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p, err := New(srv.URL, func() (string, error) { return "https://fleet.example.com", nil })
	require.NoError(t, err)

	for _, tc := range []struct {
		name     string
		call     func() error
		expected url.Values
	}{
		{
			name: "get",
			call: func() error {
				_, err := p.Get(t.Context(), tenantID, secret)
				return err
			},
			expected: url.Values{"entraTenantId": {tenantID}, "fleetServerSecret": {secret}},
		},
		{
			name: "delete",
			call: func() error {
				_, err := p.Delete(t.Context(), tenantID, secret)
				return err
			},
			expected: url.Values{"entraTenantId": {tenantID}, "fleetServerSecret": {secret}},
		},
		{
			name: "get message status",
			call: func() error {
				_, err := p.GetMessageStatus(t.Context(), tenantID, secret, messageID)
				return err
			},
			expected: url.Values{"entraTenantId": {tenantID}, "fleetServerSecret": {secret}, "messageId": {messageID}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotQuery = nil
			require.NoError(t, tc.call())
			require.Equal(t, tc.expected, gotQuery)
		})
	}
}

func TestProxyCreateRequiresCredentials(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "empty object", body: `{}`},
		{name: "no server key", body: `{"entra_tenant_id":"tid"}`},
		{name: "no tenant ID", body: `{"fleet_server_secret":"s"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			p, err := New(srv.URL, func() (string, error) { return "https://fleet.example.com", nil })
			require.NoError(t, err)

			resp, err := p.Create(t.Context(), "tid")
			require.ErrorContains(t, err, "missing tenant ID or secret")
			require.Nil(t, resp)
		})
	}
}
