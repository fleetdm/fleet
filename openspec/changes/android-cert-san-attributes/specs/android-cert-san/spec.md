## ADDED Requirements

### Requirement: Persist optional subject alternative name on certificate templates

The certificate template entity SHALL include an optional `subject_alternative_name` field of type string. The field SHALL be
persisted to the `certificate_templates` table as a nullable `TEXT` column (matching the existing `subject_name TEXT` column).
A whitespace-only value SHALL be stored as NULL and treated equivalently to "no SAN". A non-empty value SHALL be stored
**verbatim** — no per-token trimming, no leading/trailing-whitespace mutation — to preserve admin intent and keep GitOps
round-trips idempotent. JSON serialization uses Go's `omitempty`, so the response **deterministically omits** the
`subject_alternative_name` key when the stored value is NULL or empty.

#### Scenario: Create with SAN value, stored verbatim

- **WHEN** an admin POSTs a certificate template with
  `subject_alternative_name = "DNS=example.com, UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME"`
- **THEN** the template SHALL be stored with that exact string (byte-identical) in the `subject_alternative_name` column
- **AND** the response body SHALL echo the same value under the same key

#### Scenario: Create with surrounding whitespace, stored verbatim

- **WHEN** an admin POSTs `subject_alternative_name = "  DNS=a.example.com  ,  UPN=marko@x  "` (deliberate whitespace)
- **THEN** the persisted value SHALL be the exact original string, byte-identical, with no per-token or outer trimming on
  the server
- **AND** GitOps round-trip (apply -> store -> generate-gitops -> apply) SHALL preserve the exact same bytes

#### Scenario: Create without SAN value (existing behavior preserved)

- **WHEN** an admin POSTs a certificate template with no `subject_alternative_name` field, or with an empty string
- **THEN** the template SHALL be stored with NULL in `subject_alternative_name`
- **AND** the response body SHALL omit the `subject_alternative_name` key entirely (matching `json:",omitempty"`), with no
  validation error

#### Scenario: Whitespace-only SAN treated as NULL

- **WHEN** an admin POSTs a certificate template with `subject_alternative_name = "   "` (whitespace only, no other content)
- **THEN** the value SHALL be stored as NULL (using `strings.TrimSpace(value) == ""` as the test)
- **AND** the response body SHALL omit the `subject_alternative_name` key entirely

### Requirement: Lightweight SAN format validation at create time

The certificate template create endpoint SHALL perform format-only (not value-content) validation on `subject_alternative_name`
at create time. Specifically: every non-empty comma-separated token MUST contain exactly one `=`; the KEY (left of `=`,
case-insensitive) MUST be in `{DNS, EMAIL, UPN, IP, URI}`; the total length of the SAN string MUST be under 4096 bytes. The
server SHALL NOT validate the value-content (right of `=`) — value content can include `$FLEET_VAR_*` references that have not
yet been expanded at create time, and value-content parsing belongs to the agent at delivery time. Failures SHALL return a 422
invalid-argument error scoped to the `subject_alternative_name` field, with a message naming the specific token, KEY, or
condition that failed.

This requirement is conditional on designer confirmation (see design.md Open Questions). If the designer rejects format
validation, the server-side validation is limited to the variable allow-list (next requirement) and the agent becomes the
sole gatekeeper.

#### Scenario: Token missing `=`

- **WHEN** the create payload contains `subject_alternative_name = "DNS=ok, OOPS"`
- **THEN** the server SHALL return 422 with a message identifying the token `OOPS` as missing `=`
- **AND** no template SHALL be persisted

#### Scenario: Unknown KEY

- **WHEN** the create payload contains `subject_alternative_name = "FOO=bar"` or `"RFC822=user@x"`
- **THEN** the server SHALL return 422 with a message identifying the offending KEY and listing the allowed set

#### Scenario: Length exceeds cap

- **WHEN** the create payload contains a `subject_alternative_name` of 4097+ bytes
- **THEN** the server SHALL return 422 with a message identifying the length cap

#### Scenario: Value content with unexpanded variable accepted

- **WHEN** the create payload contains `subject_alternative_name = "IP=$FLEET_VAR_HOST_UUID"` (unexpanded)
- **THEN** the server SHALL accept the create — value content is not validated, the variable allow-list passes
- **AND** at delivery time the agent SHALL fail to parse the resulting unexpanded literal `$FLEET_VAR_HOST_UUID` as IP, and
  the failure SHALL surface in the host's "OS settings" modal (this is expected — admins should not put non-IP-shaped
  variables in `IP=` slots)

