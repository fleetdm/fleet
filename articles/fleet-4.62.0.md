# Fleet 4.62.0 | Custom targets and automatic policies for software, secrets in configuration profiles and scripts

<div purpose="embedded-content">
   <iframe src="TODO" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.62.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.62.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Custom targets for software installs
- Automatic policies for custom packages
- Hide secrets in configuration profiles and scripts

### Custom targets for software installs

IT admins can now install Fleet-maintained apps and custom packages only on macOS, Windows, and Linux hosts within specific labels. This lets you target installations more precisely, tailoring deployments by department, role, or hardware. Learn more about deploying software [here](https://fleetdm.com/guides/deploy-software-packages).

### Automatic policies for custom packages

Fleet now creates policies automatically when you add a custom package. This eliminates the need to manually write policies, making it faster and easier to deploy software across all your hosts. Learn more about automatically installing software [here](https://fleetdm.com/guides/automatic-software-install-in-fleet).

### Hide secrets in configuration profiles and scripts

Fleet ensures that GitHub or GitLab secrets, like API tokens and license keys used in scripts (Shell & PowerShell) and configuration profiles (macOS & Windows), are hidden when viewed or downloaded in Fleet. This protects sensitive information, keeping it secure until it’s deployed to the hosts. Learn more about secrets [here](https://fleetdm.com/secret-variables).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.62.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-01-08">
<meta name="articleTitle" value="Fleet 4.62.0 | Custom targets and automatic policies for software, secrets in configuration profiles/scripts">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.62.0-1600x900@2x.png">