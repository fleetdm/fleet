# Proposal: EJBCA REST CA integration (POC)

## Summary

Add EJBCA as a new Certificate Authority type in Fleet's CA framework, enrolling
certificates via EJBCA's REST API over mTLS. The integration mirrors the existing
DigiCert CA in shape (CSR-based enrollment, PKCS12-wrapped delivery via MDM profile
variable substitution) but uses client-certificate authentication instead of a
static API token, since EJBCA does not offer a static API key.

This proposal scopes the **POC**: a working end-to-end path that exercises every
new piece of plumbing once, against a real EJBCA instance, with a documented
customer setup flow. Production polish (rotation UX, audit-grade error messages,
full test coverage, OAuth bearer auth) is deferred to the follow-up implementation
captured in [fleet#30986](https://github.com/fleetdm/fleet/issues/30986).

## Why

- **Customer commitment.** [`prospect-homerus`](https://github.com/fleetdm/fleet/issues/29176)
  is migrating from AD CS to EJBCA and wants Fleet to deploy WiFi/VPN certificates
  to macOS hosts. Tagged `~customer promise`.
- **Milestone**: 4.87.0 (POC, this proposal). The customer-facing implementation
  is 4.88.0 (#30986, T-shirt M).
- **Strategic fit.** EJBCA is one of the dominant non-AD-CS enterprise CAs.
  Supporting it removes a recurring blocker for prospects with PKI teams.
- **Reuses existing CA framework.** Endpoints, DB schema, and MDM
  profile-variable substitution are already in place from the DigiCert
  work. Net new surface is small. (GitOps support exists in the framework
  but is deferred for the EJBCA POC.)

## Scope

### In scope

- New `CertificateAuthority` type `ejbca` with mTLS authentication.
- EJBCA REST client supporting two operations: connection verification
  (`GET /v1/ca/status`) and certificate enrollment (`POST /v1/certificate/pkcs10enroll`).
- **Feature parity with DigiCert's certificate-templating surface.** The
  EJBCA CA accepts the same per-enrollment templating fields the DigiCert
  CA does today:
  - `username_template` (used as both the EJBCA end-entity username AND the
    CSR CommonName — parallels DigiCert's `certificate_common_name`)
  - `certificate_user_principal_names []string` — UPNs included in the CSR's
    SAN extension as `otherName` entries (OID 1.3.6.1.4.1.311.20.2.3). Matches
    what DigiCert's integration supports.
  Other SAN types (DNS, email, IP) are out of scope — DigiCert doesn't
  template them today, so EJBCA matches.
- **Apple MDM payload coverage matches DigiCert's**: same payload types
  Fleet supports for DigiCert today (Wi-Fi EAP-TLS, VPN). No new MDM
  payload work in this change.
- **Positioning vs the existing SCEP-EJBCA path.** Fleet's custom-SCEP-proxy
  CA already works against EJBCA's SCEP endpoint. The REST integration is an
  **alternative** path for customers who prefer or require the REST API —
  not a migration or replacement. Customers can choose either or both. This
  is intentionally stated to avoid product-positioning confusion on review.
- DB migration adding columns for the client cert, private key, and optional
  trust CA bundle (encrypted at rest).
- Validation on CA create/update: probe the EJBCA endpoint and confirm
  authentication succeeds.
- MDM profile variable substitution: `FLEET_VAR_EJBCA_DATA_<name>` and
  `FLEET_VAR_EJBCA_PASSWORD_<name>` mirroring the DigiCert prefix pattern.
- Minimal frontend form to create/edit/delete an EJBCA CA (parity with the
  DigiCert form, not polished UX).
- Developer-facing dev guide for running the POC against a local EJBCA Community
  Edition Docker container, as a sibling to the existing
  `docs/Contributing/product-groups/security-compliance/ejbca-scep-testing.md`.

### Out of scope (deferred)

- OAuth 2.0 bearer-token authentication. EJBCA supports it (see
  [research.md](./research.md)) but adds an IdP dependency and meaningful new
  surface area. Note in spec for follow-up. Likely shares code with Fleet's
  existing SSO/OIDC support.
- PEM-direct client cert/key upload (separate `.pem` cert + `.pem` key files).
  POC accepts PKCS#12 only. PEM-direct is a small follow-up — same backend,
  just a different upload path — and can be pulled in later if customers
  request it.
- User-configurable enrollment code. POC generates the EJBCA `password`
  field internally per issuance. A customer-supplied static value (for
  workflows that require it — see proposal decision #5) is a small follow-up.
- **GitOps support for the EJBCA CA type.** POC does not extend the
  `certificate_authorities` GitOps spec, the `ValidateCertificateAuthoritiesSpec`
  parser, or `BatchApplyCertificateAuthorities`. Follow-up work — same
  pattern as the existing DigiCert / NDES / SCEP types — to be picked up
  alongside the production implementation in #30986. POC scope is the
  API + UI path only.
- **In-process BER → DER normalization for the PKCS#12 decode path.**
  POC shells out to the `openssl` binary to convert EJBCA's BER P12
  output before parsing (both Go PKCS#12 libraries are strict-DER). The
  subprocess approach **must not ship to production** — it adds a
  runtime binary dependency and broadens the trust surface. Production
  (#30986) must replace it with a pure-Go BER → DER normalizer. See
  research.md → "Open follow-ups" for design options.
- Distinct error messages for cert-revoked vs cert-expired vs role-misconfigured
  TLS handshake failures (POC returns a generic "TLS handshake failed" with
  the wrapped server error; production tightens this).
- A dedicated "Replace credentials" button on the CA list. In the POC, rotation
  happens through the regular edit modal — re-upload the P12, save. No
  separate action surface.
- Comprehensive test suite. Per the POC's learning intent, write tests only
  where they accelerate iteration, not for coverage.
- Activity-log entries beyond what reusing the generic CA activity wiring gives
  for free.
- Windows / Linux / Android cert delivery paths. macOS only (matches #30986's
  scope and the customer's stated need).
- Production-grade frontend polish, file-format validation UX, drag-and-drop
  upload, etc.

## Auth decisions requiring customer confirmation

These are the decisions that materially affect the customer's EJBCA admin work
and the field shape of Fleet's UI. They cannot be answered from EJBCA docs alone
— each is a per-customer choice. **Confirm with `prospect-homerus` before
implementation begins.**

1. **mTLS as the auth method for the POC.** ✅ Confirmed with customer.
   - Decision: client-certificate authentication, no OAuth bearer in the POC.
   - Customer impact: their EJBCA admin must enroll a service certificate for
     Fleet, bind it to an Administrator Role with appropriate access rules, and
     hand Fleet a `.p12` bundle (with its password) plus the issuing CA chain.
   - Status: customer confirmed mTLS is their preferred auth method.
     OAuth bearer remains a real follow-up (see research.md).

2. **Client cert input format — P12 only in POC.**
   - Decision: the POC accepts **PKCS#12 only** (file + password). PEM-direct
     upload (separate cert + key files) is deferred. P12 is what EJBCA's RA
     Web emits by default, so this is the friendlier first-time path and
     matches the expected customer workflow.
   - Customer question: confirm this is acceptable. Does their security team
     require PEM-direct upload (e.g., to avoid handling P12 passwords, or
     because their secrets tooling exports PEM)? If yes, raise PEM-direct
     out of "deferred" and into the POC.

3. **Trust CA bundle for EJBCA's server cert.**
   - Proposed: optional field. If empty, Fleet uses its system root store.
   - Customer question: is EJBCA fronted by a publicly-trusted TLS cert in
     their environment, or is it Management-CA-issued / self-signed? If the
     latter, they must provide the CA chain.

4. **End entity auto-creation requirement.**
   - Proposed: Fleet assumes their EJBCA is configured to auto-create end
     entities during `pkcs10enroll`. This requires their End Entity Profile
     to permit auto-add (EJBCA Enterprise feature) or for them to pre-create
     end entities per host (CE limitation, untenable at scale).
   - Customer question: confirm they will run EJBCA Enterprise in production
     and that their EE profile is configured to auto-create. If they're on
     Community Edition or unwilling to enable auto-create, the integration is
     non-viable for fleet-scale provisioning.

5. **Enrollment code / password handling.**
   - Decision: Fleet generates a cryptographically-random `password` per
     enrollment internally and never persists it. EJBCA's `pkcs10enroll`
     API requires a non-null password (the backend rejects null for any
     CA with `useUserStorage=true`), but in the typical auto-create-EE
     configuration the value isn't authenticating anything — the mTLS
     client cert + EJBCA role scope is the actual security boundary.
     Generating internally satisfies the API requirement without adding a
     persistent shared secret on Fleet's side or a customer-facing
     configuration field.
   - Customer question: are they planning to use a workflow that
     *requires* a specific configurable enrollment code? Examples:
     - Their EE profile is configured to validate the password against a
       specific server-side value rather than accepting any
     - They want EJBCA audit logs to contain a recognizable
       "fleet-integration" string rather than per-call random values
     - Internal governance requires explicitly-configured integration
       credentials
     If yes for any of these, pull forward the deferred "user-configurable
     enrollment code" feature from the spec's Deferred section — small
     follow-up, doesn't block the POC.

6. **Least-privilege admin role.**
   - Proposed: encourage the customer to bind the Fleet service cert to a
     **custom** EJBCA admin role with narrow access rules (read on issuing CA,
     create/edit on relevant EE profile, issue on cert profile) rather than
     Super Administrator.
   - Customer question: are they willing to set up a custom role, or will
     their PKI admin prefer Super Administrator for simplicity? This is a
     security-posture conversation, not a Fleet feature flag.

7. **Cert rotation cadence.**
   - Mechanism (POC): the standard edit modal on the CA accepts a new P12 +
     password. Saving re-validates against EJBCA and replaces the stored
     PEM cert+key. Existing host certificates are unaffected — they remain
     installed on devices and valid (per REQ-CA-EJBCA-11). Next enrollment
     uses the new mTLS material.
   - Fleet shows "expires in N days" on the CA list (REQ-CA-EJBCA-12) so the
     admin sees rotation coming and doesn't get caught by a silent stoppage
     of new enrollments.
   - Customer question: what is their typical service-cert lifetime (1y is
     usual)? Are they comfortable with the standard "edit modal → re-upload
     P12" rotation workflow?

These are the questions for the customer call. Capture answers in
[research.md](./research.md) → "Decisions confirmed with customer".

## Success criteria for the POC

The POC is complete when all of these are demonstrably true on a developer
machine:

1. ✅ A Fleet admin can create an EJBCA CA via API and via the UI, supplying
   URL, client cert/key, profile names, username template, and password.
   *Verified end-to-end against `keyfactor/ejbca-ce` 9.x.*
2. ✅ Fleet rejects the CA on save if the connection probe fails (bad URL,
   wrong client cert, revoked cert, untrusted server cert).
   *Verified — each failure mode surfaces a distinct error message; see the
   troubleshooting ladder in the dev guide.*
3. ✅ A macOS host with a configuration profile referencing
   `FLEET_VAR_EJBCA_DATA_<name>` / `FLEET_VAR_EJBCA_PASSWORD_<name>` receives
   a working certificate provisioned through EJBCA.
   *Verified — host installs the issued cert; EJBCA logs show the
   `pkcs10enroll` request and `CERT_CREATION` for the host's CN.*
4. ✅/🔲 The CA list page shows the client-cert expiry for each EJBCA CA,
   and a re-upload of the P12 via the edit modal rolls the stored cert+key
   without disturbing existing host certificates. *Backend exposes
   `client_cert_expires_at`; CA list badge UI was deferred from the POC
   (see Phase 8 in tasks.md). Edit-modal P12 rotation is wired and
   functional.*
5. ✅ The dev guide for running the POC against a local EJBCA Community
   Edition container exists in
   `docs/Contributing/product-groups/security-compliance/` and an engineer
   who has never touched EJBCA can follow it from scratch.
   *Verified — guide includes a Troubleshooting ladder mapping every
   failure mode hit during testing to its fix.*
6. The customer's open questions (above) are captured with answers in
   research.md.
