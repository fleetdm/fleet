## 1. Backend types and database

- [ ] 1.1 Add `SubjectAlternativeName string` to `fleet.CertificateTemplate`, `fleet.CertificateRequestSpec`,
      `CertificateTemplateResponseSummary`, `CertificateTemplateResponse`, and `CertificateTemplateResponseForHost` in
      `server/fleet/certificate_templates.go` with `json:"subject_alternative_name,omitempty"` and `db:"subject_alternative_name"`
- [ ] 1.2 Run `make migration name=AddSubjectAlternativeNameToCertificateTemplates`, edit the generated migration to add a
      nullable `subject_alternative_name` column to `certificate_templates` matching the type/length of the existing
      `subject_name` column
- [ ] 1.3 Add a unit test for the migration in the generated `_test.go` file using the standard Fleet migration test pattern
- [ ] 1.4 Update the datastore layer (`server/datastore/mysql/certificate_templates.go`) to read and write
      `subject_alternative_name` in the create / get-by-id / list / get-for-host queries; trim whitespace and store empty as NULL

## 2. Service layer: validation, variable expansion, Premium gate

- [ ] 2.1 In `server/service/certificates.go`, add `SubjectAlternativeName *string` to `createCertificateTemplateRequest`
      (pointer so we can distinguish "omitted" from "empty") and pass it into the service method
- [ ] 2.2 Update `Service.CreateCertificateTemplate` to accept the SAN argument, call
      `validateCertificateTemplateFleetVariables` on it, normalize empty/whitespace to "" before persistence, and pass through
      to the datastore
- [ ] 2.2a Implement lightweight SAN format validation per the "Lightweight server-side validation" decision in design.md:
      every non-empty comma-separated token contains exactly one `=`; KEY (case-insensitive) is in the allow-list
      `{DNS, EMAIL, UPN, IP, URI}`; total SAN string is under 4096 bytes. Failures return a 422 invalid-argument error against
      the `subject_alternative_name` field with a message naming the offending token / KEY. **Confirm with the designer
      before implementing** — the Figma dev note says "we won't validate SAN" and this deviates by adding format-only checks
      (not value content). If the designer rejects, drop this task and let the agent be the only gatekeeper.
- [ ] 2.3 If a Premium check is not already enforced upstream of `CreateCertificateTemplate`, add `svc.License.IsPremium()`
      gating consistent with how other Premium-only writes work, returning the standard Premium-required error
- [ ] 2.4 Extend `Service.replaceCertificateVariables` (or its caller in `GetDeviceCertificateTemplate`) to render
      `subject_alternative_name` for the host with the same error semantics as `subject_name`; on failure the cert state
      transitions to `failed`
- [ ] 2.5 Add unit tests in `server/service/certificate_templates_test.go` covering: SAN with supported variable, SAN with
      unsupported variable (rejected), empty SAN (NULL persisted), whitespace-only SAN (NULL persisted), variable expansion
      success, expansion failure for missing host data, and Premium gate

## 3. Device-facing endpoint plumbs SAN through to Android agent

- [ ] 3.1 Confirm `GetDeviceCertificateTemplate` (the per-host endpoint behind `/api/fleetd/certificates/{id}` that the Android
      Fleet agent reads) returns the expanded `subject_alternative_name` whenever non-empty, omits it (or returns `""`)
      otherwise
- [ ] 3.2 Add an integration test in `server/service/integration_*_test.go` that creates a template with SAN, simulates the
      Android agent fetch for a host with an IdP username, and asserts the rendered SAN comes back correctly

## 4. Android Fleet agent (Kotlin, `android/app/src/main/`)

- [ ] 4.1 In `java/com/fleetdm/agent/ApiClient.kt`, add `@SerialName("subject_alternative_name") val subjectAlternativeName:
      String? = null` to `GetCertificateTemplateResponse` (around line 588-626). Confirm `Json` is configured with
      `ignoreUnknownKeys = true` so older agents on newer servers continue to deserialize.
