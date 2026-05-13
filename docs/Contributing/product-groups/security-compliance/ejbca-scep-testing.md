# EJBCA SCEP testing

This guide sets up a local EJBCA Community CA suitable for end-to-end testing of Fleet's
custom SCEP integration. It complements the `micromdm/scep` setup documented in
[`custom-scep-integration.md`](../mdm/custom-scep-integration.md) and is preferable when you
need behavior closer to a production enterprise CA — particularly:

- **Full SAN preservation, including `otherName` (Microsoft UPN, OID `1.3.6.1.4.1.311.20.2.3`).**
  `step-ca` and `micromdm/scep` both rebuild SAN from typed CSR fields
  (`csr.DNSNames`, `csr.EmailAddresses`, `csr.IPAddresses`, `csr.URIs`) and discard
  `otherName`. EJBCA preserves the full SAN extension when **Allow Extension Override**
  is enabled on the certificate profile — what this guide configures.
- **RFC 5280-max (20-byte) serial numbers by default.** `step-ca` and `micromdm/scep`
  default to 128-bit serials, which can mask client- or server-side bugs that only
  appear with full-size serials.
- **Production-style profile and end-entity model.** Tests cross more of the same code
  paths Fleet customers will hit when wiring up Microsoft AD CS, NDES, or commercial CAs.

## Prerequisites

- Docker
- Fleet dev server running locally (default ports)
- A device or test harness that can drive SCEP enrollment against Fleet (Android agent or
  equivalent)

## Set up EJBCA in Docker

```bash
docker run -d --name ejbca \
  -h ejbca.local \
  -p 8480:8080 -p 8443:8443 \
  -e TLS_SETUP_ENABLED=simple \
  keyfactor/ejbca-ce
```

Wait ~3 min for Wildfly + EJBCA to deploy:

```bash
until curl -sk -o /dev/null -w "%{http_code}" https://localhost:8443/ejbca/publicweb/healthcheck/ejbcahealth 2>/dev/null | grep -q "200"; do sleep 10; done
echo "EJBCA up"
```

`TLS_SETUP_ENABLED=simple` exposes the Admin UI on HTTPS without requiring a client cert. The
server still presents a `CertificateRequest` so the first time you visit
`https://localhost:8443/ejbca/adminweb/` in a browser you will see a client-cert chooser —
just cancel it. You will also see a TLS interstitial because the cert is self-signed for
`ejbca.local`; click "Advanced" → "Proceed".

## Important EJBCA-CE limitation: no SCEP RA mode

EJBCA's SCEP **RA mode** auto-creates end entities from incoming CSRs. It is **gated behind
EJBCA Enterprise**. EJBCA-CE returns `SCEP RA mode is enabled, but not included in the
community version of EJBCA` on every request when the alias is set to RA mode.

This guide uses **CA mode**, which requires you to pre-create one end entity per cert
template you intend to test. The end entity's DN must match the CSR DN; the entity's
password becomes the SCEP challenge.

## Create the certificate profile (clone ENDUSER, enable extension override)

Admin UI → **CA Functions → Certificate Profiles**:

1. Type `FleetQACertProfile` in the **Identifier** field.
2. Click **Clone** on the row for `ENDUSER`.
3. Click **Edit** on the new `FleetQACertProfile` row.
4. Check **Allow Extension Override**. This is the field that decides whether SAN entries
   from the CSR — including `otherName` — flow through to the issued cert. Without it,
   EJBCA rebuilds SAN from typed fields only.
5. **Save**.

## Create the end entity profile (clone EMPTY, enable SAN, add DN fields)

Admin UI → **RA Functions → End Entity Profiles**:

1. Type `FleetQAEEP` in the **Identifier** field and click **Add Profile**.
2. Select `FleetQAEEP` in the dropdown and click **Edit End Entity Profile**.
3. Under **Subject Alternative Name**, add each of these field types (select from dropdown,
   click Add):
   - DNS Name
   - RFC 822 Name (e-mail address)
   - IP Address
   - Uniform Resource Identifier (URI)
   - MS UPN, User Principal Name
4. Under **Subject DN Attributes**, add `O, Organization` (the default profile only has CN).
   Add any other DN attributes your test cert templates will use.
5. Under **Main Certificate Data**:
   - **Default Certificate Profile** → `FleetQACertProfile`
   - **Available Certificate Profiles** → highlight `FleetQACertProfile`
   - **Default CA** → `ManagementCA`
   - **Available CAs** → highlight `ManagementCA`
6. **Save**.

## Configure the SCEP alias

Admin UI → **System Configuration → SCEP Configuration**:

1. If `fleetqa` (or whatever alias you choose) doesn't exist, **Add** it.
2. **Edit** the alias.
3. Set **Operation Mode** to `CA`. (Setting RA here works but every request will fail with
   the CE limitation; CA mode is the only working option.)
4. **Save**.

You can also do this from the EJBCA CLI:

```bash
docker exec ejbca /opt/keyfactor/bin/ejbca.sh config scep addalias fleetqa
docker exec ejbca /opt/keyfactor/bin/ejbca.sh config scep updatealias fleetqa --key operationmode --value CA
```

## Wire EJBCA into Fleet as a custom SCEP CA

When you register a SCEP CA, Fleet validates the URL by issuing a real `GetCACert` probe and
will reject EJBCA's self-signed HTTPS cert. Use the HTTP port (`8480` from the docker mapping)
instead:

