# Conditional access

Fleet's conditional access feature lets IT and security teams enforce access controls on macOS and Windows hosts based on policy status. When a host fails a particular policy in Fleet, access to third-party apps can be blocked until the issue is resolved.

Fleet currently has built-in conditional access integrations with Okta (macOS only) and Entra (macOS and Windows):
- [Okta setup guide](https://fleetdm.com/guides/okta-conditional-access-integration)
- [Entra setup guide](https://fleetdm.com/guides/entra-conditional-access-integration)

You can also use Fleet's API for conditional access with [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) (macOS, Windows, and Linux).

To only require that hosts are managed by Fleet, without checking policies, see [Require Fleet-managed hosts in Duo](https://fleetdm.com/guides/require-fleet-managed-hosts-in-duo) or [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile).

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
