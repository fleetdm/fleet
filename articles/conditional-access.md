# Conditional access

Fleet's conditional access feature lets IT and security teams block access to third-party apps from hosts that don't meet your requirements. You can require that hosts:

- Are managed by Fleet
- Are passing specific Fleet policies. When a host fails one of these policies, access is blocked until the issue is resolved.

Fleet has built-in conditional access integrations that check policies:
- [Okta](https://fleetdm.com/guides/okta-conditional-access-integration) (macOS)
- [Entra](https://fleetdm.com/guides/entra-conditional-access-integration) (macOS and Windows)

You can also use Fleet's API for conditional access with:
- [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) (macOS, Windows, and Linux): require managed hosts that are passing critical policies
- [Duo](https://fleetdm.com/guides/duo-conditional-access-integration) (macOS, Windows, and Linux): require managed hosts

To require managed Windows hosts in Okta, see [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile).

You can also use Fleet's API for conditional access with:
- [Google](https://fleetdm.com/guides/google-conditional-access-integration) (iOS and iPadOS): require managed hosts

## How it works

1. IT enables the conditional access automation for the policies which determine access.
2. Fleet evaluates policies on each host.
3. Fleet communicates compliance status to the identity provider (IdP).
4. The IdP enforces access decisions, blocking users who are failing the policies from logging into protected apps.
5. Users remediate issues on their hosts and refetch to verify. Once the host passes all required policies, access is restored.

<meta name="articleTitle" value="Conditional access">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-04-27">
<meta name="description" value="Learn how Fleet's conditional access feature works to enforce access controls on hosts.">