### Requirement: Validate variables in SAN at create time

The system SHALL validate any `$FLEET_VAR_*` references inside `subject_alternative_name` against the same allowed set already
applied to `subject_name` (`fleetVarsSupportedInCertificateTemplates` in `server/service/certificate_templates.go`). At time of
writing, that set is exactly:

- `HOST_UUID`
- `HOST_HARDWARE_SERIAL`
- `HOST_END_USER_IDP_USERNAME`

Unsupported variables SHALL be rejected with a 422 invalid-argument error. If the allow-list grows later, both `subject_name`
and `subject_alternative_name` SHALL pick up the new entries via the same shared list (no SAN-specific divergence).

#### Scenario: Supported variable accepted

- **WHEN** the SAN string references `$FLEET_VAR_HOST_UUID`, `$FLEET_VAR_HOST_HARDWARE_SERIAL`, or
  `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` (each tested individually)
- **THEN** the create call SHALL succeed

#### Scenario: Unsupported variable rejected

- **WHEN** the SAN string references `$FLEET_VAR_HOST_PLATFORM` (defined globally but not in the cert-template allow-list), or
  `$FLEET_VAR_HOST_END_USER_IDP_GROUPS`, or any other `FLEET_VAR_*` not in the allow-list above
- **THEN** the create call SHALL fail with a 422 invalid-argument error
- **AND** the error message SHALL identify both the offending variable and that it was found in `subject_alternative_name`

### Requirement: Expand variables in SAN at delivery time

When the device-facing certificate template endpoint returns a template for a specific host, the system SHALL expand
`$FLEET_VAR_HOST_*` references in `subject_alternative_name` using the host's values, with semantics identical to the existing
expansion of `subject_name`.

#### Scenario: Successful expansion for both fields

- **GIVEN** a host with UUID `H-123` and IdP username `marko@example.com`
- **AND** a template with `subject_name = "/CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME"` and
  `subject_alternative_name = "DNS=example.com, UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME"`
- **WHEN** the agent fetches the template for that host
- **THEN** the response SHALL contain `subject_name = "/CN=marko@example.com"`
- **AND** `subject_alternative_name = "DNS=example.com, UPN=marko@example.com"`

#### Scenario: Host missing data for SAN variable

- **GIVEN** a host without an IdP username
- **AND** a template whose SAN references `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`
- **WHEN** delivery is attempted
- **THEN** the system SHALL transition the host's certificate template state to `failed`
- **AND** the failure SHALL surface the same way as a `subject_name` expansion failure (no silent issuance)

#### Scenario: Empty SAN passes through unchanged

- **WHEN** the template has no `subject_alternative_name` (NULL in storage)
- **THEN** the device-facing response SHALL omit the `subject_alternative_name` key entirely (per `omitempty`), and delivery
  SHALL behave exactly as it does today for templates without SAN

### Requirement: Android Fleet agent includes SAN extension in PKCS#10 CSR

The Android Fleet agent (Kotlin source under `android/app/src/main/java/com/fleetdm/agent/`) SHALL parse the
`subject_alternative_name` field returned by `/api/fleetd/certificates/{id}` and SHALL include a `subjectAlternativeName`
extension (OID `2.5.29.17`) in the PKCS#10 Certificate Signing Request it submits to the SCEP CA. The extension SHALL be
attached as part of the CSR's `extensionRequest` attribute (PKCS#9 OID `1.2.840.113549.1.9.14`), with `critical = false`
hard-coded (per RFC 5280 §4.2.1.6, since `subject_name` is always non-empty in Fleet). The extension SHALL contain one
`GeneralName` entry per parsed token, in document order. The agent SHALL recognize five KEYs (case-insensitive): `DNS`,
`EMAIL`, `UPN`, `IP`, `URI`. These are the SAN attribute types real-world enterprise PKI actually deploys for the use cases
this feature targets (Wi-Fi/EAP-TLS, internal mTLS, S/MIME, modern service identity).

Multiple values of the same KEY are allowed. Each `KEY=value` token in the comma-separated string produces one
`GeneralName` entry, so `"DNS=a.example.com, DNS=b.example.com, EMAIL=alice@x, EMAIL=alice@y"` produces four entries (two
`dNSName`, two `rfc822Name`) preserving the order they were written. This matches RFC 5280 §4.2.1.6, which permits any number
of `GeneralName` entries of the same type.

