# Conditional access

Fleet's conditional access feature lets IT and Security teams enforce access controls on macOS hosts based on policy compliance. When a host fails a policy in Fleet, access to third-party apps can be blocked until the issue is resolved.

Conditional access works by integrating Fleet with your identity provider (IdP). Fleet evaluates policies on each host and communicates compliance status to your IdP. Your IdP then enforces access decisions — allowing or blocking login to apps — based on that status.

Fleet currently supports conditional access with the following identity providers:

- **Okta** — Fleet acts as an IdP authenticator factor in Okta. When a host fails a selected policy, the user is blocked from logging in to apps that require Fleet as an authentication factor. An mTLS reverse proxy is required for this integration.

- **Microsoft Entra** — Fleet reports device compliance status to Microsoft Intune, which Entra uses to enforce conditional access policies. When a host fails a selected policy, Fleet marks it as non-compliant in Entra, and the user is blocked from logging in to apps protected by Entra conditional access policies.

## Setup guides

For detailed setup instructions, see the guide for your identity provider:

- [Conditional access: Okta](https://fleetdm.com/guides/okta-conditional-access-integration) — Integrate Fleet with Okta to enforce conditional access on macOS hosts.
- [Conditional access: Entra](https://fleetdm.com/guides/entra-conditional-access-integration) — Integrate Fleet with Microsoft Entra to enforce conditional access on macOS hosts.

## How it works

1. **Fleet evaluates policies** on each host to determine compliance.
2. **Fleet communicates compliance status** to your IdP (directly for Okta, via Intune for Entra).
3. **Your IdP enforces access decisions**, blocking users on non-compliant hosts from logging in to protected apps.
4. **Users remediate issues** on their host and refetch to verify compliance. Once the host passes all required policies, access is restored.

## Bypassing conditional access

End users can temporarily bypass conditional access from their **My device** page if their host is not failing any critical policies. A bypass allows the user to complete a single login even with failing policies, and is consumed immediately upon successful login.

If any failing policy is marked critical, the bypass option is not available — the user must resolve the issue to regain access.

Bypass is enabled by default but can be disabled in **Settings > Integrations > Conditional access**.

<meta name="articleTitle" value="Conditional access">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-04-22">
<meta name="description" value="Learn how Fleet's conditional access feature works with Okta and Microsoft Entra to enforce access controls on macOS hosts.">
