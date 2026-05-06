## Context

Fleet currently delivers SCEP-issued client certificates to Android hosts so end users can authenticate to corporate Wi-Fi. The
certificate template (`fleet.CertificateTemplate`, table `certificate_templates`) carries a `subject_name` (e.g.
`"/CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID"`) which is rendered per-host at delivery time and then signed
by a custom SCEP CA. Many real-world EAP-TLS deployments match the user identity in a Subject Alternative Name (UPN, RFC822, DNS,
URI) instead of (or in addition to) the subject DN, so without SAN support the cert is rejected by the network.

PR #43318 already merged the user-facing contract: `subject_alternative_name` (singular string, comma-separated `KEY=value` pairs)
on the REST API and YAML for the certificate template. None of the corresponding backend, frontend, GitOps, or device-delivery
plumbing has shipped yet. The acceptance criterion is that Fleet can deliver the SANs in the test certs in the linked Google Drive
folder for issue #41472.

## Goals / Non-Goals

**Goals:**

- Round-trip the new optional `subject_alternative_name` field end-to-end: REST create -> DB -> device-facing API -> Android
  cert delivery, plus GitOps apply, GitOps generate-gitops, and the Add/Edit Certificate UI.
- Apply the same `$FLEET_VAR_HOST_*` expansion to `subject_alternative_name` that already runs on `subject_name`, with the same
  error semantics when a host lacks the data needed for an expansion.
- Keep the feature gated to Fleet Premium on both backend and frontend, matching the existing certificate template feature.
- Migrate cleanly: existing certificate templates have no SAN today, so the column is nullable and the behavior with NULL must
  match today's behavior exactly (no SAN included in the CSR).

**Non-Goals:**

- iOS/macOS SAN support — Apple SCEP profiles already accept SANs through the Apple-side configuration profile payload.
- SAN types beyond `DNS`, `EMAIL`, `UPN`, `IP`, `URI`. Exotic types (`directoryName`, `registeredID`, `x400Address`,
  `ediPartyName`) are not used in modern enterprise authentication and stay out of scope.
- Server-side validation of the SAN string syntax. The Figma dev note explicitly says the server "won't validate SAN, will let
  it through, and if it fails, surface error on the host details > OS settings modal." Variable allow-list checks remain
  (shared with `subject_name`), but no KEY=value parsing happens server-side.
