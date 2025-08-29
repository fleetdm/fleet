# Fleet 4.73.0 | TODO

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/NagFKf2BErQ?si=X-iavois5ZU9Bs28" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.73.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.73.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Linux OS vulnerabilities
- Custom severity (CVSS) filters
- Schedule scripts
- Custom variables in scripts and configuration profiles
- BitLocker PIN enforcement
- Linux software usage
- Enroll BYOD with IdP authentication
- Windows configuration profile variable: Hardware UUID

### Linux OS vulnerabilities

See and prioritize vulnerabilities in Linux operating systems, not just software packages. This gives you a fuller picture of risk across your environment. Learn more about [vulnerability detection in Fleet](https://fleetdm.com/guides/vulnerability-processing).

### Custom severity (CVSS) filters

Filter software by a custom severity (CVSS base score) range, like CVSS ≥ 7.5, to focus on what matters to your security team.

### Schedue scripts

TODO

### Linux software usage

See the last time a Linux app was opened. Optimize license spend by identifying unused software.

### BitLocker PIN enforcement

TODO

### Enroll BYOD with IdP authentication

TODO

### Custom variables in scripts and configuration profiles

TODO

### Windows configuration profile variable: UUID

Use Fleet's new built-in `$FLEET_VAR_HOST_UUID` variable in your Windows configuration profiles to help deploy unqiue certificates to connect hosts to third-party tools like Okta Verify. See other built-in variables in Fleet's [YAML reference docs](https://fleetdm.com/docs/configuration/yaml-files#macos-settings-and-windows-settings).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.73.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-09-04">
<meta name="articleTitle" value="Fleet 4.73.0 | Linux OS vulnerabilities, schedule scripts, custom variables, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.73.0-1600x900@2x.png">
