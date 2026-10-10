# v5.0: Uniform overload semantics and endpoint priority tiers (breaking)

> **Template:** 🎟 Story · **Labels:** `story`, `reliability` · **Effort:** L · **Wave:** 2–3 · **Milestone:** v5.0.0 · **Depends on:** 02, 04

## Goal

| User story |
|:--|
| As a Fleet operator with a very large fleet, |
| I want the server to shed low-priority work predictably and keep critical flows alive under any load, |
| so that an overload degrades gracefully instead of taking down enrollment, live queries, and the UI together. |

## Why v5.0

Today's API has no overload contract: clients can't know which status codes mean "back off," whether `Retry-After` will be present, or which operations may be deferred. Defining one properly is a behavioral breaking change (new 503s on endpoints that never returned them, enforced client backoff expectations), so the major release is the window to make it **uniform across the whole API** instead of endpoint-by-endpoint special cases.

## Changes

**The contract (documented in the REST API reference as a top-level section):**
- `503` + `Retry-After` is the single overload signal, on every endpoint. `429` remains reserved for per-client quota violations (login/SSO style limits).
- All agent-facing write endpoints are documented idempotent, with an audit to make that true where it isn't — retried requests must not amplify into duplicate writes.
- Clients (fleetd first, API consumers by documentation) MUST honor `Retry-After` with jitter; the contract states the server may drop non-compliant clients' requests first.

**Priority tiers (wired into the shedding from issues 02/04):**

| Tier | Shed order | Contents (first pass — product sign-off needed) |
|--|--|--|
| 0 — never shed | last | health, login/SSO, license, the minimal UI read set an operator needs during an incident |
| 1 — critical agent | late | enrollment, orbit config, live query results |
| 2 — deferrable agent | early | detail/inventory refresh, scheduled query log submission, vitals |
| 3 — batch/expensive | first | report exports, large list endpoints, non-urgent sync |

- Tier assignment lives in the route registration so new endpoints MUST declare a tier (compile-time or startup check — the uniform-by-construction part this breaking window buys us).

**Product checklist (story template items):**
- UI changes: No changes (UI keeps working — that's the point).
- REST API changes: new "Overload behavior" reference section; PR to reference docs release branch.
- fleetd changes: see issue 06 (Retry-After compliance).
- Compatibility: breaking in v5.0.0 — notify CS per handbook; release notes entry required.

## Engineering notes

- Risk level: High — touches every route registration; mitigate with the tier declared defaulting to tier 2 for unlisted agent routes during rollout.
- Load testing: full-scale terraform run demonstrating tiered shedding before GA; local rig scenario in issue 17.

## Condition of satisfaction

- Under 5× capacity load on the local rig: tier 3 sheds first, tier 0 never sheds, UI remains usable throughout, recovery is automatic.
- Every registered route has an explicit tier; adding a route without one fails CI or startup.
