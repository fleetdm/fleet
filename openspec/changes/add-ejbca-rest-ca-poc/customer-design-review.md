# EJBCA integration — design review

A summary of the proposed Fleet ↔ EJBCA integration, the design choices
we've made (with rationale and recommendations), and the open questions
we'd like to confirm before we cut the production release.

Audience: product, design, and the prospective customer
([`prospect-homerus`](https://github.com/fleetdm/fleet/issues/29176)).
The technical companion to this doc is
[`research.md`](./research.md).

## At a glance

- **What**: A new "EJBCA" certificate authority option in Fleet's
  Settings → Integrations → Certificate authorities, alongside the
  existing DigiCert, NDES SCEP, Smallstep, Hydrant, and Custom-SCEP
  integrations.
- **Why**: Lets IT admins migrating from AD CS to EJBCA continue
  deploying certificates to macOS hosts (WiFi / VPN) through Fleet
  without managing PKI on the side.
- **How (one sentence)**: The admin uploads a service certificate from
  their EJBCA to Fleet; Fleet uses it to enroll a per-host certificate
  via EJBCA's REST API and delivers it to the host through MDM.
- **State**: POC complete and verified end-to-end against EJBCA
  Community Edition; this doc is the bridge to the customer
  conversation and the production-release scoping.

## Background

The customer is moving their certificate authority from Microsoft
**AD CS to EJBCA** and wants to keep using Fleet to deploy WiFi/VPN
certificates to macOS hosts. They're a cert-native shop — the team
already operates PKI day-to-day — so the design assumes PKI fluency on
their side. Fleet already integrates with EJBCA over SCEP (the
existing "Custom SCEP Proxy" path); this work adds EJBCA's **REST
API** as a parallel, more modern path that customer expectations
increasingly assume.

## What we're building

From the IT admin's perspective:

1. Their EJBCA admin enrolls a service certificate that represents
   Fleet's identity inside EJBCA, and binds it to a custom admin role
   with narrow permissions (only the right CA, only the right
   end-entity profile).
2. They add an **EJBCA certificate authority** in Fleet's UI: upload
   the service certificate (as a `.p12` bundle with its password),
   paste the URL and a few EJBCA-side names, save.
3. They create a Fleet **configuration profile** that references the
   new CA (`$FLEET_VAR_EJBCA_DATA_<name>` /
   `$FLEET_VAR_EJBCA_PASSWORD_<name>` in the `<data>` and `<password>`
   fields of a `com.apple.security.pkcs12` payload).
4. As macOS hosts pick up the profile, Fleet enrolls a unique
   certificate per host from EJBCA and the host installs it into its
   keychain. WiFi or VPN clients then use the cert to authenticate.

The admin doesn't write code or schedule certificate renewals — Fleet
handles per-host enrollment automatically.

## What the customer needs to have configured on EJBCA's side

These are not Fleet design decisions; they're prerequisites the
customer's PKI admin must have in place on their EJBCA before the
integration can function. We document them so the customer-onboarding
guide can call them out as a checklist.

- **End Entity Profile configured to auto-create end entities during
  enrollment.** Without this, every macOS host that requests a cert
  fails until someone manually pre-creates an entry in EJBCA. This is
  a standard EJBCA Enterprise capability; the customer's PKI admin
  toggles it in the EE profile. EJBCA Community Edition cannot do it
  (which limits us to single-host testing for the POC against CE; the
  customer's production EJBCA Enterprise handles it natively).
- **A custom administrator role in EJBCA bound to Fleet's service
  certificate**, with narrowly-scoped access rules (only the right
  issuing CA, only the right End Entity Profile). We strongly
  recommend against using EJBCA's built-in Super Administrator role
  for Fleet's identity — least privilege is the right shape and means
  a compromise of Fleet's host doesn't expose the rest of the customer's
  PKI.
- **The relevant REST API protocols enabled** in EJBCA's system
  configuration (specifically "REST Certificate Management" — covers
  both the connection probe and certificate enrollment endpoints).
  Disabled by default in some EJBCA versions; enabled by default in
  production Enterprise deployments. We'll surface a clear error if
  the customer hits a disabled-protocol response.

## Already confirmed with the customer

### Authentication method: mTLS (mutual TLS)

Fleet authenticates to the customer's EJBCA by presenting a client
certificate — the same way EJBCA admins authenticate to its UI. The
customer's PKI admin enrolls a service certificate for Fleet, hands it
to us as a `.p12`, and binds that certificate to an admin role in
EJBCA. Confirmed with the customer; OAuth (the alternative we
considered) remains a deferred follow-up for shops that prefer it.

## Design decisions for review

Each subsection states **what's being decided**, **the
recommendation**, and **why**. Where there are realistic alternatives
we mention them briefly.

### Client certificate upload format

**Decision: Fleet accepts a PKCS#12 (`.p12`) file with its password
for the service certificate.**

When an EJBCA admin enrolls Fleet's service certificate, EJBCA's web
UI hands them a `.p12` file (a standard PKI bundle containing both
the certificate and its private key, encrypted with a password). We
accept that format directly in Fleet's UI: drag-and-drop or click-to-
upload the `.p12`, enter the password the admin set when downloading
it, and Fleet handles the rest.

The alternative format would be **PEM** — separate text files for the
certificate and the private key. Admins who export from a secrets
manager often have PEM; admins who download straight from EJBCA's UI
get `.p12`. For v1 we ship the `.p12` path only because that's the
default EJBCA workflow. PEM-direct upload is a small follow-up if
customers request it.

### Per-enrollment password handling

**Decision: Fleet generates a unique random password per enrollment
internally; no admin-visible field.**

EJBCA's enrollment API requires a `password` field on every
certificate request. Historically, this dates back to a pattern where
a human admin would pre-create an "end entity" with a known password,
then hand the password to the user, who would present it during
enrollment as proof of authorization. In modern automated integrations
(Fleet → EJBCA), this two-party flow doesn't apply: the mTLS client
certificate already authenticates Fleet, and EJBCA's access rules
control what Fleet can do.

Under the customer's typical configuration (auto-create end entities,
permissive password policy), the password EJBCA receives isn't
actually authenticating anything — it's a required field with no
gating role. Rather than surface a configurable field that's
misleading about its security value, **Fleet generates a unique
random password on each enrollment and discards it.** No persistent
shared secret to rotate, no audit story confused by reused passwords,
no admin-facing field to explain.

If a customer's EJBCA is configured to require a specific shared
password (uncommon but possible), the configurable-field variant is a
small follow-up. Customer question #7 below asks if that's the case
for `prospect-homerus`.

### Trust CA bundle (verifying EJBCA's TLS certificate)

**Decision: optional but strongly recommended field in the UI.**

When Fleet talks to EJBCA, the connection is over HTTPS, which means
Fleet has to verify EJBCA's TLS certificate. If the customer's EJBCA
is fronted by a publicly-trusted certificate (e.g., from a public CA
like DigiCert), Fleet's built-in trust store handles verification
automatically and the customer doesn't need to do anything. If
EJBCA's TLS cert is signed by an internal CA (the more common case
for self-hosted EJBCA — most of our deployments), Fleet needs to be
explicitly told to trust that internal CA.

We expose a **"Trust CA bundle"** field on the EJBCA add-CA form
where the admin can paste the CA chain. It's labeled optional because
the publicly-trusted case is real, but UI help text and error
messages strongly nudge customers in the more-common self-hosted
case to fill it in. When the connection probe fails because of an
untrusted cert, Fleet returns a specific error guiding them to this
field.

### mTLS service certificate: rotation and expiry notification

**Decision (recommendation: pull into v1 scope): show client-cert
expiry on the CA list and warn before it expires.**

Fleet's service certificate inside EJBCA has a lifetime (commonly 1
year). When it expires, Fleet's enrollment requests start failing
silently — existing host certificates keep working, but new
enrollments stop. From the admin's perspective, the failure mode is
"WiFi worked last month, why is it broken for new hires now?" — a
very expensive bug to debug.

