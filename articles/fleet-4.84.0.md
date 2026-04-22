# Fleet 4.84.0 | Python scripts, Entra for Windows, auto-rotate Recovery Lock, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/tNi7AcMH_sk?si=YBDsDvKPc4H3cbg7" title="0" allowfullscreen></iframe>
</div>

Fleet 4.84.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.84.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Automatically rotate Recovery Lock passwords
- Run Python scripts on macOS & Linux
- Entra conditional access for Windows
- Remove settings from Windows when profile is deleted
- GitOps mode exceptions

### Automatically rotate Recovery Lock passwords

Fleet now automatically rotates macOS Recovery Lock passwords after an IT admin views them. Previously, Fleet escrowed a unique password per host and let IT admins rotate it on demand — but rotation was a manual step. Now, after a password is viewed, Fleet schedules an automatic rotation (1 hr after view) so passwords aren't reused.

Admins can still trigger a manual rotation at any time from the Host details page. The rotation generates an audit log entry so the action is traceable.

GitHub issue: [#41003](https://github.com/fleetdm/fleet/issues/41003)

### Run Python scripts on macOS & Linux

Fleet now supports Python scripts alongside shell (`.sh`) and PowerShell (`.ps1`) scripts. IT admins can upload `.py` files in **Controls > Scripts** and run them on macOS and Linux hosts — on demand, in bulk, or as a policy automation.

Python scripts follow the same rules as other script types: they respect Fleet's [script timeout](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout), support [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles), and can be defined in GitOps.

GitHub issue: [#38793](https://github.com/fleetdm/fleet/issues/38793)

### Entra conditional access for Windows

Fleet now supports [Microsoft Entra conditional access](TODO) for Windows hosts. IT admins can mark policies as conditional access policies targeting Windows hosts — when a host fails one of those policies, Entra blocks the end user from accessing corporate resources such as Microsoft Teams and Office.

This extends the existing macOS conditional access integration to Windows, using the same Fleet + Entra setup. To configure, head to **Integrations > Conditional access** and enable conditional access on any policy targeting Windows.

GitHub issue: [#38041](https://github.com/fleetdm/fleet/issues/38041)

### Remove settings from Windows when profile is deleted

When an IT admin deletes a Windows configuration profile in Fleet, Fleet now actively removes those settings from enrolled hosts. This ensures that hosts match the intended configuration state regardless of when they enrolled.

Previously, deleting a profile only prevented it from being applied to newly enrolled hosts. Existing hosts retained the settings silently. Now, Fleet sends a removal command so the configuration is reverted on all affected hosts.

GitHub issue: [#33418](https://github.com/fleetdm/fleet/issues/33418)

### GitOps mode exceptions

Fleet now lets IT admins opt specific resources out of GitOps enforcement. When GitOps mode is enabled, admins can configure exceptions for software, labels, and enroll secrets — allowing those resources to be managed via the UI or API instead of git.

This makes it easier to ramp up with GitOps incrementally: start by managing policies and profiles in git, then add software and labels later as the team gets comfortable. Exceptions are configured per resource type and require global admin permissions. If an exception is enabled and the corresponding key is present in a YAML file, GitOps will surface a clear error during the dry run to prevent the UI-managed changes from being silently overwritten.

Heads up that after upgrading, existing Fleet instances will have the labels exception enabled automatically. This way, your next GitOps run after upgrade doesn't wipe any labels not defined in git. 

If your GitOps YAML files include a `labels:` key, you will encounter new errors. To resolve, either remove `labels:` from your YAML files (to manage labels via the UI or API going forward) or disable the labels exception in **Settings > Integrations > GitOps** (to manage labels via GitOps). If you disable the exception, make sure you move any labels managed via the UI into your YAML, otherwise your next GitOps run will wipe them out. Feel free to [reach out to Fleet](https://fleetdm.com/support) if you need a hand.

GitHub issue: [#40171](https://github.com/fleetdm/fleet/issues/40171)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.84.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-04-22">
<meta name="articleTitle" value="Fleet 4.84.0 | Python scripts, Entra for Windows, auto-rotate Recovery Lock, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.84.0-1600x900@2x.png">
