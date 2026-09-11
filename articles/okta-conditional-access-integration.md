# Conditional access: Okta

With Fleet, you can integrate with Okta to enforce conditional access on macOS hosts.

When a host fails a policy in Fleet, IT and Security teams can block access to third-party apps until the issue is resolved.

## Prerequisites

> This section is only for self-hosted instances of Fleet. If you are on a cloud-managed instance, skip down to [Step 1](#step-1-download-idp-signature-certificate-from-fleet).

Okta's [Adaptive MFA](https://www.okta.com/learn/adaptive-mfa/) SKU is required to use this feature.

Conditional access with Okta requires two reverse proxies in front of your Fleet server. The Fleet server itself must not be reachable from the public internet. All traffic reaches it through one of these proxies:

- **Main TLS reverse proxy** at your Fleet server URL (e.g., `https://fleet.example.com`). It forwards all requests to your internal Fleet address (e.g., `http://fleet.internal:8080`), except requests to `/api/fleet/conditional_access/idp/sso`. It redirects those to the mTLS reverse proxy.
- **mTLS reverse proxy** on an `okta.` subdomain of your Fleet server URL (e.g., `https://okta.fleet.example.com`). It requires a client certificate (mTLS) and validates it against Fleet's SCEP CA public certificate, which you download from Fleet in step 1 below. It then forwards requests to the same internal Fleet address and adds the `X-Client-Cert-Serial` header.

> **Warning:** Fleet trusts the `X-Client-Cert-Serial` header on `/api/fleet/conditional_access/idp/sso`. Only the mTLS reverse proxy verifies certificates, so the main reverse proxy must never forward that path to Fleet. Without the redirect, anyone who can reach your Fleet server URL can set the header themselves and impersonate an enrolled host.

If you're a managed-cloud customer, please reach out to Fleet to set up the mTLS infrastructure for you.

If you would like to set up a testing environment, see the [Okta conditional access testing guide](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/okta-conditional-access-testing.md).

If you use [fleet-terraform](https://github.com/fleetdm/fleet-terraform) modules for AWS hosting, the [okta-conditional-access addon](https://github.com/fleetdm/fleet-terraform/tree/main/addons/okta-conditional-access) sets up this model for you. It creates the mTLS load balancer, and its `redirect_rules` output adds the redirect to your main load balancer.

Otherwise, you'll need to:

1. **Get the mTLS CA certificate**: Download the CA certificate from Fleet's SCEP endpoint at `/api/fleet/conditional_access/scep?operation=GetCACert`. This is the certificate that signs the client certificates deployed to your hosts.

> **Note:** The certificate is provided in DER format. If your mTLS reverse proxy requires PEM format, convert it with the following command. Replace `fleet-scep-ca.cer` with the filename you used when downloading the certificate.

`openssl x509 -inform der -in fleet-scep-ca.cer -out fleet-scep-ca.pem`

2. **Create a DNS record**: Point the `okta.` subdomain of your Fleet server URL (e.g., `okta.fleet.example.com`) to your mTLS reverse proxy. Fleet builds the mTLS URL by adding `okta.` in front of your Fleet server URL's hostname, so the subdomain must match exactly.

3. **Configure the mTLS reverse proxy**: Set up a reverse proxy that:
   - Requires client certificate authentication using your CA certificate
   - Forwards requests to your internal Fleet address
   - Adds the `X-Client-Cert-Serial` header with the client certificate's serial number

4. **Redirect the SSO endpoint on the main reverse proxy**: Configure the reverse proxy at your Fleet server URL to redirect all requests to `/api/fleet/conditional_access/idp/sso` to the same path on the mTLS reverse proxy (e.g., `https://okta.fleet.example.com/api/fleet/conditional_access/idp/sso`). Redirect every HTTP method. Fleet accepts the SAML request as either `GET` or `POST`.

#### Example Caddy configuration

Here's an example `Caddyfile` for both reverse proxies:

```caddyfile
# Main TLS reverse proxy
fleet.example.com {
  # Never forward the SSO endpoint to Fleet from here
  redir /api/fleet/conditional_access/idp/sso https://okta.fleet.example.com{uri} 307

  reverse_proxy fleet.internal:8080
}

# mTLS reverse proxy
okta.fleet.example.com {
  tls {
    client_auth {
      mode require_and_verify
      trusted_ca_cert_file /etc/caddy/fleet-scep-ca.pem
    }
  }

  reverse_proxy fleet.internal:8080 {
    # Forward client certificate serial number to Fleet
    header_up X-Client-Cert-Serial {http.request.tls.client.serial}
  }
}
```

> **Note:** The example uses a `307` redirect so a `POST` request keeps its method and body. A `301` or `302` also works if that's all your proxy supports. The redirect only needs to keep the request away from Fleet. In the normal sign-in flow, Okta sends the browser to the mTLS reverse proxy directly.

> **Important:** Caddy sends the certificate serial number in decimal format, while AWS ALB sends it in hexadecimal format. When using Caddy, you must configure Fleet to parse the serial number in decimal format by setting [`conditional_access.cert_serial_format`](https://fleetdm.com/docs/configuration/fleet-server-configuration#conditional-access) to `decimal`.

Replace:
- `fleet.example.com` with your Fleet server URL's hostname
- `okta.fleet.example.com` with your mTLS subdomain
- `fleet.internal:8080` with your internal Fleet address
- `/etc/caddy/fleet-scep-ca.pem` with the path to your SCEP CA certificate

#### Verify the proxies

From any computer, send a request to the SSO endpoint through your Fleet server URL:

```bash
curl -si -X POST https://fleet.example.com/api/fleet/conditional_access/idp/sso -H 'X-Client-Cert-Serial: 1'
```

The response must be a redirect to your `okta.` subdomain. Any other response means the main reverse proxy is forwarding the path to Fleet.

Then send a request to the mTLS reverse proxy without a client certificate:

```bash
curl -si https://okta.fleet.example.com/api/fleet/conditional_access/idp/sso
```

The connection must fail during the TLS handshake. If you get an HTTP response, the proxy isn't requiring a client certificate.

## Step 1: Download IdP signature certificate from Fleet

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. In the modal, go to **Identity provider (IdP) signature certificate**. Click **Download certificate**.

## Step 2: Deploy user scope profile

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. In the modal, find the read-only **User scope profile**.
3. Copy the profile to a new `.mobileconfig` file and save.
4. Follow the instructions in the [custom OS settings](https://fleetdm.com/guides/custom-os-settings) guide to deploy the profile to the hosts where you want conditional access to apply.

Deploying this profile will deploy a SCEP certificate to your hosts. These certificates are valid for 1 year and 33 days and Fleet will automatically renew them. [Learn more](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#renewal).

> **Upgrading from Fleet 4.85 or earlier?** Your existing Conditional Access deployment continues to work, but auto-renewal activates only on profiles redeployed in 4.86 or later. To opt in, re-download the User scope profile above and re-deploy via custom OS settings.

> If using GitOps, use the challenge in a [secret variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles), instead of hardcoding into the profile.

## Step 3: Create IdP in Okta

1. In the Okta Admin Console, go to **Security** > **Identity Providers**.
2. Click **Add Identity Provider**.
3. Select **SAML 2.0 IdP**.
4. Set **Name** to "Fleet".
5. Set **IdP Usage** to **Factor only**
6. Set the following values (replace `fleet.example.com` with your Fleet server domain):
   - **IdP Issuer URI**: `https://fleet.example.com/api/fleet/conditional_access/idp/metadata`
   - **IdP Single Sign-On URL**: `https://okta.fleet.example.com/api/fleet/conditional_access/idp/sso` (note the `okta.` prefix)
   - **Destination**: `https://okta.fleet.example.com/api/fleet/conditional_access/idp/sso` (note the `okta.` prefix)
7. For **IdP Signature Certificate**, upload the IdP signature certificate downloaded from Fleet.
8. Click **Finish**.

## Step 4: Configure Okta settings in Fleet

Once you've created the identity provider in Okta, click on the Fleet identity provider to view its settings. You'll need to copy these values into Fleet.

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. Copy the **IdP ID** from Okta to the **IdP ID** field.
3. Copy the **Assertion Consumer Service URL** from Okta to the **Assertion consumer service URL** field.
4. Copy the **Audience URI** from Okta to the **Audience URI** field.
5. In the Okta Admin Console, go to **Security** > **Identity Providers**, select **Actions** for the Fleet identity provider and choose **Download certificate**.
6. In Fleet, for **Okta certificate**, upload the certificate downloaded from Okta.

## Step 5: Add Fleet IdP authenticator in Okta

1. In the Okta Admin Console, go to **Security** > **Authenticators**.
2. Click **Add authenticator**.
3. Find **IdP Authenticator** and click **Add**.
4. In the **Identity Provider** dropdown, select **Fleet**.
5. For the logo, download the [Fleet logo](http://fleetdm.com/images/press-kit/fleet-logo-mark.svg) and upload it.
6. Click **Add**.

## Step 6: Add Fleet to an authentication policy

Create an authentication policy rule that requires Fleet verification for macOS hosts:

1. In the Okta Admin Console, go to **Security** > **Authentication policies**.
2. Select the policy you want to modify (or create a new one).
3. Click **Add rule**.
4. Set a **Rule name** (e.g., "Require Fleet for macOS").
5. Under **AND Device platform is**, select **One of the following platforms** and **macOS** to ensure this rule only applies to macOS hosts.
6. Under **AND User must authenticate with**, select **Authentication method chain** (recommended) and add the Fleet IdP authenticator created in Step 5 as one of the authentication methods.
7. Click **Save**.

> To apply this policy to specific apps, go to **Applications** > select an app > **Sign On** tab > **Authentication policy** and assign the policy.

## Step 7: Configure conditional access policies in Fleet

Once Okta is configured in settings, head to **Policies**. Select the fleet that you want to enable conditional access for.

1. Go to **Manage automations** > **Conditional access** and enable conditional access.
2. Select the policies you want to block login via Okta.
3. Save.

Once enabled, if a user tries to log in to an app that requires Fleet as a factor and their host is failing a selected policy, they will be blocked from logging in. To regain access, the user must fix the issue on their host and then click **Refetch** on the **My device** page to verify the policy is now passing.

## Disabling Okta conditional access

> **Warning:** You must disable conditional access on the Okta side first. If you only disable it on the Fleet side, users may be unable to log in to apps that still require Fleet as an authentication factor.

To disable conditional access on the Okta side:

1. In the Okta Admin Console, go to **Security** > **Authentication policies**.
2. Either delete the authentication policy rule that requires Fleet, or remove the policy from all apps by going to **Applications** > select an app > **Sign On** tab > **Authentication policy** and assigning a different policy.

Once disabled on the Okta side, you can delete the conditional access configuration on Fleet's side from **Settings** > **Integrations** > **Conditional access** > **Okta** and clicking the delete button.

## Bypassing conditional access

End users can temporarily bypass conditional access from their **My device** page if their host is not failing any critical policies. To trigger a bypass, click a non-critical failing policy labeled **Action required**, select **Resolve later**, and confirm in the following modal. The bypass allows the user to complete a single login even with failing policies and is consumed immediately upon successful login.

If a host is failing multiple conditional access policies, the bypass option is only available if **no** failing policy is marked critical. If any one of the failing policies is marked critical, the end user will not see the option to bypass and must resolve the issue to regain access. (You can update a policy's `critical` setting on the **Edit policy** page.)

This feature is enabled by default, but can be disabled by unchecking the **Bypass for non-critical policies** checkbox in **Settings** > **Integrations** > **Conditional access**. 




### Per-policy bypass

> **Experimental feature.** The per-policy bypass setting is experimental, and will be replaced with a reference to the policy's `critical` setting in Fleet 4.83.0. To ensure a seamless upgrade, please avoid enabling bypass for policies marked critical.

By default, all conditional access policies allow bypassing. You can control which policies allow bypass individually in **Manage automations** > **Conditional access**. Each policy with conditional access enabled has an additional checkbox to allow or disallow bypass.

If a host is failing multiple conditional access policies, the bypass option is only available if **every** failing policy allows bypass. If any one of the failing policies does not allow bypass, the end user will not see the option to bypass and must resolve the issue to regain access.

<meta name="articleTitle" value="Conditional access: Okta">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-12-04">
<meta name="description" value="Learn how to enforce conditional access with Fleet and Okta.">
