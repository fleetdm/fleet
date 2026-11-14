# Migrate from JumpCloud to Fleet and Okta

Fleet is an open-source device management platform for macOS, Windows, Linux, iOS, iPadOS, Android, and Chromebooks. You can do MDM, deploy software, patch, and verify device settings from one place, and manage it all as code. Fleet doesn't include a directory or single sign-on, so you pair it with an identity provider (IdP) like Okta, which Fleet uses for sign-in during setup, user provisioning, and device checks.

You can do this whole migration by asking Claude. This guide moves the two jobs JumpCloud does for you to two tools: directory, SSO, and MFA go to Okta, and device management goes to Fleet. You'll run both side by side, move devices in batches, and retire JumpCloud last. Each step starts with a prompt you can paste into Claude to get walked through it, followed by the details so you can check its work.

## Prerequisites

- A Fleet Premium instance with admin access. Setup experience, IdP authentication, and Fleet-maintained apps are Premium features.
- An Okta org with admin access. Some Okta features need specific SKUs, and the steps below call them out.
- Admin access to JumpCloud, Apple Business (AB), and, if you manage Android, Google.
- An inventory of what JumpCloud manages today: users, groups, SSO apps, policies, commands, software, and devices by OS.

> **Warning:** Don't turn anything off in JumpCloud until the device group and the users depending on it have moved. Keep your JumpCloud tenant until the last step so you can roll back.

<!-- TODO: JumpCloud-specific export options (users, groups, policies, commands, software, MFA enrollments) are not covered by Fleet docs. A reviewer who knows JumpCloud should confirm what can be exported and add links to JumpCloud's docs. -->

## Step 1: Stand up Okta next to JumpCloud

> Ask Claude: "I'm moving from JumpCloud to Okta. Walk me through exporting my users and groups from JumpCloud, importing them into Okta, and moving my SSO apps over one at a time with a pilot group. Ask me how many users, groups, and SSO apps I have first."

Create your users and groups in Okta, then add the apps your people sign in to. Move SSO apps one at a time and test each with a pilot group. Plan for users to set a new Okta password and enroll in MFA, since you can't count on carrying passwords over.

<!-- TODO: verify with JumpCloud and Okta docs whether any password or MFA enrollment migration is supported. Also verify the JumpCloud features that don't have a 1:1 Okta equivalent (LDAP, RADIUS, password manager) and link to Okta's equivalents. -->

## Step 2: Connect Fleet to Okta

> Ask Claude: "Walk me through connecting Fleet to Okta. I need SSO for Fleet admins, a separate SAML app for end user authentication during device setup, and SCIM so Fleet maps Okta users to hosts. Use Fleet's setup experience and foreign vitals guides."

Do these in Fleet before you move any devices:

