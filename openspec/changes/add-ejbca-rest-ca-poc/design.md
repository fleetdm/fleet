# Design: EJBCA REST CA integration (POC)

## Architecture

EJBCA slots into Fleet's existing pluggable CA framework. The integration sits
in the same place DigiCert does, with the same caller shape:

```
   Apple MDM profile processor
   ─────────────────────────────
   (server/mdm/apple/profile_processor.go)
              │
              │ for each FLEET_VAR_EJBCA_DATA_<name>:
              │   1. resolve <name> → EJBCACA row
              │   2. expand FLEET_VAR_* on per-host fields
              │   3. call ejbcaService.GetCertificate(ctx, ca)
              │   4. substitute base64 PKCS12 + password into profile XML
              ▼
   ejbca.Service        ◄──── new package
   ─────────────────────
   (ee/server/service/ejbca/ejbca.go)
              │
              │ HTTP over mTLS
              ▼
   ┌──────────────────────────────────┐
   │  EJBCA REST API                  │
   │  GET  /v1/ca/status              │  ← verify connection
   │  POST /v1/certificate/pkcs10enroll │  ← enroll certificate
   └──────────────────────────────────┘
```

## Data model

### New type: `fleet.EJBCACA`

In `server/fleet/certificate_authorities.go`, two related shapes — the stored
record and the create/update payload. (Same pattern as `DigiCertCA` /
`DigiCertCAUpdatePayload`.)

```go
// Stored / GET-returned shape. After the server has decoded the uploaded
// PKCS#12, the cert and key live here as PEM.
type EJBCACA struct {
    ID   uint   `json:"id,omitempty"`
    Name string `json:"name"`

    URL string `json:"url"` // https://ejbca.example.com:8443

    // Server-derived from the uploaded P12. Returned masked when
    // includeSecrets=false. The original P12 and its password are NOT stored.
    ClientCertPEM string `json:"client_cert"`     // PEM cert chain (public)
    ClientKeyPEM  string `json:"client_key"`      // PEM private key (sensitive, encrypted at rest)

    // Optional trust override for EJBCA's HTTPS cert.
    // Empty = use system root store.
    TrustCABundlePEM string `json:"trust_ca_bundle,omitempty"`

    // EJBCA-side names — free text, must match EJBCA config exactly.
    CertificateAuthorityNameEJBCA string `json:"certificate_authority_name_ejbca"`
    CertificateProfileName        string `json:"certificate_profile_name"`
    EndEntityProfileName          string `json:"end_entity_profile_name"`

    // Per-enrollment fields. Username template may use Fleet vars.
    UsernameTemplate              string   `json:"username_template"`
    CertificateUserPrincipalNames []string `json:"certificate_user_principal_names,omitempty"` // optional UPNs for SAN otherName
}

// Create/update input shape. The user uploads a P12 + password; the server
// decodes it, populates the PEM fields on EJBCACA, then discards both the
// P12 bytes and the password. Neither is ever persisted.
type EJBCACACreatePayload struct {
    Name string `json:"name"`
    URL  string `json:"url"`

    ClientP12         []byte `json:"client_p12"`           // raw P12 bytes
    ClientP12Password string `json:"client_p12_password"`  // discarded after decode

    TrustCABundlePEM string `json:"trust_ca_bundle,omitempty"`

    CertificateAuthorityNameEJBCA string `json:"certificate_authority_name_ejbca"`
    CertificateProfileName        string `json:"certificate_profile_name"`
    EndEntityProfileName          string `json:"end_entity_profile_name"`

    UsernameTemplate              string   `json:"username_template"`
    CertificateUserPrincipalNames []string `json:"certificate_user_principal_names,omitempty"`
}
```

The `CertificateAuthority` polymorphic wrapper gains an `EJBCA *EJBCACA` field
in `CertificateAuthorityPayload`, plus update-payload and validation methods
that mirror `DigiCertCA`. PEM-direct upload is deferred — when it lands, it
becomes an additional set of optional fields on `EJBCACACreatePayload`
(`client_cert_pem`, `client_key_pem`) with a server-side "exactly one of
P12 or PEM" check.

### Constants

