package cli

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthz(t *testing.T) {
	failing := healthz(map[string]interface{}{
		"mock": healthcheckFunc(func() error {
			return errors.New("health check failed")
		})})
	ok := healthz(map[string]interface{}{
		"mock": healthcheckFunc(func() error {
			return nil
		})})

	var httpTests = []struct {
		wantHeader int
		handler    http.Handler
	}{
		{200, ok},
		{500, failing},
	}
	for _, tt := range httpTests {
		t.Run("", func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/healthz", nil)
			tt.handler.ServeHTTP(rr, req)
			assert.Equal(t, rr.Code, tt.wantHeader)
		})
	}

}

type healthcheckFunc func() error

func (fn healthcheckFunc) HealthCheck() error {
	return fn()
}
