package conditional_access_microsoft_proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProxyStatusErrorCapturesBody(t *testing.T) {
	t.Run("captures status code and body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"upstream boom"}`))
		}))
		defer srv.Close()

		p, err := New(srv.URL, func() (string, error) { return "https://fleet.example.com", nil })
		require.NoError(t, err)

		_, err = p.SetComplianceStatus(t.Context(), "tenant", "secret", "device", "upn", true, "name", "macOS", "14.0", false, time.Now())
		require.Error(t, err)

		se, ok := errors.AsType[interface {
			error
			StatusCode() int
		}](err)
		require.True(t, ok)
		require.Equal(t, http.StatusInternalServerError, se.StatusCode())

		be, ok := errors.AsType[interface {
			error
			Body() string
		}](err)
		require.True(t, ok)
		require.Contains(t, be.Body(), "upstream boom")
	})

	t.Run("captures full body", func(t *testing.T) {
		const bodyLen = 1000
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(strings.Repeat("x", bodyLen)))
		}))
		defer srv.Close()

		p, err := New(srv.URL, func() (string, error) { return "https://fleet.example.com", nil })
		require.NoError(t, err)

		_, err = p.SetComplianceStatus(t.Context(), "tenant", "secret", "device", "upn", true, "name", "macOS", "14.0", false, time.Now())
		require.Error(t, err)

		be, ok := errors.AsType[interface {
			error
			Body() string
		}](err)
		require.True(t, ok)
		require.Len(t, be.Body(), bodyLen)
	})
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