```go
// server/fleet/certificate_authorities.go
const CATypeEJBCA = "ejbca"

// server/fleet/mdm.go
const (
    FleetVarEJBCADataPrefix     FleetVarName = "EJBCA_DATA_"
    FleetVarEJBCAPasswordPrefix FleetVarName = "EJBCA_PASSWORD_" //nolint:gosec
)
```

Both prefixes added to the allow-list in `mdm.go`'s prefix check (line ~99) and
to the EJBCA-aware branches in `profile_processor.go` (`isCAConfigured`,
`expandVars`).

### Migration

Add columns to the `certificate_authorities` table:

```sql
ALTER TABLE certificate_authorities
  ADD COLUMN client_cert_pem            BLOB,
  ADD COLUMN client_key_encrypted       BLOB,
  ADD COLUMN trust_ca_bundle_pem        BLOB,
  ADD COLUMN ejbca_ca_name              VARCHAR(255),
  ADD COLUMN ejbca_certificate_profile  VARCHAR(255),
  ADD COLUMN ejbca_end_entity_profile   VARCHAR(255),
  ADD COLUMN ejbca_username_template    VARCHAR(255);
```

The `type` ENUM is extended with `'ejbca'`. There is no enrollment-code
column in the POC — the EJBCA `password` value is generated at issuance
time and never persisted.

Encryption follows the existing pattern: `serverPrivateKey` symmetric
encryption for `client_key_encrypted`. The client cert is **not
encrypted** — it's a public artifact and we display the subject / expiry
in the UI later. Trust CA bundle is likewise public.

Postprocessing on read masks the PEM private key with
`fleet.MaskedPassword` when `includeSecrets=false`.

## HTTP client: `ee/server/service/ejbca`

New package mirroring `ee/server/service/digicert`. Two exported methods:

```go
type Service struct {
    logger  *slog.Logger
    timeout time.Duration
}

var _ fleet.EJBCAService = (*Service)(nil)

func (s *Service) VerifyConnection(ctx context.Context, cfg fleet.EJBCACA) error
func (s *Service) GetCertificate(ctx context.Context, cfg fleet.EJBCACA) (*fleet.EJBCACertificate, error)
```

### TLS plumbing

EJBCA requires mTLS, so the client needs a custom `tls.Config`. Add a small
helper:

```go
func buildTLSClient(timeout time.Duration, cfg fleet.EJBCACA) (*http.Client, error) {
    cert, err := tls.X509KeyPair([]byte(cfg.ClientCertPEM), []byte(cfg.ClientKeyPEM))
    if err != nil {
        return nil, ctxerr.Wrap(ctx, err, "loading client cert keypair")
    }

    tlsCfg := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }
    if cfg.TrustCABundlePEM != "" {
        pool := x509.NewCertPool()
        if !pool.AppendCertsFromPEM([]byte(cfg.TrustCABundlePEM)) {
            return nil, errors.New("trust_ca_bundle did not contain any usable certificates")
        }
        tlsCfg.RootCAs = pool
    }

    return fleethttp.NewClient(
        fleethttp.WithTimeout(timeout),
        fleethttp.WithTLSClientConfig(tlsCfg),
    ), nil
}
```

`fleethttp.WithTLSClientConfig` already exists (verified in
`pkg/fleethttp/fleethttp.go:36`) — no fleethttp changes needed.

### `VerifyConnection`

```go
// GET {URL}/ejbca/ejbca-rest-api/v1/ca/status
// 200 → ok
// 401/403 → mTLS auth failed; report distinctly
// timeout / x509 error → wrap and surface
```

The response body is JSON `{"status":"OK","version":"...","revision":"..."}`.
We require `status == "OK"`.

### `GetCertificate`

