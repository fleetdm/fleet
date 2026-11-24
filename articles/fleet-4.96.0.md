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

- Hardened how fleetd reports the Entra device ID on macOS for Microsoft Entra conditional access. If you use Entra conditional access on macOS, add `com.fleetdm.orbit` to the `AppAllowList` in your Platform SSO configuration profile, and make sure your Macs have fleetd 1.64.0 or later. Fleet doesn't report a Mac's compliance to Intune until the Mac reports its Entra device ID again. Learn how in the [Entra conditional access guide](https://fleetdm.com/guides/entra-conditional-access-integration#deploy-platform-sso-configuration-profile).
- Changed Android zero-touch enrollment to ask the end user for a passcode during setup. Zero-touch configurations created before this release keep working, but the end user isn't asked for a passcode until after setup. To ask during setup, copy the new DPC extras from **Settings > Integrations > Mobile device management (MDM) > Android zero-touch** into each configuration in the zero-touch portal.

### IT Admins

- Added Google conditional access (Fleet Premium): Fleet marks company-owned iOS, iPadOS, and Android hosts as managed in Google, so Google can block sign-in from hosts Fleet doesn't manage. Connect Google in **Settings > Integrations > Conditional access**, through the REST API, or with GitOps. iOS and iPadOS end users sign in to Google once from the self-service web clip. Android hosts are registered with no sign-in.
- Added a passcode requirement during Android setup: when a fleet has an Android configuration profile with `passwordPolicies` or `passwordRequirements` and no labels, the end user sets a passcode before reaching the home screen.

### Bug fixes and improvements

- Changed `PUT /api/v1/fleet/hosts/:id/google_device_id` to require Google conditional access to be connected and a company-owned host, and to mark the device as managed in Google. `DELETE` unmarks it.
- Changed Android configuration profiles with `passwordPolicies` or `passwordRequirements` to reject Fleet variables, custom variables, and custom host vitals.
- Improved the performance of Apple MDM check-ins for hosts with a long MDM command history.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.96.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-11-23">
<meta name="articleTitle" value="Fleet 4.96.0 | Google conditional access for iOS, iPadOS, and Android, screen lock during Android setup, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.96.0-1600x900@2x.png">