The encoding for each KEY:

- `DNS` -> `GeneralName.dNSName` with `DERIA5String(value)`. BouncyCastle tag 2.
- `EMAIL` -> `GeneralName.rfc822Name` with `DERIA5String(value)`. BouncyCastle tag 1. (Internal X.509 type is `rfc822Name`;
  the user-facing key is `EMAIL=`. `RFC822=` is not accepted as a synonym.)
- `URI` -> `GeneralName.uniformResourceIdentifier` with `DERIA5String(value)`. BouncyCastle tag 6.
- `IP` -> `GeneralName.iPAddress`, BouncyCastle tag 7. The agent SHALL parse the value as IPv4 dotted-quad or IPv6
  colon-hex (e.g. via `InetAddress.getByName(value)`) and emit a `DEROctetString` containing the raw 4-byte (IPv4) or 16-byte
  (IPv6) address. Values that fail to parse SHALL cause the agent to hard-fail (see scenario below).
- `UPN` -> `GeneralName.otherName` (BouncyCastle tag 0) carrying an `OtherName` SEQUENCE:
  `{ type-id ASN1ObjectIdentifier("1.3.6.1.4.1.311.20.2.3"), value [0] EXPLICIT DERUTF8String(value) }` per Microsoft KB258605
  / RFC 4556 §3.2.1. The `[0] EXPLICIT` tag on the value is required — without it, the resulting `otherName` is
  uninterpretable to Windows / NPS / Intune supplicants.

#### Scenario: SAN absent — CSR unchanged from current behavior

- **GIVEN** a template whose response contains no `subject_alternative_name` (null or empty)
- **WHEN** the agent builds the CSR
- **THEN** the CSR SHALL NOT carry an `extensionRequest` attribute *for SAN* (the existing challenge-password attribute is
  still present)
- **AND** the resulting cert SHALL be byte-identical (modulo timestamps and serial) to what the current agent produces today

#### Scenario: SAN with a single DNS entry

- **GIVEN** the response carries `subject_alternative_name = "DNS=wifi.example.com"`
- **WHEN** the agent builds the CSR
- **THEN** the CSR SHALL contain exactly one SAN extension whose only entry is a `dNSName` with value `wifi.example.com`

#### Scenario: SAN with a UPN entry encoded as OtherName

- **GIVEN** the response carries `subject_alternative_name = "UPN=marko@corp.example.com"`
- **WHEN** the agent builds the CSR
- **THEN** the CSR's SAN extension SHALL contain one `otherName` with type-id OID `1.3.6.1.4.1.311.20.2.3`
- **AND** the `value` shall be a `DERUTF8String` whose contents decode to `marko@corp.example.com`

#### Scenario: SAN with IPv4 entry

- **GIVEN** the response carries `subject_alternative_name = "IP=10.0.0.1"`
- **WHEN** the agent builds the CSR
- **THEN** the CSR's SAN extension SHALL contain exactly one `iPAddress` entry whose 4-byte octet string is `0a 00 00 01`

#### Scenario: SAN with IPv6 entry

- **GIVEN** the response carries `subject_alternative_name = "IP=2001:db8::1"`
- **WHEN** the agent builds the CSR
- **THEN** the CSR's SAN extension SHALL contain exactly one `iPAddress` entry whose 16-byte octet string corresponds to the
  parsed IPv6 address

#### Scenario: SAN with URI entry

- **GIVEN** the response carries `subject_alternative_name = "URI=spiffe://example.com/workload/payments"`
- **WHEN** the agent builds the CSR
- **THEN** the CSR's SAN extension SHALL contain exactly one `uniformResourceIdentifier` entry with that exact value

#### Scenario: SAN with mixed entries

- **GIVEN** `subject_alternative_name = "DNS=wifi.example.com, UPN=marko@corp.example.com, EMAIL=marko@corp.example.com,
  IP=10.0.0.1, URI=spiffe://example.com/workload/wifi"`
- **WHEN** the agent builds the CSR
- **THEN** the CSR's SAN extension SHALL contain five entries in document order: a `dNSName`, an `otherName` (UPN), an
  `rfc822Name`, an `iPAddress`, and a `uniformResourceIdentifier`
- **AND** all five values decode to the values the server returned

#### Scenario: SAN with multiple values of the same KEY

- **GIVEN** `subject_alternative_name = "DNS=primary.example.com, DNS=secondary.example.com, EMAIL=alice@x.example.com,
  EMAIL=alice@y.example.com"`
