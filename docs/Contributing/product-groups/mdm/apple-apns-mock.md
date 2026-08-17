# Mock APNs push server

Fleet disables MDM push notifications during load tests today, so every simulated host checks in on a timer. That hides how Fleet behaves with realistic push-driven check-ins. `cmd/apple-apns-mock` is an in-memory stand-in for Apple's push notification service (APNs) that closes this gap: Fleet pushes to it exactly as it pushes to `api.push.apple.com`, and simulated devices receive wake-ups instead of polling.

## Architecture

```
+----------------+  POST /3/device/<token>   +---------------------+
|  Fleet server  | ------------------------> |   apple-apns-mock   |
|    (buford)    |   {"mdm":"<PushMagic>"}   |                     |
+----------------+                           | one pending push    |
                                             | per token, 24h TTL  |
                                             +---------------------+
                                               |     |         |
                                               | GET /events?token=
                                               | (SSE, one per device)
                                               v     v         v
                                            osquery-perf simulated devices
                                              (pkg/mdm/apnsmock client)
```

- Fleet's push path is unchanged: `server/mdm/nanomdm/push/buford` sends `POST /3/device/<token>` with body `{"mdm":"<PushMagic>"}` and no `apns-*` headers. Only the base URL will change (#31311).
- Simulated devices hold a long-lived server-sent events (SSE) connection, the mock's stand-in for a real device's persistent APNs courier connection. Each `event: ping` should trigger an MDM check-in.
- Everything is in memory. One binary, no dependencies.

The wire contract (request and response shapes, header semantics, error bodies) is documented in [`cmd/apple-apns-mock/README.md`](../../../../cmd/apple-apns-mock/README.md) and on the handlers in `cmd/apple-apns-mock/handlers.go`. `TestE2EBufordCompatibility` pins the contract against the actual buford client Fleet uses in production.

## Store-and-forward and coalescing

This is the behavior the old timer-driven setup couldn't model, and the main reason the mock exists. Semantics match real APNs:

- **Connected device**: the push is delivered immediately and never stored. APNs doesn't redeliver. If the connection is replaced or drops before the push reaches the wire, it goes back to pending so the device gets it on reconnect. A wake-up genuinely lost mid-flight is recovered by Fleet's `apns_push_to_pending_hosts` cron, which re-pushes hosts with pending commands.
- **Offline device**: the push is kept as the token's single pending push and delivered when the device connects. A newer push overwrites the older one (APNs keeps only the most recent notification per device). Default retention is 24h; an `apns-expiration` header overrides it, with `0` meaning deliver-now-or-discard.
- **Restart**: pending pushes are lost. Real APNs makes no delivery guarantee either.

Expiry is lazy (checked on connect and push) plus a periodic sweep for tokens that never reconnect. Implementation and invariants are documented on the `store` type in `cmd/apple-apns-mock/store.go`.

## Device tokens

Simulated clients derive their own tokens, so no coordination with the mock is needed: mdmtest clients send `hex("token" + serial)` with PushMagic `"pushmagic" + serial` in their TokenUpdate (`pkg/mdm/mdmtest/apple.go`). Fleet stores these in `nano_enrollments` like real tokens and pushes to them like real tokens.

The mock accepts any even-length hex token. Real APNs also enforces the 32-byte token length (verified against `api.push.apple.com` with `tools/mdm/apple/apnspush -direct`), but enforcing that would break the derived tokens. This is a deliberate divergence.

## Client library

`pkg/mdm/apnsmock` is the Go client simulated devices use: connect, auto-reconnect with jittered backoff, deliver each push on a channel. Every reconnect backs off, including after a stream that ended cleanly, so a mock restart doesn't turn 300k agents into a reconnect storm. The channel is buffered 1 and drops pushes while full, the same coalescing rule as the server, because an MDM wake-up carries no unique data.

## Scale

Target is 300k concurrent connections. The token registry is sharded 256 ways to survive the connect stampede at ramp-up, and `pkg/mdm/apnsmock` supports initial jitter to spread it. Plan roughly 16 KB of memory per connection, dominated by `net/http` (two goroutines plus buffers per connection), so 8 GB or more of RAM at 300k. Raise the file descriptor limit (`ulimit -n 1000000`) and set `GOMEMLIMIT` below available memory.

Debugging aid: `tools/mdm/apple/apnspush -direct` sends raw pushes from a Fleet database to any APNs endpoint (real, sandbox, or this mock) and dumps the raw response. Its `-fake` flag derives mdmtest-style tokens for hosts that aren't in `nano_enrollments`.

## Configuring Fleet to use a custom APNs server

To configure Fleet to use the mock APNs server, set `FLEET_DEV_MDM_APPLE_PUSH_SERVER_URL` to the base URL of the mock server. 

This requires Fleet to be started with `--dev`, and does **not** work alongside `FLEET_DEV_MDM_APPLE_DISABLE_PUSH`.

## Configuring osquery-perf to use custom APNs server

To run osquery-perf with MDM enabled, it is required to set the `mdm_apns_url` flag. osquery-perf runs on a ping channel rather than interval tickers.

For macOS if user enrollments is enabled, it will start two sessions against the APNs server, one for the device channel and one for the user channel.

It should still keep us way below the 8GB limit on osquery-perf containers, even at 5k each.


## Terraform and load test (#31314)

To be written when the infrastructure lands.
