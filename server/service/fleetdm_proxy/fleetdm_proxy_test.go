package fleetdm_proxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

const origin = "https://fleet.example.com"

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, func() (string, error) { return origin, nil })
	require.NoError(t, err)
	return c
}

func TestRequests(t *testing.T) {
	var got struct {
		method, path, origin, contentType, body string
		query                                   url.Values
	}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got.method, got.path, got.origin, got.contentType, got.body = r.Method, r.URL.Path, r.Header.Get("Origin"), r.Header.Get("Content-Type"), string(b)
		got.query = r.URL.Query()
		_, _ = w.Write([]byte(`{"answer":"42"}`))
	})
	// Values that would smuggle extra parameters if not encoded.
	query := url.Values{"id": {"a&secret=smuggled"}, "secret": {"s=1&b"}}
	var resp struct {
		Answer string `json:"answer"`
	}

	t.Run("post", func(t *testing.T) {
		resp.Answer = ""
		require.NoError(t, c.Post(t.Context(), "/api/v1/thing", map[string]string{"k": "v"}, &resp))
		require.Equal(t, "42", resp.Answer)
		require.Equal(t, http.MethodPost, got.method)
		require.Equal(t, "/api/v1/thing", got.path)
		require.Equal(t, origin, got.origin)
		require.Equal(t, "application/json", got.contentType)
		require.JSONEq(t, `{"k":"v"}`, got.body)
		require.Empty(t, got.query)
	})

	t.Run("get", func(t *testing.T) {
		resp.Answer = ""
		require.NoError(t, c.Get(t.Context(), "/api/v1/thing", query, &resp))
		require.Equal(t, "42", resp.Answer)
		require.Equal(t, http.MethodGet, got.method)
		require.Equal(t, origin, got.origin)
		require.Empty(t, got.contentType)
		require.Empty(t, got.body)
		require.Equal(t, query, got.query)
	})

	t.Run("delete ignoring the response", func(t *testing.T) {
		require.NoError(t, c.Delete(t.Context(), "/api/v1/thing", query, nil))
		require.Equal(t, http.MethodDelete, got.method)
		require.Empty(t, got.contentType)
		require.Empty(t, got.body)
		require.Equal(t, query, got.query)
	})
}

func TestStatusError(t *testing.T) {
	t.Run("captures status code and body", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"upstream boom"}`))
		})
		err := c.Get(t.Context(), "/x", nil, nil)

		se, ok := errors.AsType[interface {
			error
			StatusCode() int
			Body() string
		}](err)
		require.True(t, ok)
		require.Equal(t, http.StatusInternalServerError, se.StatusCode())
		require.Contains(t, se.Body(), "upstream boom")
		require.False(t, fleet.IsNotFound(err))
	})

	t.Run("captures full body", func(t *testing.T) {
		const bodyLen = 1000
		c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(strings.Repeat("x", bodyLen)))
		})
		err := c.Get(t.Context(), "/x", nil, nil)
		se, ok := errors.AsType[*StatusError](err)
		require.True(t, ok)
		require.Len(t, se.Body(), bodyLen)
	})
}

func TestNotFoundOn404(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no such record"}`))
	})
	err := c.Delete(t.Context(), "/x", nil, nil)

	// A 404 is a plain status error unless the caller says it means "no such record".
	require.False(t, fleet.IsNotFound(err))
	notFound := NotFoundOn404(err)
	require.True(t, fleet.IsNotFound(notFound))
	se, ok := errors.AsType[*StatusError](notFound)
	require.True(t, ok)
	require.Contains(t, se.Body(), "no such record")

	other := &StatusError{Code: http.StatusConflict}
	require.Equal(t, error(other), NotFoundOn404(other))
}

func TestMissingOrigin(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, func() (string, error) { return "", nil })
	require.NoError(t, err)

	err = c.Get(t.Context(), "/x", nil, nil)
	require.ErrorContains(t, err, "missing origin")
	require.False(t, called)
}

func TestTransportErrorMasksSecretQueryParams(t *testing.T) {
	// Close the server so the request fails at the transport layer, where Go's
	// url.Error would otherwise print the full URL, query string included.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()
	c, err := New(srv.URL, func() (string, error) { return origin, nil })
	require.NoError(t, err)

	const secret = "s3cr3t-value"
	err = c.Get(t.Context(), "/x", url.Values{"entraTenantId": {"tid"}, "fleetServerSecret": {secret}}, nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
	require.Contains(t, err.Error(), "fleetServerSecret=MASKED")
	require.Contains(t, err.Error(), "entraTenantId=tid")
}

func TestEmptySuccessBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, c.Delete(t.Context(), "/x", nil, nil))

	// A caller that expects a response must not get a zero value it can't tell
	// apart from a real one.
	var resp struct {
		Answer string `json:"answer"`
	}
	require.ErrorContains(t, c.Delete(t.Context(), "/x", nil, &resp), "empty response body")
}

func TestResponseBodyTooLarge(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"answer":"` + strings.Repeat("x", maxResponseBytes) + `"}`))
	})
	var resp struct {
		Answer string `json:"answer"`
	}
	err := c.Get(t.Context(), "/x", nil, &resp)
	require.ErrorContains(t, err, "response body exceeds")
}

func TestNewRequiresSchemeAndHost(t *testing.T) {
	getter := func() (string, error) { return origin, nil }
	for _, uri := range []string{"", "fleetdm.com", "/api", "https://"} {
		_, err := New(uri, getter)
		require.Error(t, err, uri)
	}
	_, err := New("https://fleetdm.com", getter)
	require.NoError(t, err)
}
