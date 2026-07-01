package conditional_access_microsoft_proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

		p, err := New(srv.URL, "key", func() (string, error) { return "https://fleet.example.com", nil })
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

		p, err := New(srv.URL, "key", func() (string, error) { return "https://fleet.example.com", nil })
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
