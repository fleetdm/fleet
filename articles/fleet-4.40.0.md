# Fleet 4.40.0 | More Data, Rapid Security Response, CIS Benchmark updates.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/8xNtquy9HFw?si=JkI5GrZvIEymRAt4" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.40.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.40.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* More osquery tables
* RSR version in host details
* CIS Benchmarks for Windows 10 updates


### More osquery tables

Fleet has introduced an enhancement by adding new osquery [tables](https://fleetdm.com/tables) into
the `fleetd` daemon, expanding the range of queryable data points for Fleet users. This development
aligns with Fleet's values of openness and ownership, harnessing the collective intelligence of the
osquery community and thinking long-term. Users can now utilize an enriched dataset from
community-driven extensions, enabling them to query and gather detailed data on various aspects of
their devices, such as FileVault status for macOS, Firefox preferences, the status of Windows
updates, and more.

### RSR version in host details

Fleet continues to iterate by incorporating Apple's macOS [Rapid Security Responses](https://support.apple.com/en-us/102657) (RSRs) into the host details. This feature, accessible through the user interface, REST API, or CLI, provides users with visibility into which macOS hosts have received the latest security patches. RSRs are an innovative approach by Apple to enhance security by delivering crucial updates swiftly and efficiently without necessitating a system restart.

These RSRs address various critical security issues that may affect Safari, WebKit, system libraries, or other components, including patches for vulnerabilities known to be exploited in the wild. By integrating this information into Fleet, administrators can ensure their managed devices are always up to date with the latest protections. It underscores Fleet's commitment to rooting out bottlenecks to empower IT professionals to maintain robust security standards across their device fleet. Incorporation of RSR information into host details enables organizations to leverage this proactive defense mechanism, aligning with the value of resilience in an ever-evolving threat landscape.


### CIS Benchmarks for Windows 10 updates

_Available in Fleet Premium and Fleet Ultimate_

Fleet has expanded its security capabilities for Windows 10 Enterprise by incorporating updates and additions to the CIS (Center for Internet Security) [benchmark policies](https://fleetdm.com/docs/using-fleet/cis-benchmarks). These benchmarks represent a consensus-driven set of best practices designed to mitigate a broad range of common vulnerabilities and are considered a cornerstone in hardening environments.

New policies include hardening measures such as disabling Internet Explorer 11 as a standalone browser to reduce the attack surface, enabling Administrator account lockout to prevent brute force attacks, and configuring RPC (Remote Procedure Call) settings to enforce packet-level privacy and authentication, thus elevating the security of inter-system communications. Additionally, adjustments such as disabling NetBIOS over public networks further protect against unnecessary exposure of system services.

Updates also reflect changes from the latest Windows 11 Release 22H2 Administrative Templates. For example, the 'Turn on PowerShell Transcription' setting has been updated from 'Disabled' to 'Enabled,' providing a more secure default state by ensuring that all PowerShell commands are logged, which is crucial for auditing and forensic activities.

These updates provide security administrators with enhanced tools and configurations to ensure their Windows 10 Enterprise machines are fortified against the latest security challenges, maintaining a robust defense against potential vulnerabilities.

## Changes

* **Endpoint operations**:
  - New tables added to the fleetd extension: app_icons, falconctl_options, falcon_kernel_check, cryptoinfo, cryptsetup_status, filevault_status, firefox_preferences, firmwarepasswd, ioreg, and windows_updates.
  - CIS support for Windows 10 is updated to the lates CIS document CIS_Microsoft_Windows_10_Enterprise_Benchmark_v2.0.0.

* **Device management (MDM)**:
  - Introduced support for MS-MDM management protocol.
  - Added a host detail query for Windows hosts to ingest MDM device id and updated the Windows MDM device enrollment flow.
  - Implemented `--context` and `--debug` flags for `fleetctl mdm run-command`.
  - Support added for `fleetctl mdm run-command` on Windows hosts.
  - macOS hosts with MDM features via SSO can now run `sudo profiles renew --type enrollment`.
  - Introduced `GET mdm/commandresults` endpoint to retrieve MDM command results for Windows and macOS.
  - `fleetctl get mdm-command-results` now uses the new above endpoint.
  - Added `POST /fleet/mdm/commands/run` platform-agnostic endpoint for MDM commands.
  - Introduced API for recent Windows MDM commands via `fleetctl` and the API.

* **Vulnerability management**:
  - Added vulnerability data support for JetBrains apps with similar names (e.g., IntelliJ IDEA.app vs. IntelliJ IDEA Ultimate.app).
  - Apple Rapid Security Response version added to macOS host details (requires osquery v5.9.1 on macOS devices).
  - For ChromeOS hosts, software now includes chrome extensions.
  - Updated vulnerability processing to omit software without versions.
  - Resolved false positives in vulnerabilities for Chrome and Firefox extensions.

* **UI improvements**:
  - Fleet tables in UI reset rows upon filter/search/page changes.
  - Improved handling when deleting a large number of hosts; operations now continue in the background after 30 seconds.
  - Added the ability for Observers and Observer+ to view policy resolutions.
  - Improved app settings clarity for premium users regarding usage statistics.
  - UI buttons for live queries or policies are now disabled with a tooltip if live queries are globally turned off.
  - Observers and observer+ can now run existing policies in the UI.

### Bug fixes and improvements

* **REST API**:
  - Overhauled REST API input validation for several endpoints (hosts, carves, users).
  - Validation error status codes switched from 500 to 400 for clarity.
  - Numerous new validations added for policy details, os_name/version, etc.
  - Addressed issues in /fleet/sso and /mdm/apple/enqueue endpoints.
  - Updated response codes for several other endpoints for clearer error handling.

* **Logging and debugging**:
  - Updated Apple Business Manager terms logging behavior.
  - Refined the copy of the ABM terms banner for better clarity.
  - Addressed a false positive CVE detection on the `certifi` python package.
  - Fixed a logging issue with Fleet's Cloudflare WARP software version ingestion for Windows.

* **UI fixes**:
  - Addressed UI bugs for the "Turn off MDM" action display and issues with the host details page's banners.
  - Fixed narrow viewport EULA display issue on the Windows TOS page.
  - Rectified team dropdown value issues and ensured consistent help text across query and policy creation forms.
  - Fixed issues when applying config changes without MDM features enabled.

* **Others**:
  - Removed the capability for Premium customers to disable usage statistics. Further information provided in the Fleet documentation.
  - Retired creating OS policies from host OSes in the UI.
  - Addressed issues in Live Queries with the POST /fleet/queries/run endpoint.
  - Introduced database migrations for Windows MDM command tables.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.40.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-11-06">
<meta name="articleTitle" value="Fleet 4.40.0 | More Data, Rapid Security Response, CIS Benchmark updates.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.40.0-1600x900@2x.png">
