# Fleet 4.65.0 | GitOps mode, automatically install software, certificates in host vitals

Fleet 4.65.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.65.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- GitOps mode
- Automatically install software
- Certificates in host vitals

### GitOps mode

You can now put Fleet in "GitOps mode" which puts the Fleet UI in a read-only mode that prevents edits. This points users in the UI to your git repo and ensures that changes aren’t accidentally overwritten by your GitHub action or GitLab CI/CD gitops runs.

### Automatically install software

Fleet now allows IT admins to install App Store apps on all your hosts without writing custom policies. This saves time when deploying apps across many hosts, making large-scale app deployment easier and more reliable. Learn more about installing software [here](https://fleetdm.com/guides/automatic-software-install-in-fleet).

### Certificates in host vitals

The **Host details** page now displays a list of certificates for macOS, iOS, and iPadOS hosts. This helps IT teams quickly diagnose Wi-Fi or VPN connection issues by identifying missing or expired certificates that may be preventing network access. See more host vitals [here](https://fleetdm.com/vitals/battery).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.65.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-03-14">
<meta name="articleTitle" value="Fleet 4.65.0 | GitOps mode, automatically install software, certificates in host vitals">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.65.0-1600x900@2x.png">