Rotation itself works through the standard CA edit flow: open the
EJBCA CA in Fleet, re-upload the new `.p12`, save. We strongly
recommend **also shipping a proactive warning**:

- On the certificate authorities list page, show the days-until-expiry
  for each EJBCA CA as a badge
- Visual warning state ≤30 days, error state ≤7 days
- Optional: email or in-app notification at the same thresholds

The backend already parses and exposes the expiry date as part of the
POC; the work is purely surfacing it in the UI. Without this, the
customer's first rotation experience will be a production outage.

### Apple MDM payload coverage

**Decision: same coverage as the existing DigiCert integration.**

Fleet's existing DigiCert integration supports a defined set of
macOS configuration profile payload types (WiFi with EAP-TLS, VPN,
generic PKCS#12 distribution). The EJBCA integration adopts the same
surface so customers migrating from DigiCert have a 1:1 experience,
and so we don't expand the payload-validation surface area beyond
what the existing flow already handles. Subject Alternative Name
support also follows DigiCert: Common Name (always) plus Microsoft
User Principal Name (optional, embedded as an `otherName` SAN
extension).

### Relationship to the existing SCEP-EJBCA integration

**Decision: both paths coexist as alternatives; no migration required.**

Fleet already integrates with EJBCA via the "Custom SCEP Proxy" CA
type, which uses EJBCA's SCEP endpoint rather than its REST API.
That integration continues to work; customers using it don't need to
switch. The new EJBCA (REST) integration is offered alongside,
positioned for customers who:

- Are setting up a new EJBCA integration today (REST is the more
  modern path that EJBCA itself documents first)
- Want to take advantage of EJBCA Enterprise's auto-create-EE flow
  that doesn't require pre-registering every host

The two paths are independently configurable; a customer could even
run both side-by-side against the same EJBCA if they had a reason to.

### Platform scope

**Decision: macOS only in v1.**

The customer's stated use case is macOS WiFi/VPN. Each additional
platform (Windows, Linux, Android) has its own MDM profile schema
and would expand the validation/templating surface area meaningfully.
Other platforms remain on the roadmap as follow-ups for any customer
that asks; the EJBCA backend client we're building is platform-
agnostic, so this is a UI-and-validation scope decision, not a deep
architectural one.

### GitOps configuration

**Decision (recommendation: in v1): EJBCA CAs configurable via GitOps
alongside the UI and API paths.**

Fleet's existing CA types — DigiCert, NDES SCEP, Custom SCEP,
Smallstep, Hydrant, Custom EST — all support being configured via
GitOps YAML in the `certificate_authorities` block, in addition to
the UI and API. Customers using GitOps to manage their Fleet
configuration as code expect new CA types to participate in that same
flow on day one; shipping without it creates a real wart for
infrastructure-as-code shops who'd otherwise have to configure their
EJBCA CA out-of-band and remember not to overwrite it.

The mechanical pattern follows the existing CA types directly. The
one EJBCA-specific shape: because GitOps YAML can't carry binary
files natively, the service certificate (`.p12`) is supplied as a
base64-encoded string in the YAML — typically combined with Fleet's
existing `$ENV_VAR` substitution so the actual bytes stay in a
secrets manager and don't get committed to git.

```yaml
certificate_authorities:
  ejbca:
    - name: Corp_EJBCA
      url: https://ejbca.corp.example.com:8443
      client_p12_base64: $FLEET_SECRET_EJBCA_P12_B64
      client_p12_password: $FLEET_SECRET_EJBCA_P12_PASSWORD
      trust_ca_bundle: |
        -----BEGIN CERTIFICATE-----
        ...
      certificate_authority_name_ejbca: WifiIssuingCA
      certificate_profile_name:        WifiClientProfile
      end_entity_profile_name:         WifiUsers
      username_template:               $FLEET_VAR_HOST_HARDWARE_SERIAL
```

The POC scoped GitOps out for time; the production work brings it in.

## Notes on existing Fleet behavior that applies here

Things the customer should know about, but that aren't decisions —
these match how every other CA type in Fleet already works.

- **Updating a CA in Fleet does not re-issue certificates on hosts.**
  When the admin edits a CA configuration (rotates credentials,
  changes EJBCA-side profile names, updates the trust bundle), only
  *future* enrollments use the new config. Existing host certificates
  remain installed and valid until they expire naturally. This is the
  safe default and matches every other CA type — auto-reissuing on
  every config edit would hammer the customer's CA and force
  unnecessary cert churn on hosts.
- **Rotating Fleet's service certificate is the standard edit flow.**
  When Fleet's service cert in EJBCA expires (or before), the admin
  opens the EJBCA CA in Fleet, re-uploads the new `.p12`, saves.
  Fleet re-verifies against EJBCA before persisting; existing host
  certs are unaffected; new enrollments use the new credentials.

## Open questions for the customer

We've grouped these by what hinges on the answer.

**Functional prerequisites — confirming these unblocks the rest:**

1. Is your production EJBCA configured to **auto-create end entities**
   during certificate enrollment? *(EJBCA Enterprise supports this;
   we assume yes given your migration plan, but it's worth a sanity
   check.)*
2. Will you be providing a **trust CA bundle** for Fleet (your EJBCA's
   internal issuing CA chain), or is your EJBCA fronted by a
   publicly-trusted TLS certificate?
3. Are you comfortable binding Fleet's service certificate to a
   **custom least-privilege admin role** in EJBCA rather than Super
   Administrator? We'll document the minimum set of access rules.

**Strategic / posture questions — informs whether to bring OAuth
forward:**

4. Does your organization run an **OIDC Identity Provider** (Okta,
   EntraID, Keycloak, PingID) you'd want Fleet's EJBCA access to flow
   through eventually, or is mTLS the long-term fit?
5. What's your **expected lifetime** for Fleet's service certificate,
   and how do you typically rotate machine credentials?
6. Will Fleet be deployed as **Fleet Cloud (hosted)** or **self-hosted**
   on your infrastructure?

**Workflow questions — confirms our chosen defaults fit how you'll
actually use this:**

7. Will any of your existing automation depend on Fleet sending a
   **specific, known enrollment code** (a shared secret you've
   configured in your EJBCA's End Entity Profile)? If yes, we'd add a
   configurable field; if no, the simpler "Fleet generates one
   internally" design fits.
8. Are you planning to use **UPN SAN** for the issued certificates
   (Microsoft User Principal Name embedded in Subject Alternative
   Name)? We support it; if you're using DNS or email SAN instead, the
   defaults for new certs would change.

## What's deferred to a later release

Not gone, just not in v1. Each is captured in the technical design
with enough notes for the production work to pick up.

- **OAuth 2.0 bearer-token authentication** (alternative to mTLS, for
  shops that prefer OIDC-based service-to-service auth)
- **PEM-direct upload** for the service certificate (alternative to
  `.p12`)
- **A dedicated "Replace credentials" UX action** (rotation today goes
  through the standard edit modal, which is functional but not as
  prominent as a dedicated workflow)
- **User-configurable enrollment code** (Fleet currently generates a
  random password per call internally; configurable is a follow-up if
  any customer has the use case)
- **Windows / Linux / Android cert delivery** paths
- A small set of engineering-side cleanups around how Fleet packages
  EJBCA's certificate bundle for upload (production-grade replacement
  for a tactical POC shortcut — see `research.md`)

## Success criteria

We'll know we got this right when:

1. The customer's IT admin can add an EJBCA CA in Fleet, save it, and
   immediately see Fleet's connection probe to their EJBCA succeed.
2. A macOS host receiving the relevant configuration profile picks up
   a real, unique certificate issued by their EJBCA and uses it
   successfully for WiFi/VPN authentication.
3. Rotating Fleet's service certificate (when it expires) is a
   no-drama edit-and-save flow that doesn't disturb already-deployed
   host certificates **and the admin is warned via UI before it
   expires** (per the recommendation in §"mTLS service certificate:
   rotation and expiry notification").
4. The customer can run the full setup end-to-end from our
   contributor docs without us on the call.

All four are demonstrably true in the POC against a local EJBCA
container. Customer confirmation will validate them against a real
production EJBCA shape.

## What we're asking of you (the customer)

A 30–45 minute call to walk through:

- The eight open questions above
- Any pieces of the workflow that don't match how your team would
  actually use this
- Whether anything in the deferred list is a blocker for your initial
  rollout
- Confirmation on whether the **expiry-notification work in v1** (our
  recommendation) is something you'd actively use, or whether you'd
  prefer we ship without it and add it later

We'll capture your answers in our internal design doc and use them
to scope the production release for **Fleet 4.88.0**.
