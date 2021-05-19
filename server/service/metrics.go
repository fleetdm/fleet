package service

import (
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/metrics"
)

type metricsMiddleware struct {
	kolide.Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

// NewMetrics service takes an existing service and wraps it
// with instrumentation middleware.
func NewMetricsService(
	svc kolide.Service,
	requestCount metrics.Counter,
	requestLatency metrics.Histogram,
) kolide.Service {
	return metricsMiddleware{
		Service:        svc,
		requestCount:   requestCount,
		requestLatency: requestLatency,
	}
}
