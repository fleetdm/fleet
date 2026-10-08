# v5.0: Mandatory pagination on all list endpoints (breaking)

> **Template:** 🎟 Story · **Labels:** `story`, `reliability` · **Effort:** M–L · **Milestone:** v5.0.0 · **Wave:** 3

## Goal

| User story |
|:--|
| As a Fleet operator with a very large fleet, |
| I want every list API to return bounded pages, |
| so that no single request — from the UI, fleetctl, GitOps, or an integration — can ask the server to materialize 400k rows. |

## Why v5.0

Several list endpoints return everything when no paging params are passed, and clients (including our own UI, fleetctl, and GitOps flows) depend on that. Changing the default from "all" to "first page" breaks those clients — exactly the kind of change the major release exists for. Doing it uniformly also lets us enforce it structurally instead of endpoint-by-endpoint.

## Changes

- **Contract:** every list endpoint has a default `per_page` (e.g. 100) and an enforced maximum (e.g. 500–1000, per-endpoint where justified). Omitting paging params returns page one, not everything. Response envelope always includes paging metadata.
- **Keyset where it matters:** hosts, software, vulnerabilities, and other fleet-scale tables get keyset (`after`/cursor) pagination; deep OFFSET on large tables is itself a scale bug.
- **Internal callers:** audit service/datastore code paths that call list methods without limits (including "list to count" and fan-out helpers) — internal callers get the same bounds or an explicit paginating iterator.
- **Own clients first:** UI tables, fleetctl commands, and GitOps generate/apply paths updated to paginate; these are the compatibility canaries.
- **Enforcement:** shared list-options handling rejects unbounded requests, so a new endpoint can't opt out silently.

**Product checklist:**
- REST API changes: pagination section + per-endpoint updates, PR to reference docs release branch.
- CLI changes: fleetctl list commands paginate transparently (user-invisible except speed).
- Compatibility: breaking in v5.0.0 — release notes + CS notification; document the migration ("add paging loop or raise per_page to max").

## Engineering notes

- Risk level: High (breadth, not depth) — mitigate by landing the shared enforcement last, after per-endpoint PRs have migrated internal callers.
- Load testing: list-endpoint latency on the 400k-host seeded snapshot (issue 16) before/after; deep-page access patterns included.

## Condition of satisfaction

- No endpoint returns an unbounded collection; CI check or shared-code enforcement prevents regressions.
- UI, fleetctl, and GitOps work unchanged from a user's perspective against a 400k-host seeded DB.
