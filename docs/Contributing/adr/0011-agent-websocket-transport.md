# ADR-0011: Agent WebSocket Transport

## Status

Proposed

## Authors

- @lucasmrod
- @lukeheath
- @sharon-fdm

## Date

2026-08-04

## Table of Contents

1. [What & Why](#1-what--why)
2. [How Orbit Proxies osquery](#2-how-orbit-proxies-osquery)
3. [Security Analysis](#3-security-analysis)
4. [Deployment & Rollout](#4-deployment--rollout)
5. [Consequences](#consequences)
6. [Alternatives Considered](#alternatives-considered)
7. [References](#references)

---

## 1. What & Why

### The problem

Today, every Fleet agent polls the server on fixed timers. The biggest offender is `distributed/read` (live query check-in): every host asks "anything for me?" every 10 seconds, and 99.7% of the time the answer is "no."

At 50k hosts this produces ~1.2 billion requests/day. 96.7% carry no useful payload.

| Metric (50k hosts) | Value |
|---|---|
| Daily requests | ~1.2B |
| Empty responses | 99.7% of distributed/read |
| Infra cost/day | $122 |

### The solution

Replace polling with a persistent WebSocket connection per agent. The server pushes a "check now" nudge only when there is actual work. The agent then makes one normal HTTP request to fetch it.

### How it works today (polling)

> *The diagrams below are simplified for illustration. See the full sequence diagrams in sections below.* 

```mermaid
sequenceDiagram
    participant Agent
    participant Server

    loop every 10s
        Agent->>Server: anything for me?
        Server-->>Agent: no (99.7%)
    end

    Note over Server: live query created
    loop next 10s tick
        Agent->>Server: anything for me?
        Server-->>Agent: yes, here's your query
    end
```

### How it works with WebSockets (push)

```mermaid
sequenceDiagram
    participant Agent
    participant Server

    Agent->>Server: poll config (existing mechanism)
    Server-->>Agent: config includes websocket_enabled=true
    Agent->>Server: open WebSocket (with jitter delay)
    Note over Agent,Server: silent until needed

    Note over Server: live query created
    Server->>Agent: check now (WebSocket push)
    Agent->>Server: distributed/read (HTTP)
    Server-->>Agent: here's your query
```

### Who decides and who connects

The **server** decides whether WebSocket transport is active, not the agent. When the feature flag is enabled on the server, it includes a WebSocket directive in the agent's configuration response (delivered through the existing config polling mechanism). On the next config poll:

- **New agent (supports WebSockets):** reads the directive and opens a WebSocket connection to the server. To avoid a thundering herd when the feature is first enabled, each agent applies a random jitter delay before connecting.
- **Old agent (no WebSocket support):** ignores the unknown directive and continues polling as before. No harm done.

> **The server is always in control.** Disabling the feature flag immediately stops new WebSocket connections on the next config cycle. Every agent falls back to polling. No agent action required, no rollback needed, no downtime.

### What travels over `distributed/read` today

The `distributed/read` endpoint is not just for live queries. Three distinct features share it, each with its own server-side interval:

| Feature | How it gets to the agent | Server includes it when... |
|---|---|---|
| **Live queries** | `distributed/read` | A campaign targets the host |
| **Policies** | `distributed/read` | PolicyUpdateInterval has elapsed (~1 hour) |
| **Software ingestion** | `distributed/read` (detail queries) | DetailUpdateInterval has elapsed (~1 hour) |

The agent polls `distributed/read` every 10 seconds. The server decides what to include based on these intervals. In steady state, most polls return empty because no live query is active and the hourly intervals have not elapsed.

**Scheduled queries (reports)** use a different channel: they are delivered via `/api/osquery/config` as part of the osquery pack configuration. This is covered by the config endpoint, not `distributed/read`.

### How each feature works today (polling)

> The diagrams below are simplified for illustration.

**Live queries** (the primary target of this ADR):

```mermaid
sequenceDiagram
    participant Agent
    participant Server
    loop every 10s
        Agent->>Server: distributed/read
        Server-->>Agent: empty (no active campaign)
    end
    Note over Server: admin runs a live query
    Agent->>Server: distributed/read (next 10s tick)
    Server-->>Agent: here is your query
    Agent->>Server: distributed/write (results)
```

**Policies** (delivered via distributed/read, but only every ~1 hour):

```mermaid
sequenceDiagram
    participant Agent
    participant Server
    loop every 10s
        Agent->>Server: distributed/read
        Server-->>Agent: empty (policy interval not elapsed)
    end
    Note over Server: 1 hour elapses
    Agent->>Server: distributed/read
    Server-->>Agent: policy queries
    Agent->>Server: distributed/write (pass/fail results)
```

**Software ingestion** (delivered via distributed/read as detail queries, ~1 hour):

```mermaid
sequenceDiagram
    participant Agent
    participant Server
    loop every 10s
        Agent->>Server: distributed/read
        Server-->>Agent: empty (detail interval not elapsed)
    end
    Note over Server: 1 hour elapses
    Agent->>Server: distributed/read
    Server-->>Agent: software inventory queries
    Agent->>Server: distributed/write (software list)
```

**Scheduled queries / reports** (different channel, via config):

```mermaid
sequenceDiagram
    participant Agent
    participant Server
    loop every 60s
        Agent->>Server: /api/osquery/config
        Server-->>Agent: config with pack/schedule (usually unchanged)
    end
    Note over Agent: osquery runs scheduled queries locally on their own timers
    Agent->>Server: /api/osquery/log (result logs)
```

### The WebSocket is a notification channel only

The WebSocket does **not** replace any existing functionality. It acts purely as a notification channel: the server sends a short "check now" signal, and the agent then performs the same HTTP calls it always has. No query data, no config payloads, no results travel over the WebSocket. Everything that works today continues to work exactly the same way. The only difference is that the agent no longer asks on a blind timer; it asks when told to.

Because live queries, policies, and software ingestion all share `distributed/read`, a single WebSocket nudge type covers all three. The agent does not need to know which feature triggered the nudge; it just calls `distributed/read` and the server returns whatever is due.

**Phase 1 (POC):** The notification channel covers `distributed/read` only (live queries, policies, software ingestion). This is the biggest offender (38.2% of all traffic, 99.7% empty) and proves the mechanism end to end.

**Future phases:** The same WebSocket carries notifications for additional channels:

| Phase | Channel | Current polling | What the nudge means |
|---|---|---|---|
| 1 (POC) | `distributed/read` | every 10s | "there is work for you" (live queries, policies, or software ingestion) |
| Future | orbit config | every 30s | "your config changed" |
| Future | Fleet Desktop check-in | every 5m | "there is something to show the user" |
| Future | osquery config | every 60s | "your osquery config changed" |

Each new channel is a new message type on the same connection. The pattern is always the same: the server sends a short nudge, the agent performs the corresponding HTTP call. One socket, many notification types, added incrementally.

### Keepalive and agent liveness

The server sends a WebSocket ping every **5 minutes**. This serves two purposes: confirming the agent is still alive, and preventing the load balancer (e.g. AWS ALB) from killing the connection due to idle timeout. The ALB idle timeout should be configured to a value longer than the ping interval (e.g. 10 minutes). If a pong is not received within 30 seconds, the server considers the connection dead and closes it. The agent's normal reconnection logic (with jitter) handles recovery.

### Feature flag and activation

The WebSocket transport is controlled by a server-side command-line flag (e.g. `--enable_websocket_transport`). It is not exposed in the UI, not documented, and not available through any API or GitOps configuration. The only way to enable it is to start the Fleet server with this flag. When disabled (the default), the directive is absent and all agents poll as usual.

### Thundering herd mitigation

When the feature flag is first enabled, every agent receives the WebSocket directive on its next config poll. If all agents immediately open a WebSocket, the server receives tens of thousands of connection attempts within seconds. The same problem occurs when the server restarts: every held connection drops, and all agents try to reconnect at once.

To prevent this, each agent applies a random jitter delay before connecting:

| Scenario | Jitter window | Rationale |
|---|---|---|
| First connection (feature just enabled) | 0 to 5 minutes | No urgency, spread load wide |
| Reconnection (server restart, network drop) | 0 to 30 seconds | Reconnect fast but avoid stampede |

If the connection fails after the jitter delay, the agent retries with exponential backoff (starting at 5s, capped at 5 minutes) plus a small random jitter on each retry.

The server also protects itself: each instance rate-limits new WebSocket upgrades (e.g. 200/second) and enforces a max connection cap. Excess attempts receive `503 Retry-After`, and the agent retries with its backoff logic.

**At 50k hosts with 5-minute initial jitter:** ~167 new connections/second on average, well within normal HTTP capacity.

### Connection balancing across server instances

Unlike HTTP polling (where the ALB routes each request independently), WebSocket connections are sticky. Imbalances accumulate when instances are added, restarted, or scaled.

Three layers handle this:

1. **ALB least-connections routing** sends new WebSocket upgrades to the least-loaded instance.
2. **Per-instance max connection cap.** At the limit, new upgrades get `503 Retry-After` and the ALB reroutes them. No inter-instance coordination needed.
3. **Graceful rebalancing.** Instances report connection counts to Redis. If one holds more than ~120% of the cluster average, it sends "please reconnect" to excess agents. They reconnect with jitter and the ALB redistributes them.

> **Example:** 100k agents on 2 instances. A third is added. A and B each shed ~17k connections; agents reconnect with jitter across all three.

### Key design choices

- **Nudge, not full push.** The WebSocket says "check now," then the agent does a normal HTTP call. The existing ingest pipeline stays untouched.
- **Server-driven activation.** The server controls whether agents use WebSockets via an undocumented command-line flag. Agents never decide on their own.
- **fleetd becomes osquery's distributed plugin.** osquery's 10s poll becomes a local call answered from memory. Nothing leaves the host in steady state.
- **One socket, many uses.** The same connection can eventually carry config updates, Desktop check-ins, and more, replacing multiple polling loops.
- **Fully backward compatible.** Old agents ignore the directive. New agents with an old server never receive it. Both keep polling.
- **Jittered connections.** Agents apply random delays to avoid thundering herd on activation or reconnection.

---

## 2. How Orbit Proxies osquery

osquery core is unaware of this change. Orbit acts as a local proxy using osquery's existing extension plugin system.

Today, osquery talks directly to the Fleet server over HTTP for `distributed/read` and `distributed/write`. With WebSocket transport enabled, orbit registers itself as osquery's `distributed` plugin via the osquery-go extension manager (the same mechanism orbit already uses for custom tables in `orbit/pkg/table/extension.go`). The server configures orbit to flip osquery's `--distributed_plugin` flag so osquery's 10-second poll becomes a localhost thrift call to orbit instead of an HTTP call to the Fleet server.

osquery does not know the difference. It keeps its 10-second loop, but the call never leaves the machine.

```mermaid
sequenceDiagram
    autonumber
    participant OSQ as osquery
    participant Orbit as orbit (distributed plugin)
    participant S2 as Fleet server B (holds this agent's WebSocket)
    participant S1 as Fleet server A (creates campaign)
    participant Redis

    Note over OSQ,S1: SETUP
    Orbit->>S2: poll config (existing HTTP)
    S2-->>Orbit: config with websocket_enabled=true
    Orbit->>Orbit: register as osquery's distributed plugin
    Orbit->>Orbit: flip --distributed_plugin to local extension
    Orbit->>S2: open WebSocket (with jitter delay)
    S2->>S2: hold connection

    Note over OSQ,Orbit: STEADY STATE (nothing to do)
    loop every 10s
        OSQ->>Orbit: any queries? (localhost thrift)
        Orbit-->>OSQ: no (answered from memory)
    end
    Note over OSQ,S2: zero network traffic

    Note over S1,Redis: LIVE QUERY CREATED (on server A)
    S1->>Redis: write targeting state + PUBLISH wake-up (new)
    Redis-->>S2: server B receives wake-up (new)
    S2->>Orbit: check now (over WebSocket)
    Orbit->>S2: distributed/read (HTTP)
    S2-->>Orbit: query payload
    Orbit->>Orbit: cache query in memory

    Note over OSQ,Orbit: NEXT LOCAL POLL (within 10s)
    OSQ->>Orbit: any queries? (localhost thrift)
    Orbit-->>OSQ: yes, here is your query
    OSQ->>OSQ: execute query
    OSQ->>Orbit: results (localhost thrift)
    Orbit->>S2: distributed/write (HTTP)
```

**Note on Redis pub/sub (new mechanism):** Today, campaign targeting is written to Redis keys and each server instance discovers it independently when a host polls. There is no inter-instance notification. With WebSockets, the server instance holding an agent's connection may not be the one that created the campaign. To solve this, campaign creation adds a Redis `PUBLISH` on a wake-up channel. All server instances subscribe to this channel and nudge the relevant agents they hold. Redis pub/sub already exists in Fleet for streaming query results back; this extends the same pattern for the wake-up signal.

**Orbit proxies all osquery traffic, not just distributed queries.** With orbit acting as osquery's proxy, all osquery-to-server communication flows through orbit. This includes:

- `distributed/read` and `distributed/write` (live queries, policies, software ingestion)
- **Scheduled query result logs** (`/api/osquery/log`). Today osquery sends these directly to the server. With the proxy, osquery sends results to orbit locally, and orbit forwards them to the server via HTTP.
- Config fetches (`/api/osquery/config`), if orbit also registers as the config plugin in a future phase.

Orbit is the single point of contact between the host and the server. osquery communicates only with orbit on localhost; orbit handles all external network communication.

**Why the extension plugin approach (not an HTTP proxy):**

- Orbit already runs an osquery extension manager for custom tables. Registering a `distributed` plugin is the same mechanism.
- No HTTP proxying, no TLS interception, no URL rewriting needed.
- The localhost thrift call has zero network overhead.
- osquery requires zero code changes and has zero awareness of the WebSocket.

---

## 3. Security Analysis

### Transport encryption

WebSocket connections use `wss://` (WebSocket Secure), which runs over TLS. The WebSocket upgrade starts as a standard HTTPS request and then upgrades the connection in place. It inherits the same TLS certificate, cipher suites, and certificate validation as all other Fleet HTTPS traffic. No additional encryption configuration is needed. Unencrypted `ws://` connections must be rejected by the server.

### Authentication

The WebSocket upgrade request is authenticated using the **orbit node key**, the same credential orbit already uses for all its HTTP calls to the Fleet server. The server validates the node key during the HTTP upgrade handshake, before the connection is promoted to a WebSocket. If the key is invalid or revoked, the upgrade is rejected with a `401` and no WebSocket is established.

Once connected, no further authentication is needed per message because the WebSocket is a persistent, authenticated session. If the node key is revoked while a connection is open, the server should close that connection on the next keepalive cycle.

### Connection exhaustion (DoS)

Holding 100k-200k open WebSocket connections increases the server's attack surface for resource exhaustion:

| Resource | Risk | Mitigation |
|---|---|---|
| File descriptors | Each WebSocket consumes one fd per server instance | Set OS-level fd limits (`ulimit`) appropriately; enforce per-instance max connection cap |
| Memory | Each connection holds a small buffer (~4-8 KB) | At 200k connections across a cluster, this is ~800 MB total, distributed across instances |
| CPU | Idle connections consume near-zero CPU; pings every 5 min are negligible | No special mitigation needed |
| Unauthenticated connection attempts | An attacker could flood the upgrade endpoint | Rate-limit WebSocket upgrades per source IP; reject upgrades that fail authentication immediately before allocating resources |

**Key mitigation:** The WebSocket upgrade endpoint must authenticate the node key **before** allocating connection resources. A failed auth check should return `401` and close the TCP connection immediately, not hold it open.

### Attack surface comparison

The WebSocket does not introduce new data flows or new trust boundaries:

| Concern | Today (polling) | With WebSocket |
|---|---|---|
| Encryption | TLS (HTTPS) | TLS (WSS), same certs |
| Authentication | Orbit node key per request | Orbit node key at upgrade, then persistent |
| Data on the wire | Full query payloads, config, results | Only "check now" nudges (a few bytes); all payloads still go over HTTP |
| Server-to-agent channel | None (pull only) | Yes, but carries no sensitive data |
| Spoofing risk | Attacker needs valid node key | Same, node key required at upgrade |

The new server-to-agent channel (the nudge) carries no sensitive information. The worst an attacker could do if they compromised the WebSocket channel is send a false "check now" signal, which would cause the agent to make one extra HTTP `distributed/read` call. This is equivalent to what already happens every 10 seconds today.

### Redis pub/sub

The wake-up signal is published through Redis pub/sub so all server instances can relay nudges to agents connected to them. Redis should be deployed on a private network with authentication enabled (Fleet's existing Redis configuration). The pub/sub message contains only the campaign ID and target host identifiers, not query content or results.

---

## 4. Deployment & Rollout

The feature flag gives us full control over when and where WebSocket transport is enabled. The proposed rollout is incremental, with validation at each stage before moving to the next.

### Proposed rollout order

**Stage 1: Dogfood.** Enable the feature flag on Fleet's internal Dogfood server. Validate stability, connection behavior, keepalive, reconnection, and fallback. This is a low-risk environment where we can observe the feature under real (but internal) usage.

**Stage 2: Volunteer customer.** Identify a customer willing to opt in early. Coordinate with Customer Success on whether to offer an incentive (e.g. reduced hosting cost) in exchange for being an early adopter. Run with the feature enabled, monitor closely, and gather feedback.

**Stage 3: Broader managed cloud rollout.** Expand to additional customers. Volunteers first, then progressively enable on more deployments as confidence grows. At each step, validate cost savings match expectations and no regressions occur.

**Stage 4: Document and publish.** Once the feature is proven at scale across managed cloud, document the feature flag and make it available to self-hosted customers who want to enable it on their own infrastructure.

> **Open question for @lucasmrod and @lukeheath:** What do you think of this rollout order? Any concerns or suggestions?

---

## Consequences

### Where the cost savings come from

With ECS Fargate, we pay per container (vCPU + memory allocation), not per CPU cycle. Keeping 15 instances idle saves nothing on compute. The savings come from three areas:

1. **Fewer/smaller server instances.** With 1.2B fewer HTTP requests/day, the cluster needs far less compute capacity. The instance count can be reduced (e.g. 15 to 8) or instances can be downsized.
2. **Reduced ALB costs.** ALB pricing is based on LCUs (new connections, active connections, bandwidth). Eliminating most HTTP requests dramatically reduces LCU usage.
3. **Reduced data transfer.** Fewer HTTP responses means less egress.

| | Today | With WebSockets |
|---|---|---|
| Server instances | 15 | Fewer (right-sized to actual load) |
| HTTP requests/day | ~1.2B | Near zero in steady state |
| Infra cost/day | $122 | ~$52 (measured in load test) |
| What drives the savings | -- | Fewer instances + lower ALB/transfer costs |

### The tension: fewer instances vs. WebSocket headroom

Reducing instance count increases WebSocket density per instance. Each instance must still handle HTTP bursts when nudges fire (agents do `distributed/read` and `distributed/write` over HTTP). The connection budget per instance must account for both.

**Worked example at 50k hosts:**

The OS file descriptor limit is typically 65,536. AWS ALB supports up to 100,000 concurrent connections per target. Neither is the bottleneck, but we must budget within them.

Worst case: a query targets all 50k hosts. Every agent is nudged and makes an HTTP `distributed/read` call. The ALB distributes these across all instances evenly.

| | Today (15 instances) | WebSocket option A (8 instances) | WebSocket option B (5 instances) |
|---|---|---|---|
| WebSocket connections/instance | 0 | ~6,250 | ~10,000 |
| Peak concurrent HTTP (worst case: all 50k nudged) | ~3,333 (50k/15) | ~6,250 (50k/8) | ~10,000 (50k/5) |
| Other (Redis, MySQL, internal) | ~200 | ~200 | ~200 |
| **Total connections/instance** | **~3,533** | **~12,700** | **~20,200** |
| Headroom (vs 65k fd limit) | 62k free | 53k free | 45k free |
| Memory for WebSockets/instance | 0 | ~50 MB | ~80 MB |

In the worst case, with 5 instances each instance handles ~10k WebSockets + ~10k concurrent HTTP requests simultaneously. Total ~20k connections is still well under both the OS fd limit (65k) and ALB target limit (100k). However, the HTTP burst is short-lived (each request completes in milliseconds), so the actual concurrent HTTP count at any instant will be lower than the total 10k.

> **Key takeaway:** The number of connections is not the limiting factor. We can safely reduce instance count to save money. The right sizing is driven by how much HTTP burst capacity each instance needs when nudges fire, not by connection limits.

The exact instance count and sizing should be determined by load testing with WebSockets enabled, measuring both steady-state resource usage and burst HTTP capacity under nudge scenarios.

---

## Alternatives Considered

### Long polling

Each agent holds an open HTTP request to the server. The server responds only when there is work, or after a timeout (e.g. 2 minutes), at which point the agent immediately opens a new request.

- **Pros:** Simpler than WebSockets. No upgrade handshake, no new protocol. Works through any HTTP proxy.
- **Cons:** One parked request per channel. Cannot multiplex: adding config, Desktop, and orbit notifications would require separate long-poll connections per agent. Each timeout-and-reconnect cycle creates a new TLS connection (vs. WebSocket which holds one). At 50k hosts with a 2-minute timeout, that is 25k new TLS connections/minute just from timeouts.
- **Why not chosen:** WebSockets support multiplexing multiple notification types on a single connection, which is critical for future phases. Long polling forfeits this and adds connection churn.

### Server-Sent Events (SSE)

The server pushes events to the agent over a long-lived HTTP response using the `text/event-stream` content type. The agent opens one GET request and the server streams events as they occur.

- **Pros:** Simpler than WebSockets. Built on standard HTTP, works through most proxies. Native reconnection with `Last-Event-ID`. One connection can carry multiple event types.
- **Cons:** Unidirectional (server to agent only). The agent cannot send data back over the same connection, so all agent-to-server communication still requires separate HTTP calls (which is also true of our WebSocket design). More critically, SSE requires HTTP/1.1 chunked transfer or HTTP/2. ALB-to-target communication in Fleet's topology is HTTP/1.1, which limits concurrent SSE streams per browser/client. Enterprise middleboxes (TLS-inspecting proxies) often buffer chunked responses, breaking the real-time delivery that SSE depends on.
- **Why not chosen:** The middlebox buffering problem is the same class of issue that led Kolide to add an HTTP fallback for gRPC. WebSockets have a cleaner upgrade mechanism that middleboxes handle better in practice. Fleet already runs WebSockets in production (live query results in the UI), so operational experience exists.

### gRPC streaming

Replace the HTTP API with gRPC bidirectional streaming. The agent holds a persistent gRPC stream to the server.

- **Pros:** Strong typing via protobuf. Bidirectional streaming. Efficient binary protocol.
- **Cons:** Requires HTTP/2 end-to-end. Fleet's ALB terminates TLS and forwards HTTP/1.1 to backend tasks, making gRPC unreachable without infrastructure changes (h2c or network load balancer). Enterprise middleboxes break HTTP/2 far more often than plain HTTPS. Kolide launcher shipped gRPC first and had to add a plain-HTTPS fallback for exactly this reason.
- **Why not chosen:** Blocked by Fleet's deployed ALB topology and unreliable through enterprise network gear.

### ETag / conditional requests (ADR-0012)

Reduce response size by having agents send an ETag with each request. The server returns a minimal "not modified" response when the config hasn't changed. See [ADR-0012](0012-osquery-config-conditional-requests.md).

- **Pros:** Small, self-contained change. Benefits every deployment including self-hosted without WebSockets. No infrastructure changes needed.
- **Cons:** Does not eliminate the requests themselves, only shrinks responses. The agent still polls on a fixed timer. Requires an upstream osquery change.
- **Why not chosen (as a replacement):** The two are complementary, not exclusive. ETag makes the polling fallback path cheap. WebSockets eliminate the polling entirely. Together they cover both managed cloud (WebSocket-enabled) and self-hosted (polling with ETag) deployments.

---

## References

- [Agent transport phase 1: `distributed/read` POC](https://github.com/fleetdm/confidential/issues/17019)
- Load test cost data (July baseline)
