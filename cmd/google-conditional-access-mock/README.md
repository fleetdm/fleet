# google-conditional-access-mock

A mock of fleetdm.com's Google conditional access API and of the parts of Google's Cloud Identity API the Fleet server calls. With it, you can run the whole Google conditional access flow from a phone or a browser without a Google Workspace tenant or Fleet's allowlisted OAuth client. Tests can also mount the `mock` package on `httptest.NewServer` to drive the real HTTP clients.

## Run

```sh
go run ./cmd/google-conditional-access-mock --listen :9998 --insecure
```

| Flag | Default | Description |
| --- | --- | --- |
| `--listen` | `:9998` | host:port to listen on |
| `--fleet-server-url` | registered `Origin` | Where the callback sends the inbound register call. By default, the mock uses the `Origin` the Fleet server registered with, as fleetdm.com does. |
| `--public-url` | from each request | The mock's externally reachable URL, used for the OAuth redirect URI and for `token_uri` in generated keys. Set it when the mock is behind a tunnel. |
| `--google-client-id` / `--google-client-secret` | — | A real Google Web OAuth client. Setting both turns on forwarding mode. The secret can also come from `GOOGLE_CLIENT_SECRET`. |
| `--google-partner-id` | — | Fleet's partner ID, sent on the iOS lookup in forwarding mode |
| `--customers` | accept any | Comma-separated Google customer IDs that `devices.list` accepts, in addition to registered ones. Set this to exercise the 403 from the delegation check. |
| `--latency` / `--error-rate` | `0` | Simulated latency and 429/500 failures on server-to-server routes |
| `--insecure` | `false` | Skip TLS verification when calling the Fleet server, for the self-signed dev certificate |

## Two-terminal recipe

`google_conditional_access.proxy_uri` points the Fleet server's fleetdm.com calls at the mock, and `FLEET_DEV_GOOGLE_CLOUD_IDENTITY_ENDPOINT` does the same for its Cloud Identity calls. The endpoint override only takes effect when Fleet runs with `--dev`.

1. Terminal 1, start the mock:

   ```sh
   go run ./cmd/google-conditional-access-mock --listen :9998 --insecure
   ```

2. Terminal 2, start Fleet pointed at the mock. Google conditional access stores the service account key encrypted, so the server private key must be set.

   ```sh
   FLEET_GOOGLE_CONDITIONAL_ACCESS_PROXY_URI=http://localhost:9998 \
   FLEET_DEV_GOOGLE_CLOUD_IDENTITY_ENDPOINT=http://localhost:9998/ \
   FLEET_SERVER_PRIVATE_KEY=<32+ characters> \
   ./build/fleet serve --dev ...
   ```

3. Connect in the UI (**Settings > Integrations > Conditional access > Google**) or through the API. Any customer ID and admin email work. For the key, use one from the mock. The mock doesn't verify signatures, but the Fleet server must be able to sign with the key, and the key's `token_uri` points token requests at the mock.

   ```sh
   curl -s http://localhost:9998/mock/service-account-key > /tmp/key.json
   fleetctl api -X PATCH /api/v1/fleet/config -d "$(jq -n --slurpfile key /tmp/key.json \
     '{integrations: {google_conditional_access: [{customer_id: "C01234abc", admin_email: "fleet-admin@example.com", api_key_json: $key[0]}]}}')"
   ```

4. Open the Self-service web clip on a company-owned iPhone or iPad, or the Fleet app on a company-owned Android device. To skip the phone on iOS, call the status route and open the returned `sign_in_url` in a browser:

   ```sh
   fleetctl api /api/v1/fleet/device/<host uuid>/conditional_access/google
   ```

   The mock's sign-in page continues on its own and lands on the done, not-allowed, or error page, depending on the Fleet server's answer.

5. Check the result:

   ```sh
   curl -s http://localhost:9998/mock/state | jq
   ```

