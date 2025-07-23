# Fleet 4.71.0 | IdP labels, user certificates, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/29tyuFGgGMI?si=4c4K_hDDBVuOC_1r" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.71.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.71.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Labels based on identity provider (IdP) groups and departments
- IdP foreign vitals
- Deploy user certificates
- Software installation status improvements

### Labels based on identity provider (IdP) groups and departments

IT admins can now build labels based on users’ IdP groups and departments. This enables different apps, OS settings, queries, and more based on group and department. Learn how to map IdP users to hosts in the [foreign vitals guide](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts).

### IdP foreign vitals

Fleet now supports using end users’ IdP department info in [configuration profile variables](https://fleetdm.com/docs/configuration/yaml-files#:~:text=In%20Fleet%20Premium%2C%20you,are%20sent%20to%20hosts). This allows IT admins to deploy a [property list](https://en.wikipedia.org/wiki/Property_list) (via configuration profile) so that third-party tools (i.e. Munki) can automate actions based on department data.

### Deploy user certificates

Fleet can now deploy and renew certificates from Microsoft Network Device Enrollment Service (NDES), DigiCert, and custom Simple Certificate Enrollment Protocol (SCEP) certificate authorities (CAs) directly to the login (user) Keychain. This makes it easier to connect employees to third-party tools that require user-level certificates. Learn more in the ["Connect end users to Wi-Fi or VPN" guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

### Software installation status improvements

Fleet now marks [App Store (VPP) apps](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet) as installed once they're visible via Apple MDM inventory, rather than as soon as the installation MDM command is acknowledged by the device. Successful installs and uninstalls (for VPP, [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps), and [custom packages](https://fleetdm.com/guides/deploy-software-packages)) also now automatically trigger a host vitals refetch, ensuring that software inventory and policy statuses quickly reflect changes made as a result of adding or removing software, rather than taking up to an hour by default.

This release also introduces a clearer differentiation between software installed on a host (Inventory) and software available for install on a host (Library) when viewing software via the Host details page. Further improvements on this page, as well as on the My device page, are [coming soon](https://github.com/fleetdm/fleet/issues/30240).

## Changes

### Security Engineers
- Updated CIS benchmarks for Windows 10 to version 3.
- Added support for IdP-based labels.
- Added last opened time for Windows applications.
- Updated `GET /hosts/:id/encryption_key` to return most recently archived encryption key if current key is not available.
- Added support for ingesting user's "Department" via SCIM and added support to set the `FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` variable on configuration profiles.
- Cleaned up false-positive vulnerabilities on Amazon Linux 2 hosts reported in Fleet <= 4.55.

### IT Admins
- Added the verification of user-scoped profiles on macOS.
- Added last opened time for Windows applications.
- Updated Windows Custom OS Settings including Win32/Desktop Bridge ADMX policies to now be marked verified after the host has acknowledged the MDM install command.
- Added support for "Host Vitals" label, starting with IdP-based labels which update automatically including after software installs.
- Displayed VPP apps installed on a host in the UI after command is acknowledged.
- Updated `GET /hosts/:id/encryption_key` to return most recently archived encryption key if current key is not available.
- Increased how often Fleet checks for new Fleet-maintained apps, from once per day to once per hour.
- Improved GitOps speed when managing software with hashes on a large number of teams.
- Separated host details software list into two separate sections: Inventory (software installed on a host) and Library (software available for installation on a host).
- Updated Apple profile verification code to disallow uploading profiles with the same identifier but differing PayloadScopes.
- Recorded installer URL when a Fleet-maintained app is added via the web UI or REST API.
- Added support for ingesting user's "Department" via SCIM and added support to set the `FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` variable on configuration profiles.
- Added support for the Apple MDM user channel. When a mobileconfig with a payloadscope of User is targeted for a host with a user channel connection, it will now be sent to the user channel.
- Added ability to add EULA end user sees during setup experience via gitops.

### Other improvements and bug fixes
- Added user property `api_only` to backend activity details.
- Replaced email with user full name for login activity.
- Added a new avatar for API-only users in the activity feed.
- Updated side navigation styles across the app.
- Added premium tier messaging to the certificates section on the integrations page.
- Removed ability to upload a EULA in the UI if gitops is enabled.
- Migrated from `aws-sdk-go` v1 to `aws-sdk-go-v2`.
- Optimized database queries for MDM enrollment checks when one host is being checked at a time.
- Replaced own SAML implementation with https://github.com/crewjam/saml.
- Increased page size for software versions shown on the software view page from 5 to 10.
- Added retries in `PATCH` policies API requests to fix deadlock errors in "Manage automations" page.
- Added missing team_name property on `/api/v1/fleet/hosts/identifier/:id` endpoint.
- Added missing "url" parameter when exporting YAML on software packages that have a URL specified (thanks @drvcodenta!)
- Improved performance when pulling team settings on osquery config and distributed read endpoints.
- Allowed team selection and name updates when saving a copy of an existing query as a new query.
- Updated Fleet maintained apps uninstall script to use `pkgutil` to remove applications files.
- Added functionality for verifying installation of VPP apps.
- Moved the SSO and Host status webhook settings from Settings > Organization to Settings > Integrations.
- Updated software installed activities created during setup experience correctly categorized as from automation.
- Fixed cases where valid operating system vulnerabilities would be periodically incorrectly purged.
- Fixed details not showing when the device page URL was edited.
- Fixed an issue where the `fleetctl` codesignature requirements couldn't be used to verify the codesignature of `fleetctl`.
- Fixed issue where IdP integration page did not show the premium feature message.
- Fixed bug present on gitops cmd when importing no-team.yml with scripts without default.yml.
- Fixed a bug where Fleet-maintained app updates via GitOps wouldn't pull the latest version of Google Chrome on each run, and would display an invalid SHA256 hash in the UI and API.
- Fixed host API to returns empty array (instead of 404) if software title or version is not found on hosts on that team consistent with other host filters.
- Fixed bug with the run script modal on the Hosts page when running under FreeTier due to invalid teamId filter.
- Fixed a case where host software counts wouldn't be updated if the host_software database table included one or more rows with a zero `software_id`.
- Fixed issue where attempting to lock an MDM-unenrolled macOS host was not returning the expected error.
- Fixed error when deleting a calendar event for a Google Workspace user that no longer exists.
- Fixed `fleetctl` panic caused by missing SSO settings during gitops generate.
- Fixed software title ID + installer status filters to return an empty array with 0 count instead of 404 when an installer is not present on a team.
- Fixed issue where iOS devices were not refetching at the expected cadence when re-enrolled without first deleting the host.
- Fixed cases where valid operating system vulnerabilities would be periodically incorrectly purged.
- Fixed issue with `PATCH /fleet/scim/Groups/<group name>` endpoint handling duplicate entries.
- Fixed bug with calendar/webhook endpoint that caused an error if the calendar event relates to a deleted host.
- Fixed host details > MDM OS settings tooltips from flashing during a host refetch.
- Fixed an issue where `macos_setup` would not always be exported by `fleetctl generate-gitops` when it should have been.
- Fixed host certificate source recording (including associated performance/database load issues) when multiple hosts share the same certificate on user keychains with differing usernames.
- Fixed software package version output in generated GitOps YAML.
- Fixed truncation of the MDM server url value on the about card on host details page.
- Fixed a bug that prevented users from adding VPP apps to macOS setup experience if the iOS version of the app was also added to their team software library.
- Fixed cases where installed-then-uninstalled software would show up in software inventory.
- Fixed automation tooltip not showing the correct filesystem log destination.
- Fixed SSO settings page returning 500 when SSO settings are undefined.
- Fixed the linux uninstall script.
- Fixed broken macOS users causing errors during query ingestion.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.71.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-07-23">
<meta name="articleTitle" value="Fleet 4.71.0 | IdP labels, user certificates, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.71.0-1600x900@2x.png">
