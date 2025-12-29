# Fleet 4.77.0 | Deploy enterprise packages for iOS/iPadOS, edit IdP username, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/2oLyaV7rIXM?si=FgWAi9K8KhEXWd_f" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.77.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.77.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Deploy enterprise iOS/iPadOS packages
- Edit IdP username
- Enforce authentication during enrollment
- Connect end users on Windows and Linux to Wi-Fi/VPN
- Activity for deleted hosts
- More Fleet-maintained apps


### Deploy enterprise iOS/iPadOS packages

You can now deploy enterprise (`.ipa`) packages to iPhones and iPads using Fleet’s [best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#software) and [API](https://fleetdm.com/docs/rest-api/rest-api#add-package). Perfect for distributing pre-release internal apps to testers or employees.

### Edit IdP username

You can now update a host’s identity provider (IdP) username directly from the Fleet UI (**Host details** page) or [API](https://fleetdm.com/docs/rest-api/rest-api#update-human-device-mapping). This makes it easier to maintain [human-to-host mapping](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts), especially if you don't require end user authentication during new host setup.

### Enforce authentication during enrollment

You can now require end users to authenticate with your IdP before Fleet installs software or runs policies on company-owned Windows and Linux setup. This ensures only authenticated users get access to company resources. Learn more in the [Windows and Linux setup guide](https://fleetdm.com/guides/windows-linux-setup-experience#end-user-authentication).

Also, you can require end users to authenticate when turning on and/or enrolling a Mac via profile-based device enrollment. [Learn more](https://fleetdm.com/guides/apple-mdm-setup#manual-enrollment).

### Connect end users on Linux and Windows to Wi-Fi/VPN

You can deploy certificates from any certificate authority (CA) that supports [Simple Certificate Enrollment Protocol (SCEP)](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificates to Windows hosts. This enables access to corporate Wi-Fi and VPN resources. [Learn how](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#custom-scep-simple-certificate-enrollment-protocol). Currently, certificates can only be deliverd to the host's device scope. User scope is coming soon. 

Also, you can now deliver certificates from any CA that supports [Enrollment over Secure Protocol (EST)](https://en.wikipedia.org/wiki/Enrollment_over_Secure_Transport) certificates to Linux hosts. This way, you can connect end users to Wi-Fi, VPN, or internal tools. [Learn how](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#custom-est-enrollment-over-secure-transport).

### Activity for deleted hosts

Deleted host events are now included in the activity feed. If a host is removed, you’ll still have a record for audits and historical tracking.

### More Fleet-maintained apps

Fleet added [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) for macOS (Claude, ChatGPT, Outlook, Webex, Spotify) and Windows (Slack, Zoom, Firefox), plus many more apps. See all Fleet-maintained apps in the [software catalog](https://fleetdm.com/software-catalog).

## Changes

### Security Engineers
- Added activity log entries for: host deletion and expiration, updating or deleting host IdP mappings.
- Resolved multiple false positive vulnerability matches for the VSCode golang extension.
- Resolved false positive CVE matches for [`Logi Bolt.app`](https://support.logi.com/hc/en-us/articles/4418089333655-Logi-Bolt-App).
- Detected vulnerabilities in JetBrains IDE plugins.

### IT Admins
- Updated MDM enrollment flow for BYOD macOS hosts to enable end user authentication prior to downloading the MDM profile via the "My device" page.
- Added self-service install support for custom IPA apps on iOS and iPadOS.
- Added support for in-house (".ipa") apps to `fleetctl gitops`.
- Updated existing `POST /setup_experience/script` endpoint to allow updating the macOS setup experience script in-place, and modified GitOps to remove the `DELETE` call.
- Added support for Custom EST certificate authorities.
- Added ability to deploy certificates from Custom SCEP certificate authorities on Windows.
- Added status counts to batch script detail page tabs.
- Added `InstallAnywhere` as a self-extracting archive for PE metadata extraction.
- Added ingestion of `upgrade_code`s from Windows software, and provided to all relevant software endpoints.

### Other improvements and bug fixes
- Improved performance of `/api/latest/fleet/software/versions` API endpoint.
- Updated host expiry logic to not delete macOS hosts that checkin via MDM protocol but not via `fleetd`.
- Optimized the cleanup Apple host profiles query to reduce probability of DB locking.
- Implemented UI logic to call existing manual update IdP API functionality.
- Implemented UI logic and new DELETE endpoint to manually remove host IdP mappings.
- Added experimental `FLEET_MDM_ENABLE_CUSTOM_OS_UPDATES_AND_FILEVAULT` configuration to allow deploying custom OS settings including Filevault payloads and macOS and Windows update settings.
- Added ability to change software display names in the UI.
- Fixed table styling for selecting table rows.
- Simplified setup experience configuration UI.
- Added better error messages when using build-in labels on GitOps and on the LabelSpecs endpoint.
- Hid software host count and version table when no hosts have the software installed.
- Adjusted UI section headers and layout of Settings > Integrations in Fleet Free.
- Added vulnerability seeding and performance testing tools.
- Moved end user authentication SSO settings under Integrations > SSO in global settings.
- Removed the premium check for host OS settings in host summary UI.
- Reduced Android device reconciler frequency to 1 hour.
- Reduced Android API usage by listing devices instead of getting and checking Android Enterprise disconnects hourly.
- Set the order of software installed during the setup experience to alphanumeric.
- Updated Go to 1.25.3.
- Fixed a layout issue on the script batch details page.
- Fixed installer for Cisco Secure Client not showing as installed in inventory/library due to using the wrong bundle identifier. This application should show up correctly now in the software inventory.
- Fixed errors when trying to run the `apple_mdm_iphone_ipad_refetcher` cron job.
- Fixed bug that prevented users from editing custom EST certificates URLs.
- Fixed incorrect UI placeholder element by replacing it with it's actual value.
- Fixed issue where vulnerabilities would occasionally show as missing.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.77.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-12-04">
<meta name="articleTitle" value="Fleet 4.77.0 | Deploy enterprise packages for iOS/iPadOS, edit IdP username, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.77.0-1600x900@2x.png">
