# Custom disk encryption profiles

_Available in Fleet Premium_

Fleet's built-in disk encryption enforcement covers the most common FileVault and BitLocker setups. If your organization needs settings Fleet doesn't manage directly, such as a specific FileVault deferral policy or a BitLocker encryption method, you can upload your own configuration profile instead.

By default, Fleet blocks configuration profiles that touch FileVault or BitLocker settings, so a profile meant to work alongside Fleet's enforcement can't accidentally conflict with it. Turning on custom disk encryption profiles lifts that block.

## Prerequisites

- Fleet Premium.
- Access to Fleet's server configuration. Only self-managed deployments can change this setting. If you're a Fleet managed cloud customer, contact Fleet to have it enabled for you.
- macOS or Windows MDM [turned on](https://fleetdm.com/guides/macos-mdm-setup).

## Turn on custom disk encryption profiles

Set the `mdm.enable_custom_disk_encryption` server configuration option to `true`:

```yaml
mdm:
  enable_custom_disk_encryption: true
```

Or with the environment variable:

```
FLEET_MDM_ENABLE_CUSTOM_DISK_ENCRYPTION=true
```

See the [server configuration reference](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-enable-custom-disk-encryption) for more on applying server configuration changes.

> **Warning:** Custom disk encryption configuration profiles can conflict with the profiles Fleet manages when [Fleet's disk encryption enforcement](https://fleetdm.com/guides/enforce-disk-encryption) is also turned on for the same hosts. Test your profile on a team with Fleet's disk encryption enforcement turned off before rolling it out more broadly.

## Upload a custom FileVault profile

With custom disk encryption profiles turned on, you can upload a `.mobileconfig` profile that configures FileVault directly, including [FDEFileVault](https://developer.apple.com/documentation/devicemanagement/fdefilevault), [FDEFileVaultOptions](https://developer.apple.com/documentation/devicemanagement/fdefilevaultoptions), and [FDERecoveryKeyEscrow](https://developer.apple.com/documentation/devicemanagement/fderecoverykeyescrow) payloads.

1. In the Fleet UI, head to **Controls > OS settings > Configuration profiles**.
2. Select the fleet you want to add the profile to.
3. Select **Add profile** and choose your `.mobileconfig` file.

For the full upload workflow, including the API and GitOps, see [Configuration profiles](https://fleetdm.com/guides/custom-os-settings).

## Upload a custom BitLocker profile

With custom disk encryption profiles turned on, you can upload a Windows configuration profile that targets any [BitLocker CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/bitlocker-csp), for example `./Device/Vendor/MSFT/BitLocker/AllowStandardUserEncryption`.

1. In the Fleet UI, head to **Controls > OS settings > Configuration profiles**.
2. Select the fleet you want to add the profile to.
3. Select **Add profile** and choose your Windows configuration profile XML file.

For the full upload workflow, including the API and GitOps, see [Configuration profiles](https://fleetdm.com/guides/custom-os-settings).

## Troubleshoot

**"Couldn't add. The configuration profile can't include FileVault settings."** The profile includes a FileVault payload type, and custom disk encryption profiles aren't turned on. Turn on `mdm.enable_custom_disk_encryption` and try again.

**"Couldn't add. The configuration profile can't include BitLocker settings."** The profile targets a BitLocker CSP, and custom disk encryption profiles aren't turned on. Turn on `mdm.enable_custom_disk_encryption` and try again.

## Further reading

- [Enforce disk encryption](https://fleetdm.com/guides/enforce-disk-encryption)
- [Configuration profiles](https://fleetdm.com/guides/custom-os-settings)
- [Server configuration reference](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-enable-custom-disk-encryption)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2026-08-29">
<meta name="articleTitle" value="Custom disk encryption profiles">
<meta name="description" value="Upload your own FileVault or BitLocker configuration profile alongside or instead of Fleet's built-in disk encryption enforcement.">
