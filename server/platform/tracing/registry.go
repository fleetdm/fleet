package tracing

import (
	"regexp"
	"sync"
)

// Tier classifies a span for sampling. Higher value spans (TierAlways) always get sampled. High volume noise (TierHighVolume)
// is downsampled aggressively. Liveness probes (TierNever) are dropped unconditionally.
type Tier int

const (
	// TierAlways is the catch all. Spans not registered with the Registry fall here, and are sampled at 100%. Cron jobs, enroll,
	// MDM checkin, SCEP, GitOps batch (all the load bearing flows) should never need an explicit registration. Their absence from
	// the noisy lists is what keeps them safe by default.
	TierAlways Tier = iota

	// TierHighVolume is for routes that dominate request volume without being individually interesting (osquery distributed
	// read/write, orbit ping/config, device desktop/ping). Sampled at the configured high volume ratio (default 0.1%).
	TierHighVolume

	// TierStandard is for routes with moderate volume and moderate diagnostic value (admin reads, dashboard endpoints, asset
	// paths). Sampled at the configured standard ratio (default 2%).
	TierStandard

	// TierNever drops the span unconditionally, even under ForceFull. Reserved for high volume zero diagnostic value paths like
	// liveness probes, the version endpoint, and the Prometheus scrape path.
	TierNever
)

// Registry maps normalized route names to tiers. Bounded contexts register their own routes at startup via Register. The
// sampler reads via Lookup on every span. Routes not present in the registry fall to TierAlways.
//
// The Registry is the policy seam. platform/tracing provides the mechanism (sampler and tier enum). Each bounded context
// provides the policy: which of its routes belong in which tier.
type Registry struct {
	mu     sync.RWMutex
	routes map[string]Tier
}

// NewRegistry returns an empty Registry. Routes are added via Register.
func NewRegistry() *Registry {
	return &Registry{routes: make(map[string]Tier)}
}

// Register classifies a method and path. Paths are normalized on write the same way they are on Lookup. The gorilla/mux
// version segment is collapsed to "_version_". Regex constrained params (e.g. "{id:[0-9]+}") are stripped to "{id}". Callers
// can therefore register in either the readable form ("/api/_version_/fleet/hosts/{id}") or the raw template form, and lookups
// will still match. Both alternate path forms (e.g. "/api/v1/osquery/..." and "/api/osquery/...") still need separate
// registrations.
//
// Re-registering the same normalized method+path overwrites the prior tier.
func (r *Registry) Register(method, path string, tier Tier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[normalizeSpanName(spanKey(method, path))] = tier
}

// Lookup returns the tier for a span name and a bool indicating whether the route was found in the registry. Span names are
// normalized before lookup so the gorilla version regex is stripped to "_version_". Cron and other non HTTP span names (no
// leading "METHOD ") simply won't be in the registry and return (TierAlways, false).
func (r *Registry) Lookup(spanName string) (Tier, bool) {
	normalized := normalizeSpanName(spanName)
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.routes[normalized]
	return t, ok
}

// versionTemplatePattern matches the gorilla/mux version segment Fleet inserts via /_version_/, replacing it with
// /{fleetversion:(?:v1|2022-04|latest)}/ at registration time. We normalize it back to /_version_/ so the registry stays
// decoupled from the configured version list.
var versionTemplatePattern = regexp.MustCompile(`\{fleetversion:[^}]*\}`)

// muxParamRegexPattern strips regex constraints from gorilla/mux path params. For example {id:[0-9]+} becomes {id}, and
// {fleet_id:[0-9]+} becomes {fleet_id}. Fleet uses this style extensively, and the route tier policy in
// server/service/tracing_tiers.go registers the simpler {id} form. Without this normalization, spans whose mux template
// includes the regex would miss the registry and silently fall to TierAlways at 100% sampling.
var muxParamRegexPattern = regexp.MustCompile(`\{([a-zA-Z0-9_]+):[^}]+\}`)

func normalizeSpanName(name string) string {
	// Apply versionTemplatePattern first because it strips both the param name and the surrounding braces (replacing with
	// `_version_`). muxParamRegexPattern preserves the braces. Running it first would turn {fleetversion:...} into
	// {fleetversion}, which would then no longer match versionTemplatePattern.
	name = versionTemplatePattern.ReplaceAllString(name, "_version_")
	name = muxParamRegexPattern.ReplaceAllString(name, "{$1}")
	return name
}

func spanKey(method, path string) string {
	return method + " " + path
}
