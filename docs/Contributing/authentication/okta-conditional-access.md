# Okta conditional access research

## Okta Verify

**What it is.** Okta Verify is Okta’s endpoint agent that enables FastPass (passwordless), device registration, and device posture collection for policy evaluation. Device assurance policies and posture checks are designed to work natively with Okta Verify and are evaluated at app sign-in.

**osquery support.** On macOS and Windows, Okta Verify bundles an osquery component to run **advanced posture checks** defined by admins (SQL-style checks that return pass/fail). This is conceptually similar to how fleetd leverages osquery, but here it’s packaged and invoked by Okta Verify for policy use.

**EDR integrations.** Okta Verify can ingest trust signals from supported EDR sources (for example, CrowdStrike and Windows Security Center) and expose them to policies via device assurance/custom expressions.

**Ecosystem support.** Other MDMs (e.g., Kandji, Jamf) document Okta Device Trust/Verify–based integrations as the standard way to gate access to Okta-protected apps.

**Recommendation.** If we want **Fleet** to control the device-gate experience or be the source of truth for posture, we should pursue a different path than using Okta Verify -- e.g., a **possession factor** (factor-only IdP) or Okta’s **Device Posture Provider** (EA) to feed Fleet’s signals into policies without relying on Okta Verify. ([Integrate Okta with Device Posture Provider](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/device-assurance-device-posture-idp.htm))

### How does Okta Verify talk to Okta during login?

* Okta Verify's FastPass first tries a localhost connection to the Okta Verify app
* If that can’t be used, the browser falls back to interactive deep links (custom URL scheme or app/universal links) to open Okta Verify.

---

## The key challenge: passing device information to web browser

Eventually, we want to support all major web browsers and mobile browsers on all major platforms.

Accessing device information using localhost is fragile:
- Chrome is restricting localhost access (Private Network Access)
- Safari blocks mixed content aggressively
- Firefox requires specific security headers
- Corporate proxies/firewalls may block localhost ports

We could use a browser extension to pass device information to the browser
- Requires a separate extension for each browser
- Does not work on mobile browsers
- Used by Kolide

We could use mTLS for our IdP endpoint
- Single implementation works across all browsers (Chrome, Safari, Firefox, Edge)
- No platform-specific code: same approach for Windows, macOS, Linux
- No browser extensions to develop and maintain
- No localhost battles with browser security policies
- Standard TLS: well-understood, battle-tested protocol
- We must manage certificates and renewal
- Infra becomes more complex
  - Proposal: load balancer terminates mTLS and passes the client cert as header
- Potential UX issues: users may need to picks a certificate from a list (this can be mitigated with a config profile)

## Using a possession factor instead of Okta Verify

**Overview.** Instead of relying on Okta Verify for device trust, we can make **Fleet** act as a **possession factor** (Okta IdP authenticator with *factor-only* usage). Okta will require this factor during login; Fleet validates the device (via **fleetd**) and returns pass/fail.

**How it works (happy path).**

```mermaid
sequenceDiagram
  autonumber
  participant U as User
  participant A as Target app (SAML/OIDC)
  participant O as Okta (IdP)
  participant F as Fleet server (SAML IdP)
  participant D as Device (fleetd)

  U->>A: Open app
  A->>O: Redirect to Okta
  O-->>O: Auth policy requires "Fleet device check" factor
  O->>F: Send AuthnRequest (factor-only)
  D->>F: fleetd proves device presence + compliance
  F-->>O: Factor assertion (pass)
  O-->>A: Issue assertion
  A-->>U: Access granted
```