- **WHEN** the agent builds the CSR
- **THEN** the CSR's SAN extension SHALL contain four entries in document order: two `dNSName`, then two `rfc822Name`
- **AND** the same principle SHALL apply to repeated `UPN`, `IP`, and `URI` keys (each repetition produces one additional
  `GeneralName` entry of the corresponding type)

#### Scenario: SAN with unknown KEY -> hard fail

- **GIVEN** `subject_alternative_name = "FOO=bar"` or `"RFC822=user@x"` (not a synonym for `EMAIL=`)
- **WHEN** the agent builds the CSR
- **THEN** the agent SHALL throw a `ScepCsrException` (or equivalent) naming the offending KEY
- **AND** the agent SHALL NOT submit a CSR to the SCEP CA
- **AND** the certificate template's host-side state SHALL surface the failure in the host details "OS settings" modal (no
  silent issuance of a cert lacking the intended SAN), per the Figma dev note

#### Scenario: SAN with unparseable IP -> hard fail

- **GIVEN** `subject_alternative_name = "IP=not.an.address"`
- **WHEN** the agent builds the CSR
- **THEN** the agent SHALL throw with a clear message indicating the value could not be parsed as IPv4 or IPv6
- **AND** the agent SHALL NOT submit a CSR

#### Scenario: SAN with malformed token -> hard fail

- **GIVEN** `subject_alternative_name` contains a token without `=` (e.g. `"DNS=ok, UPN"`)
- **WHEN** the agent builds the CSR
- **THEN** the agent SHALL throw with a clear message identifying the bad token
- **AND** the agent SHALL NOT submit a CSR

#### Scenario: Whitespace tolerance

- **WHEN** the SAN string contains leading/trailing whitespace around tokens or around `=`
  (e.g. `"  DNS = a.example.com ,  UPN=  marko@x  "`)
- **THEN** the agent SHALL trim whitespace from each KEY and value before building `GeneralName`, producing the same CSR as
  the un-spaced form

### Requirement: GitOps apply round-trips SAN

GitOps YAML under `controls.android_settings.certificates[]` SHALL accept an optional `subject_alternative_name` key per entry
and apply it to the corresponding certificate template. Validation matches the REST API: same variable allow-list, same
empty-vs-NULL normalization.

#### Scenario: Apply YAML with SAN

- **WHEN** a GitOps run applies a certificates entry whose YAML contains
  `subject_alternative_name: "DNS=example.com, UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME"`
- **THEN** the resulting certificate template SHALL have that value persisted
- **AND** subsequent device delivery SHALL render it as in "Expand variables in SAN at delivery time"

#### Scenario: Apply YAML without SAN

- **WHEN** a GitOps run applies a certificates entry that omits `subject_alternative_name`
- **THEN** the resulting certificate template's `subject_alternative_name` SHALL be NULL
- **AND** no validation warning SHALL fire

### Requirement: generate-gitops emits SAN

`fleetctl generate-gitops` SHALL include a `subject_alternative_name` key for each certificate template whose stored value is
non-NULL. Templates with NULL SAN SHALL omit the key entirely so existing GitOps files do not pick up spurious diffs.

#### Scenario: Round-trip with SAN

- **GIVEN** a template stored with `subject_alternative_name = "DNS=example.com"`
- **WHEN** an admin runs `fleetctl generate-gitops`
- **THEN** the emitted YAML for that certificates entry contains `subject_alternative_name: "DNS=example.com"`
- **AND** re-applying the emitted YAML produces an identical template (idempotent round-trip)

#### Scenario: Round-trip without SAN

- **GIVEN** a template stored with NULL `subject_alternative_name`
- **WHEN** an admin runs `fleetctl generate-gitops`
- **THEN** the emitted YAML for that certificates entry SHALL NOT contain the `subject_alternative_name` key

### Requirement: Add/Edit Certificate UI exposes SAN

The "Add certificate" / "Edit certificate" modal in Manage > Controls > OS Settings > Certificates SHALL include an optional
text input labeled per the Figma wireframe (node 3462-252) that maps to `subject_alternative_name`. The input SHALL be visible
only when the existing certificate-template feature is available (i.e. for Premium tenants), with no separate gate.

#### Scenario: Submit form with SAN

- **WHEN** an admin opens the modal, fills the SAN input with a valid value, and submits
- **THEN** the request payload SHALL include `subject_alternative_name` with the entered value
- **AND** on success, the certificates list SHALL show the new template

