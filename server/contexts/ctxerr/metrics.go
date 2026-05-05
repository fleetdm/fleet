package ctxerr

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("fleet")

	// clientErrorsCounter counts client errors (4xx) by type.
	// These are errors caused by client issues (bad requests, auth failures, etc.)
	// and per OTEL semantic conventions should not be treated as server errors.
	clientErrorsCounter metric.Int64Counter

	// serverErrorsCounter counts server errors (5xx) by type.
	// These are errors caused by server issues and should be investigated.
	serverErrorsCounter metric.Int64Counter
)

func init() {
	var err error

	clientErrorsCounter, err = meter.Int64Counter(
		"fleet.http.client_errors",
		metric.WithDescription("Count of client errors (4xx) by error type"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		panic(err)
	}

	serverErrorsCounter, err = meter.Int64Counter(
		"fleet.http.server_errors",
		metric.WithDescription("Count of server errors (5xx) by error type"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		panic(err)
	}
}

// clientErrorCounterAttrs returns the metric attributes for client error counters.
func clientErrorCounterAttrs(errorType string) metric.AddOption {
	return metric.WithAttributes(
		attribute.String("error.type", errorType),
	)
}

// serverErrorCounterAttrs returns the metric attributes for server error counters.
func serverErrorCounterAttrs(errorType string) metric.AddOption {
	return metric.WithAttributes(
		attribute.String("error.type", errorType),
	)
}
