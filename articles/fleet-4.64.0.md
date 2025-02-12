# Fleet 4.64.0 | Custom targets for software, Bash scripts, Fleetctl for Linux ARM

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/JM-0PKO6xvY" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.64.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.64.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Custom targets for software
- Bash scripts
- Fleetctl for Linux ARM

### Custom targets for software

IT admins can now install App Store apps only on macOS hosts that match specific labels. This allows for precise app deployment based host attributes like operating system (OS) version, hardware type, and more, ensuring the right apps reach the right devices.

### Bash scripts

Fleet now supports running Bash scripts (`#!/bin/bash`) on macOS and Linux. IT teams can execute scripts with ["bashisms"](https://mywiki.wooledge.org/Bashism) instead of rewriting these scripts to run in Z shell (Zsh).

Also, IT admins can now edit scripts within the Fleet UI. This eliminates the need to download, modify, and re-upload scripts, making it faster to fix typos or make small adjustments on the fly.

### Fleetctl for Linux ARM

Fleet users with Linux ARM workstations can now use the fleetctl command-line interface (CLI) to run scripts, queries, and more. This expands Fleet’s CLI capabilities, allowing users to manage hosts on their preferred operating system (OS). Learn more about fleetctl [here](https://fleetdm.com/guides/fleetctl).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.64.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-02-04">
<meta name="articleTitle" value="Fleet 4.64.0 | TODO">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.64.0-1600x900@2x.png">
