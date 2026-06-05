# Tasks: EJBCA REST CA integration (POC)

Implementation checklist for the POC. Tasks are listed in the order they
unblock each other. Skip tests except where they accelerate iteration —
this is a learning POC, not a production ship.

## 0. Pre-implementation

- [ ] Get customer confirmation on the seven auth decisions in
      [proposal.md → "Auth decisions requiring customer confirmation"](./proposal.md#auth-decisions-requiring-customer-confirmation).
      Capture answers in [research.md → "Decisions confirmed with customer"](./research.md#decisions-confirmed-with-customer).
- [ ] Stand up a local EJBCA Community Edition Docker container per the
      existing
      [ejbca-scep-testing.md](../../../docs/Contributing/product-groups/security-compliance/ejbca-scep-testing.md)
      Docker section. We'll reuse the container.

## 1. EJBCA-side dev setup (one-time, in your local EJBCA)

- [ ] Create a `fleetRESTAdmin` Certificate Profile (CA Functions →
      Certificate Profiles): clone ENDUSER, EKU = Client Authentication.
- [ ] Create a `fleetRESTAdmin` End Entity Profile (RA Functions → End Entity
      Profiles), default cert profile = above.
- [ ] Enroll a service certificate via RA Web → Make New Request. Username
      `fleet_rest_service`. Download as PKCS#12 with a known password.
- [ ] Create an Administrator Role bound to that cert (System Functions →
      Roles and Access Rules). Use RA Administrators template, narrow Access
      CAs and EE profiles to just what the enrollment requires.
- [ ] Add the cert to the role's Members: Match X509:CN = "Fleet REST Service",
      CA = Management CA.
- [ ] Export the Management CA cert as PEM (RA Web → CA Certificates and
      CRLs). This becomes the trust bundle for the POC.
- [ ] Verify `curl --cert fleet_rest_service.p12 ...` against
      `https://localhost:8443/ejbca/ejbca-rest-api/v1/ca/status` returns 200.

## 2. Backend types

- [x] Add `EJBCACA` struct to `server/fleet/certificate_authorities.go`
      (mirror `DigiCertCA`).
- [x] Add `CATypeEJBCA = "ejbca"` constant.
- [x] Extend `CertificateAuthority` polymorphic struct with EJBCA pointer
      fields.
- [x] Extend `CertificateAuthorityPayload` and `CertificateAuthorityUpdatePayload`
      with `EJBCA *EJBCACA`.
- [x] Add `Equals`, `Preprocess`, and `EJBCACAUpdatePayload.Validate` methods.
- [x] Add `EJBCAService` interface and `EJBCACertificate` struct in
      `server/fleet/ejbca.go` (new file).
- [x] Add `FleetVarEJBCADataPrefix` and `FleetVarEJBCAPasswordPrefix`
      constants in `server/fleet/mdm.go` and add both to the allow-list at
      `mdm.go:99`.
- [x] Add a `ClientCertExpiresAt time.Time` field to `EJBCACA` (omitempty
      JSON). Populated server-side at read time by parsing `notAfter` from
      the stored PEM client cert. Used by the frontend to render the
      "Expires in N days" badge (REQ-CA-EJBCA-12).
- [x] Add a `CertificateUserPrincipalNames []string` field to `EJBCACA`
      and `EJBCACACreatePayload` (optional, json omitempty). Mirrors
      DigiCert's same-named field.
- [x] Add `ActivityAddedEJBCA`, `ActivityEditedEJBCA`,
      `ActivityDeletedEJBCA` activity types in `server/fleet/activities.go`
      (mirror the existing Hydrant/Smallstep types). Wiring into the
      service methods happens in Phase 5.

## 3. HTTP client package

- [x] Create `ee/server/service/ejbca/ejbca.go`.
- [x] Implement `buildTLSClient` helper that builds a `tls.Config` from
      `ClientCertPEM` + `ClientKeyPEM` + optional `TrustCABundlePEM`. Pass
      via `fleethttp.WithTLSClientConfig` (already exists in
      `pkg/fleethttp/fleethttp.go:36`).
- [x] Implement `VerifyConnection`: `GET /ejbca/ejbca-rest-api/v1/ca/status`,
      decode `{"status":"OK"}`.
- [x] Implement `GetCertificate`: generate RSA 2048, build CSR with CN =
      username, POST `pkcs10enroll`, decode base64-DER cert, wrap in PKCS12,
      return `EJBCACertificate`.
- [x] In `GetCertificate`, generate a 32-byte cryptographically-random
      value (via `crypto/rand`) hex-encoded as the `password` field on
      each `pkcs10enroll` call. Do not persist it. Required because
      EJBCA's backend rejects null `password` for any CA with
      `useUserStorage=true` (verified in `SignSessionBean.java`).
- [x] In `GetCertificate`, if `CertificateUserPrincipalNames` is non-empty,
      attach a `subjectAltName` extension to the CSR with one `otherName`
      per UPN (OID `1.3.6.1.4.1.311.20.2.3`, value type UTF8String).
      Construct via raw ASN.1 — Go's stdlib doesn't have first-class UPN
      otherName support. Unit-tested via `ejbca_test.go`.
- [x] Error distinction: 401/403 → "EJBCA rejected the Fleet client cert
      (revoked or not bound to a role with sufficient access)". 404 → "CA or
      profile name not found in EJBCA". 422 → "EE profile rejected the CSR".
      Other → wrap message verbatim from EJBCA's `error_message`.

## 4. Datastore + migration

- [x] Write migration adding columns (per [design.md → Migration](./design.md#migration)).
- [x] Extend `type` ENUM with `'ejbca'`.
- [x] Update insert / select / encrypt / decrypt helpers in
      `server/datastore/mysql/certificate_authorities.go`.
- [x] Update `GroupCertificateAuthoritiesByType` in
      `server/fleet/certificate_authorities.go`.
- [x] Update `postprocessRetrievedCertificateAuthority` to mask
      the encrypted PEM private key when `includeSecrets=false`.
      Also parses `notAfter` from the stored client cert PEM into
      `ClientCertExpiresAt` for REQ-CA-EJBCA-12.
- [x] After datastore method additions, run `go test ./server/service/`
      to confirm mocks aren't broken. No new datastore interface
      methods were added — only SQL queries and helpers — so mocks
      are unaffected.

## 5. Service layer wiring

- [x] Add `ejbcaService fleet.EJBCAService` to the EE service struct;
      inject in `cmd/fleet` setup. Also added to the core service.Service
      and threaded through svctest + testing_utils + mdm_external_test
      callers.
- [x] Implement `validateEJBCA(payload)`: name + URL + profile-name
      checks, P12 + password required, decode P12 once (extract cert +
      key as PEM, discard the bundle and password), trust bundle parse
      if provided, Fleet-var allow-list check on username_template + UPNs,
      probe EJBCA via `VerifyConnection`.
- [x] Wire `NewCertificateAuthority` EJBCA branch: validate (which
      includes the VerifyConnection probe) → set caToCreate fields →
      datastore insert → activity log.
- [x] Wire `UpdateCertificateAuthority` EJBCA branch: validateEJBCAUpdate
      decodes any new P12 and probes the merged (new + existing) config
      before persistence. Returns the decoded PEM cert/key for the
      caller to plumb into caToUpdate.
- [x] Wire `DeleteCertificateAuthority` EJBCA branch (ActivityDeletedEJBCA).
- [x] Skip GitOps for the POC. `BatchApplyCertificateAuthorities` and
      `ValidateCertificateAuthoritiesSpec` are NOT extended for EJBCA in
      this change — see proposal.md "Out of scope" and design.md "GitOps
      — deferred". Follow-up alongside the production implementation.

## 6. MDM profile processor

- [ ] Add EJBCA branches in `server/mdm/apple/profile_processor.go`:
      - `isCAConfigured` → recognize `FLEET_VAR_EJBCA_*` prefixes.
      - validation phase → ensure the named CA exists.
      - expansion phase → per host, deep-copy CA, expand
        `UsernameTemplate`, call `ejbcaService.GetCertificate`, substitute
        `FLEET_VAR_EJBCA_DATA_<name>` (base64 PFX) and
        `FLEET_VAR_EJBCA_PASSWORD_<name>` in the profile XML.

## 7. Endpoints

- [ ] Confirm `request_certificate` endpoint dispatches correctly to EJBCA
      via type switch (it should — endpoint is generic).
- [ ] Smoke-test all five CRUD endpoints (`POST`, `GET list`, `GET id`,
      `PATCH`, `DELETE`) with EJBCA payloads.

## 8. Frontend (minimal POC)

- [ ] Create `frontend/pages/admin/IntegrationsPage/cards/CertificateAuthorities/components/EJBCAForm/` —
      mirror `DigicertForm.tsx`.
- [ ] Add EJBCA option in `AddCertAuthorityModal`.
- [ ] Add EJBCA branch in `EditCertAuthorityModal`.
- [ ] Form accepts **only** PKCS#12 upload + password. Reject any other
      filetype client-side with a clear "EJBCA expects a .p12 file" message.
- [ ] Server-side: decode the P12 with the supplied password using
      `software.sslmate.com/src/go-pkcs12`, extract cert + key, persist as
      PEM (key encrypted). Discard the P12 password — it is never stored.
- [ ] Edit modal exposes the same fields as create. Re-uploading a P12 in
      edit mode replaces the stored cert+key (the rotation workflow — no
      separate "Replace credentials" button in the POC).
- [ ] On the CA list page, parse `not_after` from the stored EJBCA client
      cert and render an "Expires in N days" badge per REQ-CA-EJBCA-12.
      Warning visual <30 days; error visual <7 days.
- [ ] No polish work — just enough for an end-to-end manual test.

## 9. Dev guide

- [ ] Write `docs/Contributing/product-groups/security-compliance/ejbca-rest-testing.md`
      as a sibling to the existing SCEP guide. Cover:
      - Reuse of the same Docker container.
      - The one-time EJBCA-side setup from §1 above.
      - How to create the EJBCA CA in Fleet (curl examples for both API and
        the create-via-UI flow).
      - End-to-end: create CA → push a profile with `FLEET_VAR_EJBCA_*` to
        an enrolled host → verify the cert installs.
      - Gotchas:
        - CE requires pre-creating end entities (covered by reusing the
          existing SCEP guide's `bin/ejbca.sh ra addendentity` snippet);
          EE auto-creates if the EE profile permits.
        - **"Allow Extension Override" must be checked** on the
          Certificate Profile if you use UPN templating — otherwise EJBCA
          drops the SAN extension during issuance. Same gotcha as the
          existing SCEP guide.
        - GitOps is deferred for the POC — the dev guide covers the API
          and UI paths only. (Fleet generates the EJBCA `password`
          internally per enrollment; there is no enrollment-code field
          anywhere in the POC surface.)

## 10. Manual verification (POC success criteria)

- [ ] All five proposal success criteria are demonstrably true on a
      developer machine.
- [ ] Customer auth-decision questions are answered and captured.
- [ ] Run the full new dev guide from scratch on a clean machine.

## 11. Skipped on purpose (call out in PR description)

- Tests: write only what helps debug. No comprehensive coverage.
- Activity logging beyond what the generic CA wiring gives for free.
- Cert-expiry surfacing, in-place rotation UX, distinct error UI.
- OAuth bearer auth (file as follow-up issue referencing #30986).
- Windows / Linux / Android cert delivery paths.

## 12. Linting + final hygiene

- [ ] `make lint-go-incremental` — clean.
- [ ] `make lint-js` — clean.
- [ ] `go test ./server/fleet/... ./server/service/...` — at minimum no
      mock crashes.
- [ ] PR description: link to this proposal, list what was skipped, note
      open questions for the customer call.
