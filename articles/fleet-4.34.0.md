# Fleet 4.34.0 | ChromeOS tables, CIS Benchmark load testing.

![Fleet 4.34.0](../website/assets/images/articles/fleet-4.34.0-1600x900@2x.png)

Fleet 4.34.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Fleet adds support for ChromeOS
* Boosted compliance with 'verified' status


### Additional tables for ChromeOS

In line with Fleet's value of 🟢 Results, we work relentlessly to enhance your experience. Our aim is to deliver results, focusing on pragmatic and meaningful improvements. With this in mind, we are delighted to introduce new ChromeOS-specific tables: screenlock, system_state, privacy_preferences, and disk_info. These additions not only represent our commitment to iterative progress but also our dedication to enhancing Fleet's utility for managing and understanding your ChromeOS devices better.


### Load testing CIS Benchmarks for macOS

Embodying Fleet's values of 🟠 Ownership and 🟢 Results, our team is always ready to tackle challenges head-on for the sake of delivering a reliable and high-performing product. Recently, we pondered the performance impact of running the comprehensive set of 100 CIS Benchmarks for macOS, known colloquially as "eating our own dogfood."

Upon digging deeper, our engineers identified CIS queries 5.1.5, 5.1.6, and 5.1.7 as the three primary outliers in terms of CPU usage and memory footprint. These queries were found to be causing process terminations due to high resource usage.

The queries, which are designed to verify appropriate permissions for system-wide applications (5.1.5) and ensure no world-writable files exist in the System Folder (5.1.6) or Library Folder (5.1.7), had to be refined for efficiency.

With a clear focus on achieving results and owning the challenges we face, this rigorous load testing has led not only to the improvement of the 5.1.5, 5.1.6, and 5.1.7 queries but also to the development of additional tooling for future load testing. This is another stride in our continued effort to enhance Fleet and osquery's performance, reliability, and user experience.


## More new features, improvements, and bug fixes

* Added execution of programmatic Windows MDM enrollment on eligible devices when Windows MDM is enabled.

* Microsoft MDM Enrollment Protocol: Added support for the RequestSecurityToken messages.

* Microsoft MDM Enrollment Protocol: Added support for the DiscoveryRequest messages.

* Microsoft MDM Enrollment Protocol: Added support for the GetPolicies messages.

* Added `enabled_windows_mdm` and `disabled_windows_mdm` activities when a user turns on/off Windows MDM.

* Added support to enable and configure Windows MDM and to notify devices that are able to programmatically enroll.

* Added ability to turn Windows MDM on and off from the Fleet UI.

* Added enable and disable Windows MDM activity UI.

* Updated MDM detail query ingestion to switch MDM profiles from "verifying" or "verified" status to "failed" status when osquery reports that this profile is not installed on the host.

* Added notification and execution of programmatic Windows MDM unenrollment on eligible devices when Windows MDM is disabled.

* Added the `FLEET_DEV_MDM_ENABLED` environment variable to enable the Windows MDM feature during its development and beta period.

* Added the `mdm_enabled` feature flag information to the response payload of the `PATCH /config` endpoint.

* When creating a PolicySpec, return the proper HTTP status code if the team is not found.

* Added CPEMatchingRule type, used for correcting false positives caused by incorrect entries in the NVD dataset.

* Optimized macOS CIS query "Ensure Appropriate Permissions Are Enabled for System Wide Applications" (5.1.5).

* Updated macOS CIS policies 5.1.6 and 5.1.7 to use a new fleetd table `find_cmd` instead of relying on the osquery `file` table to improve performance.

* Implemented the privacy_preferences table for the Fleetd Chrome extension.

* Warnings in fleetctl now go to stderr instead of stdout.

* Updated UI for transferred hosts activity items.

* Added Organization support URL input on the setting page organization info form.

* Added improved ABM 400 error message to the UI.

* Hide any osquery tables or columns from Fleet UI that has hidden set to true to match Fleet website.

* Ignore casing in SAML response for display name. For example, the display name attribute can be provided now as `displayname` or `displayName`.

* Provide feedback to users when `fleetctl login` is using EMAIL and PASSWORD environment variables.

* Added a new activity `transferred_hosts` created when hosts are transferred to a new team (or no team).

* Added milliseconds to the timestamp of the auto-generated team name when creating a new team in `GET /mdm/apple/profiles/match`.

* Improved dashboard loading states.

* Improved UI for selecting targets.

* Made sure that all configuration profiles and commands are sent to devices if MDM is turned on, even if the device never turned off MDM.

* Fixed bug when reading FileVault key in osquery and created new Fleet osquery extension table to read the file directly rather than via filelines table.

* Fixed UI bug on host details and device user pages that caused the software search to not work properly when searching by CVE.

* Fixed not validating the schema used in the Metadata URL.

* Fixed improper HTTP status code if SMTP is invalid.

* Fixed false positives for iCloud on macOS.

* Fixed styling of copy message when copying fields.

* Fixed a bug where an empty file uploaded to `POST /api/latest/fleet/mdm/apple/setup/eula` resulted in a 500; now returns a 400 Bad Request.

* Fixed vulnerability dropdown that was hiding if no vulnerabilities.

* Fixed scroll behavior with disk encryption status.

* Fixed empty software image in sandbox mode.

* Fixed improper HTTP status code when `fleet/forgot_password` endpoint is rate limited. 

* Fixed MaxBurst limit parameter for `fleet/forgot_password` endpoint.

* Fixed a bug where reading from the replica would not read recent writes when matching a set of MDM profiles to a team (the `GET /mdm/apple/profiles/match` endpoint).

* Fixed an issue that displayed Nudge to macOS hosts if MDM was configured but MDM features weren't turned on for the host.

* Fixed tooltip word wrapping on the error cell in the macOS settings table.

* Fixed extraneous loading spinner rendering on the software page.

* Fixed styling bug on setup caused by new font being much wider.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.34.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-07-12">
<meta name="articleTitle" value="Fleet 4.34.0 | ChromeOS tables, CIS Benchmark load testing">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.34.0-1600x900@2x.png">
