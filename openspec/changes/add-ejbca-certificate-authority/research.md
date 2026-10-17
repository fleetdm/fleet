# EJBCA live POC findings (issue #30986)

Notes from running a real EJBCA and exercising the PoC client (`origin/add-ejbca-rest-ca-poc`, `ee/server/service/ejbca/`) against it. Use these to correct `docs/Contributing/product-groups/security-compliance/ejbca-rest-testing.md` and the openspec design later.

Tested against `keyfactor/ejbca-ce` (EJBCA 9.6.3 Community, arm64) on 2026-10-15.

## Local setup that worked

- `docker run -d --name ejbca -h localhost -p 8480:8080 -p 8443:8443 -e TLS_SETUP_ENABLED=simple -e HTTPSERVER_HOSTNAME=localhost keyfactor/ejbca-ce`. Using `localhost` as the hostname means the TLS cert matches `https://localhost:8443` and no `/etc/hosts` entry (sudo) is needed, unlike the `ejbca.local` approach in the current doc.
- Health: `https://localhost:8443/ejbca/publicweb/healthcheck/ejbcahealth` returns 200 after about 1 minute on arm64.
- Enable the REST protocol from the CLI: `docker exec ejbca bin/ejbca.sh config protocols enable --name "REST Certificate Management"`, then `docker restart ejbca`. Before the restart, curl gets an empty reply (exit 52) instead of a 403 page.
- Service cert without the UI: `ra addendentity` (token `P12`), `ra setclearpwd`, `batch -dir /tmp/p12` produces `<username>.p12`. Then `roles addrole`, `roles addrolemember --role ... --caname ManagementCA --with WITH_COMMONNAME --value "Fleet REST Service"`, and `roles changerule "<role>" "<rule>" ACCEPT`.
- Export the CA cert for the trust bundle: `ca getcacert --caname ManagementCA -f /tmp/mgmtca.pem`.
- The admin UI at `https://localhost:8443/ejbca/adminweb/` needs the self-signed TLS warning accepted once; no client cert is needed.
- Hand-written profile XML via `ca importprofiles` does not work: EJBCA resets or breaks the profile (null booleans, `availableEcCurves is null`). Create profiles through the admin UI (clone ENDUSER for the cert profile, Add for the end entity profile).
- The `/etc/hosts` and the BER P12 (RA Web export) notes in the current doc were not exercised in this run.

## Minimal role rules the Fleet service cert needs

- `/administrator/`
- `/ca/<CA name>/`
- `/ca_functionality/create_certificate/`
- `/ra_functionality/create_end_entity/` and `/ra_functionality/edit_end_entity/`
- `/endentityprofilesrules/<end entity profile>/create_end_entity/` and `.../edit_end_entity/` for every end entity profile Fleet enrolls against

Without the `edit_end_entity` rules the first enrollment for a username succeeds, but any re-enrollment for the same username returns 403 `Administrator not authorized to end entity profile <id>`. Re-enrollment happens whenever Fleet reissues after a CA edit or renewal, so the docs must list the edit rules.

## Corrections to the existing PoC doc and design

1. Community Edition does auto-create the end entity on `pkcs10enroll`. The doc says CE cannot, and tells testers to pre-create one end entity per host with `ra addendentity`. That step is unnecessary, and CE is enough for multi-host testing. The end entity is created with the username and CSR subject. After issuance its status is 40 (GENERATED) and re-enrolling works because of the edit rules above.
2. "Allow Extension Override" is not required for the UPN SAN to survive. With the built-in `ENDUSER` cert profile (override off, confirmed in the UI) and the `EMPTY` end entity profile, a CSR with a UPN SAN produced a cert with the UPN intact. Re-test before keeping the gotcha in any doc.
3. `GET /v1/ca/status` is unauthenticated: it returns 200 with no client cert. `VerifyConnection` only proves reachability and TLS trust, not that the client cert is accepted or that the role can issue. A role with no rules also passes it.
4. EJBCA error shapes seen from `pkcs10enroll` (all `{"error_code":N,"error_message":"..."}`):
   - Unknown CA: 400 `CA with name "X" doesn't exist`
   - Unknown cert profile: 400 `Certificate profile with name "X" doesn't exist`
   - Unknown end entity profile: 400 `End Entity Profile of name "X" was not found`
   - Cert profile not listed in the end entity profile: 400 `Couldn't find certificate profile (<id>) among available certificate profiles.`
   - Role lacks rules: 403 `Administrator is not authorized to resource /administrator` (or `/ca_functionality/create_certificate`, or `not authorized to end entity profile <id>`)
   - Malformed SAN: 500 `General failure.` (log shows `illegal object in getInstance: DLSequence`)
   The PoC's `mapEnrollError` maps 404 and 422 to the "name does not exist" cases, so the real 400s fall into the generic branch. The 403 message ("revoked, expired, or not bound to a role") is misleading when the real cause is a missing access rule.
