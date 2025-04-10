# Fleet 4.63.0 | Automatically install software, faster employee onboarding

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/JM-0PKO6xvY" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.63.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.63.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Automatically install software
- Faster employee onboarding
- GitHub (SLSA) attestation

### Automatically install software

Fleet can now automatically install App Store (VPP) apps when a macOS host fails a policy. This removes the need for third-party automation tools, making large-scale app deployment easier and more reliable. Learn more about installing software [here](https://fleetdm.com/guides/automatic-software-install-in-fleet).

### Faster employee onboarding

During new employee onboarding, Macs can now optionally download bootstrap packages and software from the nearest CloudFront region. This speeds up onboarding for organizations that onboard new employees at different headquarters across the world. Learn more [here](https://fleetdm.com/guides/cdn-signed-urls).

### GitHub (SLSA) attestation

Fleet and Fleet's agent (`fleetd`) release binaries and images now include Supply-chain Level Software Attestation (SLSA). This allows security-conscious teams to verify that the artifacts they deploy are the exact ones produced by Fleet’s official GitHub workflows, ensuring integrity and preventing tampering. Learn more [here](https://fleetdm.com/guides/fleet-software-attestation). 

## Changes

## Device management (MDM)
- Allowed the delivery of bootstrap packages and software installers using signed URLs from CloudFront CDN. To enable, configured the following server settings:  
  - `s3_software_installers_cloudfront_url`
  - `s3_software_installers_cloudfront_url_signing_public_key_id`
  - `s3_software_installers_cloudfront_url_signing_private_key`
- Downgraded the expected or common "BootstrapPackage not found" server error to a debug message. This occurred when the UI or API checked if a bootstrap package existed.
- Removed the arrow icon from the MDM solution table on the dashboard page.

## Orchestration
- Added the ability to install VPP apps on policy failure.
- Added SLSA attestation to release binaries and images.
- Implemented user-level settings and used them to persist a user's selection of which columns to display on the hosts table. 
- Included a host's team-level queries when the user selected a query to target a specific host via the host details page.
- Included osquery pre-releases in the daily UI constant update GitHub Actions job.
- Displayed the correct path for agent options when a key was placed in the wrong object.
- When running a live query from the edit query form, considered the results of the run in calculating an existing query's performance impact if the user did not change the query from the stored version.  
- Improved the validation workflow on the SMTP settings page.
- Clarified the expected behavior of policy host counts, dashboard controls software count, and controls OS updates versions count.
- Rendered the default empty value when a host had no UUID.
- Used an email logo compatible with dark modes.
- Improved readability of the success message on email update by never including the sender address.

## Software
- Added the ability to install VPP apps on policy failure.
- Allowed filtering of titles by "any of these platforms" in `GET /api/v1/fleet/software/titles`.
- Added VPP apps to the automatic installation dropdown for failed policies and included auto-install information on the VPP app details page.
- Updated Fleet-maintained app install scripts for non-PKG-based installers to allow the apps to be installed over an existing installation.
- Clarified that editing VPP teams would remove App Store apps available to the team, not uninstall apps from hosts.
- Pushed the correct paths to the URL on the "My device" page when self-service was not enabled for the host.
- Displayed command line installation instructions when a package was generated.
- Added a fallback for extracting the app name from `.pkg` installers that had default or incorrect title attributes in their distribution file.
- Stopped VPP apps from being removed from teams whenever the VPP token team assignment was updated.
- Improved software installation for failed policies by adding platform-specific filtering in the software dropdown so that only compatible software was displayed based on each policy's targeted platforms.
- Added a timestamp for the software, OS, and vulnerability detail pages for the host count last update time.

## Bug fixes and improvements
- Fixed an issue where the vulnerabilities cron failed in large environments due to large SQL queries.
- Fixed two broken links in the setup experience.
- Fixed a UI bug on the "My device" page where the "Software" tab included filter elements that did not match the expected design.
- Fixed a UI bug on the "Controls" page where incorrect timestamp information was displayed while the "Current versions" table was loading.
- Fixed an issue for batch upload of Apple DDM profiles with `fleetctl gitops` where the activity feed showed a change even when profiles did not actually change.
- Fixed a software name overflow in various modals.
- Fixed form validation behavior on the SSO settings form.
- Fixed MSI parsing for packages that included long interned strings (e.g., licenses for the OpenVPN Connect installer).
- Fixed a software actions dropdown styling bug.
- Fixed an issue where identical MDM commands were sent twice to the same device when the replica database was being used.
- Fixed a redirect when clicking on any column in the Fleet Maintained Apps table.
- Fixed an issue where deleted Apple config profiles were installed on devices because the devices were offline when the profile was added.
- Fixed a CVE-2024-10327 false positive on Fleet-supported platforms (the vulnerability was iOS-only and iOS vulnerability checking was not supported).
- Fixed missing capabilities in the UI for team admins when creating or editing a user by exposing more information from the API for team admins.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.63.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-02-04">
<meta name="articleTitle" value="Fleet 4.63.0 | Automatically install software, faster employee onboarding, GitHub (SLSA) attestation">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.63.0-1600x900@2x.png">
