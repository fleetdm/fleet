# Proposal

## Why

IT admins who run EJBCA as their certificate authority (CA) cannot deploy per-host certificates to Apple hosts through Fleet for Wi-Fi and VPN (EAP-TLS). Fleet already supports DigiCert, NDES, custom SCEP, Smallstep, Hydrant, and custom EST. A customer moving from AD CS to EJBCA needs the same flow with EJBCA's REST API (customer promise, issue #30986). A proof of concept on `add-ejbca-rest-ca-poc` was verified, and a live EJBCA run (`ai/ejbca/live-poc-findings.md`) found defects and doc errors that this change fixes before production.

## What Changes

- Add an `ejbca` certificate authority type, configurable in the UI (Settings > Integrations > Certificate authorities), the REST API (`/certificate_authorities`), and GitOps (`org_settings.certificate_authorities.ejbca`).
- Fleet authenticates to EJBCA over mutual TLS with an admin-supplied client certificate (PKCS#12 plus password) and an optional trust CA bundle, and enrolls certificates with `POST /ejbca/ejbca-rest-api/v1/certificate/pkcs10enroll`.
- Apple (macOS, iOS, iPadOS) configuration profiles can use `$FLEET_VAR_EJBCA_DATA_<CA_NAME>` and `$FLEET_VAR_EJBCA_PASSWORD_<CA_NAME>` in a `com.apple.security.pkcs12` payload. Fleet generates the keypair and CSR per host, enrolls, and substitutes the resulting PKCS#12.
- Mirror DigiCert behavior from issue #52368: Fleet variables in the username template and UPNs, resend of matching profiles when certificate-affecting fields change, and reissue when a host's IdP values change.
- Add `added_ejbca`, `edited_ejbca`, and `deleted_ejbca` activities, `fleetctl generate-gitops` output, and secret masking on read.
- Fleet Premium only. A database migration adds the EJBCA type, its columns, and the managed-certificate type.
- Fix the defects found against a live EJBCA: malformed UPN SAN encoding, misleading error mapping, a connection probe that does not authenticate, and the runtime `openssl` dependency for BER PKCS#12 files.

## Capabilities

### New Capabilities
- `ejbca-certificate-authority`: Configure an EJBCA CA in Fleet and issue per-host PKCS#12 certificates to Apple hosts through it, including validation, secrets handling, variable substitution, reissue behavior, activities, and GitOps.

### Modified Capabilities

None. No capability specs exist on `main` yet, so the DigiCert-aligned behavior is described in the new capability.

## Impact

- Backend: `server/fleet/certificate_authorities.go`, `server/fleet/mdm.go` (variable prefixes and allow-list), `ee/server/service/certificate_authorities.go` and a new `ee/server/service/ejbca/` client, `server/datastore/mysql/certificate_authorities.go` plus a migration, `server/mdm/apple/profile_processor.go`, `server/service/mdm_profiles.go`, `server/variables/variables.go`, activities in `server/fleet/activities.go`, wiring in `cmd/fleet/serve.go` and `server/service/service.go`.
- Frontend: EJBCA form and add/edit modal wiring under `frontend/pages/admin/IntegrationsPage/cards/CertificateAuthorities/`, activity feed text, interfaces and API service.
- GitOps and CLI: `fleetctl gitops`, `fleetctl generate-gitops`, batch apply for certificate authorities.
- Docs: REST API, YAML files, audit logs, fleet variables, and the Wi-Fi certificate guide (reference docs go in a separate PR off the docs branch). The reference docs from PR #54901 list only `name`, `url`, and `password` and need extending.
- No new runtime dependency; the PoC `openssl` subprocess is removed.
