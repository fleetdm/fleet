Managing identity across macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android usually means juggling separate user directories for every application a fleet touches. Federated authentication consolidates that sprawl into a single identity provider, and the same trust model becomes the foundation for tying device compliance to application access. This guide covers how federated authentication works, how it differs from SSO and traditional identity, and how it connects to device management and Zero Trust.

## What is federated authentication?

Federated authentication lets one trusted identity provider (IdP) handle login on behalf of multiple applications and services. Instead of each application maintaining its own usernames and passwords, applications hand off the login process to a central IdP. Users sign in once at the IdP, and the applications they access accept that sign-in without requiring separate credentials.

An organization running Microsoft Entra ID or Okta can grant employees access to internal tools, cloud services, and partner applications without duplicating user accounts. The IdP becomes the authoritative source for identity, and managing access centrally simplifies both provisioning and deprovisioning.

Federation also extends across organizational boundaries through trust relationships. Partners, contractors, or an acquired company's users can authenticate with your IdP or, in some cases, their home IdP and access your resources without parallel accounts. Cross-organizational access is a common driver for federation, though the same protocol-based trust model applies any time an application delegates authentication to a separate IdP.

## Why federated authentication matters for security and compliance

Centralizing authentication through federation can improve security and compliance outcomes compared to distributed identity models.

### Reduced attack surface

Every application that stores its own credentials is another breach target. Federation often removes password storage from individual applications entirely, so a breach at that vendor doesn't expose user passwords.

### Centralized policy enforcement

With federation, teams define authentication policies once at the IdP. Multi-factor authentication (MFA) requirements, password rules, session timeouts, and risk-based access rules then apply consistently across federated applications. Without federation, enforcing MFA everywhere often requires per-application configuration and vendor-specific limitations.

### Faster offboarding

When an employee leaves, disabling the IdP account blocks new authentication attempts to federated services. Existing sessions or unexpired tokens may still allow access until they expire or an admin revokes them. In traditional identity models, former users can retain access in applications overlooked during offboarding. This risk grows when ticket-driven, app-by-app deprovisioning is the primary process.

### Compliance framework alignment

Federation simplifies compliance because authentication controls live in one place rather than scattered across individual applications. Audit teams can review a single set of policies, access logs, and enforcement rules instead of gathering evidence from every application separately. The National Institute of Standards and Technology's (NIST) [Digital Identity Guidelines](https://pages.nist.gov/800-63-4/) (SP 800-63) include a dedicated Federation Assurance Level (FAL) dimension. The latest revision ([SP 800-63-4](https://csrc.nist.gov/pubs/sp/800/63/4/final)) defines progressively stricter requirements at each level, so teams can match controls to risk.

### Security risks to watch

Federation isn't without risk. If an attacker compromises your IdP, they can gain access to every federated service. Token forgery attacks, where an attacker crafts valid-looking assertions, are particularly dangerous because they can impersonate any identity and are harder to detect than credential theft. You can reduce risk by protecting your IdP with phishing-resistant MFA (FIDO2 security keys or passkeys) and locking down key management. On managed devices, MDM configuration profiles can deploy those credentials and bind them to the hardware, extending phishing resistance to device login as well as federated application access. Monitor federation relationships for anomalies such as unexpected certificate changes or unusual token issuance patterns.

## When federated authentication fits

Federated authentication fits best in environments with specific characteristics.

- Multi-application environments: When users access more than a handful of SaaS tools and internal applications daily, federation reduces password sprawl by eliminating separate credentials.
- Cross-organizational collaboration: When teams share resources with external partners or an acquired company, federation lets their users sign in with their own IdP or with identities managed in yours. Your team doesn't need to provision guest accounts in each application.
- Compliance-driven access control: When controls map to frameworks like NIST 800-53 or FedRAMP, federation consolidates policy enforcement, audit trails, and access reviews into a single layer.
- Multi-platform device fleets: When organizations manage macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android devices together, enrollment and compliance checks can tie back to a single identity source.

Federation adds architectural complexity, so it often isn't worth the overhead for a small team with only a few applications. Once access spans a large application portfolio, maintaining separate user directories and handling per-app deprovisioning typically costs more than federation itself.

## Federated authentication vs. SSO and traditional identity

These terms describe different things. Traditional identity is about where credentials are stored. SSO is about the user's login experience. Federation is about how separate identity systems trust each other.

