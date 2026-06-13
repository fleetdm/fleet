# Certificate Authorities — EJBCA capability

## Added requirements

### REQ-CA-EJBCA-1: EJBCA CA type

Fleet SHALL support a new `CertificateAuthority` type with `type = "ejbca"`.
An EJBCA CA represents the configuration needed to enroll TLS/client
certificates from a customer's EJBCA instance via EJBCA's REST API over
mTLS.

### REQ-CA-EJBCA-2: Required configuration fields

An EJBCA CA SHALL require, at create time, the following fields:

- `name` — unique within the EJBCA CA type.
- `url` — base URL of the EJBCA REST API
  (e.g., `https://ejbca.example.com:8443`).
- `client_p12` — PKCS#12 bundle containing the client certificate chain and
  matching private key. **POC accepts P12 only**; PEM-direct upload is
  deferred. The server decodes the P12 once at submission using
  `client_p12_password`, extracts the cert chain and private key, persists
  them as PEM (private key encrypted), and discards the P12 password.
- `client_p12_password` — password protecting the uploaded P12. **Not
  persisted** — used only to decode the bundle at submission time.
- `certificate_authority_name_ejbca` — name of the issuing CA inside
  EJBCA, used as `certificate_authority_name` in `pkcs10enroll` requests.
- `certificate_profile_name` — name of the certificate profile in EJBCA.
- `end_entity_profile_name` — name of the end entity profile in EJBCA.
- `username_template` — string used as both the EJBCA end-entity username
  and the CSR CommonName. MAY contain Fleet variables from the existing
  allow-list (`$FLEET_VAR_HOST_HARDWARE_SERIAL`, `$FLEET_VAR_HOST_PLATFORM`,
  `$FLEET_VAR_HOST_END_USER_EMAIL_IDP`).

The `password` field that EJBCA's `pkcs10enroll` API requires SHALL be
generated internally by Fleet per-enrollment (REQ-CA-EJBCA-7). It is not
a user-configurable field in this iteration. See "Deferred" for the
user-configurable variant.

### REQ-CA-EJBCA-2a: Subject Alternative Name templating (UPN)

An EJBCA CA SHALL accept an optional list of Microsoft User Principal
Names to embed in the CSR's `subjectAltName` extension as `otherName`
entries (OID `1.3.6.1.4.1.311.20.2.3`):

- `certificate_user_principal_names` — `[]string`, optional. Each entry MAY
  contain Fleet variables from the allow-list. Empty list means no SAN
  extension is added to the CSR.

This matches the templating surface Fleet's DigiCert CA supports today.
Other SAN attribute types (DNS, email, IP, URI) are out of scope for this
change. The customer's EJBCA Certificate Profile MUST have "Allow Extension
Override" enabled to preserve the SAN through issuance (documented in the
dev guide).

### REQ-CA-EJBCA-3: Optional configuration fields

- `trust_ca_bundle` — PEM-encoded CA bundle used to verify EJBCA's HTTPS
  server certificate. When empty, Fleet uses its system root store.

### REQ-CA-EJBCA-4: Encrypted-at-rest secrets

The following EJBCA CA fields SHALL be encrypted with `serverPrivateKey`
when stored in MySQL:

- the PEM private key extracted from the uploaded P12

The PEM certificate chain extracted from the P12 and the `trust_ca_bundle`
are public artifacts and SHALL be stored unencrypted (matching existing
patterns for public PEM material).

The `client_p12_password` supplied at submission SHALL NOT be persisted.
The per-enrollment `password` value Fleet generates for `pkcs10enroll`
SHALL NOT be persisted — it is created at issuance time, sent, and
discarded.

### REQ-CA-EJBCA-5: Secret masking on read

When `includeSecrets=false`, reads of EJBCA CA records SHALL return
`fleet.MaskedPassword` for the stored PEM private key, matching the
existing DigiCert `api_token` masking behavior.

### REQ-CA-EJBCA-6: Connection verification on create/update

On create and update of an EJBCA CA, Fleet SHALL:

1. Validate field presence and parseability (URL parses; uploaded P12
   decodes with the supplied password; extracted cert and key pair
   correctly; trust bundle parses if present).
2. Build an HTTP client with the configured mTLS material and call
   `GET {url}/ejbca/ejbca-rest-api/v1/ca/status`.
3. Reject the create/update with a wrapped error if:
   - any TLS handshake fails (server cert untrusted, client cert revoked or
     not bound to a role, expired, etc.),
   - the HTTP response is not 200 with `status == "OK"`.

### REQ-CA-EJBCA-7: Certificate enrollment

To enroll a certificate from an EJBCA CA, Fleet SHALL:

1. Generate a fresh RSA 2048 keypair in-memory.
2. Build a CSR with `CommonName` set to the expanded
   `username_template`. If `certificate_user_principal_names` is non-empty,
   include a `subjectAltName` extension on the CSR with one `otherName`
   entry per UPN (OID `1.3.6.1.4.1.311.20.2.3`, value type `UTF8String`,
   value = the expanded UPN string).
3. Generate a cryptographically-random 32-byte value, hex-encode it, and
   use it as the `password` field in the request body. This value is
   not persisted; it satisfies EJBCA's API requirement (the backend
   rejects null `password` for any CA with `useUserStorage=true`) and
   is otherwise transparent to Fleet's operation.
4. POST to `{url}/ejbca/ejbca-rest-api/v1/certificate/pkcs10enroll` with a
   JSON body containing the PEM CSR, profile names, expanded username, the
   generated password, `include_chain=false`, and `response_format="DER"`.
