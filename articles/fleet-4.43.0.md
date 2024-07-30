# Fleet 4.43.0 | Query performance reporting, host targeting improvements.

![Fleet 4.43.0](../website/assets/images/articles/fleet-4.43.0-1600x900@2x.png)

Fleet 4.43.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.43.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Enhanced user-device association
* Disable scripts
* Osquery version


### Enhanced user-device association

Fleet introduces a significant improvement regarding the management of user information. This update allows Fleet users to set the end-user email address directly within Fleet without depending on external sources like a Google Chrome profile or an Identity Provider (IdP). This enhancement simplifies and streamlines associating user email addresses with their respective devices. Administrators have greater flexibility and control over user data by enabling direct input and managing end-user email addresses within the Fleet platform. This feature represents Fleet's ongoing commitment to user-friendly and adaptable device management solutions.

### Disable scripts

A feature to fully disable remote script execution across an organization has been added in this release. This feature aligns with Fleet's value of ownership, as it gives administrators greater control over their Fleet environment. For organizations that want a "read-only" Fleet, this ensures they can tailor the platform to their organization's specific security policies and operational requirements.

Implementing disabling remote script execution reflects Fleet's commitment to adaptable and secure device management solutions. It acknowledges different organizations' diverse needs and security concerns, offering the flexibility to opt out of this capability if it doesn't align with their particular security posture or operational strategy. This update is a testament to Fleet's dedication to providing a versatile, user-centric platform that prioritizes its users' unique needs and preferences in a straightforward, no-frills manner.


### Osquery version

_Available in Fleet Premium_

Administrators can now specify the version of `osqueryd`, `fleetd`, and Fleet desktop to be used on an endpoint, offering options such as "stable," "edge," or a specific version number. A fallback version can also be specified if the preferred version is unavailable. This provides greater flexibility and control over the deployment of Fleet and aligns with Fleet's commitment to delivering tailored and efficient device management solutions.

By enabling the specification of versions through server overrides, Fleet demonstrates its dedication to openness and ownership, empowering users with more personalized and adaptable tools. This feature is especially beneficial for organizations that require precise version control to meet specific security, compatibility, or testing needs. The ability to choose between stable releases, cutting-edge versions, or particular version numbers ensures that Fleet users can optimize their endpoint management strategies in alignment with their unique operational requirements.




## Changes

* **Endpoint operations**:
  - Added new `POST /api/v1/fleet/queries/:id/run` endpoint for synchronous live queries.
  - Added `PUT /api/fleet/orbit/device_mapping` and `PUT /api/v1/fleet/hosts/{id}/device_mapping` endpoints for setting or replacing custom email addresses.
  - Added experimental `--end-user-email` flag to `fleetctl package` for `.msi` installer bundling.
  - Added `host_count_updated_at` to policy API responses.
  - Added ability to query by host display name via list hosts endpoint.
  - Added `gigs_total_disk_space` to host endpoint responses.
  - Added ability to remotely configure `fleetd` update channels in agent options (Fleet Premium only, requires `fleetd` >= 1.20.0).
  - Improved error message for osquery log write failures.
  - Protect live query performance by limiting results per live query.
  - Improved error handling and validation for `/api/fleet/orbit/device_token` and other endpoints.

* **Device management (MDM)**:
  - Added check for custom end user email fields in enrollment profiles.
  - Modified hosts and labels endpoints to include only user-defined Windows MDM profiles.
  - Improved profile verification logic for 'pending' profiles.
  - Updated enrollment process so that `fleetd` auto-installs on Apple hosts enabling MDM features manually.
  - Extended script execution timeout to 5 minutes.
  - Extended Script disabling functionality to various script endpoints and `fleetctl`.

### Bug fixes and improvements
  - Fix profiles incorrectly being marked as "Failed". 
    - **NOTE**: If you are using MDM features and have already upgraded to v4.42.0, you will need to take manual steps to resolve this issue. Please [follow these instructions](https://github.com/fleetdm/fleet/issues/15725) to reset your profiles. 
  - Added tooltip to policies page stating when policy counts were last updated.
  - Added bold styling to profile name in custom profile activity logs.
  - Implemented style tweaks to the nudge preview on OS updates page.
  - Updated sort query results and reports case sensitivity and default to sorting.
  - Added disk size indication when disk is full. 
  - Replaced 500 error with 409 for token conflicts with another host.
  - Fixed script output text formatting.
  - Fixed styling issues in policy automations modal and nudge preview on OS updates page.
  - Fixed loading spinner not appearing when running a script on a host.
  - Fixed duplicate view all hosts link in disk encryption table.
  - Fixed tooltip text alignment UI bug.
  - Fixed missing 'Last restarted' values when filtering hosts by label.
  - Fixed broken link on callout box on host details page. 
  - Fixed bugs in searching hosts by email addresses and filtering by labels.
  - Fixed a bug where the host details > software > munki issues section was sometimes displayed erroneously.
  - Fixed a bug where OS compatibility was not correctly calculated for certain queries.
  - Fixed issue where software title aggregation was not running during vulnerability scans.
  - Fixed an error message bug for password length on new user creation.
  - Fixed a bug causing misreporting of vulnerability scanning status in analytics.
  - Fixed issue with query results reporting after discard data is enabled.
  - Fixed a bug preventing label selection while the label search field was active.
  - Fixed bug where `fleetctl` did not allow placement of `--context` and `--debug` flags following certain commands.
  - Fixed a validation bug allowing `overrides.platform` to be set to `null`.
  - Fixed `fleetctl` issue with creating a new query when running a query by name.
  - Fixed a bug that caused vulnerability scanning status to be misreported in analytics.
  - Fixed CVE tooltip bullets on the software page.
  - Fixed a bug that didn't allow enabling team disk encryption if macOS MDM was not configured.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.43.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-01-09">
<meta name="articleTitle" value="Fleet 4.43.0 | Query performance reporting, host targeting improvements.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.43.0-1600x900@2x.png">