- Traditional identity management: Each application stores and validates its own credentials independently. IT teams provision, deprovision, and reset passwords per application. This can work for small environments but increases the chance of missed accounts during offboarding as application counts grow.
- Single sign-on (SSO): Users authenticate once and access multiple applications without re-entering credentials. SSO can work inside a single organization's directory without federation. SSO describes the login experience, not how different identity systems connect.
- Federated authentication: Federation extends authentication across trust boundaries between separate identity systems. It lets users from one organization access applications managed by another without needing separate accounts. Federation can exist without SSO, and SSO can exist without federation, but most enterprise deployments combine both.

When you evaluate an identity architecture, the practical question is which combination of these three approaches your application portfolio and partner relationships require.

## How federated authentication works under the hood

Federation relies on token exchange between three parties: the user, the identity provider, and the relying party (the application). When a user tries to access an application, the application redirects them to the IdP. The user signs in at the IdP, and the IdP sends a signed token back to the application confirming who the user is. The application trusts that token and grants access without needing its own credential store.

The two most common protocols are Security Assertion Markup Language (SAML) 2.0 and OpenID Connect (OIDC). SAML is widely used for enterprise web applications and legacy integrations, exchanging identity information as signed Extensible Markup Language (XML) documents. OIDC is often a better fit for cloud-native applications, mobile apps, and API-driven services, exchanging identity information as JSON Web Tokens (JWTs). Many organizations run both protocols simultaneously, since their application portfolio spans both categories.

A few practical considerations apply regardless of protocol. Certificate management is a common failure point: if the signing certificate between the IdP and an application expires, authentication breaks for every user of that application. Deciding which user attributes (email, department, group membership) to share with each application also matters, since sending too much data increases exposure and sending too little breaks application functionality. Getting these decisions right during initial setup avoids rework as you add more applications to the federation.

## How federated authentication intersects with devices, device management, and Zero Trust

The model above applies most cleanly to web applications and modern apps. Authentication beyond that becomes messier at device login, across networks, and over legacy protocols. That is where MDM matters: most of these edges sit in the device management layer rather than in the IdP itself. Zero Trust programs treat the two as one problem. Access decisions tie device state and user identity together, evaluating both signals before granting access.

### Platform-specific device authentication

Each operating system handles federated authentication differently, which is why multi-platform environments can feel inconsistent.

