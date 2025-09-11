# Fleet 4.73.0 | Linux OS vulnerabilities, schedule scripts, custom variables, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/NagFKf2BErQ?si=X-iavois5ZU9Bs28" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.73.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.73.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Linux OS vulnerabilities
- Custom severity (CVSS) filters
- Schedule scripts
- BitLocker PIN enforcement
- IdP authentication before BYOD enrollment
- Custom variables in scripts and configuration profiles
- Windows configuration profile variable: UUID

### Linux OS vulnerabilities

See and prioritize vulnerabilities in Linux operating systems, not just software packages. This gives you a fuller picture of risk across your environment. Learn more about [vulnerability detection in Fleet](https://fleetdm.com/guides/vulnerability-processing).

### Custom severity (CVSS) filters

Filter software by a custom severity (CVSS base score) range, like CVSS ≥ 7.5, to focus on what matters to your security team.

### Schedule scripts

Choose a specific time for a script to run. This is ideal for maintenance windows, policy changes, or planned rollouts. Learn more in the [scripts guide](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### BitLocker PIN enforcement

Require a [BitLocker PIN](https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/countermeasures#preboot-authentication) (not to be confused with BitLocker recovery key) at startup. Fleet Desktop now shows a banner instructing users to create a PIN, and reports who has or hasn’t set one.

### IdP authentication before BYOD enrollment

Add a layer of security by requiring users to authenticate with your identity provider (IdP) before enrolling their personal (BYOD) iPhone, iPad, or Android device. Learn more in the [end user authentication guide](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication).

### Custom variables in scripts and configuration profiles

You can now manage variables (used in scripts and config profiles) directly in the Fleet UI. No need to touch the API or GitOps if you don't want to. Learn more in the [custom variables guide](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

### Windows configuration profile variable: UUID

Use Fleet's new built-in `$FLEET_VAR_HOST_UUID` variable in your Windows configuration profiles to help deploy unqiue certificates to connect hosts to third-party tools like Okta Verify. See other built-in variables in Fleet's [YAML reference docs](https://fleetdm.com/docs/configuration/yaml-files#macos-settings-and-windows-settings).

## Changes

### Security Engineers
- Added new detail query, only executed if TPM PIN enforcement is required, for determining whether a BitLocker PIN is set.
- Added host identity certificate renewal support for TPM-backed certificates (Linux-only). When a certificate is within 180 days of expiration, orbit will automatically renew it using proof-of-possession with the existing certificate's private key.
- Added new global activity created when a new disk encryption key is escrowed.
- Added issuer and issued cells to the host details and my device page certificates table.
- Allowed filtering host and team software by minimum and maximum CVSS score in the Fleet UI.
- Updated UI to display kernel vulnerabilities in the operating system details page for Linux systems.
- Updated macOS 13 CIS policies to align with CIS Benchmark v3.1.0 (from v3.0.0).
- Updated macOS 14 CIS policies to align with CIS Benchmark v2.1.0 (from v2.0.0).
- Updated macOS 15 CIS policies to align with CIS Benchmark v1.1.0 (from v1.0.0).
- Updated Fleet's certificate ingestion to accept non-standard country codes of longer than 2 characters. In addition, updated ingestion of other fields to truncate long values and log an error instead of failing.

### IT Admins
- Added API endpoints for adding, deleting and listing secret variables.
- Added ability to add and delete custom variables in the UI.
- Added API endpoints to get and list batch scripts. 
- Added cron job to launch scheduled batch scripts.
- Added API endpoint to cancel scheduled batch script run.
- Added the ability to cancel batch script runs directly from the UI summary modal.
- Added ability to schedule batch script runs in advance to the "Run scripts" modal.
- Added the ability to filter the hosts list to those hosts that were incompatible with the script in a batch run.
- Added side navigation on the Controls > Scripts page, with the previous Scripts page content under the "Library" tab and a new "Batch progress" tab containing details about started, scheduled, and finished scripts.
- Added batch execution IDs to script run activities.
- Added IdP SSO authentication to the BYOD mobile devices enrollment if that option is enabled for the team.
- Allowed overriding install/uninstall scripts, and specifying pre-install queries and post-install scripts, for Fleet-maintained apps in GitOps.
- Added support of `$FLEET_VAR_HOST_UUID` in Windows MDM configuration profiles.
- Added additional logging information for Windows MDM discovery endpoint when errors occur.
- Added support for last opened time for Linux software (DEB & RPM packages).
  - NOTE: Package will need to be updated out-of-band once, because the pre-removal script from previously-generated packages is called upon an upgrade. The old pre-removal script stopped Orbit unconditionally. `fleet-osquery` can safely be updated through the Software page only _after_ a new package generated with this version of fleetctl has been installed through other means.
- Added indication of whether software on a host was never opened, vs. being a software type where last opened time collection is not supported.
- Added automatic install policies into host software responses.
- Updated `fleetctl api` to now support sending data in the body of non-GET requests using the `-F` flag. (Thanks @fuhry!) 

### Other improvements and bug fixes
- Added permissions to OS updates page so that only global admins and the team admin can see the page.
- Cleared label membership when label platform changes (via GitOps).
- Improved public IP extraction for Fleet Desktop requests.
- Marked DDM profiles as failed if response comes back with Unknown Declaration Type error, and improve upload validation for declaration type.
- Modified `PUT /api/v1/fleet/spec/secret_variables` endpoint to only accept secret variables with uppercase letters, numbers and underscores.
- Updated software inventory so that when multiple version of a software are installed the last used timestamp for each version is properly returned.
- Revised stale vulnerabilities deletion (for false positive cleanup) to clear vulnerabilities touched before the current vulnerabilities run, instead of using a hard-coded threshold based on how often the vulns cron runs.
- Removed unintended broken sort on Fleet Desktop > Software > type column.
- Validated Gitops mode URL on frontend and backend.
- Updated to not log an error if EULA is missing for the `/setup_experience/eula/metadata` endpoint.
- Loosened validation during GitOps dry runs for software installer install/uninstall scripts that contain Fleet secrets.
- Added missing checks for invalid values before trying to store them in DB.
- Updated styles for turn on MDM info banner button.
- Updated so that DEB and RPM packages generated by `fleetctl package` to now be safe to upgrade in-band through the Software page.
- Updated so that individual script executions from batch jobs are now hidden from the global feed.
- Updated to attest the signed Windows Orbit binary instead of the unsigned one.
- Updated both Fleet desktop and osquery for macOS and Windows artifacts to attest the binaries inside archives.
- Made sure that if disk encryption is enabled and a TPM PIN is required, the user is able to set a TPM PIN protector.
- Removed `DeferForceAtUserLoginMaxBypassAttempts` from FileVault profile, to use default value of 0 to indicate the FileVault enforcement can not be deferred on next login.
- Updated go to 1.24.6.
- Fixed cases where the uninstall script population job introduced in Fleet 4.57.0 would attempt to extract package IDs on software that we don't generate uninstall scripts for, causing errors in logs and retries of the job.
- Fixed potential panic in error handler when Redis is down.
- Fixed a potential race condition issue, where a host might get released because no profiles has been sent for installation before releasing the device, by checking the currently installed profiles against what is expected.
- Fixed invalid rate limiting applied on Fleet Desktop requests for which a public IP could not be determined.
- Fixed VPP token dropdown to allow user to choose "All teams" selection.
- Fixed an issue where Windows configuration profiles fails to validate due to escaping data sequence with `<![CDATA[...]]>` and profile verifier not stripping this away.
- Fixed an issue where a host could be stuck with a "Unlock Pending" label even if the unlock script was canceled.
- Fixed 5XX errors on `/api/v1/fleet/calendar/webhook/*` endpoint due to missing authorization checks.
- Fixed server panic when listing software titles for "All teams" with page that contains a software title with a policy automation in "No team".
- Fixed operating system icons from bleeding into software icons.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.73.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-09-08">
<meta name="articleTitle" value="Fleet 4.73.0 | Linux OS vulnerabilities, schedule scripts, custom variables, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.73.0-1600x900@2x.png">
