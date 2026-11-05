package conditional_access_microsoft_proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
