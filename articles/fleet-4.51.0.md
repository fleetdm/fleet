# Fleet 4.51.0 | Global activity webhook, macOS TCC table, and software self-service.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/K1KN0BrBncw?si=VbxhfEBwcQ95yBoB" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.51.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.51.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Global activity webhook
* macOS TCC table
* Software self-service
* Simplified APNs and ABM token uploads


## Global activity webhook

Fleet adds webhook support for global activities, broadening automation and real-time notification capabilities. This feature allows IT administrators to set up webhooks triggered by specific events within Fleet, such as changes in MDM features or re-enrollment activities. This also supports reporting mechanisms, enabling administrators to monitor the alignment between the number of devices enrolled and employees onboarded.

This update enhances operational efficiency by automating workflows and providing timely data, helping administrators manage device configurations and compliance more effectively. By leveraging webhooks for these critical events, Fleet ensures that administrators can maintain continuous oversight and respond swiftly to changes, ultimately bolstering the organization's device management and security frameworks.


## macOS TCC table

Fleet adds to its monitoring capabilities for macOS devices with support for querying the macOS TCC (Transparency, Consent, and Control) databases. This gives administrators valuable insights into applications' permissions on individual devices, particularly concerning accessing sensitive user data. The TCC framework is a critical component of macOS, designed to safeguard user privacy by managing app permissions across the system. With this update, Fleet enables IT teams to audit and verify that applications comply with organizational policies and privacy standards by accessing detailed, granular permission settings. This capability is essential for maintaining stringent security and privacy protocols, ensuring that only authorized applications can access sensitive information, and enhancing organizations' overall security posture by utilizing macOS within their fleets.


## Software self-service

Fleet aims to streamline the software installation process across organizations through software self-service. IT administrators can easily add software packages to Fleet and make them available for end-users to install via Fleet Desktop. Administrators can offer a curated list of pre-approved and organizationally vetted software directly to users, simplifying the installation process and ensuring compliance with organizational software standards. This addition not only empowers users by providing them with the autonomy to install necessary applications as needed but also ensures that all software deployed across the organization is secure and authorized, thereby maintaining high standards of IT security and operational efficiency.


## Simplified APNs and ABM token uploads

Fleet has simplified the integration of Apple Push Notification service (APNs) certificates and Apple Business Manager (ABM) tokens directly through its user interface. This update marks a significant shift from the previous requirement of using `fleetctl` commands and environmental variables for these tasks. IT administrators can effortlessly upload APNs certificates and ABM tokens via the Fleet UI, enhancing the setup process for managing Apple devices within their networks. This streamlined approach reduces the complexity of configuring necessary services for device management. It accelerates the deployment process, allowing administrators to focus more on strategic tasks than manual configurations. \


For self-managed users, the integration of these certificates requires a server private key, which is essential for activating macOS MDM features within Fleet. See Fleet's documentation for guidance on [configuring a private key](https://fleetdm.com/learn-more-about/fleet-server-private-key), which provides detailed instructions and best practices. 



## Changes

### Endpoint Operations
- Added support for environment variables in configuration profiles for GitOps.
- `fleetctl gitops --dry-run` now errors on duplicate (or conflicting) global/team enroll secrets.
- Added `activities_webhook` configuration option to allow for a webhook to be called when an activity is recorded. This can be used to send activity data to external services. If the webhook response is a 429 error code, the webhook retries for up to 30 minutes.
- Added Tuxedo OS to the Linux distribution platform list.

### Device Management (MDM)
- **NOTE:** Added new required Fleet server config environment variable when MDM is enabled,
  `FLEET_SERVER_PRIVATE_KEY`. This variable contains the private key used to encrypt the MDM
  certificates and keys stored in Fleet. Learm more at
  https://fleetdm.com/learn-more-about/fleet-server-private-key.
- Added MDM support for iPhone/iPad.
- Added software self-service support. 
- Added query parameter `self_service` to filter the list of software titles and the list of a host's software so that only those available to install via self-service are returned.
- Added the device-authenticated endpoint `POST /device/{token}/software/install/{software_title_id}` to self-install software.
- Added new endpoints to configure ABM keypairs and tokens.
- Added `GET /fleet/mdm/apple/request_csr` endpoint, which returns the signed APNS CSR needed to activate Apple MDM.
- Added the ability to automatically log off and lock out `Administrator` users on Windows hosts.
- Added clearer error messages when attempting to set up Apple MDM without a server private key configured.
- Added UI for the global and host activities for self-service software installation.
- Updated UI to support new workflows for macOS MDM setup and credentials.
- Updated UI to support software self-service features.
- Updated UI controls page language and hid CTA button for users without access to turn on MDM.

### Vulnerability Management
- Updated the CIS policies for Windows 11 Enterprise from v2.0.0 (03-07-2023) to v3.0.0 (02-22-2024).
- Fleet now detects Ubuntu kernel vulnerabilities from the Canonical OVAL feed.
- Fleet now detects and reports vulnerabilities on Firefox ESR editions on macOS.

### Bug fixes and improvements
- Fixed a bug that might prevent enqueuing commands to renew SCEP certificates if the host was enrolled more than once.
- Prevented the `host_id`s field from being returned from the list labels endpoint.
- Improved software ingestion performance by deduplicating incoming software.
- Placed all form field label tooltips on top.
- Fixed a number of related issues with the filtering and sorting of the queries table.
- Added various optimizations to the rendering of the queries table.
- Fixed host query page styling bugs.
- Fixed a UI bug where "Wipe" action was not being hidden from observers.
- Fixed UI bug for builtin label names for selecting targets.
- Removed references to Administrator accounts in the comments of the Windows lock script.

## Fleet 4.50.2 (May 31, 2024)

### Bug fixes

* Fixed a critical bug where S3 operation were not possible on a different AWS account.

## Fleet 4.50.1 (May 29, 2024)

### Bug fixes

* Fixed a bug that might prevent enqueing commands to renew SCEP certificates if the host was enrolled more than once.
* Fixed a bug by preventing the `host_id`s field from being returned from the list labels endpoint.
* Fixed a number of related issues with the filtering and sorting of the queries table.
* Added various optimizations to the rendering of the queries table.
* Fixed a bug where Bulk Host Delete and Transfer now support status and labelID filters together.
* Added the ability to automatically log off and lock out `Administrator` users on Windows hosts.
* Removed references to Administrator accounts in the comments of the Windows lock script.



## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.51.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-06-10">
<meta name="articleTitle" value="Fleet 4.51.0 | Global activity webhook, macOS TCC table, and software self-service.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.51.0-1600x900@2x.png">
