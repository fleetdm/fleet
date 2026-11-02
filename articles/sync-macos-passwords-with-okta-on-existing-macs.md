# Sync macOS passwords with Okta on Macs that are already set up

When a Mac's local password drifts from the user's Okta password, the user ends up calling the help desk. Fleet's [Apple account provisioning](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet) keeps the local password in sync with your identity provider (IdP). That guide covers new Macs that create their account during Setup Assistant. This guide covers Macs that are already enrolled and in use: you install Fleet Desktop, deploy one configuration profile, and each user links their existing local account to Okta. No wipe or reinstall is needed.

## Password sync vs. Platform SSO

These are separate features, and you can adopt one without the other.

- **Password sync with Fleet's extension (this guide):** Fleet Desktop includes a Platform SSO extension that checks the user's password with Okta and keeps the local account password matched to it. It works on Macs that are already set up. It doesn't sign users in to apps and websites, and it doesn't apply MFA at the login window.
- **Okta's Platform SSO (Desktop Password Sync with Okta Verify):** Okta's own extension adds single sign-on to Okta-protected apps. It requires the Okta Device Access SKU, Okta Verify, and Device Access SCEP certificates. Existing accounts can register too. Users get a **Registration Required** notification after the profiles arrive. Only account creation during Setup Assistant (Simplified Platform SSO) requires a fresh automatic enrollment (ADE). See [Deploying Platform SSO with Okta Device Access](https://fleetdm.com/guides/deploying-okta-platform-sso-with-fleet).

If you need SSO for apps and websites, use Okta's integration. If you only need the local password to match Okta, follow this guide.

> **Warning:** OAuth Resource Owner Password Grant (ROPG) sends the user's actual password to your IdP so the IdP can verify it. The Mac encrypts the password for your Fleet server. Fleet decrypts it and sends it to Okta's token endpoint. Okta doesn't apply MFA to this flow, so the Okta app you create must allow password-only sign-in. Review this with your security team before you roll it out.

## Prerequisites

- Fleet Premium, on Fleet 4.90.0 or later. Use Fleet 4.94.0 or later so a misconfigured Okta app shows a clear error instead of "Incorrect username or password" (see [Troubleshoot](#troubleshoot)).
- Macs running macOS 26 or later.
- Fleet Desktop 1.5.1 or later, available as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps). Earlier versions can't register an account that already exists.
- Single-user Macs. Fleet's extension supports one user per Mac.
- Okta admin access to create an app integration and assign an authentication policy.
- A fleet or a [label](https://fleetdm.com/guides/managing-labels-in-fleet) that holds only your test Macs.

The Macs don't need to have enrolled through ADE.

## Step 1: Create a password sync app in Okta

Follow [step 1 of the account provisioning guide](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet#step-1-create-an-oauth-ropg-app-in-okta) to create a **Native Application** with the **Resource Owner Password** grant and client secret authentication.

The app's authentication policy must allow **Password** as a single factor, or ROPG requests fail. This doesn't mean turning off MFA across your org:

1. Create a new authentication policy just for this app. Don't edit a policy that other apps share.
2. Add a rule that allows password-only sign-in for the users assigned to this app.
3. Assign the new policy to the password sync app on its **Sign On** tab.

You can skip step 2 of that guide (short name and full name mapping). `TokenToUserMapping` sets the account name and full name when macOS creates a new account. On these Macs the account already exists.

## Step 2: Connect Fleet to Okta

Follow [step 3 of the account provisioning guide](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet#step-3-connect-fleet-to-your-idp) to enter the token URL, client ID, and client secret in **Settings > Integrations > Account provisioning**. To use GitOps, add them under `controls.apple_account_provisioning` in `default.yml`.

> **Note:** These settings apply to "All fleets" and can't be set per fleet. Saving them doesn't change anything on your Macs. Nothing happens on a host until it has Fleet Desktop and the configuration profile from step 4. That's how you limit the rollout to test Macs.

## Step 3: Install Fleet Desktop on your test Macs

1. In Fleet, go to **Software** for the fleet that holds your test Macs, select **Add software**, open the **Fleet-maintained** tab, and add **Fleet Desktop**.
2. Install it on your test Macs. You can install it from each host's details page, offer it in self-service, or install it automatically with a policy.

With GitOps, add it to the fleet's YAML file. Scope it with a label if your test Macs share a fleet with other hosts:

```yaml
software:
  fleet_maintained_apps:
    - slug: fleet-desktop/darwin
      labels_include_any:
        - Password sync pilot
```

> **Note:** This is the standalone Fleet Desktop app (`com.fleetdm.fleet-desktop`), not the Fleet Desktop menu bar item that ships with fleetd. The two can be installed side by side. The Platform SSO extension is only in the standalone app.

## Step 4: Deploy the configuration profile

1. Start from the [example profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/fleet-sso-extension-example.mobileconfig) and replace every `fleet.example.com` with your Fleet server's domain. [Step 5 of the account provisioning guide](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet#step-5-create-and-upload-the-configuration-profile) explains each key.
2. If you skipped the custom claim in Okta, remove the `TokenToUserMapping` dictionary. Leave `EnableRegistrationDuringSetup` and `UseSharedDeviceKeys` set to `true`. Fleet rejects the profile if `UseSharedDeviceKeys` isn't `true`.
3. Upload the profile under **Controls > OS settings > Custom settings** for the fleet that holds your test Macs. Scope it with a label if that fleet also has other Macs.

With GitOps:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/fleet-sso-extension.mobileconfig
        labels_include_any:
          - Password sync pilot
```

> **Note:** The account provisioning guide says not to scope this profile with labels. That advice is for new Macs, where a labeled profile may arrive too late for Setup Assistant. Macs that are already set up don't have that timing problem.

> **Warning:** The profile also turns on account creation for any Mac that enrolls through ADE into the same fleet. If you use a dedicated test fleet, new Macs won't land in it. For new Macs, keep **Controls > Setup experience > Users** set to **Admin** or **Standard**. If **None** is selected, macOS doesn't create an account, the user's credentials don't sync, and the user can't sign in. Macs that are already set up aren't affected by this setting.

## What end users see

1. After the profile arrives, macOS shows a **Registration required** notification.
2. The user clicks it, selects **Continue**, and enters their current local Mac password.
3. A **Sign in to your organization** window appears. The user enters their Okta username and password.
4. Fleet checks the credentials with Okta and registration completes. The local password now matches Okta.

If the user selects **Cancel**, nothing is saved and macOS asks again later.

After registration, password sync works the same way it does on new Macs. When the user changes their Okta password, they should lock and unlock their Mac with the new password to sync right away. Learn more in [the account provisioning guide](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet#end-user-experience).

## Verify

On a test Mac:

1. Open **System Settings > Users & Groups** and select the user. The Platform SSO section should show **Registered**.
2. In Terminal, run `app-sso platform -s`. The user configuration's `state` should be `POUserStateNormal`. `POUserStateNeedsRegistration` means registration hasn't finished.
3. Change the user's password in Okta, then lock and unlock the Mac with the new password. macOS should show a notification that the password was synced.
4. Confirm that the user's keychain still unlocks and that FileVault unlock at startup accepts the new password.

## Roll out to the rest of your Macs

When your test Macs pass, widen the rollout in stages. Add more hosts to your label, or move the label targeting from Fleet Desktop and the profile to the fleets you want.

## Troubleshoot

**Registration fails right away and macOS says it will retry in a few minutes.**

The Mac has a Fleet Desktop version earlier than 1.5.1. Those versions can't register an existing account. Update Fleet Desktop to 1.5.1 or later and let macOS retry.

**Users see "Incorrect username or password" with the correct password.**

On Fleet versions before 4.94.0, Fleet shows this message for any error from Okta, including a wrong client secret. Check the Okta app from step 1: the **Resource Owner Password** grant is on, client authentication is **Client secret**, the user is assigned to the app, and the app's policy allows password-only sign-in. Then check the token URL, client ID, and client secret in Fleet.

**Users see "Fleet couldn't verify your credentials (HTTP 400). Try again later."**

Okta rejected Fleet's request for a reason other than a bad password. This usually means the token URL, client ID, or client secret in Fleet is wrong, or the Okta app isn't set up for ROPG. Fleet 4.94.0 and later log Okta's error code in the Fleet server logs.

**Users see "Couldn't connect to Fleet."**

The Mac can't reach your Fleet server. Check the network connection, and check that the domain in the profile matches your Fleet server.

**The Registration required notification never appears.**

Confirm Fleet Desktop 1.5.1 or later is installed in `/Applications`, the configuration profile shows as verified on the host, and the Mac runs macOS 26 or later.

**The old password still works after a password change.**

This is expected for up to 4 hours. macOS checks the password with Okta when its token is missing, expired, or more than 4 hours old. Have users lock and unlock with the new password to sync right away.

## Further reading

- [Deploying Apple account provisioning with Fleet](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet)
- [Deploying Platform SSO with Okta Device Access](https://fleetdm.com/guides/deploying-okta-platform-sso-with-fleet)
- [Apple's Extensible Single Sign-On profile reference](https://developer.apple.com/documentation/devicemanagement/extensiblesinglesignon)

<meta name="articleTitle" value="Sync macOS passwords with Okta on Macs that are already set up">
<meta name="authorFullName" value="AUTHOR_FULL_NAME">
<meta name="authorGitHubUsername" value="AUTHOR_GITHUB_USERNAME">
<meta name="publishedOn" value="YYYY-MM-DD">
<meta name="category" value="guides">
<meta name="description" value="Keep local macOS passwords in sync with Okta on Macs that are already set up, without a wipe, using Fleet Desktop's Platform SSO extension.">