- [ ] 4.2 Create `java/com/fleetdm/agent/scep/SubjectAlternativeNameParser.kt` with a single public function
      `parse(sanString: String?): GeneralNames?` per the "Android agent" decision in design.md. Returns `null` for
      null/empty/whitespace; throws
      `IllegalArgumentException` (caller wraps to `ScepCsrException`) for unknown KEY, malformed tokens, or unparseable IP
      values. Supports five KEYs: `DNS` (`dNSName`), `EMAIL` (`rfc822Name`), `URI` (`uniformResourceIdentifier`), `IP`
      (`iPAddress`, parse via `InetAddress.getByName` to 4-/16-byte octet string), `UPN` (`otherName` with OID
      `1.3.6.1.4.1.311.20.2.3`, value as `DERUTF8String` wrapped in `[0] EXPLICIT` per Microsoft KB258605).
- [ ] 4.3 Update `java/com/fleetdm/agent/scep/ScepClientImpl.kt#buildCsr` (around line 168-179) to call the parser and, when it
      returns non-null `GeneralNames`, append an `extensionRequest` attribute (`pkcs_9_at_extensionRequest`) carrying a
      single `Extension` for `subjectAlternativeName` with `critical = false`. Wrap parser exceptions in `ScepCsrException`
      so the existing failure-path observability is preserved.
- [ ] 4.4 Update `java/com/fleetdm/agent/CertificateEnrollmentHandler.kt` if needed to carry `subjectAlternativeName` from
      `GetCertificateTemplateResponse` into the SCEP client config (no logic change beyond plumbing the new field).
- [ ] 4.5 Update `app/src/test/java/com/fleetdm/agent/testutil/TestCertificateTemplateFactory.kt` to accept an optional
      `subjectAlternativeName: String? = null` parameter on `create(...)` so existing tests keep passing while new tests can
      opt in.
- [ ] 4.6 Add `app/src/test/java/com/fleetdm/agent/scep/SubjectAlternativeNameParserTest.kt` covering: null input,
      whitespace-only input, single DNS, single EMAIL, single URI, single IP (IPv4 dotted-quad), single IP (IPv6 colon-hex),
      single UPN (decode the produced `OtherName` and assert OID + UTF8 value), mixed entries (all five KEYs), repeated keys
      across multiple types (e.g. two DNS + two EMAIL → four entries in document order, asserting the same applies to
      repeated UPN / IP / URI), unknown KEY (expects throw), `RFC822=` rejected as unknown KEY (expects throw), malformed
      token (expects throw), unparseable IP (expects throw), whitespace tolerance.
- [ ] 4.7 Extend `app/src/test/java/com/fleetdm/agent/scep/ScepClientImplTest.kt` with cases that build a CSR with various SAN
      strings and decode the resulting CSR to assert the SAN extension is present, non-critical, and contains the expected
      `GeneralName` entries. Also keep an explicit regression test for "no SAN -> no SAN extension".
- [ ] 4.8 Extend `app/src/test/java/com/fleetdm/agent/CertificateEnrollmentHandlerTest.kt` to assert the SAN string flows from
      the API response through to the SCEP config that `MockScepClient` captures.
