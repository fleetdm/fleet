# Conditional access

With Fleet, you can enforce conditional access on macOS hosts by integrating with your identity provider (IdP). When a host fails a policy in Fleet, IT and Security teams can block access to third-party apps until the issue is resolved.

Fleet supports conditional access with the following providers:

- **[Okta](https://fleetdm.com/guides/okta-conditional-access-integration)** — Fleet acts as an IdP authenticator factor in Okta, blocking login when a host fails a selected policy.
- **[Microsoft Entra](https://fleetdm.com/guides/entra-conditional-access-integration)** — Fleet reports device compliance status to Entra via Intune, allowing Entra conditional access policies to block non-compliant hosts.

## How it works

Fleet evaluates policies on each host. When conditional access is enabled for a policy, Fleet communicates the host's compliance status to your identity provider. If the host is failing a selected policy, the identity provider blocks the end user from logging in to apps that require compliance.

Once the end user resolves the issue on their host, they can click **Refetch** on the **My device** page to verify the policy is now passing. After the policy passes, access is restored.

## Configure conditional access policies in Fleet

After you've completed the provider-specific setup ([Okta](https://fleetdm.com/guides/okta-conditional-access-integration) or [Entra](https://fleetdm.com/guides/entra-conditional-access-integration)), head to **Policies** in Fleet. Select the fleet that you want to enable conditional access for.

1. Go to **Manage automations** > **Conditional access** and enable conditional access.
2. Select the policies you want to enforce conditional access with.
3. Save.

## Bypassing conditional access

End users can temporarily bypass conditional access from their **My device** page if their host is not failing any critical policies. To trigger a bypass, click a non-critical failing policy labeled **Action required**, select **Resolve later**, and confirm in the following modal. The bypass allows the user to complete a single login even with failing policies and is consumed immediately upon successful login.

If a host is failing multiple conditional access policies, the bypass option is only available if **no** failing policy is marked critical. If any one of the failing policies is marked critical, the end user will not see the option to bypass and must resolve the issue to regain access. (You can update a policy's `critical` setting on the **Edit policy** page.)

This feature is enabled by default, but can be disabled by checking the **Disable bypass** checkbox in **Settings** > **Integrations** > **Conditional access**. 

### Per-policy bypass

> **Experimental feature.** The per-policy bypass setting is experimental, and will be replaced with a reference to the policy's `critical` setting in Fleet 4.83.0. To ensure a seamless upgrade, please avoid enabling bypass for policies marked critical.

By default, all conditional access policies allow bypassing. You can control which policies allow bypass individually in **Manage automations** > **Conditional access**. Each policy with conditional access enabled has an additional checkbox to allow or disallow bypass.

If a host is failing multiple conditional access policies, the bypass option is only available if **every** failing policy allows bypass. If any one of the failing policies does not allow bypass, the end user will not see the option to bypass and must resolve the issue to regain access.

<meta name="articleTitle" value="Conditional access">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-06-20">
<meta name="description" value="Learn how to enforce conditional access with Fleet using Okta or Microsoft Entra.">