From a phone, both the mock and the Fleet server must be reachable from the device and from each other. Put the mock behind a tunnel, pass the tunnel URL as `--public-url`, and use that URL as `proxy_uri`.

## Fake and forwarding modes

In **fake mode** (the default), the mock signs the end user in itself. Its sign-in page auto-submits to the callback with the `state` and `user@example.com`. The Google device ID is `mock-` plus the first 12 characters of the host UUID, and the device user is `devices/<id>/deviceUsers/<id>`. The Android register route returns the same identifiers.

In **forwarding mode**, sign-in redirects to Google with the real client and the `cloud-identity.devices.lookup` scope. The callback exchanges the code, calls `deviceUsers.lookup` with `userId=me`, `partner`, and `iosDeviceId`, and calls `devices.get` to read the device ID. Android uses `androidId` instead. Register `<public URL>/api/v1/google-conditional-access/callback` as a redirect URI on the client. In forwarding mode, keep the Fleet server on the real Cloud Identity endpoint with a real delegated key.

## Endpoints

### fleetdm.com

| Endpoint | Caller | Behavior |
| --- | --- | --- |
| `POST /api/v1/google-conditional-access` | Fleet server | Body `{googleCustomerId}`, `Origin` header required (`400` without it). Returns `{fleet_server_secret, oauth_client_id}`, where the client ID is `mock-client-id` in fake mode. Returns `409` if the customer is already registered. There's no consent step, so every registration is complete. |
| `DELETE /api/v1/google-conditional-access` | Fleet server | Query `googleCustomerId` and `fleetServerSecret`. Returns `200`, or `404` if no registration matches. |
| `GET /api/v1/google-conditional-access/sign-in` | Web clip | Query `googleCustomerId`, `token`, `hostUuid`, and `platform` (`ios` or `ipados`). Returns `404` for an unknown customer. The `state` is a single-use server-side nonce that expires after 10 minutes. |
| `GET /api/v1/google-conditional-access/callback` | Browser | Calls `POST {fleet}/api/v1/fleet/google_conditional_access/register` with `Authorization: Bearer <secret>` and `{token, google_device_id, google_device_user_name, google_email}`. A Fleet `200` redirects to `/google-conditional-access/done`, a `403` or `409` to `/not-allowed`, and anything else to `/error`. |
| `POST /api/v1/google-conditional-access/device/register` | Fleet server (Android) | Body `{googleCustomerId, fleetServerSecret, serverAuthCode, androidId, hostUuid}`. Returns `{google_device_id, google_device_user_name, google_email}`. Returns `404` for an unknown customer or secret, and `422` when the lookup finds no record. |

### Google Cloud Identity

The `/v1` routes require a token from `POST /token`.

| Endpoint | Behavior |
| --- | --- |
| `POST /token` | JWT bearer grant. The signature isn't verified, and the assertion's `sub` is recorded as the admin. |
| `GET /v1/devices?customer=customers/{id}` | `200` with one device for a known customer, `403` otherwise |
| `GET /v1/devices/{deviceId}` | `200` for a signed-in device, `404` for an unknown or stale one |
| `PATCH /v1/devices/{d}/deviceUsers/{u}/clientStates/{state}?updateMask=` | Records `managed` and `customId` (only the fields in the mask) and the admin. Returns a completed operation, or `404` for an unknown or stale device. |

### Mock controls

| Endpoint | Behavior |
| --- | --- |
| `GET /mock/state` | Registrations (with secrets), devices, and client states as JSON |
| `GET /mock/service-account-key` | A new service account key whose `token_uri` is the mock |
| `POST /mock/devices/{deviceId}/stale` | `devices.get` and `clientStates.patch` return `404` for this device, as after a factory reset. The device's next sign-in revives it. |
| `POST /mock/android/{androidId}/no-record` | The Android register route returns `422` for this `androidId` |
| `GET /mock/health` | Liveness |

State is in memory. To register a customer again, restart the mock or delete the registration with the secret from `/mock/state`.
