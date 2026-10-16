# Design

## Context

DigiCert is the model: a named CA row, `$FLEET_VAR_<TYPE>_DATA_<name>` and `_PASSWORD_<name>` variables in Apple pkcs12 payloads, per-host issuance in `server/mdm/apple/profile_processor.go`, a managed-certificate record, and (from #52368) variables in certificate fields with resend on change. The PoC branch `origin/add-ejbca-rest-ca-poc` implements most of this for EJBCA (client in `ee/server/service/ejbca/`, `EJBCACA` type, migration, form, profile hooks). This design keeps the PoC's shape and changes what the live EJBCA run (`research.md`, EJBCA 9.6.3 Community) showed to be wrong. Requirements are in the spec.

## Goals / Non-Goals

**Goals:**
- Reuse the DigiCert code paths and tests wherever the behavior is the same, so EJBCA is a small, reviewable delta.
- Keep the Fleet-side client independent of EJBCA edition (Community and Enterprise behave the same for `pkcs10enroll`).

**Non-Goals:**
- OAuth bearer authentication, PEM-direct client certificate upload, and a user-configurable enrollment password (the per-call random password was accepted by EJBCA in the live run).
- Windows, Linux, and Android delivery, SAN types other than UPN, and revoking certificates in EJBCA when hosts are deleted.
- Listing EJBCA profiles in the UI (EJBCA exposes no list endpoint for them over this REST protocol).

## Decisions

1. **Credential input is `.p12` plus password, decoded in pure Go with BER tolerance.** The PoC shells out to `openssl` because both Go PKCS#12 libraries need strict DER and RA Web emits BER. Production must not depend on a binary, so add a small BER to DER normalizer (re-encode indefinite-length and constructed-string forms) ahead of `go-pkcs12`, covered by a test fixture exported from a real EJBCA. Alternative: require PEM upload; rejected because RA Web exports `.p12` by default. The password is used once and discarded; the private key is encrypted with the server private key like other CA secrets.
2. **Save-time verification uses `GET /v1/ca`, not `GET /v1/ca/status`.** The live run showed `/ca/status` returns 200 with no client certificate, so it proves nothing about authentication. `/v1/ca` returns 403 without a client certificate, fails the TLS handshake for an untrusted one, and returns the CAs visible to the role, which lets Fleet also confirm the configured CA name. Profile names cannot be verified at save time and surface at first enrollment. Alternative: dry-run enrollment; rejected because it creates an end entity and certificate in the customer's EJBCA.
3. **End entities are never pre-created.** `pkcs10enroll` auto-created them on Community Edition, and re-enrollment of the same username works when the role has edit rights. The PoC doc's per-host `ra addendentity` step and its "Enterprise only" claim are wrong and are removed from docs and the open questions.
4. **Required EJBCA role rules are documented and diagnosed.** Minimum: `/administrator/`, `/ca/<CA>/`, `/ca_functionality/create_certificate/`, `/ra_functionality/create_end_entity/` and `edit_end_entity/`, plus create and edit on each end entity profile. Missing edit rules break only re-enrollment (403), which is exactly the resend and reissue path, so the error text must name role access, not certificate expiry.
5. **Fix UPN SAN encoding.** The PoC wraps the UTF8String in an extra SEQUENCE and EJBCA returns 500. Encode `otherName` as `[0] { OID, [0] EXPLICIT UTF8String }` with a plain string value, with a test that decodes the output with `crypto/x509`-independent ASN.1 parsing. Do not state that "Allow Extension Override" is required: the live run preserved the UPN with it off. Document it as "verify in your EJBCA".
6. **Error mapping follows EJBCA's real codes.** 400 with an error message for unknown CA, certificate profile, or end entity profile (or a certificate profile not allowed by the end entity profile); 403 for missing role access (distinct from expired or revoked, which fail at TLS); everything else includes EJBCA's `error_message`. The PoC's 404 and 422 branches are removed.
7. **Mirror DigiCert's reissue rule.** On update, resend matching Apple profiles only when a certificate-affecting field changes (common name template, username template, UPNs, EJBCA CA name, certificate profile, end entity profile), inside the same transaction as the update using the existing resend helper pattern from #55410. Connection changes (URL, client certificate, trust bundle) do not resend. The issue's test plan says "edit reissues to every host"; the DigiCert-style rule was confirmed instead, because it avoids fleet-wide reissue on a credential rotation. The test plan line needs updating to match.
8. **Variables allow-list reuses DigiCert's.** Add EJBCA variable prefixes next to DigiCert's, validate pairs and pkcs12-only placement through the existing hook (`additionalDigiCertValidation` becomes a shared or parallel EJBCA validator), and extend the profile processor's variable substitution, IdP-change resend, and failure reporting with an EJBCA branch next to the DigiCert one.
9. **Storage.** Reuse the `certificate_authorities` table and its existing columns where the meaning matches: `certificate_common_name` for the common name template and `certificate_user_principal_names` for the UPNs, both shared with DigiCert. Add the `ejbca` enum value and columns `client_cert_pem`, `client_key_encrypted` (encrypted with the server key), `trust_ca_bundle_pem`, `ejbca_ca_name`, `ejbca_certificate_profile_name`, `ejbca_end_entity_profile_name`, and `ejbca_username_template`. The credentials are columns on this table, not `ca_config_assets`, which only the pre-`certificate_authorities` design used. Add `ejbca` to the `host_mdm_managed_certificates` type enum, and the `FLEET_VAR_EJBCA_DATA_` and `FLEET_VAR_EJBCA_PASSWORD_` prefix rows to `fleet_variables`, because profiles are linked to their CA through those rows and a CA edit finds the profiles to resend with them. One migration; only the `fleet_variables` rows need a migration test.
10. **API and GitOps shape.** Key `ejbca` in create and batch-apply payloads. Follow the published reference docs for names (`name`, `url`, `password` for the PKCS#12 password) and add the fields those docs omit: the PKCS#12 itself as base64 `client_p12`, `trust_ca_bundle`, `certificate_authority_name` (the CA name inside EJBCA, distinct from Fleet's `name`), `certificate_profile_name`, `end_entity_profile_name`, `certificate_common_name`, the optional `username_template`, and `certificate_user_principal_names`. In GitOps the PKCS#12 and password are required on every entry and are supplied as environment variable references; `generate-gitops` cannot export them (only the decoded certificate and key are stored, and the password is discarded), so it emits `$EJBCA_<CA_NAME>_CLIENT_P12` and `$EJBCA_<CA_NAME>_PASSWORD` TODO placeholders like other secret fields.

11. **Update semantics.** Update payload fields are optional pointers like DigiCert's: nil or masked means keep. A supplied `client_p12` needs `password`; verification runs with the candidate credential before anything is written, so a bad rotation never replaces a working one.
12. **Expiry is derived, not stored.** `client_cert_expires_at` is computed from the stored client certificate PEM when a CA is read, so it cannot drift from the credential and needs no column. It is a read-only response field and never appears in GitOps output.

13. **Two templates: common name and username.** EJBCA accepts an end entity username that differs from the CSR common name (live test, `research.md`), and a repeat enrollment of a username replaces the end entity's DN with the new CSR subject. One shared template would force a per-user username (many devices, one end entity) when the common name is the user's email. So the CSR common name uses `certificate_common_name`, and the EJBCA username uses the optional `ejbca_username_template`, which defaults to the common name template when unset or empty. The default is resolved when enrolling, not copied at save time, so editing the common name template moves a defaulted username too. A username with its own variable, such as the host serial, keeps one end entity per device. Alternative: one shared template as in the PoC; rejected for the reason above.

## Risks / Trade-offs

- [Customer EJBCA role is broader or narrower than documented] → Document the minimal rules, diagnose 403 precisely, and add the checklist to the guide.
- [Profile names are only validated at first enrollment] → Surface EJBCA's message on the failed profile detail; document that typos show up there.
- [BER normalizer bugs] → Test against exports from RA Web and from the CLI batch tool, plus DER inputs, and fuzz the normalizer against malformed lengths.
- [Fleet's client certificate expires and new enrollments stop silently while existing certificates keep working] → Return `client_cert_expires_at` and show an expiry warning in the CA list.
- [Reissue on IdP change adds certificates and end entities in EJBCA] → Same trade-off as DigiCert; documented in the guide.
- [Docs in PR #54901 omit fields] → Extend the reference docs in a separate docs PR before the release branch cuts.

## Migration Plan

Additive migration only (new enum values, nullable columns, two `fleet_variables` prefix rows). No backfill. Rollback is not needed for correctness: older servers ignore the new type. Roll out behind no flag; the feature is inert until a CA is created.

## Open Questions

- Product and design: confirm the final API field names and the Figma form fields against this spec, since the published docs list only `name`, `url`, and `password`.
