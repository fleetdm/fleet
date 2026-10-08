# osquery endpoints must never signal node_invalid on server errors

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** S · **Wave:** 1

## Problem

When osquery receives a `node_invalid` response it discards its node key and re-enrolls. If any server error path — overload, DB timeout, shedding from issues 02/04, a panic recovery handler — ever responds to an enroll/config/distributed/log request in a way the agent interprets as node invalidation, an overload event converts into a mass re-enrollment storm: every affected host immediately issues enrollment requests, which are far more expensive than the check-ins that were shed. This is the amplification that turns an incident into an outage.

## Impact

Catastrophic amplification exactly when the server is least able to absorb it. At 400k hosts, a few minutes of mistaken node_invalid responses means hundreds of thousands of re-enrollments plus the resulting host re-identification load. Must be verified BEFORE we ship any shedding (issues 02/04), since shedding multiplies the number of error responses agents see.

## Proposed fix

- Audit every error path in the osquery service endpoints (`server/service/osquery.go` and the handler/transport layer error encoding): enumerate which responses can carry `node_invalid: true` and prove it is only set on genuine authentication failure of the node key — never on datastore errors, timeouts, or 5xx/429/503 conversions.
- Add a regression test that drives each osquery endpoint with an injected datastore failure and asserts the response is a plain HTTP error without node_invalid semantics.
- Document the invariant in the endpoint code so future error-handling refactors keep it.

## Condition of satisfaction

- Written audit result in the issue (list of paths checked).
- Regression tests merged covering enroll, config, distributed read/write, and log endpoints under injected server failure.

## Evidence

osquery TLS plugin behavior: any response with `node_invalid` triggers re-enrollment on next check-in; prior related hardening around `server/service/osquery.go:1323`.
