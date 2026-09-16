# EJBCA SCEP testing

Sets up a local EJBCA Community CA for end-to-end testing of Fleet's custom SCEP integration.
Prefer this over `step-ca` or `micromdm/scep` when you need:

- Full SAN preservation, including `otherName` (Microsoft UPN, OID `1.3.6.1.4.1.311.20.2.3`).
- RFC 5280-max (20-byte) serial numbers by default.
- Behavior closer to a production enterprise CA (AD CS, NDES, commercial CAs).

## Prerequisites

- Docker
- Fleet dev server running locally
- A device or test harness that drives SCEP enrollment against Fleet

## Set up EJBCA in Docker

```bash
docker run -d --name ejbca \
  -h ejbca.local \
  -p 8480:8080 -p 8443:8443 \
  -e TLS_SETUP_ENABLED=simple \
  keyfactor/ejbca-ce
```

Wait for the server to come up (about 3 minutes):

```bash
until curl -sk -o /dev/null -w "%{http_code}\n" \
    https://localhost:8443/ejbca/publicweb/healthcheck/ejbcahealth | grep -q 200; do
  sleep 10
done
echo "EJBCA up"
```

EJBCA's Admin UI is at `https://localhost:8443/ejbca/adminweb/`. The first visit triggers a
client-cert chooser (cancel it) and a TLS interstitial because the cert is self-signed for
`ejbca.local` (click **Advanced → Proceed**).

## Important: EJBCA-CE has no SCEP RA mode

EJBCA's SCEP RA mode (auto-create end entities from CSRs) is gated behind EJBCA Enterprise.
This guide uses **CA mode**, which requires you to **pre-create one end entity per
distinct CSR subject DN** you intend to test. The end entity's password becomes the SCEP
challenge.

## One-time setup

### 1. Certificate profile (preserves the full SAN extension)

Admin UI → **CA Functions → Certificate Profiles**:

1. Type `FleetQACertProfile` in the **Identifier** field.
2. Click **Clone** on the `ENDUSER` row.
3. **Edit** the new `FleetQACertProfile` row.
4. Check **Allow Extension Override**. Without this, EJBCA rebuilds the SAN from its typed
   fields and drops `otherName` (UPN).
5. **Save**.

### 2. End entity profile

Admin UI → **RA Functions → End Entity Profiles**:

1. Type `FleetQAEEP` in the **Identifier** field and click **Add Profile**.
2. Select `FleetQAEEP` and click **Edit End Entity Profile**.
3. Under **Subject Alternative Name**, add each of these (select from the dropdown, click
   **Add**):
   - DNS Name
   - RFC 822 Name (e-mail)
   - IP Address
   - Uniform Resource Identifier (URI)
   - MS UPN, User Principal Name
4. Under **Main Certificate Data**:
   - **Default Certificate Profile** → `FleetQACertProfile`
   - **Available Certificate Profiles** → highlight `FleetQACertProfile`
   - **Default CA** → `ManagementCA`
   - **Available CAs** → highlight `ManagementCA`
5. **Save**.

The default Subject DN attribute is just `CN`. Leave it that way unless you need richer
DNs — adding `O`/`OU` here also requires those to be present in every CSR DN you enroll,
which complicates testing. Stick with CN-only subjects.

### 3. SCEP alias

```bash
docker exec ejbca bin/ejbca.sh config scep addalias fleetqa
docker exec ejbca bin/ejbca.sh config scep updatealias \
    --alias fleetqa --key operationmode --value CA
docker exec ejbca bin/ejbca.sh config scep updatealias \
    --alias fleetqa --key ra.certificateProfile --value FleetQACertProfile
docker exec ejbca bin/ejbca.sh config scep updatealias \
    --alias fleetqa --key ra.entityProfile --value FleetQAEEP
docker exec ejbca bin/ejbca.sh config scep updatealias \
    --alias fleetqa --key ra.defaultCA --value ManagementCA
```

### 4. Wire EJBCA into Fleet as a custom SCEP CA

Fleet validates the SCEP URL with a real `GetCACert` probe and will reject EJBCA's
self-signed HTTPS cert, so register the **HTTP** port (`8480`):

