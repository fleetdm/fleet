# Fleet 4.68.0 | Scheduled query webhooks, deploy tarballs, SHA-256 verification, and more...

<div purpose="embedded-content">
   <iframe src="TODO" frameborder="0" allowfullscreen></iframe>
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

### Custom Fleet agent (fleetd) during new Mac setup (ADE)

Fleet now allows IT admins to deploy a custom fleetd during Mac Setup Assistant (ADE). This makes it possible to custom the fleetd configuration to point hosts to a custom Fleet server URL during initial enrollment, meeting security requirements without manual reconfiguration. Learn how [here](https://fleetdm.com/guides/macos-setup-experience#advanced).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.68.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-05-07">
<meta name="articleTitle" value="Fleet 4.68.0 | TODO">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.67.0-1600x900@2x.png">