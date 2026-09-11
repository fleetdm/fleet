# Mock APNs push server

Fleet disables MDM push notifications during load tests today, so every simulated host checks in on a timer. That hides how Fleet behaves with realistic push-driven check-ins. `cmd/apple-apns-mock` is a stand-in for Apple's push notification service (APNs) that closes this gap: Fleet pushes to it exactly as it pushes to `api.push.apple.com`, and simulated devices receive wake-ups instead of polling.

## Architecture

```
                       POST /3/device/<token>
+----------------+     {"mdm":"<PushMagic>"}     +-------------------+
|  Fleet server  | ----------------------------> |  ALB (any target) |
|    (buford)    |                               +-------------------+
+----------------+                                 |       |       |
                                                   v       v       v
                                            +--------+ +--------+ +--------+
                                            | mock 1 | | mock 2 | | mock 3 |
                                            +--------+ +--------+ +--------+
                                                 \        |        /
                                                  \       |       /
                                                 +-------------------+
                                                 |       Redis       |
                                                 | pending keys +    |
                                                 | push channel      |
                                                 +-------------------+
                                                   ^       ^       ^
                                            GET /events?token= (SSE)
                                                   |       |       |
                                            osquery-perf simulated devices
                                              (pkg/mdm/apnsmock client)
```

- Fleet's push path is unchanged: `server/mdm/nanomdm/push/buford` sends `POST /3/device/<token>` with body `{"mdm":"<PushMagic>"}` and no `apns-*` headers. Only the base URL differs.
- Simulated devices hold a long-lived server-sent events (SSE) connection, the mock's stand-in for a real device's persistent APNs courier connection. Each `event: ping` should trigger an MDM check-in.
- The load balancer gives a device's stream to one instance and Fleet's push for that device to another, so the instances coordinate through Redis. Redis is required; there is no single-instance mode.

The wire contract (request and response shapes, header semantics, error bodies) is documented in [`cmd/apple-apns-mock/README.md`](../../../../cmd/apple-apns-mock/README.md) and on the handlers in `cmd/apple-apns-mock/handlers.go`. `TestE2EBufordCompatibility` pins the contract against the actual buford client Fleet uses in production.

## Routing between instances

Every push is written to Redis before it is announced, so an offline device needs no detection: the key simply waits until the device connects somewhere.

```
push at any instance    SET <prefix>pending:<token> <ping> GET PX <ttl>
                        PUBLISH <prefix>push {"t":"<token>"}

every instance          holds the token? -> GETDEL <prefix>pending:<token>
                                              value -> write it to the stream
                                              nil   -> another instance claimed it

connect at any instance GETDEL <prefix>pending:<token>, then announce ownership
```

The `GETDEL` claim keeps delivery exactly-once while two instances briefly hold the same token. Connecting also announces ownership with a Redis-issued sequence number (`INCR`), so a device that reconnects elsewhere takes its token with it and the old stream stands down; the sequence is what stops an announcement delayed in flight from evicting a newer stream.

Nodes that don't hold the token stop at a local map lookup and touch Redis not at all, which is what keeps the broadcast affordable.

If Redis is unreachable, a push is answered `503 ServiceUnavailable` rather than dropped, and Fleet's `apns_push_to_pending_hosts` cron retries it. Existing streams keep working.

## Store-and-forward and coalescing

This is the behavior the old timer-driven setup couldn't model, and the main reason the mock exists. Semantics match real APNs:

- **Connected device**: the push is delivered immediately and never stored. APNs doesn't redeliver. If the connection is replaced or drops before the push reaches the wire, it goes back to pending so the device gets it on reconnect. A wake-up genuinely lost mid-flight is recovered by Fleet's `apns_push_to_pending_hosts` cron, which re-pushes hosts with pending commands.
- **Offline device**: the push is kept as the token's single pending push and delivered when the device connects. A newer push overwrites the older one (APNs keeps only the most recent notification per device). Default retention is 24h; an `apns-expiration` header overrides it, with `0` meaning deliver-now-or-discard.
- **Restart**: pending pushes survive, since they live in Redis rather than in the instance that took them.

