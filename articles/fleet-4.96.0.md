# Fleet 4.96.0 | Google conditional access for mobile and enforce screen lock during Android setup

Fleet 4.96.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.96.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Google conditional access for iOS, iPadOS, and Android](#google-conditional-access-for-ios-ipados-and-android)
- [Require a screen lock during Android setup](#require-a-screen-lock-during-android-setup)

### Google conditional access for iOS, iPadOS, and Android

IT admins who use Google Workspace can now block sign-in from iPhones, iPads, and Android devices that aren't company-owned and enrolled to Fleet. Fleet marks company-owned iOS, iPadOS, and Android hosts as managed in Google. Google's [Context-Aware Access](https://knowledge.workspace.google.com/admin/security/about-context-aware-access) then blocks devices that aren't.

Connect Google in **Settings > Integrations > Conditional access** with your Google customer ID, a Google Workspace admin's email, and a service account key. On iPhones and iPads, the end user signs in to Google once from the self-service web clip. Android hosts don't need a sign-in.

If you use the [Chrome bookmark](https://fleetdm.com/guides/google-conditional-access-integration#register-hosts-with-a-chrome-bookmark) from Fleet 4.95, connect Google in Fleet after you upgrade. The bookmark now needs it.

Learn how in the [Google conditional access guide](https://fleetdm.com/guides/google-conditional-access-integration).

GitHub issue: [#56195](https://github.com/fleetdm/fleet/issues/56195)

### Require a screen lock during Android setup

IT admins can now make sure company-owned Android hosts have a passcode before the end user reaches the home screen. Add an Android configuration profile with `passwordPolicies` or `passwordRequirements` to a fleet, and the end user sets a passcode during setup. Profiles scoped with labels are applied after setup, because a host doesn't have labels until it enrolls. Passcode profiles can't include Fleet variables, custom variables, or custom host vitals.

If you use Android zero-touch enrollment, copy the new DPC extras from Fleet into the zero-touch portal so hosts ask for the passcode during setup. See [Changes that may require action](#changes-that-may-require-action).

Learn more in the [configuration profiles guide](https://fleetdm.com/guides/custom-os-settings#target-hosts-with-labels).

GitHub issue: [#57461](https://github.com/fleetdm/fleet/issues/57461)

## Changes

### Changes that may require action

- Fleet can now ask the end user to set a passcode during Android zero-touch setup, before they reach the home screen. If you set up zero-touch before Fleet 4.96.0, your configurations in the zero-touch portal use old DPC extras. Hosts still enroll to the right fleet, but the end user isn't asked for a passcode until after setup. To ask for a passcode during setup, update each configuration:
  1. In Fleet, head to **Settings > Integrations > Mobile device management (MDM)**.
  2. Under **Android zero-touch**, select **Setup**.
  3. Pick the fleet.
  4. Copy the new **DPC extras** JSON.
  5. In the [Android zero-touch portal](https://enterprise.google.com/android/zero-touch/customers), select **Configurations** and open the configuration for that fleet.
  6. Replace the **DPC extras** with the JSON you copied, then save the configuration.
  7. Repeat for each fleet that has a passcode profile.

  The old DPC extras keep working, so there's no deadline to update them. Hosts that are already enrolled aren't affected.

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.96.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-11-23">
<meta name="articleTitle" value="Fleet 4.96.0 | Google conditional access for iOS, iPadOS, and Android, screen lock during Android setup, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.96.0-1600x900@2x.png">
