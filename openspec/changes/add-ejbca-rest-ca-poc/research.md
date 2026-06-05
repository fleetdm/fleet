# Research: EJBCA REST CA integration

Informational doc capturing what we learned during the EJBCA POC investigation,
the risks we surfaced, and the decisions that still need customer
confirmation. This is not a spec — it's the durable record of the thinking that
fed [proposal.md](./proposal.md), [design.md](./design.md), and
[tasks.md](./tasks.md). If you're picking up this work mid-flight or revisiting
it in six months, start here.

## Customer + business context

- **Customer**: `prospect-homerus`, migrating their certificate authority from
  Microsoft AD CS to EJBCA. Use case is WiFi/VPN certificates on macOS hosts.
  Tagged `~customer promise` once they sign.
- **Upstream issues**:
  - [#30986](https://github.com/fleetdm/fleet/issues/30986) — user story (P2,
    4.88.0, T-shirt M).
  - [#45505](https://github.com/fleetdm/fleet/issues/45505) — this POC
    (4.87.0, `:release` `~timebox`).
  - [#29176](https://github.com/fleetdm/fleet/issues/29176) — parent /
    business tracker, links to EJBCA REST docs.
- **No Gong links** in any of those issues. No Figma. No prior design draft.
  Spec is being authored here for the first time.
- **Customer profile inference**: AD-CS-native. They have PKI muscle. mTLS will
  be a familiar mental model. They almost certainly buy EJBCA Enterprise (not
  Community Edition) in production.

## EJBCA authentication methods

EJBCA's REST API supports three authentication modes. Notably, **there is no
static API key** like DigiCert offers — EJBCA does not issue long-lived tokens
that a user can paste into a config. Every auth method below traces back to
either a certificate or an external identity provider.

### 1. mTLS client certificate — POC choice

**What it is.** During the TLS handshake, the client (Fleet) presents an X.509
certificate that EJBCA validates against its truststore (typically the
Management CA) and matches against role-member rules. If a matching role is
found, the certificate's effective permissions are the union of that role's
access rules.

**Intended purpose per EJBCA.** This is EJBCA's **default and canonical** auth
mode for both human admins (Admin UI / RA UI) and machine clients. EJBCA is
itself a CA — they expect every customer to use certs they've issued for
their own internal authentication. It is universally available across CE and
EE.

**Fits which customer profile.** Every EJBCA customer, by definition. Best fit
for orgs whose ops muscle is cert-based (AD CS migrators, classic enterprise
PKI shops). `prospect-homerus` fits squarely here.

**Tradeoffs.**
- ✅ Universal — works with any EJBCA install, CE or EE, no extra infrastructure.
- ✅ Posture matches the customer's existing PKI workflow.
- ❌ Service certs are long-lived (typically 1y) and must be rotated by hand —
  no automatic refresh.
- ❌ If Fleet's host is compromised, the attacker has a cert valid until the
  EJBCA admin revokes it.

### 2. OAuth 2.0 bearer token (OIDC) — deferred

**What it is.** The customer points EJBCA at an external Identity Provider
(EntraID, Okta, Keycloak, PingID are EJBCA-tested). API clients obtain a
short-lived JWT from that IdP and send it as `Authorization: Bearer <token>`.
EJBCA validates the JWT signature against the IdP's JWKS, then maps claims
(usually `sub` or a custom claim) to EJBCA roles in the same Roles & Access
Rules system.

**Intended purpose per EJBCA.** Added in 7.5 (Enterprise) and 9.3 (Community).
EJBCA's own marketing positions it for two distinct audiences:

- **Human admins**: log into Admin UI / RA UI via existing corporate SSO
  instead of carrying around a P12 in their browser. Inherit MFA, conditional
  access, deprovisioning, audit from the IdP.
- **Machine-to-machine API clients**, especially cloud-native / Kubernetes
  shops, where workload identity can mint short-lived tokens with no
  long-lived secret on the calling host's disk.

**Fits which customer profile.** Orgs that already run an OIDC IdP for
everything else and want EJBCA access to live in the same identity surface.

**Tradeoffs for Fleet → EJBCA specifically.**
- ✅ Short-lived tokens at the EJBCA boundary — IdP revocation propagates fast.
- ✅ Single audit/governance plane with the customer's other M2M integrations.
- ⚠️ Fleet still stores **long-lived secrets to the IdP** (typically
  `client_id` + `client_secret` for the OAuth client-credentials grant). The
  "no static credentials" pitch only fully lands when Fleet itself runs with
  cloud workload identity (GKE WI, EKS IRSA, EntraID WI), which doesn't fit
  most Fleet deployments — Fleet Cloud and many self-hosted Fleet installs
  don't.
- ❌ Requires the customer to operate an OIDC IdP and configure it as an
  EJBCA OAuth provider. Net new infrastructure for orgs that don't have it.

**Why deferred from POC.** Adds a meaningful new surface (IdP-registered
client, token endpoint, token cache, refresh handling, JWKS rotation) without
unlocking the customer who's asking for the feature today. Likely shares code
with Fleet's existing OIDC SSO support — same JWKS verification, audience
validation, etc. — so the follow-up is smaller than its scope suggests.

### 3. Public / anonymous access — not applicable

**What it is.** EJBCA can be configured to expose a subset of RA endpoints
without authentication, for trust-on-first-use enrollment scenarios.

**Intended purpose per EJBCA.** Bootstrap end-user enrollment in trusted
networks. EJBCA's docs explicitly call this "highly discouraged" for the CA
endpoints (the ones Fleet uses for `pkcs10enroll`).

**Why irrelevant for Fleet.** The endpoints Fleet calls
(`/v1/ca/status`, `/v1/certificate/pkcs10enroll`) are CA-functionality
endpoints that always require authentication in production setups. We do not
build for or support this mode.

### Summary

| Method | POC | Customer impact | Best fit |
|---|---|---|---|
| mTLS client cert | ✅ chosen | EJBCA admin enrolls + binds a service cert | Cert-native shops (this customer) |
| OAuth bearer | deferred | Customer registers Fleet as an OAuth client in their IdP | Customers with existing OIDC IdP |
| Public/anonymous | ❌ not built | n/a | n/a — discouraged by EJBCA |

OAuth remains a real follow-up. The proposal calls it out and design.md notes
that the `EJBCAService` interface doesn't need to change; only a new auth-
layer wrapper is needed when we eventually add it.

### Customer questions about authentication

These are exploratory — not POC-implementation decisions (those live in
[proposal.md → Auth decisions](./proposal.md#auth-decisions-requiring-customer-confirmation)).
They help us understand the customer's longer-term auth posture and how soon
OAuth follow-up work is likely to be asked for.

1. **Do you run an OIDC Identity Provider for other internal systems?**
   If yes, which? (EntraID, Okta, Keycloak, PingID are the EJBCA-tested set —
   if they use something else, OAuth follow-up gets riskier.)

2. **How do your existing automated systems authenticate to AD CS today?**
   Cert-based, Kerberos/Windows-auth, NDES-with-challenge, something else?
   This calibrates whether mTLS to EJBCA feels like a natural continuation or
   a step sideways.

3. **What's your typical service-account credential lifetime, and rotation
   cadence?** A shop that rotates every 90 days experiences mTLS as
   high-friction (manual rotation work); a shop that rotates yearly barely
   notices. Influences how strongly OAuth becomes attractive over time.

4. **Will Fleet be deployed via Fleet Cloud (Fleet-hosted) or self-hosted on
   your own infrastructure?** Self-hosted in a cloud-native environment opens
   the door to workload-identity OAuth (the "no static creds on disk"
   version). Fleet Cloud and on-prem VMs don't.

5. **Does your security team have a stated policy preference between
   long-lived static credentials and short-lived IdP-issued tokens for M2M
   integrations?** If short-lived is mandated by policy, OAuth follow-up
   moves to higher priority — possibly into the 4.88.0 production
   implementation rather than a later release.

6. **For the POC window, are you OK starting with mTLS only and revisiting
   OAuth after the integration is proven end-to-end?** Sets expectations on
   delivery shape.

Capture answers in the "Decisions confirmed with customer" table below — add
rows for these strategic questions as they come up.

## EJBCA REST API surface we actually use

| Operation | Method | Path | Notes |
|---|---|---|---|
| Connection probe | `GET` | `/ejbca/ejbca-rest-api/v1/ca/status` | Returns `{"status":"OK","version":"...","revision":"..."}` |
| Enroll cert | `POST` | `/ejbca/ejbca-rest-api/v1/certificate/pkcs10enroll` | CSR-based; response is base64-DER in `.certificate` |

That's it. Two endpoints. Discovery (list of CAs / profiles / EE profiles) is
intentionally out of scope — EJBCA has no list endpoint we can probe, and the
customer's EJBCA admin will type the names by hand.

## Edition matrix (CE vs EE)

What's in each edition that affects us:

| Feature | Community | Enterprise |
|---|---|---|
| `pkcs10enroll` REST endpoint | ✅ | ✅ |
| `GET /v1/ca/status` | ✅ | ✅ |
| mTLS authentication | ✅ | ✅ |
| OAuth bearer authentication | ✅ in 9.3+ | ✅ since 7.5 |
| `endentity` REST endpoint (manage EEs via API) | ❌ | ✅ |
| Auto-create end entity during `pkcs10enroll` | ❌ | ✅ (with config) |
| SCEP RA mode | ❌ | ✅ |

**The Fleet integration code does not change between editions.** Same
endpoint, same body, same auth. The delta is operational:

- On CE, the EJBCA admin must pre-create one end entity per CSR subject. This
  is untenable at fleet scale but fine for POC testing — the existing
  [ejbca-scep-testing.md](../../../docs/Contributing/product-groups/security-compliance/ejbca-scep-testing.md)
  guide documents the `bin/ejbca.sh ra addendentity` workflow we'll reuse.
- On EE, with `Use auto-add` enabled on the End Entity Profile, EJBCA creates
  the end entity on the fly. This is the production-realistic flow.

**Customer assumption**: `prospect-homerus` will run EE in production and
their EE profile will be configured for auto-add. We need to confirm this
on the customer call (see proposal § Auth decisions, item 4).

## EJBCA Enterprise trial

EJBCA EE is paid (Keyfactor). For verifying the auto-create-EE flow we cannot
test on CE, Keyfactor offers:

- **30-day AWS/Azure free trial** at https://www.keyfactor.com/try-ejbca-enterprise/
- **Recorded demo + personalized walkthrough** (sales motion)
- **Keyfactor PQC Lab Test Drive** on Azure Marketplace — pre-configured EE
  9.2 instance

Plan: build and primarily test against the CE Docker container. Spin up the
30-day trial near the end of the POC for an EE-specific validation pass that
exercises auto-create-EE end-to-end.

## Customer setup flow (EJBCA admin side)

```
1. Certificate Profile        →  CA Functions → Certificate Profiles
   Clone ENDUSER → "fleetRESTAdmin"
   EKU: Client Authentication; KU: Digital Signature, Key Encipherment
   Validity: 1y typical

2. End Entity Profile         →  RA Functions → End Entity Profiles
   Name: "fleetRESTAdmin"
   Default + Available Cert Profile: fleetRESTAdmin
   Default + Available CA: Management CA

3. Enroll the Fleet service   →  RA Web → Make New Request
   cert. Username: fleet_rest_service. CN: Fleet REST Service
   Download as PKCS#12, password set by admin

4. Administrator Role         →  System Functions → Roles and Access Rules
   Template: RA Administrators
   Authorized CAs: <issuing CA for device certs>
   Authorized EE Profiles: <the EE profile Fleet will enroll against>
   Access rules: read CA, create/edit end entity, view+issue cert

5. Bind cert → role           →  Role → Members
   Match with: X509:CN, CA: Management CA, Value: "Fleet REST Service"

6. (Optional) Export the      →  RA Web → CA Certificates and CRLs
   Management CA cert as PEM     For the trust bundle Fleet will use to
                                  verify EJBCA's HTTPS server cert
```

What Fleet receives from the admin:
- The PKCS#12 file containing Fleet's client cert + private key
- The P12 password (used once to decrypt; not persisted)
- Optional: trust CA bundle (PEM; required when EJBCA's HTTPS cert isn't
  publicly trusted, which is the typical self-hosted case)
- EJBCA REST URL
- CA name, cert profile name, EE profile name (all free-text)
- Username template (uses Fleet vars)
- Enrollment code

POC accepts **PKCS#12 only**. PEM-direct upload (separate cert + key files)
is deferred — small follow-up, same backend.

## Risks

### High

- **End-entity auto-creation assumption.** The whole integration depends on
  EJBCA being configured to auto-create end entities. If the customer's EE
  profile doesn't permit it, every `pkcs10enroll` returns "user not found"
  and the integration is non-functional at fleet scale. This is the single
  biggest "you have to do X on your EJBCA" requirement and must be confirmed
  with the customer before commit.
- **Cert revocation drift.** If the EJBCA admin revokes the Fleet service
  cert for any reason, all enrollments break with a TLS handshake error.
  Fleet has no way to detect this proactively until the next save. POC
  acceptable; production needs a periodic probe.

### Medium

- **Self-signed EJBCA in dev.** Most dev EJBCA instances use a Management-CA-
  issued HTTPS cert. The trust-bundle upload handles this, but skipping it
  yields a confusing TLS error. Doc clearly.
- **CSR DN sensitivity.** EJBCA may enforce strict matching of CSR subject to
  end-entity DN. The existing SCEP guide notes "CN-only subjects" is the safe
  default. Stick with it; document it.
- **Username template footguns.** If the user picks a template that doesn't
  uniquify per host (e.g., a constant), EJBCA reuses the same end-entity and
  the password is consumed after first enrollment. POC should either:
  - validate that the template references at least one host-scoped Fleet var, or
  - accept this and document it.
- **No discovery, no validation of profile names.** A typo in any of the
  three names surfaces only on first enrollment, not on CA save. POC
  acceptable; production should have a better error message.

### Low

- *(none currently outstanding — see "Resolved" below for previously
  flagged items that have since been answered.)*

### Resolved (kept for posterity)

- **"How does PATCH affect existing certs?"** — confirmed by reading
  `ee/server/service/certificate_authorities.go:1072` (`UpdateCertificate-
  Authority`). PATCH is a metadata update only: validate → write row →
  log activity → return. No re-enrollment is queued, no profile is re-
  resolved, no per-host work. Existing host certificates are unaffected.
  Same behavior as every other CA type. This is the right design — auto-
  reissuance on every CA edit would hammer the customer's CA and break
  reversibility. The expiry surfacing in REQ-CA-EJBCA-12 exists precisely
  because the "no automatic action" property means expired mTLS material
  silently breaks new enrollments while existing hosts keep working.
- **"Hot-swap on the CA row vs delete-and-recreate"** — non-issue. The
  existing PATCH endpoint supports updating any field including
  `client_p12` + `client_p12_password`. The edit modal in the UI just
  re-exposes the same fields as create. No separate "hot-swap" code path
  is needed; only a dedicated "Replace credentials" button is deferred,
  which is purely a UX nicety.
- **"Migration-phase EJBCA naming churn"** — also non-issue. Renaming an
  EJBCA CA / cert profile / EE profile during the customer's PKI
  migration is handled by the same PATCH mechanism. Existing host certs
  are unaffected; admin updates Fleet's CA row to the new name and new
  enrollments resume. Same flow as a DigiCert profile_id change or a
  SCEP URL change.
- **`fleethttp.WithTLSClientConfig` already exists** at
  `pkg/fleethttp/fleethttp.go:36`. No package additions needed for the
  mTLS plumbing.
- **Enrollment code: required by API, not authenticating in our config.**
  Verified by reading `SignSessionBean.java` in the EJBCA source: the
  backend rejects `password=null` for any CA with `useUserStorage=true`
  (which is essentially all production EJBCA deployments). So we have
  to send a non-null value. Verified separately that the
  `EnrollCertificateRestRequest` JSON-schema layer has *no*
  `@NotNull`/`@NotEmpty`/`@NotBlank` annotation on `password` — the
  requirement comes purely from backend validation.

  Under the auto-create-EE + permissive-password configuration we're
  designing against, a static enrollment code doesn't authenticate
  anything — an attacker holding the mTLS cert can supply any value
  and EJBCA will accept it. So the field has essentially no security
  value at that configuration.

  Decision: POC removes the user-facing `enrollment_code` field
  entirely. Fleet generates 32 bytes of `crypto/rand` per enrollment,
  hex-encodes, sends as `password`, discards. This satisfies the API
  without introducing a persistent shared secret or a customer
  configuration question that has no real effect.

  User-configurable enrollment code is captured as a deferred follow-up
  for customers whose EE profile genuinely requires a specific value
  or whose workflow depends on a known shared secret (see proposal.md
  decision #5 — a customer-call question to determine if the follow-up
  needs to be pulled forward).

## Decisions confirmed with customer

> Fill in during/after the customer call. One row per question from
> [proposal.md → Auth decisions](./proposal.md#auth-decisions-requiring-customer-confirmation).

| # | Decision | Customer answer | Date | Notes |
|---|---|---|---|---|
| 1 | mTLS-only POC, OAuth deferred | ✅ Confirmed | 2026-06-04 | Customer confirmed mTLS is preferred auth method |
| 2 | POC accepts PKCS#12 only. PEM-direct deferred — acceptable? | TBD | | |
| 3 | Will customer provide a trust CA bundle? | TBD | | |
| 4 | EE profile auto-creates end entities? | TBD | | |
| 5 | Workflow requires a configurable enrollment code? (POC generates internally) | TBD | | |
| 6 | Custom EJBCA admin role (vs Super Admin)? | TBD | | |
| 7 | Service-cert lifetime + rotation via standard edit modal acceptable? | TBD | | |

## Open follow-ups

- **BER → DER normalization for the PKCS#12 decode path. PRODUCTION
  BLOCKER for #30986.** EJBCA's RA Web (Java/BouncyCastle) emits BER
  PKCS#12 bundles with indefinite-length sequences. Both Go PKCS#12
  libraries — `software.sslmate.com/src/go-pkcs12` and
  `golang.org/x/crypto/pkcs12` — require strict DER and reject BER. The
  POC unblocks this by shelling out to the `openssl` binary, which
  handles BER. **This subprocess approach must not ship to production**:
  it adds a runtime binary dependency Fleet doesn't otherwise have,
  expands the trust surface (subprocess hardening, PATH lookup, openssl
  version drift across host OS distros), and complicates the security
  posture around password handling.

  Production options, ranked:
  1. *Implement an in-process BER → DER normalizer in pure Go.* The
     conversion is mechanical (recursively replace indefinite-length
     constructs with definite-length equivalents). Roughly 200–300 lines
     including tests against EJBCA's actual output. No runtime
     dependency. Best long-term shape.
  2. *Patch BER support into one of the Go PKCS#12 libraries upstream
     and depend on the patched version.* Cleaner externally but slower
     (depends on maintainer cycles).
  3. *Accept an additional upload format (e.g., separate PEM cert + PEM
     key)* — sidesteps the BER issue entirely for customers who can
     produce PEM out of EJBCA via openssl on their side. Already on
     this Open follow-ups list under "PEM-direct upload".

- OAuth 2.0 bearer-token authentication. Probably shares code with Fleet's
  existing OIDC SSO support — the same JWKS verification, audience
  validation, etc. File a follow-up issue under #30986 once POC merges.
- Hot-swap of client cert/key on an existing CA row, without re-creating it.
- UI surfacing of client-cert expiry on the CA list page.
- Cert-revocation detection (periodic probe).
- User-configurable enrollment code (for customers whose workflow
  requires a specific shared value — see proposal decision #5).
- Windows / Linux / Android cert delivery paths via EJBCA.
- Profile-name validation against EJBCA's APIs (none today; would need a
  Keyfactor feature request or a `GET /endentityprofile` if EE adds one).

## Source references

- [EJBCA REST Interface](https://docs.keyfactor.com/ejbca/latest/ejbca-rest-interface)
- [EJBCA Authentication Methods](https://docs.keyfactor.com/ejbca/latest/authentication-methods)
- [EJBCA OAuth Providers](https://docs.keyfactor.com/ejbca/latest/oauth-providers)
- [Understanding OAuth in EJBCA](https://www.ejbca.org/resources/understanding-oauth-in-ejbca-and-how-it-can-help-you-get-started/) — vendor positioning
- [Tutorial — Create roles in EJBCA](https://docs.keyfactor.com/ejbca/latest/tutorial-create-roles-in-ejbca)
- [ServiceNow REST Integration — Configure EJBCA](https://docs.keyfactor.com/ejbca/latest/servicenow-rest-integration-configure-ejbca) — closest analog
- [RA Administrator Access Rules](https://docs.keyfactor.com/ejbca/latest/ra-administrator-access-rules)
- [Keyfactor/ejbca-ce Discussion #635](https://github.com/Keyfactor/ejbca-ce/discussions/635) — CE pre-creation requirement
- [Try EJBCA Enterprise (30-day trial)](https://www.keyfactor.com/try-ejbca-enterprise/)
- Local repo: [ejbca-scep-testing.md](../../../docs/Contributing/product-groups/security-compliance/ejbca-scep-testing.md) —
  existing SCEP dev guide, sibling to the REST guide we'll write.
