# Deploying Apple account provisioning with Fleet

Fleet's Apple account provisioning creates your end users' macOS local accounts during automatic enrollment (ADE) using their identity provider (IdP) credentials, and keeps the local account password in sync with the IdP afterwards. It uses Fleet's own Platform SSO extension, built into the Fleet Desktop app, which proxies authentication through your Fleet server to any IdP that supports OAuth Resource Owner Password Grant (ROPG). This guide covers setup with Okta, but any OAuth ROPG-compatible IdP works.

> This feature requires Fleet Premium.

If your IdP offers its own native Platform SSO integration, such as [Okta Device Access](https://fleetdm.com/guides/deploying-okta-platform-sso-with-fleet) or [Microsoft Entra](https://fleetdm.com/guides/deploying-entra-platform-sso-with-fleet), consider using that instead. Fleet's account provisioning is designed for cases where a native integration is unavailable or not licensed for your organization.

## What you get (and what you don't)

With Apple account provisioning enabled:

- End users authenticate with their IdP username and password during Setup Assistant, and macOS creates their local account with that password.
- The local account password stays in sync with the IdP. After a password change in the IdP, signing in with the new password at the login window, lock screen, or FileVault unlock updates the local password and keeps the keychain intact.
- The local account's short name and full name can be mapped from IdP attributes using `TokenToUserMapping`.

What you don't get (yet):

- Single sign-on to SaaS apps and websites. Fleet's extension currently handles account creation and password sync only.

> OAuth ROPG is required because password sync needs the IdP to verify the user's actual password. Other desktop password sync products use the same class of flow for provisioning and syncing. Because ROPG sends the username and password directly to the token endpoint, it bypasses MFA, and some organizations' security policies may not allow it.

## Prerequisites

- Fleet Premium
- macOS hosts running macOS 26 or later, enrolling via ADE through Apple Business
- An IdP that supports OAuth ROPG (this guide uses Okta)
- Fleet's [setup experience](https://fleetdm.com/guides/setup-experience) configured for the target fleet
- The Fleet Desktop app, which contains Fleet's Platform SSO extension, available as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps)

> Account provisioning can currently only be configured for "All fleets" and only supports single-user macOS hosts.

## Step 1: Create an OAuth ROPG app in Okta

Okta only supports the Resource Owner Password grant on Native app integrations.

1. Sign in to the Okta Admin Console and go to **Applications > Applications > Create App Integration**.

2. Select **OIDC - OpenID Connect** as the sign-in method and **Native Application** as the application type, then click **Next**.

3. Give the app a name, like "Fleet account provisioning."

4. Under **Grant type**, check **Resource Owner Password**.

5. Under **Assignments**, assign the app to the users or groups who will enroll Macs, then click **Save**.

6. On the app's **General** tab, click **Edit** in the **Client Credentials** section, set **Client authentication** to **Client secret**, and click **Save**.

7. Copy the **Client ID** and **Client secret**. You'll add these to Fleet in step 3.

Next, confirm the app can complete a password-only sign-in:

1. Go to **Applications > Applications**, open your app, and select the **Sign On** tab.

2. Make sure the authentication policy assigned to the app allows sign-in with **Password** as a single factor. If the policy requires MFA, ROPG requests will fail.

Finally, find your token URL. Fleet recommends the `default` authorization server because it supports the custom claims used for name mapping in step 2:

1. Go to **Security > API > Authorization Servers** and open **default**.

2. Your token URL is the **Issuer** URI plus `/v1/token`, for example `https://example.okta.com/oauth2/default/v1/token`.

3. On the **Access Policies** tab, make sure a policy rule assigned to your app allows the **Resource Owner Password** grant type.

> You can also use Okta's org authorization server (`https://example.okta.com/oauth2/v1/token`), but it doesn't support custom claims, so short name mapping with `TokenToUserMapping` won't be available.

## Step 2: Map short name and full name (optional)

Without any mapping, macOS uses the end user's IdP username as the local account's account name (short name), so a user signing in as `fleetie@example.com` gets `fleetie@example.com` as their account name. To get a friendlier account name like `fleetie`, add a custom claim in Okta and map it in your configuration profile with `TokenToUserMapping`.

Fleet forwards the standard `email`, `name`, and `preferred_username` claims from your IdP's ID token to the Mac, plus any custom claim whose name starts with `account`. Name your custom claims accordingly, for example `accountName` or `accountFullName`.

To add the short name claim in Okta:

1. Go to **Security > API > Authorization Servers** and open **default**.

2. On the **Claims** tab, click **Add Claim** and enter:
   - **Name:** `accountName`
   - **Include in token type:** ID Token, Always
   - **Value type:** Expression
   - **Value:** `String.substringBefore(user.login, "@")`
   - **Include in:** Any scope

3. Click **Create**.

For the full name, the standard `name` claim works out of the box when the `profile` scope is granted (Fleet requests `openid profile email` by default). You can also add a custom `accountFullName` claim the same way if you want a different value.

You'll reference these claim names in the configuration profile's `TokenToUserMapping` dictionary in step 5.

## Step 3: Connect Fleet to your IdP

1. In Fleet, go to **Settings > Integrations > Account provisioning**.

2. Enter the **Token URL**, **Client ID**, and **Client secret** from step 1, then save.

Alternatively, configure it with [GitOps](https://fleetdm.com/docs/configuration/yaml-files#apple-account-provisioning) in `default.yml`:

```yaml
controls:
  apple_account_provisioning:
    oauth_idp_token_url: https://example.okta.com/oauth2/default/v1/token
    oauth_idp_client_id: 0oa12345abcdeFGHI678
    oauth_idp_client_secret: # TODO: client secret (masked and non-exportable from the API)
```

## Step 4: Add Fleet's Platform SSO app to setup experience

The Fleet Desktop app that contains the Platform SSO extension isn't installed by default. Add it as setup experience software so it's installed during Setup Assistant, before the user reaches the sign-in screen:

1. In Fleet, head to the **Software** page for the target fleet, select **Add software**, open the **Fleet-maintained** tab, and add **Fleet Desktop**.

2. Go to **Controls > Setup experience > Install software** and select the Fleet Desktop app so it installs during setup experience.

## Step 5: Create and upload the configuration profile

The extension is activated by a single configuration profile containing 2 payloads: an **Extensible Single Sign-On** payload and an **Associated Domains** payload. Start from the [example profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/fleet-sso-extension-example.mobileconfig) and replace every occurrence of `fleet.example.com` with your Fleet server's domain.

In the Extensible Single Sign-On payload:

- **ExtensionIdentifier:** `com.fleetdm.fleet-desktop.pssoextension` and **TeamIdentifier:** `8VBZ3948LU`. Use these values exactly.
- **ExtensionData > BaseURL** and **URLs:** your Fleet server URL.
- **RegistrationToken:** `$FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN`. Fleet replaces this variable with a unique per-device registration token when the profile is delivered to each host.
- **EnableRegistrationDuringSetup** and **UseSharedDeviceKeys:** both `true`, so the user is registered during Setup Assistant.
- **TokenToUserMapping:** maps macOS account fields to claim names in the token the Mac receives. The example profile maps the short name to the `accountName` claim from step 2 and the full name to the standard `name` claim:

```xml
<key>TokenToUserMapping</key>
<dict>
    <key>AccountName</key>
    <string>accountName</string>
    <key>FullName</key>
    <string>name</string>
</dict>
```

If you skipped step 2, remove the `AccountName` key (or the whole `TokenToUserMapping` dictionary) and macOS will use the IdP username as the account name.

> `$FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN` is only allowed in the `RegistrationToken` key of a Fleet SSO extension payload. Fleet redacts the token when you view the delivered `InstallProfile` command, so it's never exposed in the UI or API.

In the Associated Domains payload, both app identifiers (`8VBZ3948LU.com.fleetdm.fleet-desktop` and `8VBZ3948LU.com.fleetdm.fleet-desktop.pssoextension`) must list `authsrv:` plus your Fleet server's domain.

Upload the profile to the target fleet under **Controls > OS settings > Custom settings**.

> Don't scope this profile with labels. Labeled profiles may not be delivered in time for Setup Assistant, and if that happens the user won't be prompted to sign in.

## End user experience

1. The user powers on the Mac and it enrolls through automatic enrollment. End user authentication is optional; if it's enabled, the user signs in with their IdP first.

2. Setup experience installs the Fleet Desktop app and delivers the configuration profile.

3. During Setup Assistant, after Fleet's setup experience window closes, the user is prompted to sign in with their IdP username and password.

4. macOS creates the local account. The password is the user's IdP password, and the account name and full name come from `TokenToUserMapping` if configured. The account creation screen will be shown, however the values are locked and the user cannot edit them at this point.

After setup, password sync works like this:

- When the user changes their password in the IdP and then signs in or unlocks with the new password, the local password syncs and a notification confirms it. The keychain stays intact.
- If the user keeps using the old password after a change, it continues to work until the Mac next checks in with the IdP, up to 4 hours later. After that, macOS prompts the user to sync at the next desktop login.
- FileVault unlock works with the synced password.
- If the Fleet server is unreachable, users are never locked out. The existing local password keeps working until connectivity returns.

> Consider directing users to lock and then unlock their Mac using their new password after completing a password change via your IdP to immediately synchronize their Mac password, rather than relying on it happening later.

## Troubleshoot

**The user isn't prompted to sign in during Setup Assistant.**

Confirm the Fleet Desktop app is set as setup experience software (step 4), the profile is uploaded to the same fleet without label scoping (step 5), and the host is running macOS 26 or later.

**Sign-in fails with valid credentials.**

Check the Okta app configuration from step 1: the **Resource Owner Password** grant must be enabled, client authentication must be set to **Client secret**, the user must be assigned to the app, and the app's authentication policy must allow password-only sign-in. Also confirm the token URL in Fleet points to the right authorization server and that its access policy allows the password grant.

**The account name is the full email address.**

The `accountName` claim isn't reaching the Mac. Confirm the custom claim exists on the same authorization server as your token URL (custom claims require a custom authorization server, not the org authorization server), that it's included in the ID token, and that its name starts with `account`.

**The user is prompted for their previous password.**

Occasionally when the user has logged out and logs back in, rather than locking and unlocking their Mac, the user will be prompted for their previous password at the desktop. This is expected and the user should enter their previous password to complete the process. If they cannot complete this process, the FileVault password may not sync with their new password.

## Further reading

- [Setup experience](https://fleetdm.com/guides/setup-experience)
- [Deploying Platform SSO with Okta Device Access](https://fleetdm.com/guides/deploying-okta-platform-sso-with-fleet)
- [Apple's Extensible Single Sign-On profile reference](https://developer.apple.com/documentation/devicemanagement/extensiblesinglesignon)
- [Okta's custom authorization server documentation](https://developer.okta.com/docs/concepts/auth-servers/)

<meta name="articleTitle" value="Deploying Apple account provisioning with Fleet">
<meta name="authorFullName" value="Jordan Montgomery">
<meta name="authorGitHubUsername" value="JordanMontgomery">
<meta name="publishedOn" value="2026-07-10">
<meta name="category" value="guides">
<meta name="description" value="Create macOS local accounts from IdP credentials during ADE enrollment and sync passwords with Fleet's Platform SSO extension and any ROPG IdP.">
