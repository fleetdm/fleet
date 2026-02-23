# Fleet 4.78.0 | iOS and Android self-service, cross-platform certificate deployment, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/xD4_GhxduAE?si=HQpPtW8V6zEtLzEm" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.78.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.78.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Self-service software on iOS and Android
- Install work apps on corporate iOS and Android during enrollment
- Cross-platform certificate deployment
- Okta conditional access

### Self-service software on iOS and Android

You can now offer self-service app access on both iOS/iPadOS and Android. Deploy a web-based self-service portal to iPhones ([learn how](https://fleetdm.com/guides/software-self-service#deploy-self-service-on-ios-and-ipados)) and surface approved Play Store apps in managed Google Play. 

This means you can now offer self-service software on macOS, Windows, Linux, iOS/iPadOS, and Android hosts. Learn more about [self-service software](https://fleetdm.com/guides/software-self-service).

### Install work apps on corporate iOS and Android during enrollment

You can now install managed work apps like Slack, Gmail, Zoom, and GlobalProtect during enrollment on personally-owned iOS/iPadOS and Android hosts. Apps are installed as managed, giving you control over corporate data while respecting user privacy. Learn more about installing software during [new host setup](https://fleetdm.com/guides/setup-experience).

### Cross-platform certificate deployment

You can now install certificates from any [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificate authority to the user scope on Windows hosts. This helps you connect your end users to Wi-Fi, VPN, and other tools.

This means you can now install certificates on macOS, Windows, Linux, and iOS/iPadOS hosts. See all certificate authorities supported by Fleet in [the guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

### Okta conditional access

Fleet now supports [Okta for conditional access](https://fleetdm.com/guides/okta-conditional-access-integration). This allows IT and Security teams to block third-party app logins when a host is failing one or more policies.

## Changes

### IT Admins
- Added support for Android setup experience software installation.
- Added support for Android self-service apps to `fleetctl gitops`.
- Added support for Android `systemUpdate` profiles. 
- Added ability to create/view/delete Google Play Store software for Android in UI.
- Added `$FLEET_VAR_HOST_PLATFORM` for Apple platforms (`macos`, `ios`, `ipados`).
- Added support for installation of setup-experience VPP apps on manually-enrolled iOS/iPadOS devices.
- Added ability to deploy user-scoped SCEP profiles for Windows hosts.
- Added a configuration option to require Windows users turn on MDM manually via work or school account, rather than have enrollment happen automatically.
- Added UI to allow Windows hosts to manually enroll into Fleet MDM.
- Added support for `$FLEET_VAR_HOST_HARDWARE_SERIAL` and `$FLEET_VAR_HOST_PLATFORM` in Windows profiles.

### Security Engineers
- Added ability to filter the activites on the dashboard page.
- Updated to regenerate FileVault profile when Apple MDM is turned on if the device's team has disk encryption enabled.
- Added Okta conditional access configuration to the Fleet UI under Settings -> Integrations -> Conditional access.
- Added endpoint for hosts to update certificate status.
- Added detail column to `host_certificate_template` table and added `certificate_templates` property with GitOps support.
- Updated `fleetd/certificates/<id>` and `fleetd/certificates/<id>/status` to authenticate using the orbit_node_key provided in the `Authentication` header.
- Updated MDM-enrolled Android devices to receive certificate templates in `managedConfigurations`.

### Other improvements and bug fixes
- Improved performance by making the `host_count` property optional in the `GET /labels` API endpoints.
- Improved performance by avoiding unneeded extra queries when fetching team information.
- Improved request validation by returning an informative error when trying to filter `software_titles` with `platform` without a `team_id`.
- Allowed users to save Fleet queries even if their SQL is deemed invalid by the Fleet UI.
- Added a new error UI for file uploaders, and applied it in the Okta Conditional Access modal.
- Returned pre-install query output in Install Details modal.
- Translated `idp` to `mdm_idp_accounts` on API responses. 
- Updated `last_restarted_at` property for hosts to be more reliable.
- Added Mosyle to the list of well-known MDM platforms.
- Changed where `mdm_enrolled` activity is created so it occures after the inital Token Update command to allowa the webhook to fire after the host can recieve additonal commands from Fleet MDM.
- Improved MDM command result endpoint response for pending Windows commands.
- Switched configurations referencing Redis 5 to Redis 6. Fleet is no longer verified to work with Redis 5 or below.
- Redacted API tokens in `fleetctl config set` to prevent accidental logging.
- Updated error message when attempting to run software install script on host with scripts disabled to refer to `--enable-scripts` flag (instead of `--scripts-enabled`).
- Updated queries APIs that drive the OS Settings UI to include the status of host cert templates.
- Updated the layout and styling of file uploader buttons across the UI.
- Updated built-in SVG icons to avoid rendering issues when certain combinations of icons are on the same page.
- Added consistant spacing to UI elements on the MDM page.
- Updated Go to 1.25.5.
- Fixed an issue where using bitwise operators in a query incorrectly marked the query as invalid.
- Fixed issue where MDM profile retry limits were interfering with Smallstep SCEP proxy renewal attempts, particularly in cases of expired SCEP challenges.
- Fixed incorrect status code on failure to interpolate certificate template variables.
- Fixed Android configuration profiles downloading as unusable .xml files with content `[object Object]`. Android profiles now download correctly as .json files with properly formatted JSON content, matching what was originally uploaded.
- Fixed the tab order of elements in the login form.
- Fixed UI bug where the option to resend MDM profiles for macOS hosts was incorrectly presented to non-admin and non-maintainer users.
- Fixed an issue that prevented GitOps from saving multiple queries with the same label.
- Fixed an issue where "Exclude Any" label scoping did work properly for iOS, iPadOS and Android hosts.
- Fixed bug that prevented filtering by platform when listing hosts with failed profiles.
- Fixed software action buttons to disable immediately on click to prevent multiple clicks.
- Fixed an issue where newly-enrolled Windows or Linux hosts were not automatically linked with existing SCIM user account data.
- Fixed UI bug in OS settings modal that caused status tooltip to flicker when refetching host details.
- Fixed a race condition when resending Apple Profiles that would not truly resend the latest profile.
- Fixed a missing redirect to the Fleet website.
- Fixed the connect message on the controls end user auth page so that it is consistant with the other set up experience subsections.
- Fixed a bug where "installed" software sometimes showed up as "uninstalled" when certain other pieces of data were not also present.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.78.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-12-19">
<meta name="articleTitle" value="Fleet 4.78.0 | iOS and Android self-service, cross-platform certificate deployment, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.78.0-1600x900@2x.png">