#### Scenario: Submit form without SAN

- **WHEN** an admin submits the modal without filling the SAN input
- **THEN** the request SHALL succeed and the persisted template SHALL have NULL `subject_alternative_name`

#### Scenario: Server validation surfaces in form

- **WHEN** the server returns a 422 because the SAN references an unsupported variable
- **THEN** the form SHALL display the server-provided error against the SAN input

### Requirement: Add Certificate modal uses always-enabled-Add with on-submit required-field highlighting

The Add/Edit Certificate modal SHALL keep its primary "Add" button enabled whenever a submit is not already in flight,
regardless of whether required fields are filled. Clicking "Add" while required fields (Name, Certificate authority,
Subject name) are empty or invalid SHALL surface an inline "<field name> must be completed" error beneath each offending
required field and SHALL NOT call the create API. Once all required fields are valid the next "Add" click SHALL submit
normally. `subject_alternative_name` is optional and SHALL NOT participate in this required-field highlighting flow. This
matches the Figma node 2:130 dev note ("Add button is always enabled. If user hit Add without all required fields, then
light up required fields").

#### Scenario: Click Add with all required fields empty

- **GIVEN** the Add Certificate modal is open and Name, Certificate authority, and Subject name are all empty
- **WHEN** the admin clicks "Add"
- **THEN** the create API SHALL NOT be called
- **AND** an inline error SHALL be displayed beneath each of the three required fields naming the field
- **AND** the Add button SHALL remain enabled

#### Scenario: Click Add with one required field empty

- **GIVEN** Name and Subject name are filled, Certificate authority is unselected
- **WHEN** the admin clicks "Add"
- **THEN** the create API SHALL NOT be called
- **AND** the inline error SHALL appear beneath the Certificate authority field only

#### Scenario: Click Add with only optional SAN unfilled

- **GIVEN** Name, Certificate authority, and Subject name are all valid; `subject_alternative_name` is empty
- **WHEN** the admin clicks "Add"
- **THEN** the create API SHALL be called and the modal SHALL close on success
- **AND** no required-field error SHALL appear for `subject_alternative_name`

#### Scenario: Fix required fields after submit-with-empty

- **GIVEN** the admin clicked "Add" with empty required fields and the inline errors are displayed
- **WHEN** the admin fills the required fields and clicks "Add" again
- **THEN** the create API SHALL be called and the modal SHALL close on success

### Requirement: Premium gating for certificate templates

The server SHALL reject any create or GitOps apply of a certificate template — with or without
`subject_alternative_name` — when the deployment is not Fleet Premium, returning `fleet.ErrMissingLicense` (HTTP 402). The
check MUST sit at the top of the service method, after authorization, before validation.

Rationale: certificate templates require a custom SCEP CA, and CAs are documented and implemented as Premium-only (see
`server/service/certificate_authorities.go` core stubs that return `fleet.ErrMissingLicense`). The whole feature is therefore
Premium-only by construction.

The `fleetctl gitops` client SHALL perform an equivalent pre-flight Premium check via `c.GetAppConfig()` whenever the YAML
declares one or more android certificates, so Free admins get a friendly error before any destructive operation runs against
the team. Free admins whose YAML omits the certificates section (or sets it to an empty list) and whose team has no existing
templates SHALL succeed with no changes.

#### Scenario: Non-Premium attempts to create a certificate template (with or without SAN)

- **WHEN** a non-Premium tenant POSTs a certificate template, regardless of whether `subject_alternative_name` is set
- **THEN** the server SHALL reject the request with `fleet.ErrMissingLicense` (HTTP 402)
- **AND** no row SHALL be written to `certificate_templates`

#### Scenario: Non-Premium GitOps apply with certificates declared

- **WHEN** a non-Premium tenant runs `fleetctl gitops` against a YAML that declares one or more
  `controls.android_settings.certificates`
- **THEN** the client SHALL fail with a `gitOpsValidationError` whose message states that Android certificate templates
  require a custom SCEP CA and are available in Fleet Premium only
- **AND** no apply request SHALL be sent to the server

#### Scenario: Non-Premium GitOps apply with no certificates declared

- **WHEN** a non-Premium tenant runs `fleetctl gitops` against a YAML that omits the certificates section (or has an empty
  `certificates: []`) and the team has no existing certificate templates
- **THEN** the GitOps apply SHALL succeed with no errors and no Premium check SHALL fire (the cert-template flow short-
  circuits before any server call)
