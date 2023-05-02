# Fleet 4.30.0 | MDM public beta, Observer+ role, Vulnerability publication dates.

![Fleet 4.30.0](../website/assets/images/articles/fleet-4.30.0-1600x900@2x.png)

Fleet 4.30.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.30.0) or continue reading to get the highlights.

For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Fleet introduces MDM public beta
* Granular roles added
* Vulnerability objects include publication date
* Go version to 1.19.8

## Fleet introduces MDM public beta

Fleet has enabled MDM features in the latest release of Fleet as a public beta broadly available to everyone. 🟣 Openness is at the heart of open-source, and we are excited to bring an open-source MDM. [Read the full announcement.](https://fleetdm.com/releases/fleet-introduces-mdm)

## Granular roles added
_Available in Fleet Premium and Fleet Ultimate_

With this update, you can take 🟠 Ownership of Fleet account roles with greater granularity. Fleet 4.30.0 includes a new user role, `observer+.`

The `observer+` user role extends the observer user role. The observer+ user role can edit and run SQL for a given query without saving the query allowing for greater discovery without overriding the original query. Users with the `observer+` role can also execute [live queries.](https://fleetdm.com/docs/using-fleet/fleet-ui#run-a-query)

## Vulnerability objects include publication date

Knowing how long ago a vulnerability was published, helps gauge the urgency of the vulnerability. Vulnerability objects now include the date a vulnerability was published in the National Vulnerability Database (NVD) to provide you with better 🟢 Results. The published date from NVD in the vulnerability object is also available in the Fleet API and when using the vulnerability webhooks.

## Go updated to 1.19.8

Fleet has updated Go to 1.19.8 in light of Go’s [crypto/elliptic](https://github.com/golang/go/issues/58647) fix. While this only affected niche configurations with very specific direct uses of crypto/elliptic, Fleet does not make any special use of crypto/elliptic, but Fleet takes 🟠 Ownership of the tools we use and ensures they are kept up to date.

## More new features, improvements, and bug fixes

### List of features

- Removed both `FLEET_MDM_APPLE_ENABLE` and `FLEET_DEV_MDM_ENABLED` feature flags.
- Automatically send a configuration profile for the `fleetd` agent to teams that use Automatic Device Enrollment (ADE).
- ADE JSON profiles are now automatically created with default values when the server is run.
- Added the `--mdm` and `--mdm-pending flags` to the `fleetctl get hosts` command to list hosts enrolled in Fleet MDM and pending enrollment in Fleet MDM, respectively.
- Added support for the "enrolled" value for the `mdm_enrollment_status` filter and the new `mdm_name` filter for the "List hosts", "Count hosts" and "List hosts in label" endpoints.
- Added the `fleetctl mdm run-command` command, to run any of the [Apple-supported MDM commands](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) on a host.
- Added the `fleetctl get mdm-command-results` sub-command to get the results for a previously-executed MDM command.
- Added API support to filter the host by the disk encryption status on "GET /hosts", "GET /hosts/count", and "GET /labels/:id/hosts" endpoints.
- Added API endpoint for disk encryption aggregate status data.
- Automatically install `fleetd` for DEP enrolled hosts.
- Updated hosts' profiles status sync to set to "pending" immediately after an action that affects their list of profiles.
- Updated FileVault configuration profile to disallow device user from disabling full-disk encryption.
- Updated MDM settings so that they are consistent, and updated documentation for clarity, completeness, and correctness.
- Added `observer_plus` user role to Fleet. Observers+ are observers that can run any live query.
- Added a premium-only "Published" column to the vulnerabilities table to display when a vulnerability was first published.
- Improved version detection for macOS apps. This fixes some false positives in macOS vulnerability detection.
- If a new CPE translation rule is pushed, the data in the database should reflect that.
- If a false positive is patched, the data in the database should reflect that.
- Include the published date from NVD in the vulnerability object in the API and the vulnerability webhooks (premium feature only).
- User management table informs which users only have API access.
- Added configuration option `websockets_allow_unsafe_origin` to optionally disable the websocket origin check.
- Added new config `prometheus.basic_auth.disable` to allow running the Prometheus endpoint without HTTP Basic Auth.
- Added missing tables to be cleared on host deletion (those that reference the host by UUID instead of ID).
- Introduced new email backend capable of sending email directly using SES APIs.
- Upgraded Go version to 1.19.8 (includes minor security fixes for HTTP DoS issues).
- Uninstalling applications from hosts will remove the corresponding entry in `software` if no more hosts have the application installed.
- Removed the unused "Issuer URI" field from the single sign-on configuration page of the UI.
- Fixed an issue where some icons would appear clipped at certain zoom levels.
- Fixed a bug where some empty table cells were slightly different colors.
- Fixed e-mail sending on user invites and user e-mail change when SMTP server has credentials.
- Fixed logo misalignment.
- Fixed a bug where for certain org logos, the user could still click on it even outside the navbar.
- Fixed styling bugs on the SelectQueryModal.
- Fixed an issue where custom org logos might be displayed off-center.
- Fixed a UI bug where in certain states, there would be extra space at the right edge of the Manage Hosts table.-

## Ready to upgrade?

Visit our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.30.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-04-11">
<meta name="articleTitle" value="Fleet 4.30.0 | MDM public beta, Observer+ role, Vulnerability publication dates">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.30.0-1600x900@2x.png">
