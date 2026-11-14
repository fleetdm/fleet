# Google conditional access mock

Stands in for fleetdm.com's Google conditional access relay, Google's Cloud Identity API,
or both, so the Google conditional access flow runs without a Workspace tenant or an
allowlisted OAuth client. State is in memory and is lost on restart. The handlers live in
`server/service/conditional_access_google/mockserver` for tests.

## Pick what to mock

The Fleet server points at each surface separately, so either can be mocked alone.

| `--mock` | Mocked | Fleet server settings | Sign-in |
|---|---|---|---|
| `fleetdm,google` (default) | both | proxy URI and endpoint override | fake, or real Google with `--google-client-id` |
| `fleetdm` | the relay | proxy URI only | real Google: pass `--google-client-id`, `--google-client-secret` and `--partner-id` |
| `google` | Cloud Identity API | endpoint override only | done by the real fleetdm.com relay |

- **Proxy URI**: `FLEET_GOOGLE_CONDITIONAL_ACCESS_PROXY_URI=http://localhost:9090` (config
  `google_conditional_access.proxy_uri`). A plain HTTP URI needs `--dev`.
- **Endpoint override**: `FLEET_DEV_GOOGLE_CLOUD_IDENTITY_ENDPOINT=http://localhost:9090/`,
  read only with `--dev`. It also points the delegated token exchange at the mock's `/token`.

With `--mock fleetdm` and no OAuth client, the fake names don't exist in real Google, so
marking them fails; the mock warns at startup. With `--mock google`, a client state patch
on a name the mock doesn't know creates it, so names that real Google returned can be
marked.

## Two-terminal recipe

```sh
# Terminal 1: a service account key to upload in Fleet, then the mock
go run ./cmd/google-conditional-access-mock --new-key > /tmp/mock-key.json
go run ./cmd/google-conditional-access-mock --listen :9090

# Terminal 2: Fleet
FLEET_GOOGLE_CONDITIONAL_ACCESS_PROXY_URI=http://localhost:9090 \
FLEET_DEV_GOOGLE_CLOUD_IDENTITY_ENDPOINT=http://localhost:9090/ \
  ./build/fleet serve --dev --dev_license
```

In Fleet, go to **Settings > Integrations > Conditional access > Google** and connect with
customer ID `C01mock00` (or one passed in `--customer-ids`), any admin email, and the key
from `--new-key`. Any RSA key works: the mock doesn't check the signature. A customer
ID the mock doesn't know fails the delegation check, which is how to test that error.

Then, for a company-owned (`On (automatic)`) iOS or iPadOS host:

- **Phone**: open Self-service in the web clip. It redirects to the mock's sign-in page,
  which continues by itself after two seconds and lands on "You're done! 🎉". The phone
  must reach both Fleet and the mock: run the mock with `--public-url
  http://<your LAN IP>:9090` (and the proxy URI to match), or behind a tunnel.
- **Browser**: get the sign-in URL from the status route and open it:
  `curl $FLEET_URL/api/latest/fleet/device/<host UUID>/conditional_access/google`.

For a company-owned Android host there is no sign-in: Fleet's sync cron looks for the Google
record whose `deviceId` is the host's Android Management API device ID
(`android_devices.device_id`). The mock has no such record until it is seeded, which stands
in for the managed Google account's first check-in:

```sh
curl -X POST localhost:9090/_mock/google/devices -d '{
  "customer_id": "C01mock00", "device_id": "<android_devices.device_id>",
  "device_type": "android", "emails": ["user@example.com"]}'
```

The cron registers the host on its next run (every five minutes; `fleetctl trigger --name
google_conditional_access` runs it now). A host with no record is tried on three runs, then
hourly. Seeding the same `device_id` twice gives the host a second, leftover record.

`curl localhost:9090/_mock/state` shows the registrations, the device records, and each
DeviceUser's client state (`MANAGED` and the host UUID as `customId` once the host is
registered).

When Fleet's server URL isn't reachable from the mock (the mock calls Fleet back at the URL
Fleet registered with), pass `--fleet-url`. Add `--fleet-insecure` for a self-signed
certificate.

A customer stays registered to the server URL that registered it, as on fleetdm.com: after
changing Fleet's server URL, connecting again is a 409 until `POST /_mock/reset` or a
restart of the mock.

