# Fleet 4.83.0 | Recovery Lock passwords, patch policies, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/DxCqKE8tNyU?si=CKZm_FL0E4UjJf1T" title="0" allowfullscreen></iframe>
</div>

Fleet 4.83.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.83.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- macOS Recovery Lock passwords
- Patch policies for Fleet-maintained apps
- Lock end user info during macOS setup
- YAML validation for extraneous keys

### macOS Recovery Lock passwords

Fleet now automatically escrows a unique Recovery Lock password for each macOS host and lets admins rotate it on demand. Learn [how to to enable](https://fleetdm.com/guides/recovery-lock-password).

The Recovery Lock passwords prevents unauthorized access to macOS Recovery Mode. When needed, admins can look up and share the password with an end user and then rotate it afterward so it can't be reused. Automatic rotation after view is [coming soon](https://github.com/fleetdm/fleet/issues/41003).

GitHub issues: [#37497](https://github.com/fleetdm/fleet/issues/37497), [#37498](https://github.com/fleetdm/fleet/issues/37498)

### Patch policies for Fleet-maintained apps

Fleet now supports patch policies for Fleet-maintained apps (FMAs). Unlike traditional policies where admins write and maintain the latest version in the SQL themselves, a patch policy automatically generates and updates the SQL when a new version of the app is released. This removes the maintenance burden of keeping patch policies in sync with new software versions. [Learn more](TODO).

GitHub issue: [#31914](https://github.com/fleetdm/fleet/issues/31914)

### Lock end user info during macOS setup

Fleet now lets IT admins control whether end users can edit their macOS local account "Full Name" and "Account Name" during the Setup Assistant (out-of-box enrollment flow). When **Lock end user info** is enabled, end users cannot modify these fields during setup.

To configure, head to **Controls > Setup experience** and expand the new **Advanced options** section. The **Lock end user info** option is only available when end user authentication is turned on. This setting is also supported via GitOps using the `controls.setup_experience.lock_end_user_info` key.

GitHub issue: [#38669](https://github.com/fleetdm/fleet/issues/38669)

### YAML validation for extraneous keys

Fleet now returns a clear error when a YAML file contains an unrecognized or misspelled key. Previously, Fleet silently ignored unknown keys, which could cause configurations to take effect without the intended settings applied. This is especially useful for catching typos and errors in AI-generated GitOps PRs early, before a misconfiguration silently takes effect.

GitHub issue: [#40496](https://github.com/fleetdm/fleet/issues/40496)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.83.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-03-31">
<meta name="articleTitle" value="Fleet 4.83.0 | Recovery Lock passwords, patch policies, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.83.0-1600x900@2x.png">
