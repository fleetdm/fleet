# Fleet 4.60.0 | Escrow Linux disk encryption keys, custom targets for OS settings, scripts preview

![Fleet 4.60.0](../website/assets/images/articles/fleet-4.60.0-1600x900@2x.png)

Fleet 4.60.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.60.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Escrow Linux disk encryption keys
- Custom targets for OS settings
- Preview scripts before run

### Escrow Linux disk encryption keys

Fleet now supports escrowing the disk encryption keys for Linux (Ubuntu and Fedora) workstations. This means teams can access encrypted data without needing the local password when an employee leaves, simplifying handoffs and ensuring critical data remains accessible while protected. Learn more in the guide [here](https://fleetdm.com/guides/enforce-disk-encryption).

### Custom targets for OS settings

With Fleet, you can now use a new "include any" label option to target OS settings (configuration profiles) to specific hosts within a team. This added flexibility allows for finer control over which OS settings apply to which hosts, making it easier to tweak configurations without disrupting broader baselines (aka Fleet [teams](https://fleetdm.com/guides/teams#basic-article)).

### Preview scripts before run

Fleet now provides the ability to preview scripts directly on the **Host details** or **Scripts** page. This quick-view feature reduces the risk of errors by letting you verify the script is correct before running it, saving time and ensuring smoother operations.

## Changes

TODO: @noahtalerman: Update when changelog PR is opened.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.60.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2024-11-25">
<meta name="articleTitle" value="Fleet 4.60.0 | Escrow Linux disk encryption keys, custom targets for OS settings, scripts preview">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.60.0-1600x900@2x.png">