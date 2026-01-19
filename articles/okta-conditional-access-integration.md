# Conditional access: Okta

With Fleet, you can integrate with Okta to enforce conditional access on macOS hosts.

When a host fails a policy in Fleet, IT and Security teams can block access to third-party apps until the issue is resolved.

## Prerequisites

Conditional access with Okta requires an mTLS reverse proxy on a separate subdomain (e.g., `okta.fleet.example.com`). All other Fleet traffic continues to use your existing Fleet server URL.

### Fleet-hosted servers

If your Fleet server is hosted by Fleet, contact your Fleet representative to set up the mTLS infrastructure for you.

### Self-hosted servers

If you use [fleet-terraform](https://github.com/fleetdm/fleet-terraform) modules for AWS hosting, see the [okta-conditional-access addon](https://github.com/fleetdm/fleet-terraform/tree/main/addons/okta-conditional-access) for streamlined mTLS proxy setup.

Otherwise, you'll need to:

1. **Get the mTLS CA certificate**: Download the CA certificate from Fleet's SCEP endpoint at `/api/fleet/conditional_access/scep?operation=GetCACert`. This is the certificate that signs the client certificates deployed to your hosts.

> Note: The certificate is provided in DER format. If your mTLS termination solution requires PEM format, you can convert it using the following command:

`openssl x509 -inform der -in fleet-scep-ca.cer -out fleet-scep-ca.pem`

Replace `fleet-scep-ca.crt` with the filename you used when downloading the certificate.

2. **Create a DNS record**: Set up a subdomain with an `okta` prefix pointing to your mTLS proxy server (e.g., `okta.fleet.example.com`).

3. **Configure an mTLS reverse proxy**: Set up a reverse proxy that:
   - Requires client certificate authentication using your CA certificate
   - Forwards the `X-Client-Cert-Serial` header to your Fleet backend

4. **Redirect the SSO endpoint**: Configure your main Fleet server to redirect `/api/fleet/conditional_access/idp/sso` to the mTLS proxy (e.g., `https://okta.fleet.example.com/api/fleet/conditional_access/idp/sso`). This ensures all authentication requests go through mTLS verification.

#### Example Caddy configuration

Here's an example `Caddyfile` for setting up the mTLS proxy:

```caddyfile
okta.fleet.example.com {
  # Enable TLS with mTLS (client certificate authentication)
  tls {
    client_auth {
      mode require_and_verify
      trusted_ca_cert_file /etc/caddy/fleet-scep-ca.crt
    }
  }

  # Reverse proxy to your Fleet server
  reverse_proxy https://fleet.example.com {
    # Forward client certificate serial number to Fleet
    header_up X-Client-Cert-Serial {http.request.tls.client.serial}
  }
}
```

Replace:
- `okta.fleet.example.com` with your mTLS subdomain
- `/etc/caddy/fleet-scep-ca.crt` with the path to your SCEP CA certificate
- `https://fleet.example.com` with your Fleet server URL


## Instructions

### Step 1: Deploy user scope profile

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. In the modal, find the read-only **User scope profile**.
3. Copy the profile to a new `.mobileconfig` file and save.
4. Follow the instructions in the [Custom OS settings](https://fleetdm.com/guides/custom-os-settings) guide to deploy the profile to the hosts where you want conditional access to apply.

### Step 2: Download IdP signature certificate from Fleet

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. In the modal, go to **Identity provider (IdP) signature certificate**. Click **Download certificate**.
3. Rename certificate extention from `.cer` to `.crt` if needed.

### Step 3: Create IdP in Okta

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
9. Back in **Security** > **Identity Providers**, select **Actions** for the Fleet identity provider and choose **Download certificate**.

### Step 4: Configure Okta settings in Fleet

Once you've created the identity provider in Okta, click on the Fleet identity provider to view its settings. You'll need to copy these values into Fleet.

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. Copy the **IdP ID** from Okta to the **IdP ID** field.
3. Copy the **Assertion Consumer Service URL** from Okta to the **Assertion consumer service URL** field.
4. Copy the **Audience URI** from Okta to the **Audience URI** field.
5. For **Okta certificate**, upload the certificate downloaded from Okta in Step 3.

### Step 5: Add Fleet IdP authenticator in Okta

1. In the Okta Admin Console, go to **Security** > **Authenticators**.
2. Click **Add authenticator**.
3. Find **IdP Authenticator** and click **Add**.
4. In the **Identity Provider** dropdown, select **Fleet**.
5. For the logo, download the [Fleet logo](https://raw.githubusercontent.com/fleetdm/fleet/main/orbit/cmd/desktop/fleet-logo.svg) and upload it.
6. Click **Add**.

### Step 6: Add Fleet to an authentication policy

Create an authentication policy rule that requires Fleet verification for macOS hosts:

1. In the Okta Admin Console, go to **Security** > **Authentication policies**.
2. Select the policy you want to modify (or create a new one).
3. Click **Add rule**.
4. Set a **Rule name** (e.g., "Require Fleet for macOS").
5. Under **AND Device platform is**, select **One of the following platforms** and **macOS** to ensure this rule only applies to macOS hosts.
6. Under **AND User must authenticate with**, select **Authentication method chain** (recommended) and add the Fleet IdP authenticator created in Step 5 as one of the authentication methods.
7. Click **Save**.

> To apply this policy to specific apps, go to **Applications** > select an app > **Sign On** tab > **Authentication policy** and assign the policy.

### Step 7: Configure conditional access policies in Fleet

Once Okta is configured in settings, head to **Policies**. Select the team that you want to enable conditional access for.

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


<meta name="articleTitle" value="Conditional access: Okta">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-12-04">
<meta name="description" value="Learn how to enforce conditional access with Fleet and Okta.">
