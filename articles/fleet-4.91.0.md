# Fleet 4.91.0 | Windows local admin accounts, Autopilot default fleets, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/db7vcJEueuY?si=zhxBZVJwtYURAmXw" title="0" allowfullscreen></iframe>
</div>

Fleet 4.91.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.91.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

Three of this release's highlights close gaps in Windows device management: a local admin account for troubleshooting, automatic fleet assignment for Autopilot-enrolled hosts, and clearer software inventory for Windows Store apps. Read on for those, plus updates to webhooks, patch policies, scripts, DDM, Apple Business Manager, and OS updates.

## Highlights

- [Windows: create a local admin account](#windows-create-a-local-admin-account)
- [Default fleet for Windows Autopilot-enrolled hosts](#default-fleet-for-windows-autopilot-enrolled-hosts)
- [Webhooks for host activities](#webhooks-for-host-activities)
- [Patch policies: install when the app is closed](#patch-policies-install-when-the-app-is-closed)
- [Built-in variables in scripts](#built-in-variables-in-scripts)
- [Release a host from Apple Business](#release-a-host-from-apple-business)
- [Apple hardware marketing names](#apple-hardware-marketing-names)
- [OS updates: update to latest after a deadline](#os-updates-update-to-latest-after-a-deadline)

### Windows: create a local admin account

_Available in Fleet Premium_

IT admins can now create managed local admin accounts on Windows hosts. Turn on **Create hidden admin** for Windows under **Controls > Setup experience > Users**, or set [`windows_settings.managed_local_account_settings` in GitOps](TODO: https://fleetdm.slack.com/archives/C0AQY8D7FM4/p1787780988221729?thread_ts=1787775061.407319&cid=C0AQY8D7FM4). Fleet creates the account, hides it from the sign-in screen, and generates a unique, escrowed password for each host. All roles in Fleet can retrieve the password from **Host details > Actions > Show managed account** when IT needs a break-glass account for troubleshooting.

On macOS, managed accounts are created only at enrollment time, and only for hosts that automatically enroll via Apple Business Manager. On Windows, they're created on every host with MDM turned on, including hosts that enrolled earlier.

GitHub issue: [#43488](https://github.com/fleetdm/fleet/issues/43488)

### Default fleet for Windows Autopilot-enrolled hosts

IT admins can now choose a default fleet for hosts that enroll through [Windows Autopilot](). New Autopilot-enrolled Windows hosts land in that fleet automatically, instead of sitting in "Unassigned" until an admin transfers them by hand. Only global admins can set the default, and it's available in [GitOps as `mdm.windows_enrollment.default_fleet`](TODO: https://fleetdm.slack.com/archives/C0AQY8D7FM4/p1787783001510159?thread_ts=1787775061.407319&cid=C0AQY8D7FM4).

GitHub issue: [#41787](https://github.com/fleetdm/fleet/issues/41787)

### Webhooks for host activities

_Available in Fleet Premium_

IT Admins and Security Engineers can now send a webhook for every activity tied to a specific host, like a script run, software install, or configuration profile install. Configure a webhook URL per fleet from that fleet's **Hosts** page, or with [`webhook_settings.host_activities_webhook` in GitOps](https://fleetdm.com/docs/configuration/yaml-files#host-activities-webhook), to feed a [SIEM](https://www.gartner.com/reviews/market/security-information-event-management) or trigger automations in a third-party automation tool (e.g. [Tines](https://www.tines.com/))

GitHub issue: [#40493](https://github.com/fleetdm/fleet/issues/40493)

### Patch policies: install when the app is closed

_Available in Fleet Premium_

Patch policies for [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) can now wait until the app is closed before installing an update, instead of forcing the update while an end user is mid-task. Choose **Patch only when app is closed** when adding or editing a Fleet-maintained app, and Fleet skips the install (logged as "Install skipped") until the app quits, then installs it on the next automation run.

GitHub issue: [#39962](https://github.com/fleetdm/fleet/issues/39962)

### Built-in variables in scripts

_Available in Fleet Premium_

Fleet's [built-in variables](https://fleetdm.com/guides/fleet-variables), like `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`, now work in shell, PowerShell, and Python script content on macOS, Windows, Linux hosts, not just configuration profiles. IT Admins can write one script that inserts a host's serial number, IdP username, or other host-specific values, instead of hardcoding a value per host. Fleet validates variable references when a script is uploaded, so a script referencing a variable that doesn't exist is rejected before it ever runs.

GitHub issue: [#46837](https://github.com/fleetdm/fleet/issues/46837)

### Release a host from Apple Business

_Available in Fleet Premium_

IT admins can now release a host from Apple Business (ABM) directly from the Fleet UI or API, without signing into the ABM portal. Use the **Release from Apple Business** action on an eligible host's details page, or release multiple hosts at once via the API. Only Fleet users with [admin role](https://fleetdm.com/guides/role-based-access) can release a host.

GitHub issue: [#47633](https://github.com/fleetdm/fleet/issues/47633)

### Apple hardware marketing names

Fleet now shows the Apple hardware marketing name, like "MacBook Pro 16-inch, M3, 2023," on the **Hosts**, **Host details**, and **Fleet Desktop > My device** pages, instead of the raw hardware model identifier (for example, "Mac15,9"). This makes it easier to tell which hardware a host is running without looking up the model identifier yourself.

GitHub issue: [#46818](https://github.com/fleetdm/fleet/issues/46818)

### OS updates: update to latest after a deadline

_Available in Fleet Premium_

IT Admins managing macOS, iOS, and iPadOS OS updates can now set enforcement to always target the latest OS version, with a deadline measured in days after each release, instead of pinning to a specific version and a fixed calendar date. As Apple ships new OS versions, Fleet updates the enforced target automatically, so admins don't have to bump the minimum version by hand every time Apple releases an update.

GitHub issue: [#39085](https://github.com/fleetdm/fleet/issues/39085)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.91.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-08-28">
<meta name="articleTitle" value="Fleet 4.91.0 | Windows local admin accounts, Autopilot default fleets, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.91.0-1600x900@2x.png">
