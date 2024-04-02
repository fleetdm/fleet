# Fleet 4.41.0 | NVD API 2.0, Windows script library.

![Fleet 4.41.0](../website/assets/images/articles/fleet-4.41.0-1600x900@2x.png)

Fleet 4.41.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.40.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* NVD API 2.0
* Windows script library


### NVD API 2.0

The National Vulnerability Database (NVD) is transitioning to its new 2.0 API, a change that significantly impacts all users of vulnerability management services, including Fleet. Effective December 15th, 2023, the NVD will exclusively support the more advanced, flexible, and user-friendly 2.0 API, rendering previous versions of Fleet incompatible. This update mandates an essential upgrade to Fleet v4.41.0 (or later) to maintain access to the latest vulnerability data and ensure continuous monitoring and security compliance. Dive into the details and prepare for a seamless transition by reading our full article at [Fleet's NVD API 2.0 Update](https://fleetdm.com/announcements/nvd-api-2.0).


### Windows script library

Fleet has expanded its script management capabilities by introducing support for Windows scripts in the UI in addition to existing [CLI and API support for script execution](https://fleetdm.com/docs/using-fleet/scripts), enhancing the versatility of its Scripts Library. In addition to macOS, Fleet users can now upload, store, and manage Windows-specific scripts PowerShell `.ps1` script files. This feature enables the execution of scripts directly from the Host Details page for Windows devices, providing a streamlined and efficient process for script management. By extending script support to Windows, Fleet demonstrates a commitment to openness, catering to a broader user base and acknowledging the diverse environments in which its users operate. This update signifies Fleet’s dedication to ownership, empowering users with robust tools to manage their devices effectively across different operating systems. The addition of Windows script support in Fleet enhances its utility as a comprehensive tool for IT administrators, allowing for seamless and flexible script management in mixed-device environments.

## Changes

* **Endpoint operations**:
  - Enhanced `fleetctl` and API to support PowerShell (.ps1) scripts.
  - Updated several API endpoints to support `os_settings` filter, including Windows profiles status.
  - Enabled `after` parameter for improved pagination in various endpoints.
  - Improved the `fleet/queries/run` endpoint with better error handling.
  - Increased frequency of metrics reporting from Fleet servers to daily.
  - Added caching for policy results in MySQL for faster operations.

* **Device management (MDM)**:
  - Added database tables for Windows profiles support.
  - Added validation for WSTEP certificate and key pair before enabling Windows MDM.

* **Vulnerability management**:
  - Fleet now uses NVD API 2.0 for CVE information download.
  - Added support for JetBrains application vulnerability data.
  - Tightened software matching to reduce false positives.
  - Stopped reporting Atom editor packages in software inventory.
  - Introduced support for Windows PowerShell scripts in the UI.
  
* **UI improvements**:
  - Updated activity feed for better communication around JIT-provisioned user logins.
  - Query report now displays the host's display name instead of the hostname.
  - Improved UI components like the manage page's label filter and edit columns modal.
  - Enabled all sort headers in the UI to be fully clickable.
  - Removed the creation of OS policies from a host's operating system in the UI.
  - Ensured correct settings visibility in the Settings > Advanced section.

### Bug fixes

  - Fixed long result cell truncation in live query results and query reports.
  - Fixed a Redis cluster mode detection issue for RedisLabs hosted instances.
  - Fixed a false positive vulnerability report for Citrix Workspace.
  - Fixed an edge case sorting bug related to the `last_restarted` value for hosts.
  - Fixed an issue with creating .deb installers with different enrollment keys.
  - Fixed SMTP configuration validation issues for TLS-only servers.
  - Fixed caching of team MDM configurations to improve performance at scale.
  - Fixed delete pending issue during orbit.exe installation.
  - Fixed a bug causing the disk encryption key banner to not display correctly.
  - Fixed various error code inconsistencies across endpoints.
  - Fixed filtering hosts with invalid team_id now returns a 400 error.
  - Fixed false positives in software matching for similar names.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.41.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-11-28">
<meta name="articleTitle" value="Fleet 4.41.0 | NVD API 2.0, Windows script library.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.41.0-1600x900@2x.png">