```go
// 1. Generate RSA 2048 keypair locally.
// 2. Build CSR with CommonName = expanded username template.
//    If CertificateUserPrincipalNames is non-empty, add a subjectAltName
//    extension to the CSR with one otherName per UPN (OID
//    1.3.6.1.4.1.311.20.2.3, value type UTF8String). This must be
//    constructed via raw ASN.1 — Go's stdlib x509.CertificateRequest
//    doesn't have first-class UPN otherName support. The customer's EJBCA
//    Certificate Profile must have "Allow Extension Override" enabled or
//    the SAN will be dropped during issuance (per the existing SCEP dev
//    guide's gotchas).
// 3. Generate a 32-byte cryptographically-random value, hex-encode it,
//    use it as the `password` field. NOT persisted — created at call
//    time, sent, discarded. EJBCA's backend rejects null password for
//    any user-storage-enabled CA, so we have to send something; the
//    value is otherwise transparent to Fleet's operation under the
//    common auto-create-EE + permissive-password configuration.
// 4. POST /ejbca/ejbca-rest-api/v1/certificate/pkcs10enroll
//      {
//        "certificate_request": <PEM CSR>,
//        "certificate_profile_name": cfg.CertificateProfileName,
//        "end_entity_profile_name": cfg.EndEntityProfileName,
//        "certificate_authority_name": cfg.CertificateAuthorityNameEJBCA,
//        "username": <expanded UsernameTemplate>,
//        "password": <generated random hex>,
//        "include_chain": false,
//        "response_format": "DER",
//      }
// 5. Response 201 → base64-DER cert in .certificate field.
//    Decode → x509.Parse → wrap (key, cert) in PKCS12 with a generated password.
// 6. Return EJBCACertificate{PfxData, Password, NotBefore, NotAfter, SerialNumber}.
```

Error handling: EJBCA returns `{"error_code":N,"error_message":"..."}` on
failures. Surface the message in `ctxerr.Wrap`. Distinguish 401/403
(authentication or authorization), 404 (CA/profile name typo), 422 (CSR
rejected by profile rules) with their own wrapped messages.

### `EJBCACertificate`

```go
// server/fleet/ejbca.go (new file)
type EJBCACertificate struct {
    PfxData        []byte
    Password       string
    NotValidBefore time.Time
    NotValidAfter  time.Time
    SerialNumber   string
}

type EJBCAService interface {
    VerifyConnection(ctx context.Context, cfg EJBCACA) error
    GetCertificate(ctx context.Context, cfg EJBCACA) (*EJBCACertificate, error)
}
```

## Service layer

`ee/server/service/certificate_authorities.go` grows EJBCA-aware branches:

- `NewCertificateAuthority` validates an EJBCA payload then calls
  `svc.ejbcaService.VerifyConnection`.
- `UpdateCertificateAuthority` follows the same pattern.
- A new private helper `validateEJBCA(payload)` mirrors `validateDigicert`:
  required fields, URL parse, PEM cert / key parse, X509 keypair match, optional
  trust bundle parse. Username template may reference the same allow-listed
  Fleet vars as DigiCert (`HOST_END_USER_EMAIL_IDP`, `HOST_HARDWARE_SERIAL`,
  `HOST_PLATFORM`).

The service struct gains an `ejbcaService fleet.EJBCAService` field, injected
at construction.

## Profile processor wiring

In `server/mdm/apple/profile_processor.go`:

- Add `FleetVarEJBCADataPrefix` and `FleetVarEJBCAPasswordPrefix` to the prefix
  scan in `isCAConfigured` and the per-host expansion branch.
- The expansion calls `ejbcaService.GetCertificate(ctx, caCopy)` where
  `caCopy` has had Fleet vars replaced in `UsernameTemplate`.
- Resulting `PfxData` is base64-encoded and substituted into the profile XML,
  matching the DigiCert pattern exactly (`ReplaceExactFleetPrefixVariableInXML`).

## Datastore

Methods are reused — `NewCertificateAuthority`, `GetCertificateAuthorityByID`,
etc. are already polymorphic on `type`. The only datastore work is:

- Add EJBCA fields to the SQL select/insert helpers in
  `server/datastore/mysql/certificate_authorities.go`.
- Add EJBCA branch to `GroupCertificateAuthoritiesByType` (in
  `server/fleet/certificate_authorities.go`).
- Add EJBCA branch to the `postprocessRetrievedCertificateAuthority` masking
  logic.

## Endpoints

No new endpoints. The existing `POST/GET/PATCH/DELETE /fleet/certificate_authorities`
endpoints handle EJBCA via type dispatch. The same is true for
`request_certificate`. The GitOps spec endpoints are not extended for
EJBCA in this change — see "GitOps — deferred" below.

## GitOps — deferred

Not in this change. See "What this design does *not* solve". When pulled
forward, the shape will mirror DigiCert / NDES / SCEP entries in the
existing `certificate_authorities` spec, with the P12 carried as a
base64-encoded string (`client_p12_base64`) plus `client_p12_password`
since YAML can't carry binary files.