```bash
FLEET_TOKEN="$(awk '/^  qa:/,/^  [a-z]/' ~/.fleet/config | awk '/token:/ {print $2}')"
curl -sk -X POST -H "Authorization: Bearer $FLEET_TOKEN" -H "Content-Type: application/json" \
  -d '{"custom_scep_proxy":{
        "name":"EJBCA_QA",
        "url":"http://localhost:8480/ejbca/publicweb/apply/scep/fleetqa/pkiclient.exe",
        "challenge":"fleet-qa-scep-challenge"
      }}' \
  https://localhost:8080/api/latest/fleet/certificate_authorities
```

Notes:

- The URL must end in `/pkiclient.exe` — EJBCA's "clean" URL form `/scep/<alias>` returns
  `Wrong URL. No alias found.`
- The `challenge` value here becomes the SCEP challenge password Fleet's proxy injects on
  the agent's behalf. It must match the password of every end entity you pre-create below.

## Per-test setup: pre-create the end entity

For each Fleet certificate template you want to enroll against EJBCA, the CSR's DN must
match a pre-created end entity. Example for a template with `subject_name = "CN=t07-ejbca,O=FleetQA"`:

```bash
docker exec ejbca /opt/keyfactor/bin/ejbca.sh ra addendentity \
  --username t07-ejbca \
  --dn "CN=t07-ejbca,O=FleetQA" \
  --caname ManagementCA \
  --type 1 \
  --token USERGENERATED \
  --password fleet-qa-scep-challenge \
  --certprofile FleetQACertProfile \
  --eeprofile FleetQAEEP
```

Match the `--password` to the SCEP challenge configured on the Fleet CA record. The
`--dn` must be an exact match for the CSR DN — DN string ordering and capitalization
both matter. EJBCA logs at `docker logs ejbca` will say `Wrong number of <FIELD> fields
in Subject DN` if the DN doesn't match what the EEP allows.

After enrollment the end entity's status flips from `NEW` (10) to `GENERATED` (40) and
the cert is stored in the EJBCA DB. To re-enroll the same end entity, reset its status:

```bash
docker exec ejbca /opt/keyfactor/bin/ejbca.sh ra setendentitystatus --username t07-ejbca --S NEW
```

## Create the Fleet certificate template

Using the Fleet CA record `id` returned by the create-CA call above:

```bash
curl -sk -X POST -H "Authorization: Bearer $FLEET_TOKEN" -H "Content-Type: application/json" \
  -d '{
    "name": "qa-test",
    "subject_name": "CN=t07-ejbca,O=FleetQA",
    "subject_alternative_name": "DNS=t07-host.example.com, UPN=qa-user@corp.example.com",
    "certificate_authority_id": <CA_ID>,
    "team_id": <TEAM_ID>
  }' \
  https://localhost:8080/api/latest/fleet/certificates
```

Then wait for the agent's 15-min poll cycle (or restart the agent app) for the cert to be
delivered, the CSR sent through the SCEP proxy, and the issued cert returned.

## Verify the issued cert

Extract the issued cert from EJBCA:

```bash
docker exec ejbca /opt/keyfactor/bin/ejbca.sh ra getendentitycert --username t07-ejbca \
  | awk '/-----BEGIN CERTIFICATE-----/,/-----END CERTIFICATE-----/' > /tmp/issued-cert.pem
```

Inspect with openssl:

```bash
openssl x509 -in /tmp/issued-cert.pem -text -noout | grep -B1 -A 5 "Subject Alternative Name"
openssl x509 -in /tmp/issued-cert.pem -noout -serial -subject -dates
```

For a cert template with `subject_alternative_name = "DNS=t07-host.example.com, UPN=qa-user@corp.example.com"`,
the SAN extension should contain both entries end-to-end:

```
X509v3 Subject Alternative Name:
    DNS:t07-host.example.com, othername: UPN::qa-user@corp.example.com
```

The `othername: UPN::` line confirms that the UPN flowed through. If you only see the DNS
entry, the cert profile's **Allow Extension Override** is not enabled — fix it and re-enroll.

## Cleanup

```bash
docker rm -f ejbca
```

If you also added EJBCA's cert to the system trust store for HTTPS testing, remove that:

```bash
sudo security delete-trusted-cert -d /path/to/ejbca-tls.crt
```

Delete Fleet templates and the CA record:

```bash
curl -sk -X DELETE -H "Authorization: Bearer $FLEET_TOKEN" \
  https://localhost:8080/api/latest/fleet/certificates/<TEMPLATE_ID>
curl -sk -X DELETE -H "Authorization: Bearer $FLEET_TOKEN" \
  https://localhost:8080/api/latest/fleet/certificate_authorities/<CA_ID>
```

## Known limitations and gotchas

- **EJBCA-CE has no SCEP RA mode.** Each test cert template requires a pre-created end
  entity. If you're iterating on many templates, scripts that read template DNs and create
  matching end entities save time.
- **EJBCA's self-signed TLS cert is for `ejbca.local`.** Fleet's SCEP URL validation and most
  HTTPS clients will reject it on `localhost`. Use the HTTP port for the Fleet→EJBCA leg.
- **End-entity status doesn't reset automatically.** After an enrollment the status moves to
  `GENERATED`; subsequent SCEP requests for the same DN/username will fail until you reset
  it back to `NEW`.
- **DN exact match required.** The EEP's allowed DN attributes must include every component
  of the cert template's `subject_name`, in compatible order. Missing fields fail with
  `Wrong number of <FIELD> fields in Subject DN`.
