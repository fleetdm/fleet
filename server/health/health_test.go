package health

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckHealth(t *testing.T) {
	checkers := map[string]Checker{
		"fail": fail{},
		"pass": Nop(),
	}

	healthy := CheckHealth(log.NewNopLogger(), checkers)
	require.False(t, healthy)

	checkers = map[string]Checker{
		"pass": Nop(),
	}
	healthy = CheckHealth(log.NewNopLogger(), checkers)
	require.True(t, healthy)
}

type fail struct{}

func (c fail) HealthCheck() error {
	return errors.New("fail")
}

func TestHealthzHandler(t *testing.T) {
	logger := log.NewNopLogger()
	failing := Handler(logger, map[string]Checker{
		"mock": healthcheckFunc(func() error {
			return errors.New("health check failed")
		})})

	ok := Handler(logger, map[string]Checker{
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
