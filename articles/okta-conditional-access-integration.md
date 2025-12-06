# Conditional access: Okta

_Available in Fleet Premium._

With Fleet, you can integrate with Okta to enforce conditional access on macOS hosts.

When a host fails a policy in Fleet, IT and Security teams can block access to third-party apps until the issue is resolved.

1. [Deploy user scope profile](#step-1-deploy-user-scope-profile.)
2. [Download certificate for Okta](#step-2-download-certificate-for-okta)
3. [Create IDP in Okta](#step-2-create-idp-in-okta)
4. [Configure Okta settings in Fleet](#step-3-configure-okta-settings-in-fleet)
5. [Configure conditional access policies](#step-4-configure-conditional-access-policies)

## Step 1: Deploy user scope profile

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. In the modal, find the read-only **User scope profile**.
3. Copy the profile to a new `.mobileconfig` file and save.
4. Follow the instructions in the [Custom OS settings](https://fleetdm.com/guides/custom-os-settings) guide to deploy the profile to the hosts where you want conditional access to apply.

## Step 2: Download certificate for Okta

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. In the modal, go to **Identity provider (IdP) signature certificate**. Click **Download certificate**.

## Step 3: Create IdP in Okta

1. In the Okta Admin Console, go to **Security** > **Identity Providers**.
2. Click **Add Identity Provider**.
3. Select **SAML 2.0 IdP**.
4. Set **Name** to "Fleet".
5. Set **IdP Usage** to **Factor only**
6. Set **IdP Issuer URI**, **IdP Single Sign-On URL**, and **Destination** to [TODO]
7. For **IdP Signature Certificate**, upload the IdP signature certificate downloaded from Fleet.
8. After saving, you'll see the Fleet IdP listed in **Security** > **Identity Providers**.


## Step 4: Configure Okta settings in Fleet

Once you've created the identity provider in Okta, you'll need to copy its values into your Fleet settings.

1. In Fleet, go to **Settings** > **Integrations** > **Conditional access** > **Okta** and click **Connect**.
2. Copy the **IdP ID** from Okta to the **IdP ID** field.
3. Copy the **Assertion Consumer Service URL** from Okta to the **Assertion consumer service URL** field.
3. Copy the **Audience URI** from Okta to the **Audience URI** field.

## Step 5: Configure conditional access policies

> TODO

## Disabling Okta conditional access

You can delete Conditional access configuration on Fleet's side from **Settings** > **Integrations** > **Conditional access** > **Okta** and clicking the delete button.

To fully disable conditional access, you will also need to disable it on the Okta side. 

> TODO steps to disable



<meta name="articleTitle" value="Conditional access: Okta">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-12-04">
<meta name="description" value="Learn how to enforce conditional access with Fleet and Okta.">