# Fleet 4.68.0 | Scheduled query webhooks, deploy tarballs, SHA-256 verification, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/Udhh-XYhb4I?si=gh9vasjviB6-3sMm" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.68.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.68.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Scheduled query webhooks
- Deploy tarballs
- SHA-256 verification
- Certificate renewal
- Configuration profile variables
- Software self-service categories
- Run scripts in bulk
- Fleet-maintained apps via GitOps (YAML)
- Generate GitOps (YAML)
- Custom Fleet agent (fleetd) during new Mac setup (ADE)

### Scheduled query webhooks

Security engineers can now send scheduled query results to a webhook URL. This makes it easy to monitor for events like new Chrome extensions or potential tampering with Fleet’s agent, helping teams respond quickly to anomalous activity.

### Deploy tarballs

Fleet now supports deploying `.tar.gz` and `.tgz packages`. Security engineers no longer need separate hosting or deployment tools, simplifying the process of distributing software across hosts. Learn more [here](https://fleetdm.com/guides/deploy-software-packages).

### SHA-256 verification

IT admins can now specify a `hash_sha256` when adding custom packages to Fleet via [GitOps (YAML)](https://fleetdm.com/docs/configuration/yaml-files#packages). Fleet will verify the hash to ensure that the uploaded software matches exactly what was intended.

### Certificate renewal

Fleet can now automatically renew certificates from DigiCert, NDES, or custom certificate authorities (CA). This ensures end users can maintain seamless Wi-Fi and VPN access without manual certificate management. Learn more [here](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

### Configuration profile variables

IT admins can now insert end users' identity provider (IdP) usernames and groups into macOS, iOS, and iPadOS configuration profiles. This allows certificates to include user-specific data and enables other tools, like Munki, to take group-based actions. See all configuration profile variables Fleet currently supports [here](https://fleetdm.com/docs/configuration/yaml-files#macos-settings-and-windows-settings).

### Software self-service categories

IT admins can now organize software in **Fleet Desktop > Self service** into categories like "🌎 Browsers," "👬 Communication," "🧰 Developer tools," and "🖥️ Productivity." This makes it easier for end users to quickly find and install the apps they need. Learn more [here](https://fleetdm.com/guides/software-self-service).

### Run scripts in bulk

IT admins can now select multiple hosts and run a script across all of them at once. This speeds up resolving issues and applying fixes across large groups of hosts.

### Fleet-maintained apps via GitOps (YAML)

IT admins can now add Fleet-maintained apps to their environment using [GitOps (YAML)](https://fleetdm.com/docs/configuration/yaml-files#fleet-maintained-apps). This enables full GitOps workflows for software management, allowing teams to manage all software alongside other configuration as code.

### Generate GitOps (YAML)

A new `fleetctl generate-gitops` command now generates GitOps (YAML) files based on your current Fleet configuration. This supports a more seamless transition from UI-based Fleet administration to GitOps.

### Custom Fleet agent (fleetd) during new Mac setup (ADE)

Fleet now allows IT admins to deploy a custom fleetd during Mac Setup Assistant (ADE). This makes it possible to custom the fleetd configuration to point hosts to a custom Fleet server URL during initial enrollment, meeting security requirements without manual reconfiguration. Learn how [here](https://fleetdm.com/guides/macos-setup-experience#advanced).

## Changes

### Security Engineers
- Built Fleet integration with Microsoft Entra to conditionally prevent single sign-on for hosts failing policies.
- Added ability to set conditional access per policy, and update host policy UI to incorporate conditional access data.
- Added CVE ID as matching criteria for host software queries, in addition to software name. Also rebuild host software querying for better maintainability.
- Updated Fleet-managed DigiCert, NDES, and SCEP certificates to be renewed 30 days before expiry for those valid longer than 30 days or when half the validity period remains for certificates valid 30 days or less. Applies to certificates requested using this release or later. 
- Added webhook as a logging configuration option.
- Added webhook query automation logging.
- Added shell and Powershell syntax highlighting when editing scripts.
- Added ability to run a script on a batch of hosts with a single user flow.
- Added download validation and existing-installer matching in GitOps via a new `hash_sha256` field in software YAML.
- Added `hash_sha256` field to the response for the `GET /software/titles` API.
- Added `fleetctl generate-gitops` command to generate gitops YAML files based on current Fleet configuration.
- Enabled saving Integrations > Advanced in GitOps mode.

### IT Admins
- Added ability to run a script on a batch of hosts with a single user flow.
- Added the ability to upload and install tarball archives (.tar.gz).
- Added support for Fleet-maintained apps in GitOps.
- Added ability to add FMA via `fleetctl` YAML files.
- Added shell and Powershell syntax highlighting when editing scripts.
- Added query ID to query automation logs.
- Added UI for the manual agent install of a bootstrap package.
- Added categorization for self-service software, including filtering on the "My device" page.
- Added number of policies triggering automatic install of software in software table.
- Added webhook as a logging configuration option.
- Added webhook query automation logging.
- Added download validation and existing-installer matching in GitOps via a new `hash_sha256` field in software YAML.
- Added `hash_sha256` field to the response for the `GET /software/titles` API.
- Added support for `FLEET_VAR_HOST_END_USER_IDP_USERNAME`, `FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` and `FLEET_VAR_HOST_END_USER_IDP_GROUPS` fleet variables in macOS MDM configuration profiles.
- Added `last_mdm_enrolled_at` and `last_mdm_checked_in_at` to host detail endpoints to return the last time a host enrolled, or re-enrolled in MDM and the last time a host checked in via MDM, respectively.
- Added `fleetctl generate-gitops` command to generate gitops YAML files based on current Fleet configuration.
- Updated Fleet-managed DigiCert, NDES, and SCEP certificates to be renewed 30 days before expiry for those valid longer than 30 days or when half the validity period remains for certificates valid 30 days or less. Applies to certificates requested using this release or later. 
- Updated host certificates with serial numbers below 2^63 will now display the decimal represntation of the serial number in addition to hex so that it is easier to match them up to what is displayed in the macOS keychain.
- Updated Install Status to correctly display available for self-service VPP apps.
- Logged invalid Windows MDM SOAP message and return 400 instead of 5XX. This change helps debug Windows MDM issues.
- Added `macos_setup.manual_agent_install` option in Mac setup experience to bypass fleetd install. Instead, fleetd should be installed via customer-customized bootstrap package.
- Allowed uploading VPP apps when GitOps mode is enabled.
- Allowed viewing the status details for an (un)install via the "My device" page.
- Updated Apple MDM enrollment flow to improve device-to-user mapping.
- Updated verification of Windows Wireless profiles to avoid resending already-applied profiles.
- Enabled saving Integrations > Advanced in GitOps mode.

### Other improvements and bug fixes
- Added hover cursors to checkbox and radio form elements.
- Added keyboard accessibility controls to activities on dashboard and host details pages.
- Added an additional statistic item to count ABM pending hosts.
- Added truncation and a conditional tooltip for long host names on the host details page.
- Updated the parser used when editing SQL in the UI to handle modern expressions like window functions.
- Updated "My device" page layout.
- Updated Google Calendar event bodies and relevant previews in the Fleet UI.
- Updated UI for Settings > Organization settings > Organization info.
- Updated LUKS escrow instrucitons.
- Updated error message and related documentation for Windows MDM configuration.
- Updated UI to show the premium feature message when viewing the GitOps mode toggle page on Fleet free.
- Cleaned up various empty and configured states on the settings pages.
- Improved performance on database migration from 4.66 and earlier for instances with large macOS host counts.
- Removed Apple MDM profile validation checks for com.apple.MCX keys (dontAllowFDEDisable and dontAllowFDEEnable) due to customer feedback.
- Removed Fleet config no team settings when the `no-team.yml` file is removed via GitOps.
- Updated Go to 1.24.2.
- Fixed an issue where the upcoming host activities showed the incorrect created at date in the tooltip.
- Fixed bug where Fleet failed to restore some "pending" hosts (i.e. hosts that remained assigned to Fleet in Apple Business Manager) when multiple hosts are deleted from Fleet.
- Fixed an issue with how names for macOS software titles were calculated and prevents duplicate entries being created if the software is renamed by end users.
- Fixed issue when Apple device was removed/re-added to ABM, it was not getting an enrollment profile.
- Fixed issue where `fleetctl gitops --dry-run` would sometimes fail when creating and using labels in the same run.
- Fixed a small bug with the way live policy result percentages were being rounded.
- Fixed an issue where selections made on the Queries page were cleared a few seconds after page load.
- Fixed an issue with the gitops command caused when trying to interpolate variables inside the 'description'/'remediation' sections.
- Fixed `fleetctl gitops` issue where creating a new team containing VPP apps caused an error.
- Fixed issue where GitOps may fail to apply new queries due to deadlocks.
- Fixed spurious install/uninstall script errors on EXE software edits when install and uninstall scripts were specified.
- Fixed issue where the host expiry window caused MDM devices assigned to Fleet in Apple Business Manager (ABM) to be repeatedly deleted and re-added to Fleet, which in some cases also caused the device to revert to the default team.
- Fixed missing To: email header.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.68.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-05-22">
<meta name="articleTitle" value="Fleet 4.68.0 | Scheduled query webhooks, deploy tarballs, SHA-256 verification, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.68.0-1600x900@2x.png">
