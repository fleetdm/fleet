# Fleet 4.74.0 | Custom software icons, batch script details, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/n35ROwlHGTU?si=tAvx2YiVbR-ycqWo" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.74.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.74.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Custom software icons
- Batch script details
- Linux setup experience
- Self-service software intrucutions
- AWS IAM autnetication for MySQL and Redis

### Custom software icons

IT Admins can now add custom icons for software. This makes it easier for end users to visually identify and install the tools they need from self-service. Learn more in the [self-service software guide](https://fleetdm.com/guides/software-self-service).

### Batch script details

A new page gives you a full view of all batch scripts that are scheduled or running. No more hunting for script statuses, output, and errors across individual hosts' **Host details** pages. Now it’s all in one place. Learn more in the [scripts guide](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### Linux setup experience

You can now install required software as part of your Linux setup process and show live progress to the end user. This ensures new Linux workstations are ready for day one, with less needing manual setup. Learn more in the [Linux setup experience guide](https://fleetdm.com/guides/windows-linux-setup-experience).

### Self-service software intructions

Fleet now includes default launch instructions for software in the **My device > Self-service** tab. End users get simple tips on how to open installed apps (e.g., “Finder > Applications” on macOS), reducing confusion and helping them get started faster. 

### AWS IAM autnetication for MySQL and Redis

Companies that self-host Fleet can now use IAM (Identity and Access Management) authentication for MySQL and Redis. This lets use short-lived credentials to algin with AWS best practices. Learn how to configure in the [Fleet server configuration reference](https://fleetdm.com/docs/configuration/fleet-server-configuration#mysql).

## Changes

### Security Engineers
- Added support for Hydrant as a Certificate Authority and added an experimental API that can be used to have Fleet request a certificate from a Hydrant.
- Added a check to disallow FLEET_SECRET variables in Apple configuration profile `<PayloadDisplayName>` fields for security.
- Added `/batch/{batch_execution_id:[a-zA-Z0-9-]+}/host-results` API endpoint to list hosts targeted in batch.
- Added `POST /api/v1/fleet/configuration_profiles/batch` API endpoint to batch modify MDM configuration profiles.
- Added a new page in the UI for batch script run details.
- Added support for AWS RDS (MySQL) IAM authentication.
- Added support for AWS ElastiCache (Redis) IAM authentication.
- Added support for hosts enrolled with Company Portal using the legacy SSO extension for Entra's conditional access.

### IT Admins
- Added setup experience software items for Linux devices.
- Added API endpoints for Linux setup experience.
  - Device API endpoints for fleetd: `POST /api/fleet/orbit/setup_experience/init` and `POST /api/v1/fleet/device/{token}/setup_experience/status`.
  - `PUT /api/v1/fleet/setup_experience/software` and `GET /api/v1/fleet/setup_experience/software` now have a `platform` argument (`linux` or `macos`, defaults to `macos`).
- Added IdP `fullname` attribute as a valid Fleet variable for Apple configuration profiles.
- Added the username of the managed user account user-scoped profiles are delivered to for macOS hosts.
- Enabled configuring webhook and ticket policy (Jira/Zendesk) automations for "No team".
- Added support for writing multiple packages in a single GitOps YAML file included under `software.packages`.
- Moved `self_service`, `labels_include_any`, `labels_exclude_any`, `categories`, and `setup_experience` declarations to team level for software in GitOps; `setup_experience` can now be set on a software package, Fleet Maintained App, or App Store app.
- Changed `GET /host/:id` to return an empty array for `software` field when `exclude_software=true`.
- Updated `generate-gitops` command to output filenames with emojis and other special characters where applicable.
- Added a Fleet-maintained app for macOS: Omnissa Horizon Client.
- Added opening instructions to self-service macOS apps and Windows programs.

### Other improvements and bug fixes
- Added index to `distributed_query_campaign_targets` table to speed up DB performance for live queries.
> **WARNING:** For deployments with millions of rows in `distributed_query_campaign_targets`, the database migration to add the index may take significant time. We recommend testing migration duration in a staging environment first. The initial cleanup of old campaign targets will occur progressively over multiple hours to avoid database overload.
- Added clean up of live query campaign targets 24 hours after campaign completion. This keeps the DB size in check for performance of large and frequent live query campaigns.
- Improved OpenTelemetry integration to add tracing to async tasks (host seen, labels, policies, query stats) and improve HTTP span naming, enabled gzip compression, reduced batch size to prevent gRPC errors.
- Updated output from `packages_only=true` so that it only returns software with available installers.
- Added tarballs summary card back into UI. 
- Improved the sorting of batch scripts in the Batch Progress UI. Batches in the "started" state now sort by started date, and batches in the "finished" state now sort by the finished date.
- Removed inaccurate host count timestamp on the software version details page.
- Downgraded "distributed query is denylisted" error to a warning on the Fleet server since this message indicates a likely issue on the host and not the server. We will surface this issue in the UI in the future.
- Improved performance for YARA rules: when modifying config (`PATCH /api/latest/fleet/config`) with a large number of yara rules and when large numbers of hosts fetch rules via /api/osquery/yara/{name} endpoint.
- Improved performance when updating multiple policies in the UI. The policies are now updated in series to reduce server/DB load.
- Added user icon to OS settings custom profiles on host details page if they are user scoped.
- Added clearer error messages when a new password doesn't meet the password criteria.
- Removed extra spacing from under disk encryption table.
- Updated `fleetctl get mdm-command-results` to show output in a vertical format instead of a table.
- Optimized os_versions API response time.
- Added logic to detect and fix migration issues caused by improperly published Fleet v4.73.2 Linux binary.
- Refactored ApplyQueries DS method so that queries are upserted in batches, this was done to avoid deadlocks during large gitops runs.
- Refactored the way failing policies are computed on host details endpoint to avoid discrepancies due to read replica delays and async computation.
- Refactored PATH fleet/config endpoint to use the primary DB node for both persisting changes and fetching modified App Config.
- Fixed missing ticket integration options in Policies -> Other workflows modal for teams.
- Fixed deduplicating bug in UI to only count unique vulns when counting software title vulnerabilities across versions in various software title vulnerabilities count, and host software title vulnerabilities count.
- Fixed cases where the default auto-install policy for .deb packages would treat installed-then-uninstalled software as still installed.
- Fixed the message rendered from user_failed_login global activities on the Activity feed if the email is not specified.
- Fixed fleetctl printing binary data to terminal in debug mode.
- Fixed a bug where incorrect CVEs were received from MSRC feed.
- Fixed Fleet-installed host count not updating after software is installed over an older version.
- Fixed UI issue in the Dashboard page. The software card is now rendered while content is been fetched to avoid the layout to jump around.
- Fixed error when updating a script to exactly match the contents of another script.
- Fixed an issue where string concatenations in a LIKE expression caused a syntax error in the query editor.
- Fixed `fleetctl gitops` issue uploading an Apple configuration profile with a FLEET_SECRET in a `<data>` field.
- Fixed Linux lock script on Ubuntu with GDM to now switch UI to text mode to work around GUI issues.
- Fixed Google Cloud Storage (GCS) support broken since Fleet 4.71.0 by implementing a workaround for AWS Go SDK v2 signature compatibility issues with GCS endpoints.
- Fixed banner link colors in UI. 
- Fixed an alignment issue on the My device page.
- Fix deadlocks when updating automations for 10+ policies at one time.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.74.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-10-06">
<meta name="articleTitle" value="Fleet 4.74.0 | Custom software icons, batch script details, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.74.0-1600x900@2x.png">
