# Fleet 4.91.0 | Windows local admin accounts, Autopilot default fleets, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/db7vcJEueuY?si=zhxBZVJwtYURAmXw" title="0" allowfullscreen></iframe>
</div>

Fleet 4.91.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.91.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

Three of this release's highlights close gaps in Windows device management: a local admin account for troubleshooting, automatic fleet assignment for Autopilot-enrolled hosts, and clearer software inventory for Windows Store apps. Read on for those, plus updates to webhooks, patch policies, scripts, DDM, Apple Business Manager, and OS updates.

## Highlights

- [Windows: create a local admin account](#windows-create-a-local-admin-account)
- [Default fleet for Windows Autopilot-enrolled hosts](#default-fleet-for-windows-autopilot-enrolled-hosts)
- [Windows Store apps now labeled clearly in software inventory](#windows-store-apps-now-labeled-clearly-in-software-inventory)
- [Webhooks for host activities](#webhooks-for-host-activities)
- [Patch policies: install when the app is closed](#patch-policies-install-when-the-app-is-closed)
- [Built-in variables in scripts](#built-in-variables-in-scripts)
- [Custom activations for DDM profiles](#custom-activations-for-ddm-profiles)
- [Release a host from Apple Business](#release-a-host-from-apple-business)
- [Apple hardware marketing names](#apple-hardware-marketing-names)
- [OS updates: update to latest after a deadline](#os-updates-update-to-latest-after-a-deadline)

### Windows: create a local admin account

_Available in Fleet Premium_

IT admins can now have Fleet create a local admin account on Windows hosts during Autopilot enrollment, the same managed local account behavior Fleet already provides on macOS. Turn on **Create hidden admin** for Windows under **Controls > Setup experience > Users**, or set `windows_settings.managed_local_account_settings` in GitOps. Fleet creates the account, hides it from the sign-in screen, and generates a unique, escrowed password for each host. Admins, maintainers, and observers can retrieve the password from **Host details > Actions > Show managed account** when IT needs a break-glass account for troubleshooting.

GitHub issue: [#43488](https://github.com/fleetdm/fleet/issues/43488)

### Default fleet for Windows Autopilot-enrolled hosts

IT admins can now choose a default fleet for hosts that enroll through Windows Autopilot. New Autopilot-enrolled Windows hosts land in that fleet automatically, instead of sitting in "Unassigned" until an admin transfers them by hand. Only global admins can set the default, and it's available in GitOps as `mdm.windows_enrollment.default_fleet`.

GitHub issue: [#41787](https://github.com/fleetdm/fleet/issues/41787)

### Windows Store apps now labeled clearly in software inventory

Apps installed from the Windows Store, like Microsoft Copilot, already showed up in Fleet's software inventory (via osquery 5.23.1 and higher), but shared the same "Program (Windows)" type as classic `.exe` installers. Fleet now labels all Windows programs "Application (Windows)" instead, so admins checking the **Software** page, host details, or a software title's details page know they're looking at the full set of installed Windows applications, not just legacy installers.

GitHub issue: [#14717](https://github.com/fleetdm/fleet/issues/14717)

### Webhooks for host activities

_Available in Fleet Premium_

IT admins and security engineers can now send a webhook for every activity tied to a specific host, like a script run, software install, or MDM enrollment. Configure a webhook URL per fleet (or for "No fleet") from that fleet's activity automations settings, or with `webhook_settings.host_activities_webhook` in GitOps, to feed a SIEM or trigger downstream automations without waiting on the global activity feed.

GitHub issue: [#40493](https://github.com/fleetdm/fleet/issues/40493)

### Patch policies: install when the app is closed

_Available in Fleet Premium_

Patch policies for [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) can now wait until the app is closed before installing an update, instead of forcing the update while an end user is mid-task. Choose **Patch only when app is closed** when adding or editing a Fleet-maintained app's automation, and Fleet skips the install (logged as "Install skipped," not "Failed") until the app quits, then installs it on the next automation run.

GitHub issue: [#39962](https://github.com/fleetdm/fleet/issues/39962)

### Built-in variables in scripts

_Available in Fleet Premium_

Fleet's built-in `$FLEET_VAR_*` variables, like `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`, now work in shell and Python script content on macOS and Linux hosts, not just configuration profiles. IT admins can write one script that resolves a host's serial number, IdP username, or other host-specific values at run time, instead of hardcoding a value per host. Fleet validates variable references when a script is uploaded, so a script referencing a variable that doesn't exist is rejected before it ever runs.

GitHub issue: [#46837](https://github.com/fleetdm/fleet/issues/46837)

### Custom activations for DDM profiles

_Available in Fleet Premium_

IT admins doing advanced Apple declarative device management (DDM) setups can now define a custom activation, with predicates, instead of relying only on Fleet's built-in activation. This scopes a DDM configuration to specific hosts the way Apple's DDM spec supports, so admins aren't blocked on Fleet shipping UI for every new DDM feature Apple releases. Turn it on with the `mdm.allow_custom_activations` server configuration option.

Only self-managed users and customers can modify Fleet server configuration. If you're a managed-cloud customer, reach out to Fleet about turning this on.

GitHub issue: [#48222](https://github.com/fleetdm/fleet/issues/48222)

### Release a host from Apple Business

_Available in Fleet Premium_

IT admins can now release a host from Apple Business (ABM) directly from the Fleet UI or API, without signing into the ABM portal. Use the **Release from AB** action on an eligible host's details page, or release multiple hosts at once via the API. Only admins can release a host, the action is instant, and it can't be undone, so Fleet asks for confirmation first.

GitHub issue: [#47633](https://github.com/fleetdm/fleet/issues/47633)

### Apple hardware marketing names

Fleet now shows the Apple hardware marketing name, like "MacBook Pro 16-inch, M3, 2023," on the hosts list, host details, and My device pages, instead of the raw hardware model identifier (for example, "Mac15,9"). This makes it easier to tell which hardware a host is running without looking up the model identifier yourself.

GitHub issue: [#46818](https://github.com/fleetdm/fleet/issues/46818)

### OS updates: update to latest after a deadline

_Available in Fleet Premium_

IT admins managing macOS, iOS, and iPadOS OS updates can now set enforcement to always target the latest OS version, with a deadline measured in days after each release, instead of pinning to a specific version and a fixed calendar date. As Apple ships new OS versions, Fleet updates the enforced target automatically, so admins don't have to bump the minimum version by hand every time Apple releases an update.

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
