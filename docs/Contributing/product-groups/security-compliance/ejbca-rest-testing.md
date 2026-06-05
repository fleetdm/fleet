# EJBCA REST testing

Sets up a local EJBCA Community CA for end-to-end testing of Fleet's REST API
integration (the `ejbca` CA type, distinct from the custom-SCEP-proxy path
covered by [ejbca-scep-testing.md](./ejbca-scep-testing.md)).

This guide is structured to mirror what an EJBCA admin would do in production:
enroll a service certificate for Fleet, bind it to a least-privilege admin
role, then configure Fleet to call EJBCA's REST API over mutual TLS.

Prefer this over the SCEP guide when you want to exercise the REST API path —
the customer scenario for [fleet#30986](https://github.com/fleetdm/fleet/issues/30986).

## Prerequisites

- Docker
- Fleet dev server running locally (with a server private key configured —
  required for encrypting the EJBCA client key at rest)
- `openssl` on `$PATH` for the Fleet server (POC only — used to convert
  EJBCA's BER-encoded P12 export to DER on upload; see "Known POC debt"
  below)
- A macOS host enrolled in your Fleet for the end-to-end test

## Known POC debt

The POC shells out to the `openssl` binary in the P12 decode path because
EJBCA's RA Web emits BER-encoded PKCS#12 bundles and both Go PKCS#12
libraries (`software.sslmate.com/src/go-pkcs12` and
`golang.org/x/crypto/pkcs12`) require strict DER. **This subprocess approach
must not ship to production.** The production implementation
([fleet#30986](https://github.com/fleetdm/fleet/issues/30986)) will replace
it with a pure-Go BER → DER normalizer so Fleet has no runtime dependency
on the openssl binary. Tracked in
`openspec/changes/add-ejbca-rest-ca-poc/research.md` → "Open follow-ups".

## Set up EJBCA in Docker

Same container as the SCEP guide. If you already have it running for SCEP
testing, skip to the next section — the configurations live side by side.

```bash
docker run -d --name ejbca \
  -h ejbca.local \
  -p 8480:8080 -p 8443:8443 \
  -e TLS_SETUP_ENABLED=simple \
  keyfactor/ejbca-ce
```

Wait for the server (~3 minutes):

```bash
until curl -sk -o /dev/null -w "%{http_code}\n" \
    https://localhost:8443/ejbca/publicweb/healthcheck/ejbcahealth | grep -q 200; do
  sleep 10
done
echo "EJBCA up"
```

EJBCA's Admin UI is at `https://localhost:8443/ejbca/adminweb/`. First visit
triggers a client-cert chooser (cancel it) and a TLS interstitial because the
cert is self-signed for `ejbca.local` (click **Advanced → Proceed**).

### Map `ejbca.local` to localhost

The EJBCA container's HTTPS cert is issued for `ejbca.local`, matching the
`-h ejbca.local` flag in `docker run`. Browsers accept it after the
interstitial click-through, but Fleet's REST client does strict TLS
verification (it's mTLS — we can't skip it) and needs the connect-time
hostname to match the cert's SAN. Add a `/etc/hosts` entry so `ejbca.local`
resolves to `127.0.0.1`:

```bash
echo '127.0.0.1 ejbca.local' | sudo tee -a /etc/hosts
```

When configuring Fleet below, use `https://ejbca.local:8443` (not
`https://localhost:8443`) for the **EJBCA REST URL**.

The existing SCEP dev guide sidesteps this by using the HTTP port (8480) —
SCEP doesn't require TLS. The REST integration does (mTLS), so the
hosts-file mapping is the cleanest dev workaround.

## CE caveats

EJBCA Community Edition has two limitations that affect what we can verify
locally:

- **No SCEP RA / no REST auto-create-EE.** End entities must be pre-created
  per CSR subject DN. The customer's production EJBCA (Enterprise) will be
  configured to auto-create on enrollment; CE cannot.
- **No REST endpoint for end-entity management.** Pre-creation has to happen
  via the Admin UI or `bin/ejbca.sh ra addendentity` CLI.

This means our local POC test uses a single pre-created end entity (matching
one host's enrollment). For the customer's auto-create-EE flow, validate
against the 30-day EJBCA Enterprise trial (see openspec
`add-ejbca-rest-ca-poc/research.md`).

## EJBCA-side setup

### 0. Enable REST API protocols

**`keyfactor/ejbca-ce` ships with all REST API protocols DISABLED.** Hitting
`/ejbca/ejbca-rest-api/v1/*` against a default container returns the
unmistakable response:

```
<html><head><title>Error</title></head><body>This service has been disabled.</body></html>
HTTP 403
```

Enable the REST protocol Fleet uses before doing anything else.

Admin UI → top menu **System Configuration** → **Protocol Configuration**
tab (it's a tab on the System Configuration page — *not* under "System
Functions"):

1. Find the **REST Certificate Management** row (resources
   `/ejbca/ejbca-rest-api/v1/ca` and `/ejbca/ejbca-rest-api/v1/certificate`)
   and click **Enable** in its Actions column. The status flips to `Enabled`
   immediately — there's no separate page Save.

That single protocol covers **both** endpoints Fleet calls: `GET /v1/ca/status`
(the connection probe at CA-save time) and `POST /v1/certificate/pkcs10enroll`
(the per-host enrollment) — both `/v1/ca` and `/v1/certificate` live under
REST Certificate Management.

Do **not** enable "REST CA Management" — despite the name, it serves only
`/v1/ca_management` (CA lifecycle operations Fleet never calls), and on
`keyfactor/ejbca-ce` it shows as `Unavailable` (can't be enabled in CE
anyway). Enabling it does nothing for Fleet.

**Heads up — you almost certainly need to restart the container after
enabling.** Clicking Enable flips the config to `Enabled` and it persists,
but on `keyfactor/ejbca-ce` the running REST app keeps serving the
"This service has been disabled" 403 page until it reloads the protocol
config on boot. Toggling the protocol off/on and waiting does not clear it.
Restart the container and re-test:

```bash
docker restart ejbca
# wait for it to come back, then:
curl -sk -o /dev/null -w "%{http_code}\n" \
    https://localhost:8443/ejbca/ejbca-rest-api/v1/certificate   # expect 401, not 403
```

A `401` (auth required — you didn't present a client cert) means the protocol
is live; a `403` with the "disabled" page means it isn't yet. (You can also
sanity-check that unauthenticated requests reach the app at all by hitting an
already-enabled protocol like `/ejbca/ejbcaws/ejbcaws?wsdl`, which returns
200.)

In a real customer EJBCA deployment, the PKI admin would have already
enabled this. For the local POC container you have to do it yourself —
there's no Docker env-var to flip it on at boot.

### 1. Certificate Profile for the Fleet service cert

Admin UI → **CA Functions → Certificate Profiles**:

1. Click **Clone** on the `ENDUSER` row. A dialog appears — enter
   `fleetRESTAdmin` as the **Name of new certificate profile** and click
   **Create from template**.
2. **Edit** the new `fleetRESTAdmin` row.
3. Under **Available Key Algorithms**, ensure RSA is checked.
4. Under **Bit Lengths**, check 2048 and 3072.
5. Under **Extended Key Usages**, set:
   - Use: ✓
   - Critical: optional
   - Add: `Client Authentication`
6. Validity: `1y`
7. **Save**.

### 2. End Entity Profile for the Fleet service cert

Admin UI → **RA Functions → End Entity Profiles**:

1. Type `fleetRESTAdmin` in **Identifier** and click **Add Profile**.
2. Select `fleetRESTAdmin` and click **Edit End Entity Profile**.
3. **Default Certificate Profile** → `fleetRESTAdmin`. **Available Certificate
   Profiles** → highlight `fleetRESTAdmin`.
4. **Default CA** → `ManagementCA`. **Available CAs** → highlight `ManagementCA`.
5. **Save**.

### 3. Enroll Fleet's service certificate

Admin UI → **RA Web** (top-right menu; the cert chooser dialog appears,
cancel it) → **Enroll → Make New Request**:

1. **Certificate Type** → `fleetRESTAdmin`.
2. **Key-pair generation** → `By the CA`.
3. **Key algorithm** → `RSA 2048 bits`. (This selector appears only after you
   pick "By the CA," and defaults to `ECDSA B-163` — you must change it, or
   you'll enroll an ECDSA service cert instead of RSA.)
4. **Subject DN** → CN: `Fleet REST Service`.
5. **Username**: `fleet_rest_service`.
6. **Enrollment code** (+ **Confirm enrollment code**): set a strong password
   (you will use this as the P12 password; Fleet stores neither this password
   nor the P12 itself — re-uploading is fine if you lose it).
7. Click **Download PKCS#12**. There is no separate "Enroll" step — this
   button enrolls the end entity, generates the keypair, and downloads the
   keystore in one action. (The sibling buttons — JKS, BCFKS, PEM — do the
   same for other keystore formats.)

You should now have `fleet_rest_service.p12` in your Downloads folder. Move
it somewhere safe. To confirm the enrollment took, Admin UI → **RA Functions
→ Search End Entities** → search status `All`: `fleet_rest_service` should
appear with status `GENERATED`.

### 4. Admin Role + Access Rules

Admin UI → **System Functions → Roles and Access Rules → Add Role**:

1. **Role Name** → `Fleet Service Account`. Click **Add**.
2. Click **Edit Access Rules** on the new role:
   - **Role Template** → `RA Administrators`
   - **Authorized CAs** → check only the CA(s) that should issue device
     certs (e.g., `ManagementCA` for the POC)
   - **Authorized End Entity Profiles** → check only the profile(s) Fleet
     will enroll against
3. **Save**.

### 5. Bind the Fleet service cert to the role

On the `Fleet Service Account` role page, click **Members**:

1. **Match with** → `X509:CN, Common name`.
2. **CA** → `ManagementCA` (the CA that just issued the Fleet service cert).
3. **Match value** → `Fleet REST Service` (exact CN from step 3 above).
4. Click **Add**.

### 6. Pre-create one end entity for the device test

For each CSR username Fleet will use, EJBCA-CE needs a pre-created end
entity. The username Fleet sends is the expanded `username_template` — for
testing, use one host's hardware serial.

Get the test host's serial from Fleet (Hosts page) then:

```bash
# Replace SERIAL with your test host's hardware serial
SERIAL='YOUR-HOST-SERIAL'

docker exec ejbca bin/ejbca.sh ra addendentity \
    --username "${SERIAL}" \
    --dn "CN=${SERIAL}" \
    --caname ManagementCA \
    --type 1 \
    --token USERGENERATED \
    --password "anyvalue" \
    --certprofile fleetRESTAdmin \
    --eeprofile fleetRESTAdmin
```

The `--password` value doesn't have to match anything Fleet sends — Fleet
generates a per-issuance random password (see openspec REQ-CA-EJBCA-7). EJBCA
auto-create-EE-enabled deployments don't need this step.

### 7. Export the trust CA

Optional but recommended for the POC since EJBCA's HTTPS cert is signed by
the Management CA, which Fleet's system root store doesn't trust:

```bash
docker exec ejbca cat /opt/keyfactor/ejbca/wildfly/standalone/configuration/keystore/truststore.pem
```

Or via Admin UI → **RA Web → CA Certificates and CRLs → Download** the
`ManagementCA` as PEM.

Save the resulting PEM contents — you'll paste them into Fleet's "Trust CA
bundle" field.

## Configure Fleet

### Via the UI

1. Log into Fleet as a global admin.
2. **Settings → Integrations → Certificate authorities → Add CA**.
3. Pick **EJBCA** from the dropdown.
4. Fill in:
   - **Name**: `Test_EJBCA` (this becomes `$FLEET_VAR_EJBCA_DATA_Test_EJBCA`
     in profiles)
   - **EJBCA REST URL**: `https://ejbca.local:8443` (requires the
     `/etc/hosts` mapping from the Docker section above)
   - **Client certificate (.p12)**: upload `fleet_rest_service.p12`
   - **PKCS#12 password**: the password you set in step 3
   - **Trust CA bundle**: paste the Management CA PEM from step 7
   - **EJBCA Certificate Authority name**: `ManagementCA`
   - **EJBCA Certificate Profile name**: `fleetRESTAdmin`
   - **EJBCA End Entity Profile name**: `fleetRESTAdmin`
   - **Username template**: `$FLEET_VAR_HOST_HARDWARE_SERIAL`
   - **UPN** (optional): leave empty for the basic test
5. Click **Save**. Fleet will probe EJBCA via `GET /v1/ca/status` over mTLS
   and reject the save if anything's off.

### Via the API

```bash
FLEET_TOKEN="$(awk '/^  qa:/,/^  [a-z]/' ~/.fleet/config | awk '/token:/ {print $2}')"

P12_BASE64="$(base64 < fleet_rest_service.p12)"
TRUST_BUNDLE="$(awk '{printf "%s\\n", $0}' management_ca.pem)"

curl -sk -X POST \
    -H "Authorization: Bearer $FLEET_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"ejbca":{
          "name":"Test_EJBCA",
          "url":"https://ejbca.local:8443",
          "client_p12":"'"$P12_BASE64"'",
          "client_p12_password":"<P12 password>",
          "trust_ca_bundle":"'"$TRUST_BUNDLE"'",
          "certificate_authority_name_ejbca":"ManagementCA",
          "certificate_profile_name":"fleetRESTAdmin",
          "end_entity_profile_name":"fleetRESTAdmin",
          "username_template":"$FLEET_VAR_HOST_HARDWARE_SERIAL",
          "certificate_user_principal_names":null
        }}' \
    https://localhost:8080/api/latest/fleet/certificate_authorities
```

Save the returned `id` — you'll need it if you want to PATCH or DELETE later.

## End-to-end test

### 1. Create a configuration profile that references the EJBCA CA

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>Password</key>
      <string>$FLEET_VAR_EJBCA_PASSWORD_Test_EJBCA</string>
      <key>PayloadCertificateFileName</key>
      <string>ejbca-test.p12</string>
      <key>PayloadContent</key>
      <data>$FLEET_VAR_EJBCA_DATA_Test_EJBCA</data>
      <key>PayloadDescription</key>
      <string>EJBCA-issued client cert (POC)</string>
      <key>PayloadDisplayName</key>
      <string>EJBCA Test Cert</string>
      <key>PayloadIdentifier</key>
      <string>com.fleetdm.ejbca.test.cert</string>
      <key>PayloadType</key>
      <string>com.apple.security.pkcs12</string>
      <key>PayloadUUID</key>
      <string>00000000-0000-0000-0000-000000000001</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>EJBCA Test Profile</string>
  <key>PayloadIdentifier</key>
  <string>com.fleetdm.ejbca.test</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>00000000-0000-0000-0000-000000000002</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Save as `ejbca-test.mobileconfig` and upload via **Controls → OS settings →
Custom settings → Add profile** in Fleet's UI (or via `fleetctl apply`).

### 2. Trigger profile delivery

The next time the test host checks in (or run **Refetch** on the host
details page), Fleet's MDM profile processor will:

1. Detect the `$FLEET_VAR_EJBCA_*` variables in the profile XML.
2. Expand `$FLEET_VAR_HOST_HARDWARE_SERIAL` in the username template to the
   actual host's serial.
3. Call EJBCA's `pkcs10enroll` REST endpoint over mTLS, generating a
   per-host RSA 2048 keypair + CSR.
4. Receive the issued cert (base64 DER), wrap it with the private key in a
   PKCS#12, and substitute the base64-encoded PFX into the profile XML.
5. Push the profile to the host via APNs.

### 3. Verify the cert installed

On the test host:

```bash
security find-certificate -a -p -c "$SERIAL" /Library/Keychains/System.keychain
```

You should see a PEM-encoded certificate with Subject CN = your host's
hardware serial, issued by EJBCA's Management CA, valid for 1 year.

### 4. Verify in EJBCA

Admin UI → **RA Web → Search End Entities** → enter the host's serial as
the username. You should see:

- Status: `GENERATED` (cert was issued)
- Last Cert: the cert Fleet just enrolled

In Admin UI → **RA Web → Search Certificates** you can also see the issued
cert listed.

## Gotchas

- **REST API protocols are disabled by default in `keyfactor/ejbca-ce`.**
  Symptom: `<html><body>This service has been disabled.</body></html>` with
  HTTP 403 on any `/ejbca/ejbca-rest-api/v1/*` URL. Fix: enable **REST
  Certificate Management** (not "REST CA Management") in Admin UI → System
  Configuration → Protocol Configuration tab (covered in the **Enable REST
  API protocols** section above). If the 403 persists after enabling,
  restart the container — the REST app re-reads protocol config on boot.
  The TLS/mTLS layer succeeds before this check, so the failure mode from
  Fleet's side is a silent EOF mid-request rather than a clean error.
- **CE-only: end-entity status is consumed on use.** After successful
  issuance the EE flips from `NEW` (10) to `GENERATED` (40). Re-enrolling
  for the same host (e.g., to test rotation) requires resetting:
  ```bash
  docker exec ejbca bin/ejbca.sh ra setendentitystatus --username "${SERIAL}" -S 10
  docker exec ejbca bin/ejbca.sh ra setpwd --username "${SERIAL}" --password "anyvalue"
  ```
  Customer's EJBCA Enterprise with auto-create-EE doesn't have this
  problem — every new enrollment creates a fresh EE record.
- **"Allow Extension Override" on the Certificate Profile.** Required if
  you set a UPN in Fleet's CA config. Without it EJBCA rebuilds the SAN
  from its typed fields and drops the otherName UPN. Same gotcha as the
  existing SCEP guide.
- **Self-signed TLS at port 8443.** Fleet's REST client requires HTTPS
  (mTLS), so we can't use the HTTP-port shortcut from the SCEP guide.
  You must provide the Management CA in the trust bundle field, otherwise
  the connection probe fails at save time with a TLS verify error.
- **The P12 password is one-shot.** Fleet uses it once during decode and
  discards it. To rotate the service cert, use the standard edit modal
  in Fleet — re-upload a new P12 and supply its password; the backend
  re-validates against EJBCA before writing the new material.
- **No managed-cert tracking row for EJBCA in the POC.** The Fleet UI's
  host certificates page may not show the EJBCA-issued cert in its
  "managed by Fleet" indicator. This is a known POC gap noted in the
  spec — the production implementation
  ([fleet#30986](https://github.com/fleetdm/fleet/issues/30986)) adds the
  required `CAConfigEJBCA` enum value and tracking write.

## Cleanup

```bash
docker rm -f ejbca

# Remove the Fleet CA via API
curl -sk -X DELETE -H "Authorization: Bearer $FLEET_TOKEN" \
    https://localhost:8080/api/latest/fleet/certificate_authorities/<CA_ID>

# Or via the UI: Settings → Integrations → Certificate authorities →
# row menu → Delete
```

## See also

- [ejbca-scep-testing.md](./ejbca-scep-testing.md) — sibling guide for the
  custom-SCEP-proxy integration path.
- OpenSpec change `add-ejbca-rest-ca-poc` for the design rationale, customer
  questions, and deferred follow-ups (`openspec/changes/add-ejbca-rest-ca-poc/`).
- [EJBCA REST Interface docs](https://docs.keyfactor.com/ejbca/latest/ejbca-rest-interface).