- [ ] 4.9 Run `./gradlew test` (or the project's standard test command — see `android/CHANGELOG.md` and `android/README.md`)
      and confirm green.
- [ ] 4.10 Add an entry to `android/CHANGELOG.md` for the SAN-extension behavior change. No new third-party dep is needed
       (BouncyCastle 1.78.1 already on classpath via `bcprov-jdk18on` + `bcpkix-jdk18on`).

## 5. GitOps apply

- [ ] 5.1 Add `SubjectAlternativeName string` (with `yaml:"subject_alternative_name,omitempty"`) to
      `fleet.CertificateTemplateSpec` in `server/fleet/app.go`
- [ ] 5.2 Update GitOps validation in `pkg/spec/gitops.go` (and `gitops_validate.go` if applicable) to validate variables in SAN
      via the same helper used for `subject_name`
- [ ] 5.3 Update the Apply path so the spec's SAN is forwarded to `CreateCertificateTemplate`
- [ ] 5.4 Add a GitOps round-trip test in `cmd/fleetctl` covering both a certificate with SAN and one without

## 6. GitOps generate

- [ ] 6.1 Update `cmd/fleetctl/generate_gitops.go` (around the certificates emit block) to include
      `subject_alternative_name` for each template whose stored value is non-NULL, omit the key entirely otherwise
- [ ] 6.2 Add or extend the existing generate-gitops golden-file test for the new field, covering both populated and empty
      cases

## 7. Frontend

- [ ] 7.1 Add `subjectAlternativeName: string` to `IAddCertFormData` and the equivalent edit-form types under
      `frontend/pages/ManageControlsPage/OSSettings/cards/Certificates/`
- [ ] 7.2 Add a SAN text input directly under the existing `subject_name` input in `AddCertificateModal.tsx`, matching the
      Figma wireframe (https://www.figma.com/design/2jRQoXofC1caxyNhWl8F0m/...?node-id=3462-252) for label/help text
- [ ] 7.3 Wire the field through the API client (`frontend/services/`) request and response types so the modal sends and reads
      `subject_alternative_name`
- [ ] 7.4 Surface server-side validation errors against the SAN input (the existing pattern for `subject_name` errors)
- [ ] 7.5 Switch `AddCertificateModal.tsx` to the always-enabled-Add pattern per the Figma's second dev note: replace
      `disabled={!formValidation.isValid || isUpdating}` (currently around line 187) with `disabled={isUpdating}`. Add an
      `attemptedSubmit` boolean state that flips to `true` on the first click of "Add". The submit handler short-circuits
      (does not call the API) when `attemptedSubmit && !formValidation.isValid`. Pass the `attemptedSubmit` flag down to each
      required field's `<InputField>` / `<DropdownWrapper>` so it renders the existing validation error inline (e.g. "Name
      must be completed", "Certificate authority must be completed", "Subject name must be completed"). Remove the existing
      tooltip on the disabled button (`disableTooltip` / `TooltipWrapper` around the Add button) — it no longer applies.
      Note: this is a modal-wide change, not SAN-specific.
- [ ] 7.6 Add a "field must be completed" message for any required field that does not already have one (today some required
      fields rely solely on the disabled-button-with-tooltip to communicate). Confirm against the Figma example screenshot
      ("Add user" modal) for exact phrasing per field.
- [ ] 7.7 Add Jest tests for the modal: (a) submitting with SAN, (b) submitting without SAN, (c) server error on SAN
      surfaces inline against the SAN field, (d) clicking Add with all required fields empty surfaces three inline errors and
      does not call the API, (e) clicking Add with one required field empty surfaces exactly that field's error, (f) the Add
      button is enabled at all times except while `isUpdating`.

## 8. Documentation and rollout

- [ ] 8.1 Update the feature guide at https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#android-deploy-certificate
      to document the new SAN field, supported variables, and an end-to-end example for Wi-Fi UPN
- [ ] 8.2 Verify the REST API docs already merged in PR #43318 still match the implemented contract (field name,
      example payload); fix if drift exists
- [ ] 8.3 Run the test plan from the issue: deliver a certificate with each of the SANs in the supplied test cert fixtures,
      confirm the resulting cert on the device contains the SAN extension and that EAP-TLS authentication succeeds against
      the reference test CA
- [ ] 8.4 Rollout order per design.md Migration Plan: the Android Fleet agent build (section 4 of this list) ships *first*,
      and only after it has rolled out broadly does the Fleet server expose SAN to admins (frontend modal, generate-gitops
      emit, REST docs visibility). The agent build is forward-compatible against older servers, so it can land independently.
      If timing forces overlapping releases, hide the UI input and the generate-gitops emit behind a feature flag until agent
      coverage is confirmed.
- [ ] 8.5 Engineer comment on issue #41472 confirming successful test plan completion (per the issue's Confirmation section)

## 9. Pre-merge verification

- [ ] 9.1 `make lint-go-incremental` and `make lint-js` clean
- [ ] 9.2 Targeted Fleet test bundles green: `service`, `mysql`, `integration-core`, `integration-mdm`, `fleetctl`, plus
      `yarn test` for the touched frontend files
- [ ] 9.3 Android agent build green: `./gradlew test` from `android/`, plus the Android lint checks the project already runs
      in CI
