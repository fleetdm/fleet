# Fleet 4.86.0 | Rotate local admin password, Windows setup experience, Platform SSO, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/PLACEHOLDER_VIDEO_ID" title="0" allowfullscreen></iframe>
</div>

Fleet 4.86.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.86.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Rotate local admin password](#rotate-local-admin-password)
- [Windows setup experience: cancel if software fails](#windows-setup-experience-cancel-if-software-fails)
- [Platform SSO during macOS Setup Assistant](#platform-sso-during-macos-setup-assistant)
- [Upload your org logo](#upload-your-org-logo)
- [Automatic SCEP and ACME certificate renewal](#automatic-scep-and-acme-certificate-renewal)
- [iOS/iPadOS software for user-enrolled hosts](#iosipados-software-for-user-enrolled-hosts)


### Rotate local admin password

Building on the [local admin account](https://fleetdm.com/articles/fleet-4.85.0#create-a-local-admin-account-during-macos-setup) introduced in 4.85, Fleet now lets admins rotate the hidden account's password directly from the **Host details** page. After an admin views the password, Fleet starts a countdown and automatically rotates it roughly an hour later — limiting how long a credential is valid after it's been seen. Admins can also rotate immediately using the **Rotate password** button at any time. Every rotation is logged as an activity, whether triggered manually or automatically by Fleet.

GitHub issue: [#37142](https://github.com/fleetdm/fleet/issues/37142)

### Windows setup experience: cancel if software fails

Fleet now gives IT admins control over what happens when setup experience software fails during a Windows Autopilot or OOBE enrollment. Turning on **Cancel setup if software fails** in **Controls > Setup experience** causes the device to display a failure screen and prompt the end user to restart if any setup software doesn't install successfully. Without this toggle, failed installs are surfaced in host details but the device proceeds to the desktop anyway. A `canceled_setup_experience` activity is logged when the feature triggers, making it easy to review what went wrong.

GitHub issue: [#38785](https://github.com/fleetdm/fleet/issues/38785)

### Platform SSO during macOS Setup Assistant

Fleet now supports configuring [Platform SSO](https://support.apple.com/guide/deployment/platform-sso-depfd9cdf8ab/web) during macOS [Automated Device Enrollment (ADE)](https://fleetdm.com/articles/apple-device-enrollment-program). With this enabled, end users log in to their Mac using the same credentials they use for their identity provider (such as Okta) — no separate local password to remember. Platform SSO can be deployed alongside Fleet's existing Setup Assistant profiles and setup experience apps, with or without end user authentication enabled.

GitHub issue: [#30674](https://github.com/fleetdm/fleet/issues/30674)

### Upload your org logo

IT admins can now upload their organization's logo directly to their Fleet instance — no external image hosting required. Separate images can be set for light and dark mode. Upload during Fleet's initial setup or update later from **Settings > Organization settings**. Logos set here appear in Fleet's masthead and can be managed via the API, [`fleetctl`](https://fleetdm.com/articles/fleetctl), or [GitOps](https://fleetdm.com/docs/configuration/yaml-files). This feature is available in Fleet Free.

GitHub issue: [#39016](https://github.com/fleetdm/fleet/issues/39016)

### Automatic SCEP and ACME certificate renewal

Fleet now automatically re-pushes configuration profiles containing SCEP or ACME certificates before they expire — including certificates that aren't proxied through Fleet. This covers common identity and network access scenarios: Okta conditional access (SCEP), Okta Verify (SCEP with a static challenge), and hardware-attested ACME certificates. The renewal logic follows the [same pattern](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#renewal) already used for Fleet-proxied SCEP certificates. No Fleet configuration changes are required; include the `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` variable in the certificate profile's Subject and Fleet handles the rest.

GitHub issue: [#40639](https://github.com/fleetdm/fleet/issues/40639)

### iOS/iPadOS software for user-enrolled hosts

Fleet now supports installing VPP and in-house (`.ipa`) apps on iOS and iPadOS hosts enrolled via Account-based User Enrollment with Managed Apple Account. Admins can install apps from the host details page, and end users can install from self-service. Setup experience software also installs automatically on user enrollment. Previously, software installs on personally enrolled iOS and iPadOS devices returned an error; this release removes that restriction for Account-based User Enrolled hosts.

GitHub issue: [#31138](https://github.com/fleetdm/fleet/issues/31138)

## Changes

### IT Admins

- Added ability to upload a custom org logo (light and dark variants) hosted directly by the Fleet instance; configurable via the UI, API, `fleetctl`, and GitOps.
- Added **Cancel setup if software fails** toggle for Windows setup experience; when enabled, Autopilot and OOBE enrollments display a failure screen and prompt a restart if any setup experience software fails to install.
- Added **Rotate password** button to the managed local admin account modal on the Host details page; password auto-rotates roughly one hour after being viewed, with activity logged for both manual and automatic rotations.
- Added support for configuring Platform SSO during macOS Setup Assistant (ADE) so end users can log in with their IdP credentials.
- Added ability to install VPP and in-house (`.ipa`) apps on Account-based User Enrolled iOS and iPadOS hosts, including self-service; setup experience software installs automatically on user enrollment.
- Added support for VPP apps from non-US Apple App Stores; the VPP settings page now shows the country for each token, and apps are fetched from the storefront matching the token's country.
- Added managed app configuration (XML) for iOS and iPadOS VPP and `.ipa` apps; configurable via the UI, API, and GitOps.
- Added Fleet Desktop app for end users' macOS Dock, with a red badge when the host is failing policies.
- Added clearing of labels, pending scripts, pending software installs, and pending MDM commands when an ABM host re-enrolls; added a **Preserve host activities on re-enrollment** option in **Settings > Organization settings > Advanced options** to retain historical activity and MDM command history.

### Security Engineers

- Added automatic re-push of configuration profiles for SCEP and ACME certificates not proxied through Fleet before expiration; supports Okta conditional access (SCEP), Okta Verify (SCEP with static challenge), and hardware-attested ACME certificates.
- Added support for subject alternative name (SAN) attributes — including UPN, email (rfc822Name), and DNS — in Android certificate profiles, enabling Wi-Fi connectivity requiring SAN-based authentication.
- Added ingestion of MDM-delivered certificates (including hardware-bound ACME) via the `CertificateList` MDM command on macOS; ACME certificates now appear on the Host details page alongside osquery-ingested certificates, deduplicated by SHA-1 fingerprint.

### Bug fixes and improvements

- See the [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.86.0) for the full list of bug fixes and improvements.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.86.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-05-28">
<meta name="articleTitle" value="Fleet 4.86.0 | Rotate local admin password, Windows setup experience, Platform SSO, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.86.0-1600x900@2x.png">