5. UPN SAN encoding bug in `buildUPNSANExtension`: the PoC encodes `[0] { SEQUENCE { UTF8String } }`, the correct form is `[0] { UTF8String }`. Fix: drop the `upnValue` struct and use `Value string` tagged `asn1:"utf8,explicit,tag:0"` in the `otherName` struct. After the fix the UPN appears in the issued cert (`othername: UPN:...`) and the Fleet client enrolled successfully with and without a UPN.

## Better save-time probe: GET /v1/ca

`GET /ejbca/ejbca-rest-api/v1/ca` is authenticated. Without a client cert it returns 403 `Error no client certificate or OAuth token received for authentication.`; with a client cert not issued by a CA EJBCA trusts, the TLS handshake fails (curl exit 000); with the Fleet service cert it returns 200 and `{"certificate_authorities":[{"id","name","subject_dn","issuer_dn","expiration_date","external"}]}` for the CAs the role can access. So a save-time check can prove the credential is accepted and that the configured CA name is visible to the role (typo or missing `/ca/<CA>/` rule). It cannot validate cert profile or end entity profile names (EJBCA exposes no list endpoint for them over this REST protocol).

## What the Fleet client sends and EJBCA needs to match

- CSR: RSA 2048, subject CN equal to `username_template` after variable expansion, optional UPN SAN.
- Body: `certificate_request` (PEM), `certificate_profile_name`, `end_entity_profile_name`, `certificate_authority_name`, `username`, `password` (random per call, accepted), `include_chain: false`, `response_format: DER`. Response is 201 with a base64 DER `certificate`.
- The end entity profile must include a CN subject DN field and list the cert profile; for the UPN case it also had the UPN subject alt name field added (the built-in `EMPTY` profile worked without it).
- The cert profile needs a CA in its available CAs (UI default `Any CA` was fine for the built-in; the clone used `ManagementCA`).

## Not tested

- Upload of a P12 exported from RA Web (BER encoding, the openssl subprocess in the PoC). Tested with PEMs extracted from a batch-generated P12.
- Fleet server CA save path, profile variable substitution, and delivery to a real macOS host through MDM.
- EJBCA Enterprise behaviors.

## PR 1 (datastore) learnings

- The PoC migration did not insert the `FLEET_VAR_EJBCA_DATA_` and `FLEET_VAR_EJBCA_PASSWORD_` prefix rows into `fleet_variables`. Profile resend after a CA edit finds profiles through `mdm_configuration_profile_variables` joined to that table, so without the rows a CA edit never resends EJBCA profiles.
- `fleet.HasCAVariables` needs the EJBCA prefixes or EJBCA profiles skip the CA install throttle in `server/mdm/apple/reconcile.go`.
- The IdP-change resend (`triggerResendCAProfilesUsingVariables` in `server/datastore/mysql/scim.go`) must scan the EJBCA username template and UPNs.
- Adding `CAConfigEJBCA` to `ListCATypesWithRenewalSupport` adds one more `RenewMDMManagedCertificates` pass per cron run, like every other type.
- For PR 3 and PR 5: the batch upsert writes NULL for any EJBCA column the caller leaves nil, so GitOps must always send the full CA, and the service must reject a masked key (`********`) as a new key.

## Username and CSR common name are separate (decision)

Live test: EJBCA accepts a username that differs from the CSR common name, and a re-enrollment of the same username replaces the end entity DN with the new CSR subject (the end entity then holds both certificates). So Fleet has two templates: the CSR common name uses `certificate_common_name` (shared with DigiCert) and the EJBCA username uses the optional `ejbca_username_template`, which defaults to the common name template when empty (`CertificateAuthority.EffectiveEJBCAUsernameTemplate`). The OpenSpec spec and design still describe one shared template and need updating.