**How it works with mTLS and load balancer (happy path).**
```mermaid
sequenceDiagram
    autonumber
    box Host
      participant U as User Browser
      participant D as Device (fleetd)
    end
    participant A as Target app (SAML/OIDC)
    participant O as Okta (IdP)
    participant LB as Load balancer
    participant F as Fleet server (SAML IdP)
    
    F->>D: Install client cert
    U->>A: Open app
    A->>O: Redirect to Okta
    O-->>O: Auth policy requires "Fleet device health" factor
    O->>U: Redirect to Fleet SAML endpoint
    U->>LB: mTLS connection with client cert
    Note over U,LB: Browser may prompt for cert selection
    LB->>LB: Validate client cert
    LB->>F: Forward request + cert as header
    F->>F: Verify device by cert serial/thumbprint
    F->>F: Check device compliance
    alt Device compliant
        F->>O: SAML assertion (pass)
        O-->>A: Issue assertion
        A-->>U: Access granted
    else Device not compliant
        F->>U: Redirect with context (direct or SAML response with error URL)
        U->>U: User fixes issues
    end
```

**Fleet requirements.**

* Expose a **factor-only SAML endpoint** that validates Okta’s AuthnRequest and returns a signed SAML response (pass/fail).
* On factor evaluation, query required device policies and return pass/fail.
* Provide a user-facing remediation URL when the factor denies.

**SAML flow with Okta using Fleet as IdP**
```mermaid
sequenceDiagram
    autonumber
    participant U as User / Browser
    participant O as Okta
    participant I as Fleet (SAML IdP)
    participant A as Downstream App

    U->>O: Navigate to app / Okta sign-in
    O->>O: Evaluate sign-on policy (requires external SAML IdP factor)

    Note over O,I: SP-initiated SAML<br/>HTTP-redirect binding to IdP SSO

    O-->>U: 302 Redirect to IdP SSO with AuthnRequest (+RelayState)
    U->>I: GET /sso?SAMLRequest=...&RelayState=...
    I->>I: Validate AuthnRequest (Issuer=Okta, ACS, Conditions)

    rect rgb(245,245,245)
    Note over U,I: Fleet performs device check
    I->>I: Check device policies and build SAML Assertion
    I->>I: Sign response/assertion with IdP private key
    end

    Note over I,O: HTTP-POST binding to Okta ACS

    I-->>U: HTML form (SAMLResponse, RelayState)
    U->>O: POST SAMLResponse to Okta ACS

    O->>O: Validation
    O->>O: Mark "external SAML factor" = SUCCESS in policy chain
    O->>O: Establish Okta session for user

    Note over O,A: Okta now acts as IdP to the downstream app

    O->>A: Send SAML response/assertion to app's ACS (via browser POST)
    A->>A: Validate Okta assertion, create app session
    A-->>U: Grant access
```

**Links.**

