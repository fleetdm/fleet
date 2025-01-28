package health

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/log"
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
	failCheck := healthcheckFunc(func() error {
		return errors.New("health check failed")
	})
	passCheck := healthcheckFunc(func() error {
		return nil
	})

	fail := Handler(logger, map[string]Checker{
		"mock": failCheck,
	})
	pass := Handler(logger, map[string]Checker{
		"mock": passCheck,
	})
	both := Handler(logger, map[string]Checker{
		"pass": passCheck,
		"fail": failCheck,
	})

	httpTests := []struct {
		handler    http.Handler
		path       string
		wantHeader int
	}{
		{pass, "/healthz", http.StatusOK},
		{fail, "/healthz", http.StatusInternalServerError},

		// Empty check name
		{pass, "/healthz?check=mock&check=", http.StatusBadRequest},
		// Bad check name
		{pass, "/healthz?check=mock&check=bad", http.StatusBadRequest},
		// Passing and failing checks
		{both, "/healthz", http.StatusInternalServerError},
		// Passing and failing checks
		{both, "/healthz?check=pass&check=fail", http.StatusInternalServerError},
		// Only run passing
		{both, "/healthz?check=pass", http.StatusOK},
		// Only run failing
		{both, "/healthz?check=fail", http.StatusInternalServerError},
	}
	for _, tt := range httpTests {
		t.Run("", func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)
			tt.handler.ServeHTTP(rr, req)
			assert.Equal(t, rr.Code, tt.wantHeader)
		})
	}
}

type healthcheckFunc func() error

func (fn healthcheckFunc) HealthCheck() error {
	return fn()
}
