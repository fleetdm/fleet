# Fleet 4.66.0 | Windows Fleet-maintained apps, DigiCert integration, Custom SCEP server

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/ApZthJXwqqM?si=CwVISKn9mmANxumz" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.66.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.66.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Fleet-maintained apps for Windows
- DigiCert certificate integration
- Custom SCEP server support

### Fleet-maintained apps for Windows

Fleet now supports [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) for Windows. This allows IT admins to easily manage and deploy trusted applications at scale, without manually packaging or scripting installations.

### DigiCert certificate integration

Fleet now integrates with DigiCert Trust Lifecycle Manager, enabling admins to [deploy DigiCert certificates](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#digicert) directly to their macOS devices via configuration profiles. This simplifies certificate management and helps streamline the provisioning process.

### Custom SCEP server support

Admins can now use their own [custom Simple Certificate Enrollment Protocol (SCEP) servers](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#custom-scep-server) with Fleet. This integration allows deployment of certificates to Macs through configuration profiles, while ensuring all traffic to the SCEP server is routed through Fleet.

## Changes

### Security
- Added integration with DigiCert Trust Lifecycle Manager. Fleet admins can now deploy DigiCert certificates to their macOS devices via configuration profiles.
- Updated activity log UI for new certificate authority features.
- Updated **Host details** > **Software** table to filter by vulnerability severity and known exploit.
- Return more granular data for live query and policy runs so it can be displayed to users.
- Added support for queries targeting hosts via label.
- Added `author_id` to labels DB table to track who created a label.
- Removed duplicate download/delete attempts for MSRC bulletins when hosts are enrolled spanning multiple builds of the same version of Windows.
- Split up expired query deletion to avoid deadlocks in zero-trust flows.
- Moved software version transformations for vulnerability matching out of software ingestion to ensure software inventory versions match what osquery reports. 
- Modified host software query to apply the vulnerability filter on VPP apps and latest software installs & uninstalls.
- Fixed false positive on macOS 15.3 by making sure we match the version format reported by Vulncheck.
- Fixed false positive for CVE-2024-6286 on non-Windows hosts.

### IT
- Added support for Fleet-maintained apps for Windows.
- Added integration with a custom SCEP server. Fleet admins can now deploy certificates from their own SCEP server to their macOS devices via configuration profiles. The SCEP server will only see traffic from the Fleet server.
- Return more granular data for live query and policy runs so it can be displayed to users.
- Added support for queries targeting hosts via label.
- Updated macOS setup experience to show an error if an App Store app installation fails due to lack of licenses.
- Added `platform` key to `software_package` and `app_store_app` keys throughout API.
- Improved error messages when Fleet admin tries to upload a FileVault (macOS) or a BitLocker (Windows) configuration profile.
- Ignored compatible Linux hosts in disk encryption statistics and filters if disk encryption is disabled.
- Allowed for any number of comments at the top of XML files for Windows MDM profile CSPs.
- Disabled unsupported automatic install option during add flow of .exe custom packages.
- Updated Fleet to treat software installer download errors as a failure for that installation attempt, which prevents the software installation from remaining in "pending".
- Added Apple Root Certificate for HTTP requests to https://gdmf.apple.com/v2/pmv. This solves the issue of minimum macOS version not being enforced at enrollment.
- Removed unreliable default (un)install scripts for .exe software packages; install and uninstall scripts are now required when adding .exe packages.
- Added software URL validation in GitOps to catch URL parse errors earlier.


### Bug fixes and improvements
- Fixed software installer download and Fleet-maintained app errors by extending the timeout for the download and FMA add endpoints.
- Fixed issue where bootstrap package was incorrectly installed during renewal of Apple MDM enrollment profiles.
- Fixed a bug to ignore Windows hosts that are not enrolled in Fleet MDM for disk encryption statistics and filters.
- Fixed policy automation with scripts to surface errors to user instead of rendering false success message.
- Fixed whitespace not being displayed correctly in policy automation calendar preview.
- Fixed bug where Windows profiles were not being resent after `fleetctl` GitOps update.
- Fixed row selection firing twice in host selection screen.
- Fixed **Dashboard** > **Software** table truncating host count.
- Fixed an error when requesting `/fleet/software/titles` endpoint unpaginated with > 33k software titles by batching the policies by software title id query
- Fixed an issue where removing label conditions on configuration profiles (e.g. `labels_include_any`, `labels_include_all` or `labels_exclude_any`) did not clear the labels associated with the profile when applied via `fleetctl gitops`.
- Updated the empty states when choosing a label scope for new software, queries, and profiles.
- Clarified meanings of various types and fields involved in live query/policy infrastructure, document, and refactor for improved code clarity.
- Added configuration to Fleet server to enable H2C (forcing http2) to get around a limitation in GCP Cloud Run for upload file sizes.
- Added validation to both org logo URL fields, and accept data URIs as valid.
- Removed redundant json array parsing in osquery pack report handler.
- Added `took` field (request duration) on server logs for requests that fail (non-2XX).
- Unified all pagination logic and styling.
- Updated the new policy flow and associated UI elements.
- Updated UI to cleanly truncate two overflowing values and display full values in a tooltip.
- Removed extra space above Next and Previous buttons in host activity feeds.
- Allowed team GitOps to run without global config.
- Added support for displaying scheduled query labels in `fleetctl`.
- Updated `fleetctl` to print an informative error message when it is authenticated with a user who is required to reset their password.
- Stopped `fleetctl` npm publishing script from tagging patch releases for old versions as `latest`. 

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.66.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2025-04-04">
<meta name="articleTitle" value="Fleet 4.66.0 | Windows Fleet-maintained apps, DigiCert integration, Custom SCEP server">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.66.0-1600x900@2x.png">

