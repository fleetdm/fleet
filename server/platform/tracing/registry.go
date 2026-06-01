package tracing

import (
	"regexp"
	"sync"
)

// Tier classifies a span for sampling. Higher-value spans (TierAlways)
// always get sampled; high-volume noise (TierHighVolume) is downsampled
// aggressively. Liveness probes (TierNever) are dropped unconditionally.
type Tier uint8

const (
	// TierAlways is the catch-all. Spans not registered with the Registry
	// fall here, and are sampled at 100%. Cron jobs, enroll, MDM checkin,
	// SCEP, GitOps batch — all the load-bearing flows — should never need
	// an explicit registration, since their absence from the noisy lists
	// is what keeps them safe by default.
	TierAlways Tier = iota

	// TierHighVolume is for routes that dominate request volume without
	// being individually interesting (osquery distributed read/write,
	// orbit ping/config, device desktop/ping). Sampled at the configured
	// high-volume ratio (default 0.1%).
	TierHighVolume

	// TierStandard is for routes with moderate volume and moderate
	// diagnostic value (admin reads, dashboard endpoints, asset paths).
	// Sampled at the configured standard ratio (default 2%).
	TierStandard

	// TierNever drops the span unconditionally, even under ForceFull.
	// Reserved for high-volume zero-diagnostic-value paths like liveness
	// probes, the version endpoint, and the Prometheus scrape path.
	TierNever
)

// Registry maps normalized route names to tiers. Bounded contexts register
// their own routes at startup via Register; the sampler reads via Lookup
// on every span. Routes not present in the registry fall to TierAlways.
//
// The Registry is the policy seam: platform/tracing provides the mechanism
// (sampler + tier enum), each bounded context provides the policy (which
// of its routes belong in which tier).
type Registry struct {
	mu     sync.RWMutex
	routes map[string]Tier
}

// NewRegistry returns an empty Registry. Routes are added via Register.
func NewRegistry() *Registry {
	return &Registry{routes: make(map[string]Tier)}
}

// Register classifies a method+path. The path should use "_version_" as
// the placeholder for the gorilla/mux fleetversion segment (the form that
// span names take after normalization). Both alternate-path forms (e.g.
// "/api/v1/osquery/..." and "/api/osquery/...") must be registered
// separately if both exist.
//
// Re-registering the same method+path overwrites the prior tier.
func (r *Registry) Register(method, path string, tier Tier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[spanKey(method, path)] = tier
}

// Lookup returns the tier for a span name and a bool indicating whether
// the route was found in the registry. Span names are normalized before
// lookup so the gorilla version regex is stripped to "_version_". Cron
// and other non-HTTP span names (no leading "METHOD ") simply won't be
// in the registry and return (TierAlways, false).
func (r *Registry) Lookup(spanName string) (Tier, bool) {
	normalized := normalizeSpanName(spanName)
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.routes[normalized]
	return t, ok
}

// versionTemplatePattern matches the gorilla/mux version segment Fleet
// inserts via /_version_/ → /{fleetversion:(?:v1|2022-04|latest)}/ at
// registration time. The sampler normalizes spans back to /_version_/
// before lookup so the registry stays decoupled from the configured
// version list.
var versionTemplatePattern = regexp.MustCompile(`\{fleetversion:[^}]*\}`)

func normalizeSpanName(name string) string {
	return versionTemplatePattern.ReplaceAllString(name, "_version_")
}

func spanKey(method, path string) string {
	return method + " " + path
}