```bash
FLEET_TOKEN="$(awk '/^  qa:/,/^  [a-z]/' ~/.fleet/config | awk '/token:/ {print $2}')"

curl -sk -X POST \
    -H "Authorization: Bearer $FLEET_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"custom_scep_proxy":{
          "name":"EJBCA_QA",
          "url":"http://localhost:8480/ejbca/publicweb/apply/scep/fleetqa/pkiclient.exe",
          "challenge":"fleet-qa-scep-challenge"
        }}' \
    https://localhost:8080/api/latest/fleet/certificate_authorities
```

The URL must end in `/pkiclient.exe` — the "clean" form `/scep/<alias>` returns
`Wrong URL. No alias found.`. The `challenge` value must match the end entity password you
set below.

## Per-test setup

### Pre-create the end entity

For each cert template, create one end entity whose username and DN match the CSR's CN.
Example for `subject_name = "CN=qa-user@example.com"`:

```bash
docker exec ejbca bin/ejbca.sh ra addendentity \
    --username 'qa-user@example.com' \
    --dn 'CN=qa-user@example.com' \
    --caname ManagementCA \
    --type 1 \
    --token USERGENERATED \
    --password fleet-qa-scep-challenge \
    --certprofile FleetQACertProfile \
    --eeprofile FleetQAEEP
```

The `--password` must match the SCEP `challenge` value on Fleet's CA record.

### Create the Fleet certificate template

Using the Fleet CA `id` returned from the create-CA call:

```bash
curl -sk -X POST \
    -H "Authorization: Bearer $FLEET_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "qa-test",
      "subject_name": "CN=qa-user@example.com",
      "subject_alternative_name": "DNS=qa-host.example.com, UPN=qa-user@corp.example.com",
      "certificate_authority_id": <CA_ID>,
      "team_id": <TEAM_ID>
    }' \
    https://localhost:8080/api/latest/fleet/certificates
```

Wait for the agent's 15-minute poll cycle, or restart the agent to trigger enrollment now.

### Verify the issued cert

EJBCA stores every cert it issues for the end entity. `getendentitycert` prints them all,
newest-first. Take the first PEM block:

```bash
docker exec ejbca bin/ejbca.sh ra getendentitycert --username 'qa-user@example.com' 2>/dev/null \
    | awk '/-----BEGIN CERTIFICATE-----/{n++} n==1 {print} /-----END CERTIFICATE-----/{if(n==1) exit}' \
    > /tmp/issued-cert.pem

openssl x509 -in /tmp/issued-cert.pem -noout -subject -dates -ext subjectAltName
```

For the template above, the SAN extension should contain both entries:

```
X509v3 Subject Alternative Name:
    DNS:qa-host.example.com, othername: UPN::qa-user@corp.example.com
```

If you only see DNS and no `othername`, the cert profile's **Allow Extension Override**
is not enabled — fix it and re-enroll.

### Re-enrolling the same end entity

After each successful enrollment, the end entity flips from status `NEW` (10) to
`GENERATED` (40) and its password is consumed. Reset both before the next enrollment:

```bash
docker exec ejbca bin/ejbca.sh ra setendentitystatus \
    --username 'qa-user@example.com' -S 10
docker exec ejbca bin/ejbca.sh ra setpwd \
    --username 'qa-user@example.com' --password fleet-qa-scep-challenge
```

## Cleanup

```bash
docker rm -f ejbca

curl -sk -X DELETE -H "Authorization: Bearer $FLEET_TOKEN" \
    https://localhost:8080/api/latest/fleet/certificates/<TEMPLATE_ID>
curl -sk -X DELETE -H "Authorization: Bearer $FLEET_TOKEN" \
    https://localhost:8080/api/latest/fleet/certificate_authorities/<CA_ID>
```

## Gotchas

- **EJBCA-CE has no SCEP RA mode.** Each new CSR subject DN needs its own pre-created
  end entity.
- **EJBCA's self-signed TLS cert is for `ejbca.local`.** Use the HTTP port (`8480`) for
  the Fleet → EJBCA leg; use HTTPS (`8443`) only for the Admin UI in your browser.
- **End entity status doesn't reset automatically.** After enrollment, status moves to
  `GENERATED`. Subsequent SCEP requests for the same username will fail with
  `User <name> not found.` (a misleading message — the user exists, but is no longer in
  status `NEW`) until you reset it.
- **CSR DN must exactly match the end entity's DN.** Ordering and capitalization both
  matter. With CN-only subjects this rarely bites; with O/OU it routinely does.
