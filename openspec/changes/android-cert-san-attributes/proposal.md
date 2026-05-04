## Why

IT admins managing Android hosts need to deliver certificates whose Subject Alternative Name (SAN) carries identifiers like RFC822/email,
UPN, DNS, or URI so end users can authenticate to corporate Wi-Fi (e.g. EAP-TLS that matches on UPN). Today Fleet only accepts a
`subject_name` for the Android certificate template, so any cert profile that requires a SAN cannot be issued from Fleet, blocking the
"connect end user to Wi-Fi with certificate" flow on Android. Tracked in issue #41472 for milestone 4.86.0.

## What Changes

PR #43318 already merged the REST API and YAML *documentation* for a new optional `subject_alternative_name` field on certificate
templates. This change implements the rest of the feature behind that contract:

- Persist `subject_alternative_name` on certificate templates (new nullable column + struct/spec field).
- Accept and validate `subject_alternative_name` on the create/update certificate template REST endpoints, including
  `$FLEET_VAR_HOST_*` variable expansion at delivery time.
- Return the rendered `subject_alternative_name` to the Android Fleet agent on the device-facing certificate-template endpoint
  (`/api/fleetd/certificates/{id}`), alongside the existing rendered `subject_name`.
- Update the **Android Fleet agent** (Kotlin source under `android/app/src/main/java/com/fleetdm/agent/` — separate from
  `fleetd`) to parse the rendered SAN string and include a non-critical SAN extension in the PKCS#10 CSR it submits to the
  SCEP CA. New parser converts the user-facing `"KEY=value, KEY=value"` format from the Figma into BouncyCastle
  `GeneralNames`. v1 covers the five SAN attribute types that real-world enterprise PKI actually deploys for the use cases in
  scope (Wi-Fi/EAP-TLS, internal mTLS, S/MIME, modern service identity): `DNS`, `EMAIL`, `UPN`, `IP`, `URI`. These map
  internally to X.509 `dNSName`, `rfc822Name`, `otherName` (UPN with OID `1.3.6.1.4.1.311.20.2.3` per Microsoft KB258605),
  `iPAddress`, and `uniformResourceIdentifier` respectively. Exotic types (`directoryName`, `registeredID`, `x400Address`,
  `ediPartyName`) remain out of scope — they are not used in modern enterprise authentication.
- Parse and apply `subject_alternative_name` from GitOps YAML (`controls.android_settings.certificates[]`).
- Emit `subject_alternative_name` from `fleetctl generate-gitops` for each certificate template.
- Add a SAN text input to the Add/Edit Certificate UI (Manage > Controls > OS Settings > Certificates), with the same validation
  shape as the documented format ("DNS=example.com, UPN=...").
- Change the Add/Edit Certificate modal's submit-button behavior to match the Figma's second dev note: the "Add" button is
  always enabled, and clicking it with required fields empty surfaces inline "field must be completed" errors against the
  empty fields (mirroring the "Add user" modal pattern shown in the Figma example screenshot). This replaces the current
  "button disabled until form valid" behavior. Note: this is a modal-wide UX change, not SAN-specific — it affects how the
  existing Name, Certificate authority, and Subject name fields surface validation errors too.
- Premium-only: feature stays gated to Fleet Premium on both backend and frontend (matches existing certificate-template behavior).
- Update the feature guide at https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#android-deploy-certificate.

Non-goals:

- iOS/macOS SAN support (Apple SCEP profile already supports SANs through the configuration profile payload — out of scope here).
- SAN attribute types beyond DNS, EMAIL, UPN, IP, URI. Exotic types (directoryName, registeredID, x400Address, ediPartyName)
  are out of scope — they are not used in modern enterprise authentication.
- Server-side validation of SAN *value content* (e.g. is the IP literal a valid IP address, does the URI parse, does the
  email have an `@`). Values can contain unexpanded `$FLEET_VAR_*` at create time, so server-side content checks would
  false-positive; value parsing belongs in the agent at delivery time. The server still performs **format-only** validation
  (token shape, KEY allow-list, variable allow-list, length cap) at create time — see "Lightweight server-side validation"
  in design.md, conditional on designer confirmation.
- Exposing the X.509 SAN-extension `critical` flag to admins. The agent always emits the SAN extension as **non-critical** per
  RFC 5280 §4.2.1.6 (subject DN is non-empty, so SHOULD non-critical) — admins do not configure this.
