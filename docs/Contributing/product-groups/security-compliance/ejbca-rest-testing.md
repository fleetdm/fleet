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
- A macOS host enrolled in your Fleet for the end-to-end test

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

### 1. Certificate Profile for the Fleet service cert

Admin UI → **CA Functions → Certificate Profiles**:

1. Type `fleetRESTAdmin` in **Identifier** and click **Clone** on the `ENDUSER` row.
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

Admin UI → **RA Web → Make New Request** (top-right menu → RA Web; the cert
chooser dialog appears, cancel it):

1. **Certificate Type** → `fleetRESTAdmin`.
2. **Key-pair generation** → `By the CA`.
3. **Subject DN** → CN: `Fleet REST Service`.
4. **Username**: `fleet_rest_service`.
5. **Enrollment code**: set a strong password (you will use this to download
   the P12; Fleet stores neither this password nor the P12 itself —
   re-uploading is fine if you lose it).
6. Click **Enroll**.
7. Download as **PKCS#12 (P12)**.

You should now have `fleet_rest_service.p12` in your Downloads folder. Move
it somewhere safe.

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
   - **EJBCA REST URL**: `https://localhost:8443`
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
          "url":"https://localhost:8443",
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
