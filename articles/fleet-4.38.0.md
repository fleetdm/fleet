# Fleet 4.38.0 | Profile redelivery, NVD details, and custom extension label support.

![Fleet 4.38.0](../website/assets/images/articles/fleet-4.38.0-1600x900@2x.png)

Fleet 4.38.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.38.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.


## Highlights

* Failed profile redelivery
* API includes NVD description and upgrade version
* Target custom osquery extensions with labels


### Failed profile redelivery

Fleet has updated the profile delivery process for MDM by automating the redelivery of failed profiles to macOS hosts within Fleet. A profile enters a _Pending_ state when it is initially uploaded. Fleet will attempt to send the profile through the MDM protocol. When the MDM protocol returns a success, an osquery check is triggered, setting the status to _Verifying_. The status moves to _Verified_ when the osquery check is successful. If the MDM protocol or the osquery check fails, the process will automatically retry once before entering a _Failed_ state, minimizing manual intervention from the IT admin.

This feature embodies Fleet's values of 🟠 Ownership and 🟢 Results. By automating the redelivery process, Fleet takes ownership of a task that would otherwise require manual effort from IT admins, thus showing commitment to finishing what it starts. The streamlined process also aims for effective results by ensuring the profiles reach their intended state with minimal hassle, optimizing the IT admin's time and focus.


### API includes NVD description and upgrade version

_Available in Fleet Premium and Fleet Ultimate_

The upcoming additions to the Fleet API aim to enrich the vulnerability dashboard with crucial information sourced from the National Vulnerability Database (NVD). The first feature will display a description of the Common Vulnerabilities and Exposures (CVE) directly in the vulnerability list view, offering users immediate context about the security issues. This information will be accessible both via the API and fleetctl. The second feature adds a "Resolved In Version" column to the dashboard, informing users which software version has addressed a given vulnerability. This new field will be included in the /fleet/software API and offers actionable data to mitigate risks quickly.

These enhancements uphold Fleet's values of 🔴 Empathy and 🟢 Results. By incorporating NVD descriptions and "Resolved In Version" data, Fleet genuinely understands the user's needs for detailed, actionable information. These additions enable users to make informed decisions more efficiently, thereby embodying the results-driven approach that Fleet values. It aims to deliver practical solutions that allow users to address vulnerabilities effectively, offering immediate value and contributing to a more secure environment.


### Target custom osquery extensions with labels

_Available in Fleet Premium and Fleet Ultimate_

The latest update allows administrators to target custom osquery extensions to specific agents based on a combination of labels within a team. This functionality enhances precision and control by enabling the deployment of extensions only to specific platforms, such as darwin or ubuntu, according to team and label specifications. If both team and labels are specified in the YAML configuration file or via the `PATCH /api/v1/fleet/config` [API endpoint](https://fleetdm.com/docs/rest-api/rest-api#modify-configuration), the settings will be applied exclusively to the hosts within that team that meet the label criteria. This addition aligns with Fleet's values of 🟠 Ownership, for providing more granular control to administrators, and 🔵 Objectivity, for focusing on efficient and effective deployment of resources.

To make use of this functionality, you must have your own The Update Framework (TUF) server set up. For information on setting up a TUF server, consult the [Fleet documentation on self-managed agent updates](https://fleetdm.com/docs/using-fleet/update-agents#self-managed-agent-updates).



## New features, improvements, and bug fixes

* Updated MDM profile verification so that an install profile command will be retried once if the command resulted in an error or if osquery cannot confirm that the expected profile is installed.

* Ensured post-enrollment commands are sent to devices assigned to Fleet in ABM.

* Ensured hosts assigned to Fleet in ABM come back to pending to the right team after they're deleted.

* Added `labels` to the fleetd extensions feature to allow deploying extensions to hosts that belong to certain labels.

* Changed fleetd Windows extensions file extension from `.ext` to `.ext.exe` to allow their execution on Windows devices (executables on Windows must end with `.exe`).

* Surfaced chrome live query errors to Fleet UI (including errors for specific columns while maintaining successful data in results).

* Fixed delivery of fleetd extensions to devices to only send extensions for the host's platform.

* (Premium only) Added `resolved_in_version` to `/fleet/software` APIs pulled from NVD feed.

* Added database migrations to create the new `scripts` table to store saved scripts.

* Allowed specifying `disable_failing_policies` on the `/api/v1/fleet/hosts/report` API endpoint for increased performance. This is useful if the user is not interested in counting failed policies (`issues` column).

* Added the option to use locally-installed WiX v3 binaries when generating the Fleetd installer for Windows on a Windows machine.

* Added CVE descriptions to the `/fleet/software` API.

* Restored the ability to click on and select/copy text from software bundle tooltips while maintaining the abilities to click the software's name to get more details and to click anywhere else in the row to view all hosts with that software installed.

* Stopped 1password from overly autofilling forms.

* Upgraded Go version to 1.21.1.

### Bug fixes

* Fixed vulnerability mismatch between the flock browser and the discoteq/flock binary.

* Fixed v4.37.0 performance regressions in the following API endpoints:
  * `/api/v1/fleet/hosts/report`
  * `/api/v1/fleet/hosts` when using `per_page=0` or a large number for `per_page` (in the thousands).

* Fixed script content and output formatting on the scripts detail modal.

* Fixed wrong version numbers for Microsoft Teams in macOS (from invalid format of the form `1.00.XYYYYY` to correct format `1.X.00.YYYYY`).

* Fixed false positive CVE-2020-10146 found on Microsoft Teams.

* Fixed CVE-2013-0340 reporting as a valid vulnerability due to NVD recommendations.

* Fixed save button for a new policy after newly creating another policy.

* Fixed empty query/policy placeholders.

* Fixed used by data when filtering hosts by labels.

* Fixed small copy and alignment issue with status indicators in the Queries page Automations column.

* Fixed strict checks on Windows MDM Automatic Enrollment.

* Fixed software vulnerabilities time ago column for old CVEs.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.38.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-09-25">
<meta name="articleTitle" value="Fleet 4.38.0 | Profile redelivery, NVD details, and custom extension label support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.38.0-1600x900@2x.png">