- Changes to `fleetd` (the cross-platform osquery agent). Android certificate delivery uses the Android Fleet agent in this
  repository's `android/` directory, which is *not* `fleetd` — those are different binaries with different release trains.
- Activity log additions (story explicitly says "no activity changes").

## Capabilities

### New Capabilities

- `android-cert-san`: Authoring, storage, GitOps round-trip, server-side variable expansion, and Android-agent CSR construction
  for the optional Subject Alternative Name on Android certificate templates, including Premium tier gating.

### Modified Capabilities

(None — no prior accepted spec covers Android certificate templates in `openspec/specs/`.)

## Impact

- Database: one additive migration adding `subject_alternative_name VARCHAR NULL` (or `TEXT NULL`) to `certificate_templates`.
- Backend types: `fleet.CertificateTemplate`, `fleet.CertificateTemplateSpec`, request/response structs in
  `server/service/certificates.go`, and the cert template datastore methods in `server/datastore/mysql/certificate_templates.go`.
- Variable expansion: extend `replaceCertificateVariables` (`server/service/certificate_templates.go`) to also expand
  `subject_alternative_name`.
- Device-facing endpoint: `CertificateTemplateResponseForHost` (and `GetDeviceCertificateTemplate`) carries the rendered SAN
  back to the Android agent. `server/mdm/android/service/service.go` (`BuildAndSendFleetAgentConfig`) and the
  `AgentCertificateTemplate` payload in `server/mdm/android/android.go` are unchanged in shape — the agent still fetches the
  full template by UUID, the new field just rides on that response.
- **Android Fleet agent (Kotlin, `android/app/src/main/java/com/fleetdm/agent/`)**:
  - `ApiClient.kt` — `GetCertificateTemplateResponse` data class gains `subjectAlternativeName: String?`.
  - `scep/ScepClientImpl.kt` — `buildCsr()` adds an `extensionRequest` attribute carrying the SAN extension when present.
  - New SAN parser (e.g. `scep/SubjectAlternativeNameParser.kt`) converting `"KEY=value, KEY=value"` to BouncyCastle
    `GeneralNames`, covering DNS, RFC822, URI, and UPN (UPN encoded as `OtherName` with OID `1.3.6.1.4.1.311.20.2.3` per
    Microsoft KB258605 / RFC 4556 §3.2.1).
  - Tests under `app/src/test/` (`ScepClientImplTest`, `CertificateEnrollmentHandlerTest`,
    `testutil/TestCertificateTemplateFactory`) and a new `SubjectAlternativeNameParserTest`.
  - No new third-party deps — BouncyCastle 1.78.1 (`bcprov-jdk18on` + `bcpkix-jdk18on`) is already on the classpath.
- GitOps: `pkg/spec/gitops.go` (parse/validate) and `cmd/fleetctl/generate_gitops.go` (export).
- Frontend: `frontend/pages/ManageControlsPage/OSSettings/cards/Certificates/components/AddCertificateModal/` (form, validation,
  helpers) and the corresponding API client typings.
- Docs: `docs/Configuration/yaml-files.md` and `docs/REST API/rest-api.md` are already updated by PR #43318. The user-facing feature
  guide at fleetdm.com/guides/connect-end-user-to-wifi-with-certificate must be updated. Android agent's `CHANGELOG.md` should
  note the SAN-extension behavior change.
- Tests: integration tests for the certificate template CRUD endpoints, GitOps round-trip test in `cmd/fleetctl`, frontend Jest
  tests for the modal, and Kotlin unit tests for the SAN parser, the CSR builder's SAN extension, and the enrollment handler.
- Release coordination: **Android agent ships first** (via the agent's release train, typically Google Play). Once the new
  agent is rolled out broadly, the Fleet server change ships and the SAN UI/YAML/API surface is exposed. Shipping the server
  first would silently break Wi-Fi auth for any admin who sets a SAN value while older agents are still in the field — those
  agents tolerate the new JSON field (no crash) but strip the SAN out of the CSR, producing certs without the requested SAN
  extension. The old certs continue to work as before, but any new cert with a SAN-bearing template would fail to authenticate.
  See design.md Migration Plan for the gated rollout.
- Risk: Low. No load testing required. Premium-only — both backend service method and frontend form must check tier.