* [1Password Extended Access Management's Okta integration](https://blog.1password.com/extended-access-management-okta-guide)
* [Connect Kolide to Okta](https://www.kolide.com/docs/admins/setting-up-kolide/connect-kolide-to-okta)

---

## Using a device posture provider instead of Okta Verify (Early Access)

**Overview.** Instead of adding a visible factor step, we can make **Fleet** act as an **Okta Device Posture Provider**. During sign-in, Okta silently calls Fleet (as a special **SAML IdP**), Fleet returns posture **facts** (for example `IsManaged`, `IsCompliant`, OS/OSVersion), and Okta enforces them via **Device Assurance + app sign-on policies**. *(Feature is **Early Access** in Okta.)* ([Integrate Okta with Device Posture Provider](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/device-assurance-device-posture-idp.htm))

**How it works (happy path).**

```mermaid
sequenceDiagram
  autonumber
  participant U as User
  participant A as Target app (SAML/OIDC)
  participant O as Okta (IdP)
  participant P as Fleet server (SAML)
  participant D as Device (fleetd)

  U->>A: Open app
  A->>O: Redirect to Okta
  O-->>O: Policy requires device posture
  O->>P: Send SAML AuthnRequest (urn:okta:saml:2.0:DevicePosture)
  D->>P: fleetd provides device facts to Fleet
  P-->>O: SAML assertion with posture facts (e.g., IsManaged/IsCompliant)
  O-->>O: Evaluate Device Assurance + sign-on policy
  O-->>A: Issue assertion
  A-->>U: Access granted
```

**Fleet requirements.**

* Expose a **SAML IdP endpoint** that accepts Okta’s AuthnRequest with `urn:okta:saml:2.0:DevicePosture` and returns a **SAML assertion** containing posture facts per Okta’s schema. At minimum include `<Fact Name="IsManaged" Value="true|false" />`; `<Fact Name="IsCompliant" ...>` and other facts are optional.
* On device posture evaluation, query required device policies and return pass/fail.
* Provide signing cert/metadata; handle error states (for example, return **AuthnFailed** when device isn't managed/known).

**Pros.**

* **No extra factor UI**; posture is evaluated invisibly in Okta's policy engine.
* Posture becomes **first-class** across apps wherever the policy applies.

**Tradeoffs.**

* **Early Access** feature: some customers may not be willing to use it.
* Could not find any examples of security/MDM vendors actually using this.

## Other alternatives to Okta Verify

### Shared Signals Framework (SSF)

**Overview.** [SSF](https://openid.net/wg/sharedsignals/) is an OpenID protocol for transmitting security events and posture signals between providers asynchronously. It enables continuous monitoring by broadcasting events like "device became non-compliant" to multiple receivers (Okta, Cisco, SGNL, Google Workspace, etc.).

**Why it doesn't solve our use case.**

SSF has no native mechanism to identify which device the user is logging in from during authentication. The critical gap:

1. **Device identification problem**: SSF events are keyed by user identifiers (email, subject) or session identifiers, but there's no standard way for the browser to communicate "I'm device X" to Okta during login.
2. **No presence proof**: Unlike Okta Verify (which proves device presence via localhost/deep links) or our mTLS approach (which uses client certificates), SSF has no mechanism to link the browser session to a specific Fleet-managed device.

**When SSF is useful.**

SSF would be valuable for continuous monitoring use cases:
- Notifying Okta when a Fleet device becomes non-compliant (triggering session revocation)
- Sharing security events with other systems (EDR, SIEM)
- Multi-vendor risk correlation

**Recommendation.** SSF does not replace the need for synchronous device identification and authentication. Consider SSF as a future enhancement for continuous monitoring, but it cannot substitute for our possession factor + mTLS approach.

### Other certificate-based alternatives

* Issue a device certificate if it is passing policies, and use the certificate as a factor in Okta flow.
  * [Use your own certificate authority for managed devices](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/config-customer-provided-ca.htm)
  * Cert remains valid until revoked/expired (potential security gap for several hours)
    * Okta checks CRL (Certificate Revocation List) endpoint a few times per day
  * Can't show custom remediation messages

### Inline hooks

* Can work for OIDC to block access, but not for SAML.

---

## OIN (Okta integration network) / partnership not required

We can integrate with Okta conditional access without being in OIN. Okta provides public APIs and admin-console configuration for SAML, OIDC, SCIM, and device trust. No formal partner agreement or OIN listing is necessary to make the integration work technically. It takes weeks to get an OIN app reviewed.

Once we do an OIN app, we can use a configurable `tenant_url` or “base URL” parameter for each customer. This allows customers with different on-prem URLs to still use our integration without a single global SaaS endpoint.

Currently, we can also use any Okta test or developer account to develop and QA our conditional access integration before rolling it out to production customers.

## Why OIN / partnership still matters

Being in OIN gives our customers one-click install and preconfigured endpoints. It also increases our visibility to Okta admins and signals enterprise readiness, which helps shorten sales cycles. Jamf has an official Okta partnership and device trust integration, and Kandji has an OIN app listed. Having Fleet server in OIN positions us alongside these and other competitors. Kandji's OIN app simply bundles SAML+SCIM, so the app itself does not provide conditional access. Thus, OIN is largely a separate discussion from this feature.

---

## Links
* [Zach's POC](https://github.com/fleetdm/fleet/pull/17304)
* [Zach's POC video](https://www.loom.com/share/8bba3a146a374cf58f8b3673414559aa)