1. Turn on [single sign-on (SSO)](https://fleetdm.com/docs/deploy/single-sign-on-sso) so your admins log in to Fleet with Okta.
2. Create a separate SAML app in Okta for [end user authentication](https://fleetdm.com/guides/setup-experience#end-user-authentication), then turn on **Require IdP authentication** under **Controls > Setup experience** for the fleet that receives new devices. This makes new hires sign in with Okta during setup.
3. Set up [SCIM](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts#okta) so Fleet knows which Okta user owns each host.

> **Note:** If you use Okta, give the end user authentication SAML app a sign-in policy that disallows FastPass. Otherwise enrollment can get stuck. Details are in the [setup experience guide](https://fleetdm.com/guides/setup-experience#end-user-authentication).

## Step 3: Turn on MDM for each platform

> Ask Claude: "Walk me through turning on Windows, Apple, and Android MDM in Fleet. Ask which platforms I manage, then give me the steps for each, using Fleet's Windows MDM setup, Apple MDM setup, and Android MDM setup guides."

Turn these on before you move any devices:

- **Windows:** [turn on Windows MDM](https://fleetdm.com/guides/windows-mdm-setup#turn-on-windows-mdm).
- **macOS, iOS, and iPadOS:** [connect Fleet to Apple Push Notification service (APNs) and Apple Business](https://fleetdm.com/guides/apple-mdm-setup).
- **Android:** [connect Fleet to Android Enterprise](https://fleetdm.com/guides/android-mdm-setup).

> **Note:** If your organization already has an Android Enterprise binding through JumpCloud, check how Google handles creating a new one before you connect Fleet.

<!-- TODO: confirm whether an existing JumpCloud Android Enterprise binding blocks or changes the Fleet signup, and whether the Android Enterprise enterprise can be reused or must be unbound. -->

## Step 4: Move a pilot group

> Ask Claude: "Here is a list of my JumpCloud policies, commands, and software. Help me translate each one into a Fleet configuration profile, script, or software package, and tell me which ones don't have a direct match. Ask me for the list first."

Pick 10 to 20 devices across every OS you manage, and run steps 5 to 7 on them first. Put them in their own fleet so you can test profiles, scripts, and software before everyone gets them.

Move JumpCloud's work into Fleet as you go. These don't map 1:1:

| JumpCloud | Fleet |
| --- | --- |
| Policies | Native OS configuration profiles (`.mobileconfig` or DDM for Apple, CSP XML for Windows, JSON for Android) |
| Commands | [Scripts](https://fleetdm.com/guides/scripts) (shell and Python on macOS and Linux, PowerShell on Windows) |
| Software management | [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps), custom packages, and App Store apps |
| Device and user groups | Fleets and labels, with group membership coming from Okta |
| Directory, SSO, MFA, LDAP, RADIUS | Okta |

<!-- TODO: confirm the JumpCloud-side terms in this table (policy templates, commands, device groups) and which JumpCloud features (for example, LDAP, RADIUS, and password management) have no Fleet equivalent and move to Okta or another tool. -->

## Step 5: Move Windows devices

> Ask Claude: "Walk me through moving Windows PCs from JumpCloud to Fleet. Start with the OEM path (Dell ReadyImage, Lenovo RTP, or my hardware provider's equivalent) for new PCs. Then cover migrating existing PCs and removing the JumpCloud agent. Use Fleet's Windows MDM setup guide and don't suggest Autopilot."

For new PCs, skip Autopilot and Intune. You don't need Windows Autopilot, Microsoft Entra ID P1, or Intune to manage Windows hosts with Fleet. Use an OEM service such as Dell ReadyImage, Lenovo RTP, or the equivalent from your hardware provider:

1. Forward the order email from Fleet to your hardware provider.
2. The provider installs Fleet's agent (fleetd) on your new PCs before they ship.
3. When the end user turns on the PC and it comes online, it enrolls in Fleet.

For PCs you already have:

1. Install fleetd on the hosts, using JumpCloud's command feature or your usual software deployment.
2. Remove the JumpCloud agent with a PowerShell script run from Fleet.
3. Under **Settings > Integrations > Mobile device management (MDM)**, select **Edit** next to **Windows MDM turned on**, and select **Automatically migrate hosts connected to another MDM solution**.

> **Note:** Leftover enrollment data or management agents from the old tool can cause enrollment errors and locked local accounts. If a host doesn't report MDM as **On**, run the [fix-windows-mdm-migration.ps1](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/fix-windows-mdm-migration.ps1) script, reboot, and select **Refetch** on the host.

You only need Entra ID P1 if you choose Entra automatic enrollment or Autopilot. Each of those needs a P1 license for every end user.

<!-- TODO: confirm with Mike whether Fleet's offer to cover the one-time OEM cost applies to JumpCloud switchers. The public Windows MDM setup guide words it for customers moving from Intune with more than 500 devices. Do not restate it here until confirmed. -->
<!-- TODO: verify that Fleet's automatic Windows migration applies to hosts enrolled in JumpCloud, and that Fleet can enroll PCs where JumpCloud manages local accounts. Add the JumpCloud agent uninstall command after checking JumpCloud's docs. -->
<!-- TODO: add a note on Okta sign-in to Windows (Okta's local account provisioning for Windows is described as coming soon in the Provision Windows devices with Fleet, Okta, and a partner guide). -->

## Step 6: Move Macs, iPhones, and iPads

> Ask Claude: "Walk me through moving Macs, iPhones, and iPads from JumpCloud MDM to Fleet using Apple Business and Apple's native migration on version 26. Include batching, FileVault keys, and what end users will see. Use Fleet's MDM migration guide."

New Macs enroll automatically through Apple Business. For devices you already have:

1. Install fleetd on the Mac.
2. In Apple Business, unassign the devices from JumpCloud and assign them to Fleet.
3. Choose a [migration workflow](https://fleetdm.com/guides/mdm-migration#step-3-choose-migration-workflow-and-migrate-hosts). On macOS, iOS, and iPadOS 26, Apple's native Device Management Migration moves devices from one MDM to another. Migrate in batches of around 100, with a one to two week deadline for each.
4. Export FileVault recovery keys before you start. Fleet escrows new keys after the user logs out or restarts.

> **Note:** Fleet doesn't run the full setup experience during a migration. The managed local account and bootstrap package aren't applied, and only Fleet's agent is installed.

If you use Okta Platform SSO, see [Deploying Platform SSO with Okta and Fleet](https://fleetdm.com/guides/deploying-okta-platform-sso-with-fleet). It needs the Okta Device Access SKU. For conditional access, the [Okta conditional access guide](https://fleetdm.com/guides/okta-conditional-access-integration) documents macOS hosts and needs Okta's Adaptive MFA SKU.

<!-- TODO: confirm how to release devices from JumpCloud MDM (the default workflow needs the old MDM to unenroll each host) and whether JumpCloud escrows FileVault keys that can be exported. Confirm the iOS and iPadOS steps for devices enrolled in JumpCloud, and whether Okta conditional access supports Windows in the release you ship against. -->

## Step 7: Move Android devices

> Ask Claude: "Walk me through moving Android phones from JumpCloud to Fleet. Cover BYOD work profiles and company-owned devices. Use Fleet's Android MDM setup and BYOD migration guides."

Android Enterprise is already on from step 3. For work profile (BYOD) devices:

1. Remove the existing work profile from JumpCloud.
2. Send end users your Fleet enrollment link or a QR code.
3. Have them select **Enroll** and follow the prompts. If you turned on IdP authentication, they sign in with Okta first.

Company-owned devices bought from an authorized reseller can use [Android zero-touch enrollment](https://fleetdm.com/guides/android-zero-touch-enrollment), a Premium feature. Follow the [Android BYOD migration guide](https://fleetdm.com/guides/android-byod-mdm-migration) for the full steps.

<!-- TODO: confirm how to move company-owned (fully managed) Android devices that are already enrolled in JumpCloud. -->

## Step 8: Cut over and retire JumpCloud

> Ask Claude: "Make a cutover and rollback plan for moving from JumpCloud to Fleet and Okta. I have devices on [Windows, macOS, iOS, Android] and [number] SSO apps. Order the batches, list what to verify after each, and tell me how to back out."

Cut over in this order, one batch at a time:

1. Move SSO apps to Okta and confirm the pilot group can sign in.
2. Migrate devices in batches: Windows, macOS, iOS and iPadOS, then Android.
3. When a batch is stable, remove the JumpCloud agent and profiles from it.
4. Turn off JumpCloud after the last device and app have moved.

To roll back, keep the JumpCloud tenant and agent in place for the devices you haven't removed it from. For a device that already moved, point the app or device back at JumpCloud and re-enroll it.

<!-- TODO: confirm the JumpCloud re-enrollment steps for rollback, and have a JumpCloud-knowledgeable reviewer check the cutover order for directory dependencies such as LDAP, RADIUS, and local account management. -->

## Verify

1. In Fleet, head to **Hosts** and confirm each migrated device reports MDM as **On**.
2. On the **Dashboard**, check the **Mobile device management (MDM)** section to see which Apple hosts have migrated.
3. In Okta, confirm users can sign in to their apps, and that Fleet's SSO and end user authentication work.

## Further reading

- [macOS MDM migration](https://fleetdm.com/guides/mdm-migration)
- [Windows MDM setup](https://fleetdm.com/guides/windows-mdm-setup)
- [Android MDM setup](https://fleetdm.com/guides/android-mdm-setup)
- [Setup experience](https://fleetdm.com/guides/setup-experience)
- [Fleet vs. JumpCloud vs. Workspace ONE](https://fleetdm.com/articles/jumpcloud-vs-workspace-one-vs-fleet)

<meta name="articleTitle" value="Migrate from JumpCloud to Fleet and Okta">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-14">
<meta name="articleImageUrl" value="../website/assets/images/articles/migrate-to-fleet-800x450@2x.png">
<meta name="description" value="Move directory, SSO, and MFA from JumpCloud to Okta and devices to Fleet, with zero-touch Windows and easy Apple and Android migration.">
