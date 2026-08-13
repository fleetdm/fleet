# apple-apns-mock

A mock of Apple's push notification service (APNs) for load testing Fleet's Apple MDM. Fleet sends it the same push requests it sends to `api.push.apple.com`, and simulated devices receive them over server-sent events (SSE). Built for [#30816](https://github.com/fleetdm/fleet/issues/30816).

See [the design doc](../../docs/Contributing/product-groups/mdm/apple-apns-mock.md) for how it fits into load testing.

### Relevant documentation
- [Sending notifications to APNs](https://developer.apple.com/documentation/usernotifications/sending-notification-requests-to-apns)
- [Handling notification responses from APNs](https://developer.apple.com/documentation/usernotifications/handling-notification-responses-from-apns)

## Run

```sh
go run ./cmd/apple-apns-mock --listen :8378
```

| Flag | Default | Description |
| --- | --- | --- |
| `--listen` | `:8378` | host:port to listen on |
| `--default-ttl` | `24h` | how long a push to an offline device is kept when the request has no `apns-expiration` header |
| `--keep-alive` | `30s` | SSE keep-alive comment interval. `0` disables. |
| `--write-timeout` | `10s` | deadline for a single SSE write. A device that stops reading is disconnected instead of pinning its token. `0` disables. |
| `--sweep-interval` | `10m` | how often expired pending pushes are dropped |
| `--debug` | `false` | debug logging |

## Endpoints

| Endpoint | Caller | Description |
| --- | --- | --- |
| `POST /3/device/{token}` | Fleet server | Accept a push. Same path shape as real APNs. |
| `GET /events?token=<hex>` | Simulated devices | Long-lived SSE stream of pushes for one device token. |
| `GET /healthz` | Infra | Liveness check. |
| `GET /stats` | Operator | Counters as JSON: connected clients, delivered, stored, coalesced, expired. |

## Push requests

The mock accepts exactly what Fleet's buford client (`server/mdm/nanomdm/push/buford`) sends today: a POST with the JSON body `{"mdm":"<PushMagic>"}` and no `apns-*` headers. `TestE2EBufordCompatibility` in `e2e_test.go` drives the mock through that client to keep this true. The mock also honors these [request headers](https://developer.apple.com/documentation/usernotifications/sending-notification-requests-to-apns#Send-a-POST-request-to-APNs) when present (parsing lives in `parsePushHeaders`, `handlers.go`):

| Header | Behavior |
| --- | --- |
| `apns-id` | Echoed in the response. Generated if absent. Not a UUID: `400 BadMessageId`. |
| `apns-push-type` | `mdm` or absent accepted. Anything else: `400 InvalidPushType`. |
| `apns-expiration` | Unix seconds. `0` or past: deliver now or discard. Absent: `--default-ttl`. |

[Responses](https://developer.apple.com/documentation/usernotifications/handling-notification-responses-from-apns) match real APNs. Success is `200` with an `apns-id` header and empty body. Errors carry a JSON body and the `apns-id` header:

```
HTTP/1.1 400 Bad Request
Apns-Id: 4FCEA7C9-78CC-0A03-2902-3473E54F9ED4

{"reason":"BadDeviceToken"}
```

All returned errors live in [errors.go](errors.go) and match the real APNs error codes.

## Behavior

Store-and-forward lives in `store.go`, with full semantics on the `store` type and its methods. The short version:

- One pending push per device token. A push to an offline device is stored until the device connects or the push expires. A newer push overwrites the pending one, matching APNs coalescing.
- A push to a connected device is delivered immediately and never stored. APNs doesn't redeliver.
- The newest connection for a token wins. An older connection for the same token is closed, like a real device reconnecting. A push that was queued for the old connection but never written to the wire goes back to pending, so the reconnecting device gets it.
- A device that stops reading is disconnected after `--write-timeout` and its pending push is kept, rather than holding the token open and swallowing later pushes.
- Pushes live in memory only. A restart drops them. Real APNs makes no delivery guarantee either.

## Simulated devices

Devices subscribe with `GET /events?token=<hex>` and hold the stream open. Each push arrives as:

```
event: ping
data: {"mdm":"pushmagicABC123"}
```

Keep-alive comment lines (`: keepalive`) flow on the `--keep-alive` interval and should be ignored. A payload containing a newline is split across several `data:` lines, per SSE, and the client rejoins them with `\n`. Use the Go client in [`pkg/mdm/apnsmock`](../../pkg/mdm/apnsmock) instead of hand-rolling this.

Smoke test with curl:

```sh
curl -N 'http://localhost:8378/events?token=746f6b656e414243' &
curl -i -X POST http://localhost:8378/3/device/746f6b656e414243 \
  -H 'Content-Type: application/json' -d '{"mdm":"pushmagicABC"}'
```

To push to a device enrolled in a local Fleet instance, or to compare against real APNs responses, use `tools/mdm/apple/apnspush -direct`.

## Differences from real APNs

- Plain HTTP, no client certificate check. Fleet's push client works over HTTP/1.1 and doesn't require HTTP/2 from the server. As we can't validate against the Apple APNs certificate.
- Any even-length hex token is accepted. Real APNs also enforces the 32-byte token length, but mdmtest and osquery-perf derive variable-length tokens (`hex("token" + serial)`).
- Not all known error codes are exercised. 

## Running at scale

Plan on **~8 KB per connection**, so roughly 2.5 GB for 300k. Measured at 75k live streams on an M-series Mac: 130 MB heap, 295 MB goroutine stacks, 598 MB total. Raise the file descriptor limit to clear the connection count (`ulimit -n 1000000`, and on macOS `kern.maxfilesperproc` too, which caps `ulimit` and defaults to 92160). Set `GOMEMLIMIT` below available memory to keep the garbage collector ahead of the ramp.

`GET /memstats` reports what the Go runtime is actually using, and `?gc=1` forces a collection first so the heap figure is live data. Prefer it to RSS: on macOS, pages the runtime has already released still count against the process, which overstated a 40k-connection run by 3x.

Most of the remaining cost is goroutine stacks — one goroutine per stream, ~4 KB each. The `net/http` per-connection buffers that would otherwise dominate (4 KB read, 4 KB write, 2 KB chunking, none of them tunable through `http.Server`) are not in the total, because the SSE handler hijacks the connection and returns instead of blocking; see `eventsSSEHandler`. `tools/apns-loadgen` is the harness these numbers come from.