- Windows: Devices can join a directory or cloud identity service directly, but device join and hybrid join are not the same thing as federated authentication. Conditional access policies can require device compliance before granting access to federated applications.
- macOS: [Platform Single Sign-On](https://support.apple.com/guide/deployment/platform-sso-for-macos-dep7bbb05313/web) (Platform SSO) connects the Mac login experience to an organization's IdP. Apple's [Extensible Single Sign-On](https://support.apple.com/guide/deployment/extensible-single-sign-on-payload-settings-depfd9cdf845/web) (Extensible SSO) framework extends that integration to Safari and native apps through configuration profiles.
- Linux: Device sign-in usually goes through SSSD (System Security Services Daemon) with Kerberos or LDAP backends. SSSD connects to directory services like Active Directory or FreeIPA, or to cloud-native identity providers via synchronization layers.
- iOS, iPadOS, and Android: Mobile platforms tie identity to MDM enrollment rather than to a device login screen. Apple's Extensible SSO framework extends to iOS and iPadOS through configuration profiles, and Android relies on managed accounts and IdP sign-in flows handled at the app layer.

Aligning device sign-in identity with application sign-in identity simplifies device ownership models, support workflows, and audit evidence. Each of these mechanisms is also a configuration that has to be deployed, monitored, and kept current across every managed device. Platform breadth often dominates real-world tooling decisions for that reason.

### Local accounts and device access

Federation centralizes identity for applications, but device login often still depends on a local account on the device itself. Most operating systems maintain a local user record so the device remains usable when no network or IdP is reachable. Many organizations sync a copy of the IdP credential into that local account during enrollment. The result is that an authenticated user can usually sign into a managed Windows or Mac device even during an IdP outage, while access to federated applications is blocked. This split matters for support workflows, recovery scenarios, and compliance reporting, since the device-level identity and the application-level identity are not always the same. Teams need to decide which device platforms maintain local accounts, how those accounts are governed, and which devices fall back to local credentials versus locking out entirely.

### Conditional access and device compliance

Conditional access policies depend on a device-management signal to verify compliance before granting access to federated applications. The IdP decides who can authenticate, but it relies on MDM to confirm whether the requesting device meets policy. Microsoft Entra ID, Okta, and Google Workspace each implement this pattern under different names: Conditional Access, Device Trust, and Context-Aware Access. Every implementation needs the same underlying device posture signal. Without that link, a user could pass identity verification on a compromised or non-compliant device. The key design decisions are which applications to gate, whether privileged roles need stricter rules, and whether corporate-owned and personally owned devices need different requirements. Depth of integration varies by operating system, which is where a multi-platform device management solution closes the gap.

### Zero Trust alignment

Zero Trust treats every access request as potentially hostile, regardless of network location. Federation provides the identity verification layer, and device compliance checks provide the device trust layer. Together, they let organizations make access decisions based on who is asking and whether their device meets security requirements, rather than relying on network location alone. In practice, the IdP requests a posture check at access time. The device management tool reports compliance state (encryption, OS version, agent presence), and the IdP combines that with identity signals to allow or block the session. This matters most for remote and hybrid workforces, where devices regularly connect from networks the organization doesn't control.

## Federated authentication and device management with Fleet

The platform-specific inconsistencies described above are a common friction point. Windows ties device compliance to conditional access, macOS uses Platform SSO and Extensible SSO, and Linux relies on SSSD with Kerberos backends. Mobile platforms add their own enrollment and sign-in patterns. Teams managing all of these end up maintaining separate identity integration patterns for each operating system.

Fleet connects federated identity to device management across macOS, iOS, iPadOS, Windows, Linux, ChromeOS, and Android from a single console. Fleet supports SAML-based SSO with major identity providers and just-in-time account provisioning. Fleet also uses the System for Cross-domain Identity Management (SCIM) protocol to map IdP attributes (username, groups, department) to managed devices, what Fleet calls "foreign vitals." Fleet can also require IdP authentication during [device enrollment](https://fleetdm.com/guides/end-user-authentication).

Fleet ships native conditional access integrations with Microsoft Entra ID and Okta. The Entra integration covers macOS and Windows; the Okta integration is shipped and documented. Both share device compliance state with the IdP, so federated app access is blocked when a device falls out of policy.

Fleet Premium also closes the loop on conditional access. When a device fails a compliance check, Fleet's policy automations can install software, run remediation scripts, fire webhooks, or open tickets in Jira, Zendesk, or ServiceNow. Each policy retries up to three times before access stays blocked.

Fleet's vulnerability detection automatically identifies CVEs on managed devices by matching installed software against NVD, KEV, and EPSS data, giving access policies a current posture signal to act on. Because every Fleet product is open source, security teams can audit how compliance is evaluated and reported to the IdP, which matters when access decisions hinge on that signal.

Fleet's [GitOps workflows](https://fleetdm.com/docs/using-fleet/gitops) let identity rules, conditional access policies, and device compliance baselines live in version-controlled YAML. Changes go through pull-request review and CI/CD before deployment, the same as any other infrastructure code.

[Schedule a demo](https://fleetdm.com/contact) to see how Fleet ties federated identity to device enrollment and compliance across your fleet.

## Frequently asked questions

### Does federation replace a local directory?

Not necessarily. Many organizations keep a local directory (like on-premises Active Directory) alongside their cloud IdP. Federation defines how identity assertions pass between systems, but it doesn't require removing existing directories. Hybrid setups are common, especially when legacy applications don't support modern federation protocols and still need Kerberos or LDAP authentication.

### How does federation handle contractors and temporary workers?

Contractors are often managed in your own IdP rather than their employer's IdP, though federation can also support partner access in some environments. You define access in the identity system you use, and users receive scoped access to specific applications. When the engagement ends, removing that access revokes it without your team needing to track individual accounts across applications.

### What happens if the identity provider goes down?

If the IdP is unavailable, users typically can't authenticate to federated applications. Depending on the configuration, there may be limited fallback options, but most designs assume an IdP outage blocks web application access. Device login can behave differently when local account syncs keep a local password copy on the machine. Users can still log into a Windows or Mac device even if the IdP is unavailable.

### How does federated authentication connect to device management?

Federated authentication ties user identity to device enrollment and compliance so access decisions can consider both. When device setup requires IdP authentication, it becomes possible to prove which identity claimed the device. Fleet supports SAML-backed [end-user authentication](https://fleetdm.com/guides/end-user-authentication) during enrollment across macOS, iOS, iPadOS, Windows, Linux, and Android. Fleet's conditional access integrations share device compliance state with Microsoft Entra ID and Okta, so federated app access can depend on device posture. Learn how Fleet brings [identity-aware device management](https://fleetdm.com/guides/setup-experience) to your environment.

<meta name="articleTitle" value="Federated authentication explained: SSO, protocols, and device management">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-08">
<meta name="description" value="Learn how federated authentication works, how it differs from SSO, and how it connects to device management and Zero Trust strategies. ">