## Frontend

New form component at
`frontend/pages/admin/IntegrationsPage/cards/CertificateAuthorities/components/EJBCAForm/`,
modeled on `DigicertForm`. Field types:

- `url`: text
- `client_p12`: file upload (`.p12` only) **— required**
- `client_p12_password`: password input **— required**
- `trust_ca_bundle`: file upload (PEM, optional but strongly recommended)
- `certificate_authority_name_ejbca`, `certificate_profile_name`,
  `end_entity_profile_name`: text
- `username_template`: text with Fleet-var helper popover

The form has no enrollment-code field. Fleet generates the EJBCA
`password` value internally per enrollment (see proposal.md decision #5
and the design's `GetCertificate` walkthrough).

Server-side accepts only PKCS#12 in the POC. It decodes the bundle once,
extracts cert and key, persists them as PEM (key encrypted at rest), and
throws the P12 password away. PEM-direct upload (separate cert + key files)
is deferred — see proposal.md "Out of scope (deferred)".

## What this design does *not* solve

- **Token-based auth (OAuth bearer)**. Deferred. When added, expect to introduce
  a `client_id`/`client_secret`/`token_url` set of columns and a token cache
  layer. The `EJBCAService` interface will not need to change; a new
  `Service` implementation in the same package can be selected by config shape.
- **Dedicated "replace credentials" UX**. The POC uses the standard edit
  modal — re-upload the P12, save, done. A separate action surface (with
  its own confirmation flow, etc.) is deferred. Client-cert expiry IS
  surfaced in the POC (REQ-CA-EJBCA-12), so admins can see rotation coming
  via the CA list page.
- **Profile/CA name validation against EJBCA.** EJBCA has no list endpoint for
  profiles; we trust user input and surface the runtime error.
- **User-configurable enrollment code.** POC generates the `password`
  field internally per issuance. A customer-configurable field for
  scenarios where their workflow requires a specific shared value is a
  small follow-up — see proposal.md decision #5 for the customer-call
  question.
- **GitOps support.** POC ships API + UI only. No changes to
  `ValidateCertificateAuthoritiesSpec`, `BatchApplyCertificateAuthorities`,
  or the GitOps YAML schema. Follow-up alongside the production
  implementation.
- **PEM-direct upload of cert + key.** POC is P12-only. PEM-direct is a small
  follow-up — backend stores PEM either way, so only the upload form and one
  payload-parsing branch are new.

## Resolved engineering decisions

These were raised during exploration and answered by mirroring DigiCert /
checking the codebase. Captured here so reviewers don't relitigate.

- **`fleethttp` TLS config plumbing.** `WithTLSClientConfig` already exists
  in `pkg/fleethttp/fleethttp.go:36`. Use it directly. No package changes.
- **CSR CommonName.** Set to the expanded `username_template`. EJBCA's EE
  profile may override based on its config; we set the most informative
  value we have. Matches DigiCert's pattern of setting CN from a templated
  field.
- **`trust_ca_bundle` parsing.** Accepts multiple PEM blocks; append all to
  the `CertPool`. If no blocks decode, return a validation error at save.
- **GitOps secret handling.** N/A — GitOps support is deferred. When
  the follow-up pulls GitOps in, `client_p12_password` SHOULD be
  referenced via Fleet's existing `$ENV_VAR` substitution rather than
  committed in plaintext, mirroring how DigiCert's `api_token` is
  handled in the same spec today.
- **Validation behavior under transient EJBCA outages.** Hard-reject the
  create/update if `VerifyConnection` fails for any reason (timeout,
  connection refused, TLS error, non-200). The admin retries. Matches
  DigiCert behavior. Save-and-warn is not implemented for the POC; the
  admin's clearer signal is a failed save.
- **Activity logging scope.** Emit dedicated activity events for the
  EJBCA CA lifecycle: `created_ejbca_ca`, `edited_ejbca_ca`,
  `deleted_ejbca_ca` (mirroring the existing DigiCert pattern at
  `ee/server/service/certificate_authorities.go:1122` etc.). Per-host
  enrollment events use the existing generic profile-delivery activity —
  no EJBCA-specific event for that path.