- Exposing the X.509 `critical` bit on the SAN extension to admins. The agent always emits SAN as non-critical (see "Android
  agent: parse SAN string and add SAN extension to PKCS#10 CSR" below).
- Activity log entries for SAN-specific events (the story explicitly says "no activity changes").
- Changes to `fleetd` (the cross-platform osquery agent). Android certificate delivery uses the **Android Fleet agent** in this
  repository's `android/` directory, which is a different binary from `fleetd`. The Android agent IS in scope for this change
  and is co-located in the same repo, but it ships from its own release train.
- A second (plural) field. The merged contract is the singular `subject_alternative_name`; we do not split it into a JSON array.

## Decisions

### Field name and shape: stay with the merged contract

PR #43318 documented `subject_alternative_name` (singular string, comma-separated `KEY=value` tokens). The story title and one
checkbox say "subject alternative names" (plural), but the merged docs are the source of truth.

- **Decision:** The persisted column, Go struct field, JSON field, YAML field, and frontend form field are all named
  `subject_alternative_name` / `SubjectAlternativeName`, matching `subject_name` exactly in shape (single string).
- **Alternative considered:** A `[]SAN{Type, Value}` array. Rejected because (a) it diverges from the already-merged docs, (b) the
  same comma-separated format is already accepted in `subject_name`, and (c) a single string survives variable expansion with the
  exact same code path.

### Variable expansion: extend the existing helper, not duplicate it

`Service.replaceCertificateVariables` (`server/service/certificate_templates.go:36`) already expands the supported
`$FLEET_VAR_HOST_*` set inside `subject_name`. The function is parameterized by the input string, so we apply it to
`subject_alternative_name` with no signature change. Validation (`validateCertificateTemplateFleetVariables`) is similarly applied
to both inputs at create time, so unsupported variables are rejected up front rather than at delivery.

This is not greenfield for Android: the integration test
`server/service/integration_android_certificate_templates_test.go` (`TestCertificateTemplateNoTeamWithIDPVariable`) already
exercises `subject_name = "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME"` end-to-end on Android, including the failure path when
the host has no associated IdP user. The IdP-to-Android-host association is wired during the Android enrollment-token flow at
`server/mdm/android/service/pubsub.go:597` (`AssociateHostMDMIdPAccount`). Extending expansion to SAN reuses this proven path.

- **Decision:** Reuse the helper. The set of supported variables is identical for SN and SAN. Today that set
  (`fleetVarsSupportedInCertificateTemplates`) is exactly three names: `HOST_UUID`, `HOST_HARDWARE_SERIAL`,
  `HOST_END_USER_IDP_USERNAME`. Other globally-defined variables — `HOST_PLATFORM`, `HOST_END_USER_IDP_USERNAME_LOCAL_PART`,
  `HOST_END_USER_IDP_GROUPS`, `HOST_END_USER_IDP_DEPARTMENT`, `HOST_END_USER_IDP_FULL_NAME`, and the legacy
  `HOST_END_USER_EMAIL_IDP` — are NOT accepted in either field.
- **Alternative considered:** Tracking SN-only vs SAN-only allowed variables. Rejected — there is no current product reason to
  diverge, and unifying keeps the contract simpler. If a customer needs a SAN-specific variable later (see Open Questions),
  add it to the shared list and both fields gain it together.

### Storage shape: nullable column, mirror `subject_name`'s type

- **Decision:** New migration adds `subject_alternative_name TEXT NULL`, matching the existing `subject_name TEXT` column on
  `certificate_templates` (verified in `server/datastore/mysql/migrations/tables/20251124140138_CreateTableCertifcatesTemplates.go`
  and `server/datastore/mysql/schema.sql`). NULL means "no SAN", which is the existing default behavior. The spec's 4096-byte
  length cap is enforced at the service layer (see "Lightweight server-side validation"), not by the column type — `TEXT`
  comfortably accommodates 4096 bytes plus headroom.
- The Go struct field is a plain `string` with `json:"subject_alternative_name,omitempty"`. The JSON response deterministically
  omits the key when empty/NULL (per `omitempty`). On the request side, both an omitted key and an empty string deserialize
  to `""` and store as NULL.
- **Whitespace policy:** `strings.TrimSpace(value) == ""` -> store NULL. Non-empty values are stored verbatim — no per-token
  trimming, no leading/trailing-whitespace mutation. This preserves admin intent and keeps GitOps idempotent (no churn). The
  Android agent applies its own whitespace tolerance at parse time (see the agent decision below); the two layers are
  independent.
- **Alternative considered:** A separate join table (one row per SAN attribute). Rejected — the format is already a single
  human-authored string, not structured data; the parsing happens once at delivery time.

### Variable expansion failure semantics on SAN match SN

If `subject_name` references `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` and the host has no IdP username, delivery fails today. Same
error must fire for SAN.

- **Decision:** `replaceCertificateVariables` is called for SAN with the same error-wrapping. The cert template moves to
  `failed` state and a delivery-failure path identical to the existing one.
- **Alternative considered:** Best-effort expansion (drop unresolved tokens). Rejected — this would silently issue certs that
  could let a misconfigured host onto Wi-Fi as the wrong identity.

### Device-facing response: extend the existing endpoint, do not add a new one

The Android Fleet agent (Kotlin source under `android/app/src/main/java/com/fleetdm/agent/`) fetches
`CertificateTemplateResponseForHost` from `/api/fleetd/certificates/{id}` to get the per-host rendered subject name, the SCEP
challenge, etc. We add `SubjectAlternativeName` to that struct (same JSON tag), so the existing endpoint returns the rendered SAN
alongside the rendered SN. No new endpoint is needed.

- **Decision:** Add `SubjectAlternativeName string` (with `json:"subject_alternative_name,omitempty"`) to
  `CertificateTemplateResponse` and its embedded summary / per-host structs. The agent's `GetCertificateTemplateResponse` data
  class in `ApiClient.kt` gains a matching `@SerialName("subject_alternative_name") val subjectAlternativeName: String? = null`.
- Backwards-compat: the agent has historically tolerated unknown fields (kotlinx.serialization with `ignoreUnknownKeys = true`),
  so a new server can ship before a new agent is rolled out without breaking older agents.
- The Android agent — not the server — owns adding the SAN extension to the CSR. That logic lives in this repo (see Decision
  9), so we control both halves.

### Android agent: parse SAN string and add SAN extension to PKCS#10 CSR

The agent must convert the rendered SAN string ("DNS=example.com, UPN=marko@corp.example.com") into BouncyCastle
`GeneralNames` and attach it to the CSR as an `extensionRequest` attribute (PKCS#9 OID `1.2.840.113549.1.9.14`). Today
`scep/ScepClientImpl.kt:168-179` (`buildCsr`) uses `JcaPKCS10CertificationRequestBuilder` and adds only the challenge-password
attribute. We extend it to also add the extensionRequest when SAN is non-empty.

- **Decision:** Introduce `scep/SubjectAlternativeNameParser.kt` (or equivalent) with a single public function
  `parse(sanString: String): GeneralNames?`. Behavior:
  - Returns `null` for empty / whitespace-only input — caller skips adding the extension.
  - Splits on `,` (no quoting support; matches the simple parsing the docs and `subject_name` already use).
  - Trims whitespace around each token.
  - Splits each token on the first `=`. Left side is the KEY (case-insensitive, normalized to upper); right side is the value
    (kept verbatim, so it can include `:` for URIs and `@` for emails). **The agent owns case normalization end-to-end:** the
    server validates the KEY allow-list case-insensitively but **persists admin input verbatim** (so an admin who types
    `dns=example.com` sees the same casing back via REST GET / GitOps export), and the agent's parser uppercases the KEY at
    parse time. This keeps storage faithful to admin intent and means a future case-related contract change lives in one place
    (the parser) rather than as a hidden server-side rewrite.
  - Maps KEY to `GeneralName` per the table below. The KEY names match what the Figma exposes to admins
    (`UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, EMAIL=$FLEET_VAR_HOST_END_USER_IDP_USERNAME` is the modal placeholder), and
    cover the full set of SAN types used for EAP-TLS / Wi-Fi authentication. Anything else throws `IllegalArgumentException`
    with a message naming the offending KEY — caller wraps it as a `ScepCsrException` so the cert state goes to `failed` and
    the failure surfaces in the host's "OS settings" modal per the Figma dev note ("if it fails, surface error on the host
    details > OS settings modal"), rather than silently issuing a cert without the requested SAN.

  Sourcing rationale for the v1 KEY set: per the Figma dev note ("Support all available fields for SAN"), v1 covers the SAN
  types that real-world enterprise PKI actually deploys. RFC 5280 §4.2.1.6 defines nine `GeneralName` choices, but four are
  effectively dead in modern enterprise authentication (`directoryName`, `registeredID`, `x400Address`, `ediPartyName`). The
  remaining five are the ones admins actually use:

  | KEY | `GeneralName` tag | BC tag # | Where it shows up in real deployments |
  |---|---|---|---|
  | `DNS` | `dNSName` | 2 | Server certs, mTLS, EAP-TLS by hostname. Effectively universal. Already in the merged Fleet REST API docs example. |
  | `EMAIL` | `rfc822Name` | 1 | User certs, S/MIME, EAP-TLS where the username is an email. In the Figma placeholder. |
  | `UPN` | `otherName` (OID `1.3.6.1.4.1.311.20.2.3`) | 0 | Active Directory-integrated EAP-TLS, NPS, Intune, smart card login. The de facto identity for Microsoft-shop Wi-Fi. In the Figma placeholder and merged docs. |
  | `IP` | `iPAddress` | 7 | Internal services, IoT, services accessed by IP literal. Common in enterprise PKI for internal hosts. |
  | `URI` | `uniformResourceIdentifier` | 6 | SPIFFE/SPIRE IDs, S/MIME identity URIs, modern cloud-native service identity. |

  Encoding details for the non-trivial cases:

  - `DNS` -> `GeneralName(GeneralName.dNSName, DERIA5String(value))`.
  - `EMAIL` -> `GeneralName(GeneralName.rfc822Name, DERIA5String(value))`. User-facing key is `EMAIL=`; `RFC822=` is not a
    synonym in v1.
  - `URI` -> `GeneralName(GeneralName.uniformResourceIdentifier, DERIA5String(value))`.
  - `IP` -> parse value with `InetAddress.getByName(value)`, then
    `GeneralName(GeneralName.iPAddress, DEROctetString(addr.address))`. 4 bytes for IPv4, 16 for IPv6. Reject anything that
    fails to parse as IPv4 or IPv6.
  - `UPN` -> `GeneralName(GeneralName.otherName, DERSequence(arrayOf(ASN1ObjectIdentifier("1.3.6.1.4.1.311.20.2.3"),
    DERTaggedObject(true, 0, DERUTF8String(value)))))` per Microsoft KB258605 / RFC 4556 §3.2.1. The `[0] EXPLICIT` tag on
    the value (the `true` arg in `DERTaggedObject`) is required — without it the `otherName` is uninterpretable to Windows /
    NPS / Intune supplicants. This is the single most error-prone encoding.

- **Decision:** In `buildCsr`, when `parse(config.subjectAlternativeName ?: "")` returns non-null, append the extensionRequest
  attribute:
  ```kotlin
  val extensions = ExtensionsGenerator().apply {
      addExtension(Extension.subjectAlternativeName, false, generalNames)
  }.generate()
  csrBuilder.addAttribute(PKCSObjectIdentifiers.pkcs_9_at_extensionRequest, extensions)
  ```
  The SAN extension is **non-critical** (`false`), and this is hard-coded — it is not exposed to admins in any UI or YAML
  field. Rationale per RFC 5280 §4.2.1.6: "If the subject field contains an empty sequence, then the issuing CA MUST include a
  subjectAltName extension that is marked as critical. ... If the subject field is non-empty, conforming CAs SHOULD mark the
  subjectAltName extension as non-critical." Fleet always requires `subject_name` to be non-empty, so the SAN is always SHOULD
  non-critical. Practical implications:
  - Modern Wi-Fi supplicants (wpa_supplicant, Intune NAC, all major Android/Windows/macOS supplicants from the last decade)
    honor SAN regardless of the critical bit. EAP-TLS identity matching works the same way.
  - Marking critical when subject DN is also present would risk CSR rejection from enterprise CAs that follow RFC 5280
    strictly, and would risk supplicants that don't fully process SAN rejecting otherwise-valid certs.
  - IT admins do not configure this — there is no checkbox. If a customer ever needs critical SAN (very rare, requires empty
    subject), that's a separate feature.
- **Alternatives considered:**
  - Doing the parse server-side and shipping a structured JSON to the agent. Rejected — the agent already has BouncyCastle and
    is the authoritative place that builds the CSR; pushing structured types over the wire would create two formats (string
    in the API, struct on the wire to the agent) that must stay in sync forever.
  - Parsing on the server only as a *validation* step (to surface bad input early). Rejected for the MVP — server-side
    validation in this change is limited to variable-allow-list checks, matching how `subject_name` works. We can add stricter
    validation later if customers footgun themselves with bad SAN syntax.
  - Marking the SAN extension *critical*. Rejected — most enterprise issuing CAs reject critical SAN when the subject DN is
    also non-empty; non-critical is the safe default.
- **Edge cases:**
  - Empty value (`"DNS="`): treat as malformed, throw with a clear message.
  - Repeated KEY (`"DNS=a, DNS=b"`, `"EMAIL=u@x, EMAIL=u@y"`, etc.): each occurrence produces one `GeneralName` entry of the
    corresponding type, in document order. RFC 5280 §4.2.1.6 permits any number of entries of the same type, and real
    deployments use this — e.g. Wi-Fi configs with multiple DNS SANs, user certs with both work and personal email SANs.
  - Trailing comma: skip empty tokens.
  - Unparseable IP (`"IP=not.an.address"`): throw — IP value must parse as IPv4 (dotted-quad) or IPv6 (colon-hex) for the raw
    byte encoding to be correct.
  - Server-side variable expansion failed and left a literal `$FLEET_VAR_*` in the value: the agent should NOT special-case
    this; it just gets passed through and the CA rejects the cert. The server should never send unexpanded variables, but the
    agent does not need to defend against this.

### GitOps: extend the spec struct in lockstep with the persisted struct

`fleet.CertificateTemplateSpec` (the YAML-facing type, `server/fleet/app.go`) gets `SubjectAlternativeName`. The YAML key under
`controls.android_settings.certificates[]` matches the docs verbatim. `pkg/spec/gitops.go` already validates the existing fields;
SAN gets the same validation (length, no unsupported variables). `cmd/fleetctl/generate_gitops.go` exports the field whenever
non-empty, omits it when empty, so existing GitOps files round-trip without churn.

### Frontend: add an optional input next to subject name, and switch the modal to the always-enabled-Add pattern

Match the Figma (node 2:130). Two modal changes ship together:

1. **Add the SAN input.** Place a `subject_alternative_name` text input directly under the existing `subject_name` input in
   `AddCertificateModal.tsx`. Help text: "Separate SAN fields by ', '." Placeholder mirrors the Figma:
   `UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, EMAIL=$FLEET_VAR_HOST_END_USER_IDP_USERNAME`. Apply the same Premium gate that
   already hides the certificate feature for non-Premium tenants (the modal is already only mounted for Premium users; no new
   gate is needed at this layer beyond not adding a non-Premium code path). The client-side validation rule on the SAN field
   is permissive (any non-empty string is accepted at form level), with server-side validation as the source of truth — this
   matches how `subject_name` is handled today.

2. **Switch the Add button to always-enabled with on-submit field highlighting.** Per the Figma's second dev note ("Add button
   is always enabled. If user hits Add without all required fields, then light up required fields"), replace the existing
   `disabled={!formValidation.isValid || isUpdating}` (`AddCertificateModal.tsx:187`) with `disabled={isUpdating}` so the
   button is enabled whenever the form is not in flight. Add an `attemptedSubmit` state that flips to `true` on the first
   click of Add. While `attemptedSubmit` is `true`, each required field's `<InputField>` / `<DropdownWrapper>` renders its
   validation error (e.g. "Name field must be completed") inline beneath the field, matching the example screenshot in the
   Figma (the "Add user" modal showing red "field must be completed" lines under empty inputs). The submit handler short-
   circuits and does not call the API while validation fails — only the visual state changes. Once the user fixes the
   required fields and the form becomes valid, submission proceeds normally.

   This is a **modal-wide UX change**, not SAN-specific. It changes how the Name, Certificate authority, and Subject name
   fields render their validation errors too. The existing tooltip on the disabled button (`disableTooltip={...}`) is
   removed — it no longer applies. Required-ness is unchanged: Name, Certificate authority, and Subject name remain required;
   `subject_alternative_name` remains optional and is never highlighted by the missing-required-field flow.

   The pattern is consistent with how `UserForm.tsx` already shows errors after submit/blur in the user-management area, so
   no new shared infrastructure is needed; we just rewire the cert modal's local state machine.

### Lightweight server-side validation: format and KEY allow-list, no value content checks

The Figma dev note ("we won't validate SAN, we will let it through, and if it fails, surface error on the host details > OS
settings modal") sets a permissive default and pushes failure handling to the agent and the host-details modal. We follow
that intent for *value content* (don't reimplement a CSR parser server-side; values may contain unexpanded `$FLEET_VAR_*` at
create time), but we add minimal validation at create time for the highest-leverage admin-error classes:

1. **Token shape:** every non-empty comma-separated token contains exactly one `=`. Rejects `"DNS=ok, OOPS"`.
2. **KEY allow-list:** KEY (case-insensitive, normalized to upper) is one of `DNS`, `EMAIL`, `UPN`, `IP`, `URI`. Rejects
   `"FOO=bar"`, `"RFC822=user@x"`, `"EMIAL=..."` (typo).
3. **Variable allow-list:** `$FLEET_VAR_*` references inside SAN values must be in
   `fleetVarsSupportedInCertificateTemplates`. This already runs for `subject_name`; we extend the same call to SAN.
4. **Length cap:** total SAN string under 4096 bytes. Cheap DOS protection.

Failure is a 422 invalid-argument error from the create endpoint, with the field name `subject_alternative_name` and a
message naming the offending token / KEY / variable.

What we explicitly do *not* validate server-side:

- Per-key value contents. `IP=$FLEET_VAR_HOST_FOO` (hypothetical future variable) cannot be IP-parsed at create time. Same
  for URI, hostname, and email regex checks. Those happen on the agent at delivery time, where the value is already expanded.
- Anything that would require a real CSR / X.509 parser. The agent is the authoritative place that builds the CSR.

**Why deviate from the strict reading of the Figma note:** the "we won't validate SAN" note's primary concern is the cost of
reimplementing a CSR parser server-side and keeping it in sync with the agent. KEY allow-listing and shape checks don't fall
into that category — they are 10 lines of code, can never drift (the allow-list is a list of strings), and prevent the most
common admin frustration: typo `RFC822=`, push to N hosts, debug for hours. This is captured as an Open Question for designer
review before merge.

**Industry precedent:** smallstep/step-ca and AWS Private CA both validate SAN format (key allow-list, length, basic shape)
at template / configuration storage time, while deferring value-content validation to the issuance step. OpenSSL `req`
validates strictly at CSR generation. Microsoft AD CS validates at the CA on CSR receipt. Validating at the boundary closest
to where the data is consumed is the universal pattern; we already do that on the agent. Adding the format-only checks at
ingress is a small UX improvement that does not violate the pattern.

### Premium tier gate is enforced server-side, frontend just hides the feature

- **Decision:** The certificate template service methods (`CreateCertificateTemplate`, etc.) gain an explicit
  `svc.License.IsPremium()` check if one is not already present, before SAN-bearing payloads are accepted. Frontend tier-gating
  remains UX-only.
- **Alternative considered:** Gating only the SAN field, not the entire feature. Rejected — Android cert templates as a whole
  are a Premium feature; SAN is just one more field.

## Risks / Trade-offs

- **[Variable-expansion error noise]** Hosts that lack `end_user_idp_username` will start failing more cert deliveries if admins
  reference that variable in SAN. -> **Mitigation:** Same delivery-failure path the SN already uses; surface in the existing
  certificate-status UI; document in the feature guide that SAN/SN variables both require the host to have the underlying data.
- **[Schema migration on a table that may have rows in production]** The column is additive and nullable, so the migration is
  safe to run online. -> **Mitigation:** standard Fleet migration tooling (`make migration`); no backfill needed.
- **[Mixed-version fleet — must ship agent first]** A fleet running an older Android Fleet agent against a newer Fleet server
  will fetch the SAN field in the API response, ignore it (kotlinx.serialization tolerates unknown fields), and submit a CSR
  without the SAN extension — silently issuing a cert that will fail the EAP-TLS match the admin expected. The "agent ignores
  unknown fields" property prevents a *crash* but not a *feature regression*. -> **Mitigation (mandatory):** ship the new
  Android agent **before** the Fleet server surfaces SAN through any user-facing path (UI, GitOps, REST). The agent build is
  forward-compatible — it reads `subject_alternative_name` if present, simply omits the SAN extension if absent — so it can
  ship against any current server with no behavior change. Optional belt-and-suspenders: server-side User-Agent check that
  refuses to return a non-empty SAN to agents below the SAN-supporting version, falling back to the cert state going to
  `failed` for that host (forces admin attention rather than silent miscertification). Ship-order is captured in Migration
  Plan.
- **[BouncyCastle ASN.1 encoding bugs]** UPN-as-OtherName is the most error-prone case: the value must be wrapped as
  `DERUTF8String` inside the `OtherName` SEQUENCE, with the right OID. A wrong wrapping produces a CSR the CA will sign but
  whose UPN is uninterpretable to network supplicants. -> **Mitigation:** Unit test `SubjectAlternativeNameParserTest` decodes
  the produced extension back through `GeneralNames.getInstance(...)` and asserts on the round-tripped values; integration
  test against the test certs in the issue's Drive folder before shipping.
- **[Behavior on empty string]** Whitespace-only `subject_alternative_name` could pass to the CSR layer and produce an invalid
  cert. -> **Mitigation:** Trim and treat empty-after-trim as "no SAN" at the service layer.
- **[GitOps round-trip drift]** If `generate-gitops` exports differently than apply parses, customers see spurious diffs. ->
  **Mitigation:** Exact-string round-trip test in `cmd/fleetctl` covering a template with and without SAN.

## Migration Plan

The agent ships **before** any server-side user-facing exposure of the SAN field. The agent build is forward-compatible against
older Fleet servers, so it can land at any time without coordinating with a specific Fleet server release.

1. **Android Fleet agent release (ships first).** Add the SAN field to `GetCertificateTemplateResponse` in `ApiClient.kt`, the
   new `SubjectAlternativeNameParser`, and the `buildCsr` extension wiring. Ship through the agent's existing release train
   (typically Google Play). The new agent: reads `subject_alternative_name` from the per-host template response if present;
   if absent or empty, builds the CSR exactly as today (no SAN extension). Wait for this build to roll out broadly to the
   field before proceeding.
2. **Server backend release.** Land the migration (additive nullable column), Go type changes, datastore CRUD,
   variable-expansion, Premium gate, and the device-facing endpoint that now returns `subject_alternative_name`. The REST
   create endpoint accepts the new field, but it is **not yet documented or surfaced in the UI**. GitOps validate accepts the
   field but generate-gitops does not yet emit it. (Documenting this step is a no-op for admins if they don't poke the REST
   API directly — agents in the field already support SAN, so even an admin who finds the field manually gets correct
   behavior on any host whose agent has updated.)
3. **GitOps generate-gitops + Frontend modal + REST docs visibility.** Round-trip tests pass; Add/Edit Certificate modal
   exposes the SAN input; the merged docs from PR #43318 (already in `docs-v4.86.0`) ship to fleetdm.com along with the
   release. This is the moment customers can discover and use the feature.
4. **Feature guide update at fleetdm.com** (https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#android-deploy-certificate).

Rollback: server side, revert the migration (column drop is safe because nothing in the old code reads or writes the column)
plus the code changes. Android agent side, agent ignores any unrecognized JSON field on the server, and treats a missing field
as "no SAN" — so rolling back the server while the new agent is still in the field has no adverse effect.

## Open Questions

- The cert-template variable allow-list today is `HOST_UUID`, `HOST_HARDWARE_SERIAL`, `HOST_END_USER_IDP_USERNAME` only. The
  Figma placeholder shows `UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, EMAIL=$FLEET_VAR_HOST_END_USER_IDP_USERNAME` — i.e. it
  assumes the requesting customer's IdP username works for both UPN and email slots (typical for tenants whose IdP usernames
  are emails). No new `HOST_END_USER_IDP_EMAIL` variable is needed for v1. Re-evaluate if the requesting customer's IdP
  usernames are not in email form.
- The lightweight server-side validation proposed under "Lightweight server-side validation" deviates from the Figma dev
  note's "we won't validate SAN, we will let it through" guidance. We believe the note's intent is to not reimplement a CSR
  parser server-side, not to let typos through unchecked, but this should be confirmed with the designer before
  implementation.
