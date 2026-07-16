// Package tracing implements a route aware head sampler for Fleet's OTEL trace export. The sampler classifies each span via a
// Registry. Each bounded context populates the Registry at startup with its own routes. The sampler then applies per tier ratio
// sampling. The goal is to prevent noisy hot agent paths from drowning out the rare load bearing paths (enroll, MDM command
// flows, cron jobs).
//
// Runtime control lives in the trace_sampler_settings MySQL row. Each Fleet replica runs StartSettingsPoller which re-reads the
// row every 60 seconds and atomically swaps the sampler's state. No restart is required to flip force_full during an incident
// debug window.
//
// Architecture: platform/tracing owns the mechanism (sampler, tier enum, registry). Each bounded context owns the policy: which
// of its routes belong in which tier. Each context registers them at startup. This keeps the platform package free of cross
// context coupling.
package tracing
