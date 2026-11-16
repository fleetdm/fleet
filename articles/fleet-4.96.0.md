# Fleet 4.96.0 | Google conditional access for iOS, iPadOS, and Android, screen lock during Android setup, and more...

<!--
<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/TODO" allowfullscreen></iframe>
</div>
-->

Fleet 4.96.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.96.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Google conditional access for iOS, iPadOS, and Android](#google-conditional-access-for-ios-ipados-and-android)
- [Require a screen lock during Android setup](#require-a-screen-lock-during-android-setup)

### Google conditional access for iOS, iPadOS, and Android

IT admins who use Google Workspace can now block sign-in from company-owned iOS, iPadOS, and Android hosts that Fleet doesn't manage. Fleet marks the hosts it manages as managed in Google, so only those hosts can access company data. This no longer requires deploying the Chrome bookmark.

Connect Fleet to Google Workspace with your customer ID, an admin email, and a service account key. iOS and iPadOS hosts register when the end user signs in with Google from the self-service web clip. Android hosts need nothing on the device. Fleet registers them automatically from the device ID it already stores. This feature is available in Fleet Premium, and only global admins can turn it on or off.

Learn more in the [Google conditional access guide](https://fleetdm.com/guides/google-conditional-access-integration). Reference docs for the API and GitOps settings: TODO.

GitHub issue: [#56195](https://github.com/fleetdm/fleet/issues/56195)

### Require a screen lock during Android setup

IT admins can now make sure company-owned Android hosts have a passcode before the end user reaches the home screen. Add an Android configuration profile with a password policy to a fleet, and the end user is prompted to set a compliant passcode during setup.

The prompt appears during setup only for profiles that apply to every host in the fleet, so profiles scoped with labels are applied after setup. Profiles with Fleet variables aren't supported for this purpose and are rejected with an error. Each fleet's zero-touch enrollment token now points to that fleet's own default Android policy. After you upgrade, copy the new DPC extras from the zero-touch settings page into the Android zero-touch portal. Old tokens keep working, but they don't prompt during setup.

Learn more in the [Android zero-touch enrollment guide](https://fleetdm.com/guides/android-zero-touch-enrollment) and the [Android MDM setup guide](https://fleetdm.com/guides/android-mdm-setup).

GitHub issue: [#57461](https://github.com/fleetdm/fleet/issues/57461)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.96.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-11-23">
<meta name="articleTitle" value="Fleet 4.96.0 | Google conditional access for iOS, iPadOS, and Android, screen lock during Android setup, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.96.0-1600x900@2x.png">
