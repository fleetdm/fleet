// Package host enables setting and reading
// the current host from context
package host

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const hostKey key = 0

// NewContext returns a new context carrying the current osquery host.
func NewContext(ctx context.Context, host *fleet.Host) context.Context {
	return context.WithValue(ctx, hostKey, host)
}

// FromContext extracts the osquery host from context if present.
func FromContext(ctx context.Context) (*fleet.Host, bool) {
	host, ok := ctx.Value(hostKey).(*fleet.Host)
	return host, ok
}

// HostAttributeProvider wraps a fleet.Host to provide error context.
// It implements ctxerr.ErrorContextProvider.
type HostAttributeProvider struct {
	Host *fleet.Host
}

// GetDiagnosticContext implements ctxerr.ErrorContextProvider
func (p *HostAttributeProvider) GetDiagnosticContext() map[string]any {
	if p.Host == nil {
		return nil
	}
	return map[string]any{
		"host": map[string]any{
			"platform":        p.Host.Platform,
			"osquery_version": p.Host.OsqueryVersion,
		},
	}
}

// GetTelemetryContext implements ctxerr.ErrorContextProvider
func (p *HostAttributeProvider) GetTelemetryContext() map[string]any {
	if p.Host == nil {
		return nil
	}
	return map[string]any{
		"host.hostname": p.Host.Hostname,
		"host.id":       p.Host.ID,
	}
}
