# ADR-0012: Conditional requests for the osquery config endpoint

## Status

Approved

## Date

2026-08-07

## Context

Every enrolled host downloads its osquery configuration by calling `POST /api/v1/osquery/config` every `config_tls_refresh` seconds (60 seconds by default). At the default interval, each host makes 1,440 config requests per day; a deployment with 50,000 hosts serves ~72 million config responses per day.

Infrastructure observations show that roughly 80% of Fleet server egress is this endpoint ([#50157](https://github.com/fleetdm/fleet/issues/50157)). Yet the response almost never changes: it is built in `Service.GetClientConfig` (`server/service/osquery.go`) from agent options (resolved per team and platform) and the pack config (scheduled reports, identical for all hosts on the same team). It only changes when an admin edits agent options or adds/edits/removes a report — rare events compared to a 60-second polling cadence. In steady state, Fleet re-marshals and re-sends the same multi-kilobyte payload to every host, every minute.

Server-side caching already exists: the pack config is cached per `(team, queryReportsDisabled)` in `packConfigCache`, which cuts database load and most of the marshaling cost. But caching does nothing for egress — the full response body still goes out on every request, and response egress is where the cost is.

The standard HTTP answer is conditional requests: the client presents a validator for the representation it already has (`If-None-Match` + ETag), and the server replies `304 Not Modified` with an empty body when nothing changed. Two constraints prevent using the mechanism literally:

- osquery's TLS config plugin does not send any validator today, and it may treat a `304` (or any non-200) response as an error. Either way, an osquery-side change is required.
- The config request is a `POST`, so standard HTTP caching semantics (designed around `GET`) don't apply cleanly, and intermediaries can't be trusted to pass conditional-request headers through unmodified.

Any design must also degrade gracefully: older agents that never send a validator must keep receiving the full config, unchanged.

## Decision

Fleet will implement ETag/304-style conditional requests for the osquery config endpoint, carried in the request and response JSON bodies rather than in HTTP headers and status codes.

1. **The server assigns the validator.** The server computes a hash of the marshaled config and returns it in the config response under an `"etag"` key. The full config for a host is a function of its team and platform (agent options are resolved per team/platform; the pack config is per team), so the etag is computed and cached per `(team, platform)`, alongside the existing `packConfigCache` and invalidated by the same events (agent options changes, report changes). Computing and caching this is cheap.

2. **The agent echoes the etag back.** osquery includes an `"etag"` field in the `POST /api/v1/osquery/config` request body containing the etag from the last config response it received (empty on its first request). The agent never computes anything — it stores the server's opaque value and echoes it. The etag deliberately acknowledges *receipt*, not successful application: if the agent echoed only its last successfully applied config, a config that fails to apply on the host would be retransmitted in full on every refresh — exactly the redundant egress this mechanism eliminates, and at 100k hosts a bad config push would recreate the full load with no benefit. Apply failures are the agent's to track and surface (osquery logs a warning and fails the config refresh on every cycle), so nothing is masked by acknowledging receipt. This requires an upstream osquery change to the TLS config plugin ([osquery/osquery#9033](https://github.com/osquery/osquery/pull/9033)); Fleet employs osquery committers, and the change reaches agents through the osquery version bundled in fleetd.

3. **On a match, the server returns a minimal "not modified" response.** If the agent's etag equals the current etag for its team/platform, the server responds `200 OK` with the constant body `{"etag":"ok"}` — the smallest response that still tells the agent its config is current — and osquery keeps its current config. On a mismatch — or an empty etag — the server returns the full config with the current `"etag"` key included.

4. **Old agents see no change.** An agent that does not send an `"etag"` field in the request has not opted in: the server always returns the full config and omits the `"etag"` key from the response, byte-for-byte identical to today's behavior.

5. **Hosts with legacy packs bypass the optimization.** Legacy packs (`ListPacksForHost`) make the response per-host rather than per-team. These hosts always receive the full config, matching the existing cache bypass in `getPackConfig`.

6. **Both sides are behind a feature flag, enabled by default.** The osquery change ships behind an osquery flag and the server change behind a Fleet server flag, both on by default so the savings apply out of the box. Either side can be switched off independently as an escape hatch: disabling the osquery flag stops the agent from sending the `"etag"` field, and disabling the server flag makes the server ignore incoming etags and always return the full config (without the `"etag"` key). Because the protocol degrades gracefully in both directions, any combination of flag states is safe — the worst case is today's behavior.

Carrying the validator in the JSON bodies instead of using a real `If-None-Match`/`304` exchange was chosen because the request is a `POST` (header-based conditional semantics would be nonstandard), because osquery's config plugin error-handles non-200 responses, and because a body field makes version negotiation trivial: an old agent simply never sends the field and never sees the new response shape. Having the server assign the etag (rather than agents hashing their applied config) keeps the validator opaque — the server can change how it computes the value at any time without coordinating with agents. The literal value `"ok"` is reserved and never used as a real etag.

## Consequences

**Positive:**

- In steady state, nearly all config responses shrink from multiple kilobytes to the 13-byte `{"etag":"ok"}` body, directly reducing the dominant share of server egress.
- The server skips unmarshaling agent options and assembling the response map on the not-modified path, saving CPU per request.
- Fully backward compatible in both directions: old agents never send an etag and always get today's exact response; new agents against an old server get the full config without an `"etag"` key and simply keep sending an empty etag.
- The mechanism also serves as the efficient fallback path for deployments that never enable push-based transport (see Alternatives), and remains useful alongside it.
- The feature flags provide an immediate kill switch on either side: if a stale-etag bug (or any misbehavior) is suspected, disabling the server flag instantly restores full-config responses for every host, with no agent action or upgrade required.

**Negative:**

- Depends on an upstream osquery change; savings only materialize as fleets upgrade to a fleetd/osquery version that sends the etag.
- A stale-etag bug could leave hosts running an outdated config indefinitely. The etag must cover the complete effective response, and its cache must be invalidated on every mutation path (agent options edits, report add/edit/delete, GitOps batch application). This is the primary correctness risk and the focus of the test plan.
- `GetClientConfig` currently persists interval changes (`UpdateHostOsqueryIntervals`) when the delivered config alters `distributed_interval`, `logger_tls_period`, or `config_refresh`. The not-modified path skips this bookkeeping; that is safe only because a matching etag implies the server already delivered exactly this config (received and matching, not necessarily applied), and the intervals were reconciled server-side at that delivery. Implementation must keep this invariant.
- Load testing is required to validate the win — before/after measurements of egress and CPU are part of the story's acceptance ([#50157](https://github.com/fleetdm/fleet/issues/50157)), and osquery-perf must be updated to simulate etag-sending agents.

## Alternatives considered

### Push-based transport (agent WebSocket nudges)

Replace polling entirely: the server holds a persistent WebSocket per agent and pushes a "your config changed" nudge, after which the agent fetches over HTTP (see ADR-0011).

- **Pros:** Eliminates the request as well as the response; one mechanism eventually covers all polling endpoints (`distributed/read`, orbit config, Fleet Desktop).
- **Cons:** Much larger scope — connection management at 100k+ connections, Redis pub/sub wake-ups, load-balancer configuration, thundering-herd handling. It is opt-in via a server flag, its first phase covers only `distributed/read`, and polling remains the permanent fallback for agents or deployments without WebSockets.
- **Why not chosen (as a replacement):** The two are complementary, not exclusive. Conditional requests are a small, self-contained change that benefits every deployment — including self-hosted servers that never enable WebSocket transport — and they make the fallback polling path cheap.

### Real HTTP `ETag` / `If-None-Match` / `304 Not Modified`

- **Pros:** Standards-based; intermediaries and tooling understand it.
- **Cons:** The config request is a `POST`, where conditional-request semantics are nonstandard; osquery's config plugin would still need to change to send the header and to not treat `304` as an error; proxy behavior around `304` on `POST` is unpredictable.
- **Why not chosen:** It requires the same osquery change anyway, with more protocol risk and no additional benefit over the body-field mechanism.

### Increase `config_tls_refresh`

- **Pros:** Zero code changes; immediately reduces request volume linearly.
- **Cons:** Slows config propagation (an admin's report edit takes longer to reach hosts); must be changed in every team's agent options, not just globally; savings are linear in the interval rather than near-total.
- **Why not chosen:** It trades responsiveness for cost instead of removing the redundancy. It remains an orthogonal knob operators can turn independently.

### Server-side caching only (status quo)

- **Pros:** Already implemented (`packConfigCache`); no agent changes.
- **Cons:** Saves database queries and marshaling, but every response still carries the full payload; does nothing for egress, which is the dominant cost.
- **Why not chosen:** It is kept — the etag cache builds on it — but it cannot address the problem on its own.

### Response compression

- **Pros:** Reduces bytes per response with no protocol change.
- **Cons:** Every response is still built and sent; savings are bounded by the compression ratio and paid for with CPU on every request, versus near-elimination of the body on the not-modified path.
- **Why not chosen:** Complementary at best; it does not remove the redundant work.

## References

- [#50157: Reduce `/api/v1/osquery/config` traffic with ETag / HTTP 304-style conditional requests](https://github.com/fleetdm/fleet/issues/50157)
- `Service.GetClientConfig` and `getPackConfig`: `server/service/osquery.go`
- [ADR-0011: Agent WebSocket Transport](0011-agent-websocket-transport.md)
- osquery TLS config plugin change: [osquery/osquery#9033](https://github.com/osquery/osquery/pull/9033)
- osquery TLS config plugin: https://osquery.readthedocs.io/en/stable/deployment/remote/
