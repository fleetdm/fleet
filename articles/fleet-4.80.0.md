# Fleet 4.80.0 | Schedule app updates for dedicated devices, easier offboarding for personal devices, and more...

<div purpose="embedded-content">
   <iframe src="TODO" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.80.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.80.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Dedicated devices: Schedule App Store app updates and view location
- Personal devices: Only report managed apps and auto uninstall apps when employees leave
- Deploy in-house and custom iOS/iPadOS apps

### Dedicated devices: Schedule App Store app updates and view location

Now you can [schedule App Store app updates](https://fleetdm.com/guides/install-app-store-apps#schedule-app-updates) on dedicated (ex. Zoom room) iOS and iPadOS devices. This way, updates happen during overnight windows to minimize user disruption. In addition, Fleet now displays the last known location for these devices, making it easier to track down lost or missing hardware. Learn how to [find the location](https://fleetdm.com/guides/lock-wipe-hosts#get-location-of-locked-ios-ipados-host).

### Personal devices: Only report managed apps and auto uninstall apps when employees leave

Fleet now reports only managed (company-deployed) apps on personal (BYO) iOS and iPadOS devices. When a device unenrolls, Fleet automatically removes any company-deployed apps, ensuring your organization’s data doesn’t stay behind on personal devices.

### Deploy in-house and custom iOS/iPadOS apps

Use Fleet to deploy in-house (`.ipa`) apps to iOS and iPadOS devices. You can also deploy custom ([unlisted](https://developer.apple.com/support/unlisted-app-distribution/)) Apple App Store apps, giving you full control over your enterprise software from one place.

## Changes

### IT Admins
- Added ability to automatically uninstall managed apps when iOS/iPadOS devices are unenrolled from MDM.
- Added ability to schedule automated software updates for iOS/iPadOS VPP apps via the Fleet admin interface.
- Added the ability to get and set auto-update schedule for VPP apps via the API.
- Added scheduled updates functionality to iOS/iPadOS managed devices.
- Added custom VPP apps to available VPP apps listing.
- Added support for in-house apps to use Cloudfront signed URLs in manifest if Cloudfront is configured.

### Security Engineers
- Added NATS as a logging destination.
- Updated NDES SCEP proxy to auto-detect response encoding, enabling compatibility with Okta CA and other UTF-8-based CAs.
- Implemented ingesting, persisting, and serving the sha256 hash and path for the CFBundleExecutable binaries of .app bundles on macOS.

### Other improvements and bug fixes
- Added validation and harmonized the error message displayed when an installer (FMA, custom package, VPP app, in-house app) conflicts with another one on the same team targeting the same platform.
- Randomized APNS query to ensure all pending Apple hosts gets a push notification.
- Updated macOS bootstrap package to no longer install during MDM migration, only initial setup.
- Updated script and software installer policy automations will retry up to three times if attempts to run them fail.
- Improved host status tag styles on host details page.
- Improved error message for user-scoped profiles on iOS/iPadOS hosts.
- Surfaced Queries within the Details tab on the Host Details page.
- Updated software ingestion of manually-enrolled (BYOD) iPhone/iPad devices to only ingest (and display in software inventory) Fleet-installed software.
- Omitted software `last_opened_at` in API responses when the data source does not support it. Return an empty string when the source does have support but there is no value.
- Updated UI for Controls > Setup experience > Install software > Android to fix inconsistent loading state.
- Updated UI to show a generic error message when attempting to delete setup experience software.
- Improved error message when trying to apply certificate authorities via gitops without the correct license.
- Added space trimming of `displayVersion` when processing VPP apps (found in some production apps).
- Updated software version search to now include results that match the software title name in addition to the version name.
- Adjusted the read-only SQL editor to appear non-interactive.
- Added information about auto-update configuration to the "edited_app_store_app" activity.
- Refactored common endpoint_utils package to support bounded contexts inside Fleet codebase. Moved it to server/platform/endpointer.
- Updated UI to inform admins of the need to accept terms and conditions for multiple Apple Business Manager accounts.
- Removed Queries tab from Host Details page.
- Revised software batch upload timeout to be 4 minutes, refreshed as every software package is downloaded from source or uploaded to object storage, from 24 hours, allowing for quicker detection of when a software batch fails due to the underlying server going offline.
- Added a tooltip to an expired ABM token and also correctly removes the banner when an expired ABM token is deleted.
- Updated error message to clarify that Fleet requires Apple (macOS, iOS, and iPadOS) configuration profiles have a unique identifier (PayloadIdentifier) and scope (PayloadScope) across teams.
- Renamed "Disk space" to "Disk space available" in Host details > Vitals.
- Truncated long strings (Operating system and Hardware model) in Host details > Vitals.
- Rolled back the change to ingest legacy Entra "device ID" from the keychain (for silent migrations) because it's not supported by Entra.
Refactored common_mysql package to support bounded contexts inside Fleet codebase. Moved it to server/platform/mysql.
- Updated Go to 1.25.6.
- Fixed an issue that allowed uploading invalid Android profiles.
- Fixed spacing and alignment for author on edit query and edit policy pages.
- Fixed an issue where VPP apps would fail with 9610 errors, by implementing a retry mechanism for VPP app installations.
- Fixed VPP versions refresh to update the latest version for all platforms of an Adam ID.
- Fixed a bug where failed software installs showed up in the host library page after transferring it to a team without that installer. 
- Fixed `fleetctl` config get/set to show proper usage information when called without required arguments.
- Fixed cases where Fleet would show the wrong current VPP app version when app versions varied by platform.
- Fixed inconsistent styling for Controls > Setup experience > Bootstrap package.
- Fixed the metadata of the "Windows App" macOS installer, as it was reported as "Microsoft AutoUpdate" instead of "Windows App".
- Fixed an issue where newly-enrolled hosts would sometimes not be linked to SCIM user data.
- Fixed FMA create form to allow input fields to work properly as only edit was working correctly.
- Fixed Android certificate enrollment failures caused by SCEP challenge expiration when devices were offline.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.80.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-01-30">
<meta name="articleTitle" value="Fleet 4.80.0 | Schedule app updates for dedicated devices, easier offboarding for personal devices, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.80.0-1600x900@2x.png">