Expiry is the Redis TTL on the pending key; there is nothing to sweep. Implementation is split between `registry.go` (which tokens an instance holds) and `coordinator.go` (the shared state).

## Device tokens

Simulated clients derive their own tokens, so no coordination with the mock is needed: mdmtest clients send `hex("token" + serial)` with PushMagic `"pushmagic" + serial` in their TokenUpdate (`pkg/mdm/mdmtest/apple.go`). Fleet stores these in `nano_enrollments` like real tokens and pushes to them like real tokens.

The mock accepts any even-length hex token. Real APNs also enforces the 32-byte token length (verified against `api.push.apple.com` with `tools/mdm/apple/apnspush -direct`), but enforcing that would break the derived tokens. This is a deliberate divergence.

## Client library

`pkg/mdm/apnsmock` is the Go client simulated devices use: connect, auto-reconnect with jittered backoff, deliver each push on a channel. Every reconnect backs off, including after a stream that ended cleanly, so an instance restart doesn't turn its share of the fleet into a reconnect storm. The channel is buffered 1 and drops pushes while full, the same coalescing rule as the server, because an MDM wake-up carries no unique data. The client needs no changes to work across instances — reconnects land wherever the load balancer sends them.

## Scale

Target is 300k concurrent connections across three or four instances, 75k–100k each. The token registry is sharded 256 ways to survive the connect stampede at ramp-up, and `pkg/mdm/apnsmock` supports initial jitter to spread it. Measured cost is roughly 8 KB per connection (75k streams: 130 MB heap, 295 MB goroutine stacks, 598 MB total), dominated by goroutine stacks because the SSE handler hijacks the connection rather than blocking. Raise the file descriptor limit (`ulimit -n 1000000`) and set `GOMEMLIMIT` below available memory.

Size Redis for the push rate, not the connection count: each push costs two round trips on the receiving instance and one `GETDEL` on the delivering one, and each announcement fans out to every instance. The instances hold only a handful of Redis connections each — one for the subscription plus a small command pool — so the 65,000-connection cap per node is never in play.

The mock gets its own ElastiCache (`cache.m7g.large`, single node) rather than sharing Fleet's, so a full-fleet wave of ~900k commands doesn't perturb the system under test. Extra nodes would be read replicas, and every operation here is a write to the primary, so they buy failover rather than throughput; node size is the only lever. Tune with `apple_apns_mock_redis_instance_size` / `_count`, and the task with `apple_apns_mock_cpu` / `_memory` (GOMEMLIMIT is derived from the latter).

Debugging aid: `tools/mdm/apple/apnspush -direct` sends raw pushes from a Fleet database to any APNs endpoint (real, sandbox, or this mock) and dumps the raw response. Its `-fake` flag derives mdmtest-style tokens for hosts that aren't in `nano_enrollments`.

## Configuring Fleet to use a custom APNs server

To configure Fleet to use the mock APNs server, set `FLEET_DEV_MDM_APPLE_PUSH_SERVER_URL` to the base URL of the mock server. 

This requires Fleet to be started with `--dev`, and does **not** work alongside `FLEET_DEV_MDM_APPLE_DISABLE_PUSH`.

## Configuring osquery-perf to use custom APNs server

Set the `mdm_apns_url` flag to the mock's base URL. Simulated hosts then check in on the ping channel instead of interval tickers. On macOS with user enrollments enabled each host opens two streams, one for the device channel and one for the user channel — still well below the 8 GB limit on osquery-perf containers at 5k hosts each.

## Load testing with mock APNS server

To spin up a mock APNs server alongside Fleet in a loadtest environment, you need to select `yes` for the "Deploy the mock Apple APNs push server and point Fleet's MDM pushes at it?" option.

This creates the container behind the internal ALB on its own hostname (`fleet-<workspace>-apns-mock.loadtest.fleetdm.com`) and points Fleet at it with `FLEET_DEV_MDM_APPLE_PUSH_SERVER_URL`. Scale it with `apple_apns_mock_instance_count`; it brings up its own Redis alongside.

osquery-perf loadtesting needs the regular MDM knobs, but also now the `--mdm_apns_url=...` set to the base url. With this it should auto initiate sessions and listen for pings to do MDM check-ins.


