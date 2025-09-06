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

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.73.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-09-04">
<meta name="articleTitle" value="Fleet 4.73.0 | Linux OS vulnerabilities, schedule scripts, custom variables, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.73.0-1600x900@2x.png">