5. On HTTP 201: base64-decode the `certificate` field, parse as an X.509
   cert, wrap (key, cert) in a PKCS#12 with a randomly generated password,
   and return an `EJBCACertificate` containing the PFX bytes, password, not-
   before / not-after, and serial number.
6. On non-201: return a wrapped error containing EJBCA's `error_message`,
   with status code mapping per REQ-CA-EJBCA-9.

### REQ-CA-EJBCA-8: MDM profile variable substitution

Apple MDM configuration profiles SHALL support two new variable prefixes:

- `$FLEET_VAR_EJBCA_DATA_<ca_name>` — replaced with the base64-encoded
  PKCS#12 PFX data for the per-host enrolled certificate.
- `$FLEET_VAR_EJBCA_PASSWORD_<ca_name>` — replaced with the randomly
  generated PKCS#12 password.

`<ca_name>` is the EJBCA CA's `name` field. Per-host Fleet variables in
the CA's `username_template` SHALL be expanded against the receiving host
before each enrollment.

### REQ-CA-EJBCA-9: Error response mapping

The EJBCA REST client SHALL distinguish the following error classes in
wrapped error messages:

- HTTP 401 / 403 → "EJBCA rejected the Fleet client certificate (likely
  revoked, expired, or not bound to a role with sufficient access)"
- HTTP 404 → "EJBCA reports the CA or profile name does not exist"
- HTTP 422 → "EJBCA end-entity profile rejected the CSR: <error_message>"
- All other non-2xx → wrap EJBCA's `error_message` verbatim with the HTTP
  status code

### REQ-CA-EJBCA-11: Update semantics — no automatic re-enrollment

Updates to an existing EJBCA CA row (via PATCH or UI edit; GitOps apply
is deferred for the POC) SHALL NOT trigger re-enrollment of certificates
previously issued to hosts.
Existing host certificates remain installed and valid; the updated CA
configuration takes effect only on subsequent enrollments triggered by
ordinary profile-delivery events (new host assignment, profile
re-application, manual `ResendHostMDMProfile`, host MDM re-enrollment).

This matches the existing behavior for all other CA types in Fleet and is
intentional: cert validity at the protocol layer (WiFi server, VPN
gateway, etc.) is independent of Fleet's stored configuration, so a
rotation of Fleet's mTLS material or EJBCA-side profile names does not
require — and SHOULD NOT cause — fleet-wide reissuance.

### REQ-CA-EJBCA-12: Client certificate expiry surfacing

Fleet SHALL parse the `notAfter` date from the stored EJBCA client
certificate and expose it on:

- the certificate authority list response (`GET /fleet/certificate_authorities`)
  as a `client_cert_expires_at` field on each EJBCA CA entry
- the certificate authority list page in the UI as an "Expires in N days"
  badge alongside each EJBCA CA name

When `client_cert_expires_at` is less than 30 days in the future, the UI
SHALL render the badge with a warning visual style. Less than 7 days
SHALL render as an error style.

This requirement exists because once Fleet's mTLS client cert expires,
*new* enrollments silently stop while *existing* host certs continue to
work — a failure mode that is undetectable from device-side health and
must be surfaced administratively.

## Deferred (not in this change)

- OAuth 2.0 bearer-token authentication as an alternative to mTLS.
- PEM-direct upload of client cert + key (separate `.pem` files). POC is
  P12-only.
- A dedicated "replace credentials" UX action on the CA list page. In the
  POC, rotating the mTLS material is done via the standard edit modal —
  re-upload the P12, save. The backend already re-validates and re-persists.
- **User-configurable enrollment code.** POC generates the `password`
  field internally per-issuance (REQ-CA-EJBCA-7). A configurable
  optional field — for customers whose EJBCA End Entity Profile is
  configured to require a specific shared password, or whose workflow
  depends on a known shared secret — is a small follow-up: one optional
  field on the create/update payload, one if-empty branch in
  `GetCertificate`. Pull forward only if a customer confirms they need
  it (see proposal.md decision #5).
- **GitOps support for the EJBCA CA type.** POC ships API + UI only.
  The `certificate_authorities` GitOps spec, the
  `ValidateCertificateAuthoritiesSpec` parser, and
  `BatchApplyCertificateAuthorities` are NOT extended for EJBCA in this
  change. Follow-up alongside the production implementation will mirror
  the existing DigiCert / NDES / SCEP shapes, with the P12 carried as a
  base64-encoded string.
- **Pure-Go BER → DER normalization for the PKCS#12 decode path.** POC
  shells out to the `openssl` binary to convert EJBCA's BER-encoded P12
  to DER before parsing — both Go PKCS#12 libraries are strict-DER and
  reject EJBCA's output. Production (#30986) must replace this with a
  pure-Go normalizer so Fleet has no runtime dependency on the openssl
  binary. See research.md → "Open follow-ups" for design options.
- **Structural `additionalEJBCAValidation` for `.mobileconfig` uploads.**
  Mirrors `additionalDigiCertValidation`'s check that EJBCA Fleet
  variables only appear inside a `com.apple.security.pkcs12` payload,
  and that the Password / PayloadContent fields exactly match the
  expected variable patterns. POC accepts the uploaded profile as
  long as the DATA + PASSWORD pair is complete and the CA name
  exists (the framework hooks are already in place — see the fifth
  parameter `additionalEJBCAValidation` on
  `validateProfileCertificateAuthorityVariables`, currently nil).
- Windows, Linux, and Android cert delivery paths.
