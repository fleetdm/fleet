# Fleet 4.92.0 | Android MDM commands, Windows Autopilot pending hosts, and more...

<div purpose="embedded-content">
   <iframe src="TODO" title="0" allowfullscreen></iframe>
</div>

Fleet 4.92.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.92.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Android: run any command](#android-run-any-command)
- [Android: scope self-service software with labels](#android-scope-self-service-software-with-labels)
- [Windows Autopilot devices appear as pending hosts](#windows-autopilot-devices-appear-as-pending-hosts)
- [Require SSO for Fleet Desktop](#require-sso-for-fleet-desktop)
- [Cancel upcoming Apple commands](#cancel-upcoming-apple-commands)
- [Policy automations: resend a configuration profile](#policy-automations-resend-a-configuration-profile)
- [Disk encryption: per-platform escrow configurations](#disk-encryption-per-platform-escrow-configurations)
- [Filter vulnerability exposure by severity](#filter-vulnerability-exposure-by-severity)

### Android: run any command

IT admins can now send any command from the [Android Management API](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand) to an Android host via `fleetctl mdm run-command` or [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command), in addition to any Apple (macOS, iOS, iPadOS) or Windows command. This unlocks automations for commands Fleet doesn't have built-in UI/API action for yet, like [requesting device info](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand#CommandType.ENUM_VALUES.REQUEST_DEVICE_INFO) or [relinquishing ownership](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand#CommandType.ENUM_VALUES.RELINQUISH_OWNERSHIP).

IT admins can see Android commands results on a host's **Host details** page or via `fleetctl get mdm-commands` and `fleetctl get mdm-command-results`, the same way we already could for Apple and Windows hosts. This makes it easier to troubleshoot a command sent to an Android host, whether it came from the Fleet UI, GitOps, or a custom command sent through the API.

See an example Android command, like rebooting a host, in the [MDM commands guide](https://fleetdm.com/guides/mdm-commands#examples).

GitHub issues: [#23232](https://github.com/fleetdm/fleet/issues/23232), [#33158](https://github.com/fleetdm/fleet/issues/33158)

### Android: scope self-service software with labels

IT admins adding software to Android hosts' managed Google Play Store can now target hosts using labels, the same targeting options already available for macOS, Windows, and Linux software. This makes it possible to make an app available in self-service on a more specific set of Android hosts instead of every host in a fleet. Learn how to [add an Android app](https://fleetdm.com/guides/install-app-store-apps#google-play-android).

GitHub issue: [#33062](https://github.com/fleetdm/fleet/issues/33062)

Scoping an different Android app configuration to specifc hosts is [coming soon](https://github.com/fleetdm/fleet/issues/47904).

### Windows Autopilot devices appear as pending hosts

_Available in Fleet Premium_

Windows devices registered in a connected Microsoft Entra tenant's Autopilot registry now show up in Fleet as pending hosts, along with their Autopilot group tag, before they ever enroll. IT admins can build an automation on top of Fleet's API to transfer a pending host to the right fleet before it enrolls, instead of waiting for it to land in "Unassigned" first.

GitHub issue: [#43481](https://github.com/fleetdm/fleet/issues/43481)

### Require SSO for Fleet Desktop

_Available in Fleet Premium_

IT admins can now require end users to sign in with SSO before they can access Fleet Desktop's My device page on macOS, Windows, Linux, or iOS/iPadOS hosts. Turn on the new "End user authentication" setting, or set `fleet_desktop.sso_enabled` in GitOps, to add an extra layer of authentication in front of Fleet Desktop.

GitHub issue: [#47116](https://github.com/fleetdm/fleet/issues/47116)

### Cancel upcoming Apple commands

IT admins can now cancel a pending Apple MDM lock, wipe, clear passcode, or enable lost mode command before a host receives it, from the host's Actions menu or via `DELETE /api/v1/fleet/hosts/:id/commands/:command_uuid`. If the host runs the command anyway before the cancellation reaches it, Fleet restores the host's lock or wipe state, including the unlock PIN, once the result comes back.

GitHub issue: [#43181](https://github.com/fleetdm/fleet/issues/43181)

### Policy automations: resend a configuration profile

_Available in Fleet Premium_

IT admins can now configure a policy automation that resends a configuration profile when a host fails the policy, the same way Fleet already supports resending scripts and software. This makes it possible to build automated fixes for configuration drift, like renewing a certificate, fixing a Wi-Fi profile, or re-enforcing a CIS benchmark setting.

GitHub issue: [#40637](https://github.com/fleetdm/fleet/issues/40637)

### Disk encryption: per-platform escrow configurations

_Available in Fleet Premium_

IT admins can now configure disk encryption enforcement and key escrow separately for macOS, Windows, and Linux, instead of one setting that applies to every platform. This makes it possible to use a custom setup, like a third-party tool such as Xcreds to handle FileVault, while Fleet still escrows the recovery key.

GitHub issue: [#48654](https://github.com/fleetdm/fleet/issues/48654)

### Filter vulnerability exposure by severity

_Available in Fleet Premium_

The vulnerability exposure chart on the dashboard can now be filtered by severity (CVSS score), giving security engineers more control over what the chart shows. The severity filter on the Software, Host details, and My device pages also only shows the min and max score inputs when custom severity is selected, so the filter stays easy to read at a glance.

GitHub issue: [#47326](https://github.com/fleetdm/fleet/issues/47326)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.92.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-09-11">
<meta name="articleTitle" value="Fleet 4.92.0 | Android MDM commands, Windows Autopilot pending hosts, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.92.0-1600x900@2x.png">
