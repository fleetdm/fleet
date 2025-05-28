# Fleet 4.69.0 | Bulk scripts improvements, Entra ID and authentik foreign vitals, and more...

<div purpose="[embedded-content](https://www.youtube.com/embed/KfWGkgaMEN0?si=XpL8tufModTR9Q_O)">
   <iframe src="TODO" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.69.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.69.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Bulk scripts improvements
- Entra ID and authentik foreign vitals
- Secondary CVSS scores
- Self-service software: uninstall
- Add custom packages via GitOps
- Bulk resend failed configuration profiles
- Turn off MDM on iOS/iPadOS

### Bulk scripts improvements

IT Admins can now run scripts in bulk using host filters. This makes it easy to target and take action on hundreds or more hosts without manually selecting them. Learn more about scripts [here](https://fleetdm.com/guides/scripts).

### Entra ID and authentik foreign vitals

Fleet now supports pulling user data—like IdP email, full name, and groups—from Entra ID or [authentik](https://goauthentik.io/) into host vitals. This helps IT Admins quickly identify the user assigned to each host. Lear nmore [here](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts).

### Secondary CVSS scores

When a vulnerability has no primary CVSS score in the [National Vulnerability Database (NVD)](https://nvd.nist.gov/), Fleet now shows the secondary score instead. This gives Security Engineers better visibility into potential risk and helps prioritize remediation.

### Add custom packages via GitOps

In GitOps mode, IT Admins can now use the UI to add a custom package and copy the corresponding YAML. This is useful for managing private software (like CrowdStrike) without a public URL. Learn how [here](https://fleetdm.com/guides/gitops-mode-software).

### Bulk resend failed configuration profiles

IT Admins can now see all hosts that failed to apply a configuration profile and resend it in one step. No need to visit each host’s **Host details** page to retry.

### Turn off MDM on iOS/iPadOS

IT Admins can now disable MDM directly from the host detail page. This makes managing MDM status more consistent across all Apple devices in your fleet.

## Changes

### Security Engineers

TODO

### IT Admins

TODO

### Other improvements and bug fixes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.69.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-06-11">
<meta name="articleTitle" value="Fleet 4.69.0 | ">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.69.0-1600x900@2x.png">
