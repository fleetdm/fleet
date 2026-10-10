# fleetd/orbit honors Retry-After with jittered backoff

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering`, `#g-orchestration` (fleetd) · **Effort:** M · **Wave:** 2 — **start early: rides the agent release cycle**

## Problem

osquery retries failed check-ins at its configured interval — tolerable natural backoff — but neither osquery nor orbit reads `Retry-After`. When the server starts shedding (issues 02/04/05), non-compliant agents all come back on their fixed intervals, so the shed load returns as the same synchronized spike instead of spreading out. Server-side shedding without client-side backoff just converts overload into a retry storm with extra steps.

## Impact

Caps the effectiveness of the entire admission-control stack. Also the longest-lead-time item in the plan: it ships with fleetd, and large customers upgrade agents slowly — every week this isn't started pushes compliant-agent coverage further past the server-side work.

## Proposed fix

- Orbit's HTTP client (and the update/config/scripts/software paths it owns): on `503`/`429` with `Retry-After`, defer the next attempt to the indicated time plus jitter (±20%), capped (e.g. 30m) and floored (min 30s) defensively.
- For osquery-owned traffic that orbit proxies or launches, apply what we control: at minimum the orbit-managed flags/extensions paths; document that raw osquery check-ins rely on interval-based backoff.
- Expose last-backoff state in `fleetctl debug`/orbit logs so support can verify behavior in the field.
- Version-gate nothing: honoring Retry-After against old servers (which never send it) is a no-op, so this can ship ahead of the server work.

## Condition of satisfaction

- Against a local server returning 503 + Retry-After: orbit's retry timing follows the header with jitter (verified in an integration test with a fake clock or short durations).
- Released in a fleetd version before the v5.0.0 server ships, so compliant agents exist in the field when tiered shedding turns on.

## Evidence

Server-side counterpart: issues 02/04/05. The v5.0 API contract (issue 05) documents the client obligation this implements.
