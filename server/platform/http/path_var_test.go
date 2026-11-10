package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestPathVar(t *testing.T) {
	cases := []struct {
		name      string
		request   func() *http.Request
		wantValue string
		wantOK    bool
	}{
		{"path value", func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, "/hosts/1", nil)
			r.SetPathValue("id", "1")
			return r
		}, "1", true},
		{"gorilla var", func() *http.Request {
			return mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/hosts/2", nil), map[string]string{"id": "2"})
		}, "2", true},
		{"missing", func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/hosts", nil)
		}, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			value, ok := PathVar(c.request(), "id")
			require.Equal(t, c.wantOK, ok)
			require.Equal(t, c.wantValue, value)
		})
	}
}
