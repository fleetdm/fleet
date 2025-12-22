package ctxerr

import "context"

// ErrorContextProvider provides contextual information for error handling.
// Implementations can provide data for both error storage and telemetry systems.
type ErrorContextProvider interface {
	// GetDiagnosticContext returns attributes stored with errors for troubleshooting.
	// Data is persisted to Redis and included in logs. Should contain diagnostic
	// information like platform, versions, and status flags. Avoid including PII.
	GetDiagnosticContext() map[string]any

	// GetTelemetryContext returns attributes sent to observability systems
	// (OpenTelemetry, Sentry). May include identifiers not stored with errors.
	// Return nil if no telemetry context is available.
	GetTelemetryContext() map[string]any
}

type errorContextProvidersKey struct{}

// WithErrorContextProviders returns a new context with the given providers.
// Multiple calls to this function will replace the previous providers.
func WithErrorContextProviders(ctx context.Context, providers ...ErrorContextProvider) context.Context {
	return context.WithValue(ctx, errorContextProvidersKey{}, providers)
}

// AddErrorContextProvider returns a new context with the given provider added to
// the existing providers. This is useful when you want to add a provider
// without replacing existing ones.
func AddErrorContextProvider(ctx context.Context, provider ErrorContextProvider) context.Context {
	existing := getErrorContextProviders(ctx)
	providers := make([]ErrorContextProvider, len(existing)+1)
	copy(providers, existing)
	providers[len(existing)] = provider
	return context.WithValue(ctx, errorContextProvidersKey{}, providers)
}

func getErrorContextProviders(ctx context.Context) []ErrorContextProvider {
	providers, _ := ctx.Value(errorContextProvidersKey{}).([]ErrorContextProvider)
	return providers
}
