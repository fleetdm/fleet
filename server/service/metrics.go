package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/metrics"
)

type metricsMiddleware struct {
	fleet.Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

// NewMetricsService service takes an existing service and wraps it
// with instrumentation middleware.
func NewMetricsService(
	svc fleet.Service,
	requestCount metrics.Counter,
	requestLatency metrics.Histogram,
) fleet.Service {
	return metricsMiddleware{
		Service:        svc,
		requestCount:   requestCount,
		requestLatency: requestLatency,
	}
}
