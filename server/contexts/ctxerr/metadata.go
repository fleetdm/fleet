package ctxerr

import "context"

// ErrorAttributeProvider is an interface for types that can provide
// additional attributes to be attached to errors. This allows bounded
// contexts to enrich errors without ctxerr needing to know about
// their specific types.
type ErrorAttributeProvider interface {
	// GetErrorAttributes returns a map of attributes to be attached to errors.
	// The returned map will be merged into the error's data field.
	GetErrorAttributes() map[string]any
}

// TelemetryAttributeProvider is an interface for types that can provide
// attributes for telemetry (OpenTelemetry, Sentry, etc.). This is separate
// from ErrorAttributeProvider because telemetry attributes have different
// requirements (e.g., they may include PII that shouldn't be stored in errors).
type TelemetryAttributeProvider interface {
	// GetTelemetryAttributes returns key-value pairs for telemetry systems.
	// Keys should be strings, values can be strings, ints, or other primitive types.
	GetTelemetryAttributes() map[string]any
}

type errorAttributeProvidersKey struct{}

// WithErrorAttributeProviders returns a new context with the given error attribute providers.
// Multiple calls to this function will replace the previous providers.
func WithErrorAttributeProviders(ctx context.Context, providers ...ErrorAttributeProvider) context.Context {
	return context.WithValue(ctx, errorAttributeProvidersKey{}, providers)
}

// AddErrorAttributeProvider returns a new context with the given provider added to
// the existing providers. This is useful when you want to add a provider
// without replacing existing ones.
func AddErrorAttributeProvider(ctx context.Context, provider ErrorAttributeProvider) context.Context {
	existing := getErrorAttributeProviders(ctx)
	providers := make([]ErrorAttributeProvider, len(existing)+1)
	copy(providers, existing)
	providers[len(existing)] = provider
	return context.WithValue(ctx, errorAttributeProvidersKey{}, providers)
}

func getErrorAttributeProviders(ctx context.Context) []ErrorAttributeProvider {
	providers, _ := ctx.Value(errorAttributeProvidersKey{}).([]ErrorAttributeProvider)
	return providers
}

type telemetryProvidersKey struct{}

// WithTelemetryProviders returns a new context with the given telemetry providers.
func WithTelemetryProviders(ctx context.Context, providers ...TelemetryAttributeProvider) context.Context {
	return context.WithValue(ctx, telemetryProvidersKey{}, providers)
}

// AddTelemetryProvider returns a new context with the given provider added to
// the existing providers.
func AddTelemetryProvider(ctx context.Context, provider TelemetryAttributeProvider) context.Context {
	existing := getTelemetryProviders(ctx)
	providers := make([]TelemetryAttributeProvider, len(existing)+1)
	copy(providers, existing)
	providers[len(existing)] = provider
	return context.WithValue(ctx, telemetryProvidersKey{}, providers)
}

func getTelemetryProviders(ctx context.Context) []TelemetryAttributeProvider {
	providers, _ := ctx.Value(telemetryProvidersKey{}).([]TelemetryAttributeProvider)
	return providers
}
