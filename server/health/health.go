// Package health adds methods for checking the health of service dependencies.
package health

import (
	"net/http"

	"github.com/go-kit/kit/log"
)

// Checker returns an error indicating if a service is in an unhealthy state.
// Checkers should be implemented by dependencies which can fail, like a DB or mail service.
type Checker interface {
	HealthCheck() error
}

// Handler returns an http.Handler that checks the status of all the dependencies.
// Handler responds with either:
// 200 OK if the server can successfully communicate with it's backends or
// 500 if any of the backends are reporting an issue.
func Handler(logger log.Logger, checkers map[string]Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthy := CheckHealth(logger, checkers)
		if !healthy {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// CheckHealth checks multiple checkers returning false if any of them fail.
// CheckHealth logs the reason a checker fails.
func CheckHealth(logger log.Logger, checkers map[string]Checker) bool {
	healthy := true
	for name, hc := range checkers {
		if err := hc.HealthCheck(); err != nil {
			log.With(logger, "component", "healthz").Log("err", err, "health-checker", name)
			healthy = false
			continue
		}
	}
	return healthy
}

// Nop creates a noop checker. Useful in tests.
func Nop() Checker {
	return nop{}
}

type nop struct{}

func (c nop) HealthCheck() error {
	return nil
}