## Fake sign-ins

The sign-in page continues as signed in after two seconds and also links to the other
outcomes: pass `--auto-continue 0` to have it wait for a click, or a longer delay. The
`code` in the callback URL picks the outcome:

| Code | Outcome |
|---|---|
| `mock-signed-in` (default) | the DeviceUser of `--email` on the host's record |
| `mock-not-found` | no names: the user hasn't signed in to Gmail |
| `mock-refused` | Google refuses the code: the error page |

A fake lookup finds the iOS records whose `deviceId` is the host UUID, and creates one when
there is none. A seeded record without the `--email` user is a lookup
with no names. `--device-records 2` gives an iOS host a second, leftover record, as after a
wipe and re-enrollment.

## Real Google sign-ins

Give the mock Fleet's Web OAuth client to run the relay as fleetdm.com will. The
environment variables keep the secret out of `ps` and the shell history:

```sh
GOOGLE_CA_MOCK_CLIENT_ID=<client ID> \
GOOGLE_CA_MOCK_CLIENT_SECRET=<client secret> \
GOOGLE_CA_MOCK_PARTNER_ID=<partner ID> \
  go run ./cmd/google-conditional-access-mock --mock fleetdm --public-url https://<tunnel host>
```

The flags `--google-client-id`, `--google-client-secret` and `--partner-id` do the same.
The iOS sign-in then redirects to Google with the `openid`, `email` and lookup scopes. The
callback exchanges the code with the client secret, reads the email from the ID token, runs
the lookup with `userId=me`, `partner` and `iosDeviceId` (the host UUID), and calls Fleet's
register route. Android hosts don't use the relay.

Which surfaces to mock:

- `--mock fleetdm` is the closest to production. The sign-in and the lookup go to real
  Google, and Fleet marks the devices in the real tenant with the customer's service
  account. Only the relay is a stand-in.
- `--mock fleetdm,google` takes real names from Google's lookup and records the marks in the
  mock, to watch them in `/_mock/state` without changing the tenant's client states.

Setup in Google Cloud for the Web client:

- Add `<public URL>/api/v1/google-conditional-access/callback` to its authorized redirect
  URIs. Google accepts plain HTTP only on `localhost`, which a phone can't reach, so for a
  device run the mock behind an HTTPS tunnel, pass the tunnel URL as `--public-url`, and
  register that URL. Point Fleet's proxy URI at the same URL or at `localhost`.
- While the OAuth consent screen is in testing mode, the account that signs in must be one
  of its test users.
- The lookup only finds a device whose user signed in to Gmail with Fleet's `partner` and
  `iosDeviceId` in its managed configuration. Otherwise the result is the "Sign in to
  Gmail" page.

Where it differs from fleetdm.com, without changing what Google or Fleet sees: the sign-in
state is kept in the mock's memory instead of in the OAuth `state`, the mock serves the
result pages itself, and the ID token's signature isn't checked, since it comes straight
from Google's token endpoint.

## Admin routes

These answer on `localhost` only, and are a 403 through a tunnel or from another machine:
the state includes the secret that authenticates the mock's calls to the Fleet server.

| Route | Body | Effect |
|---|---|---|
| `GET /_mock/state` | | dump the state |
| `POST /_mock/reset` | | forget everything |
| `POST /_mock/google/customers` | `{"customer_id": "C02abc"}` | add a known customer |
| `POST /_mock/google/devices` | `{"customer_id", "device_id", "device_type", "emails": []}` | seed a record with one DeviceUser per distinct email; returns it. `device_id` is the host UUID on iOS and the Android Management API device ID on Android |
| `POST /_mock/google/devices/{id}/state` | `{"state": "active" \| "deleted" \| "left"}` | `deleted` makes calls on it a 404, `left` a 403, for the stale checks |
| `POST /_mock/google/token-error` | `{"error": "unauthorized_client"}` | refuse delegated tokens; `""` clears |

`--latency` and `--error-rate` slow down or fail the relay and Cloud Identity calls (429 or
500), not the admin routes. The delegated token exchange only fails with a 503: Fleet reads
a 4xx there as a broken delegation. The state keeps the latest 1,000 Cloud Identity calls.
