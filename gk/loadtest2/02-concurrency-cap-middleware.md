# Per-container in-flight concurrency cap (first overload guard)

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** S–M · **Wave:** 1

## Problem

A Fleet server container accepts unbounded concurrent requests. Under overload, requests queue inside the process (goroutines pile up waiting on the DB pool), latency explodes, memory grows, and the container eventually OOMs or times everything out — the classic queuing collapse. Nothing bounds the one variable that actually kills servers: concurrent work in flight.

## Impact

This is the difference between "p99 degrades and some agent check-ins retry later" and "the deployment is down, including the UI." Highest-leverage single guard we can ship; prerequisite thinking for the adaptive shedding in issue 04.

## Proposed fix

A semaphore middleware at the top of the handler chain (`server/service/handler.go`):

- When N requests are already executing, immediately return `503` with a jittered `Retry-After` — do not queue.
- N configurable (`FLEET_SERVER_MAX_INFLIGHT` or similar), default derived from `max_open_conns` plus headroom, since DB-bound work dominates.
- Exempt a small always-on allowlist: health checks, version, login — an operator must be able to get in while agents are shed (full tiering comes in issue 05; hardcode the minimal list for now).
- Increment `shed_requests_total{reason="inflight_cap"}` (names from issue 01).
- On osquery endpoints the rejection MUST be a plain HTTP error — see issue 03.

## Condition of satisfaction

- Local proportionate rig (issue 17) or a constrained docker-compose run demonstrates: at 3× capacity load, requests beyond the cap get fast 503s, p99 for admitted requests stays bounded, memory stays flat, and the server recovers immediately when load drops.
- Config documented; shipped enabled with a generous default rather than opt-in off.

## Evidence

Fixed-quota GCRA limiting exists only for login/SSO/MFA flows (`server/service/handler.go:969–1310`); nothing covers the agent hot path or tracks actual server state.
