# Fleet 4.92.0 | Android commands, Windows Autopilot hosts, custom FileVault, and more...

<div purpose="embedded-content">
   <iframe src="TODO" title="0" allowfullscreen></iframe>
</div>

Fleet 4.92.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.92.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Android: run any command](#android-run-any-command)
- [Android: scope self-service software](#android-scope-self-service-software)
- [Fleet assignment for Windows Autopilot hosts before enrollment](#fleet-assignment-for-windows-autopilot-hosts-before-enrollment)
- [Custom macOS FileVault](#custom-macos-filevault)
- [Cancel upcoming lock/wipe commands](#cancel-upcoming-lock-wipe-commands)
- [Policy automations: resend a configuration profile](#policy-automations-resend-a-configuration-profile)
- [Filter vulnerability exposure by severity](#filter-vulnerability-exposure-by-severity)
- [Require SSO for Fleet Desktop](#require-sso-for-fleet-desktop)

### Android: run any command

IT admins can now send any command from the [Android Management API](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand) to an Android host via `fleetctl mdm run-command` or [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command), in addition to any Apple (macOS, iOS, iPadOS) or Windows command. This unlocks automations for commands Fleet doesn't have built-in UI/API action for yet, like [requesting device info](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand#CommandType.ENUM_VALUES.REQUEST_DEVICE_INFO) or [relinquishing ownership](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand#CommandType.ENUM_VALUES.RELINQUISH_OWNERSHIP).

IT admins can see Android commands results on a host's **Host details** page or via `fleetctl get mdm-commands` and `fleetctl get mdm-command-results`, the same way we already could for Apple and Windows hosts. This makes it easier to troubleshoot a command sent to an Android host, whether it came from the Fleet UI, GitOps, or a custom command sent through the API.

See an example Android command, like rebooting a host, in the [MDM commands guide](https://fleetdm.com/guides/mdm-commands#examples).

GitHub issues: [#23232](https://github.com/fleetdm/fleet/issues/23232), [#33158](https://github.com/fleetdm/fleet/issues/33158)

### Android: scope self-service software

_Available in Fleet Premium_

IT admins adding software to Android hosts' managed Google Play Store can now target hosts using labels, the same targeting options already available for macOS, Windows, and Linux software. This makes it possible to make an app available in self-service on a more specific set of Android hosts instead of every host in a fleet. Learn how to [add an Android app](https://fleetdm.com/guides/install-app-store-apps#google-play-android).

GitHub issue: [#33062](https://github.com/fleetdm/fleet/issues/33062)

Scoping an different Android app configuration to specifc hosts is [coming soon](https://github.com/fleetdm/fleet/issues/47904).

### Fleet assignment for Windows Autopilot hosts before enrollment

_Available in Fleet Premium_

Windows Autopilot now show up in Fleet as "Pending" hosts, along with their Autopilot group tag, before they ever enroll. IT admins can manually transfer a pending host to the right fleet from the **Hosts** page, or build an automation on top of Fleet's API to do it before the host ever enrolls, instead of waiting for it to land in "Unassigned" first.

GitHub issue: [#43481](https://github.com/fleetdm/fleet/issues/43481)

### Custom macOS FileVault

_Available in Fleet Premium_

IT admins can now add custom disk encryption (FileVault) settings for macOS. This makes it possible to upload a [custom `FDEFileVaultOptions` configuration profile](https://fleetdm.com/guides/custom-disk-encryption-profiles), for example, to defer FileVault until the next login, or allow a third-party tool such as Xcreds to enforce FileVault, while Fleet still escrows the recovery key.

GitHub issue: [#48654](https://github.com/fleetdm/fleet/issues/48654)

### Cancel upcoming lock/wipe commands

IT admins can now cancel a pending lock, wipe, clear passcode, or enable lost mode command on Apple (macOS, iOS, iPadOS) hosts before a host receives it, from the host's **Host details > Activity > Upcoming > MDM commands** or via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#cancel-hosts-pending-mdm-command). Even if the host runs a lock command before the cancellation reaches it, Fleet still shows the unlock PIN, once the result comes back.

Lock on Windows/Linux and wipe on Linux run as scripts, but can be canceled via **Host details > Activity > Upcoming**, or via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#cancel-hosts-upcoming-activity). Wipe on Windows and lock, wipe, and clear passcode on Android aren't cancelable yet.

GitHub issue: [#43181](https://github.com/fleetdm/fleet/issues/43181)

### Policy automations: resend a configuration profile

_Available in Fleet Premium_

IT admins can now automatically resends a configuration profile when a host fails the policy, the same way Fleet already supports automatically [running scripts](https://fleetdm.com/guides/policy-automation-run-script) and [installing software](https://fleetdm.com/guides/automatic-software-install-in-fleet). This makes it possible to build automated fixes for configuration drift, like renewing a certificate, fixing a Wi-Fi profile, or re-enforcing a CIS benchmark setting.

GitHub issue: [#40637](https://github.com/fleetdm/fleet/issues/40637)

### Filter vulnerability exposure by severity

_Available in Fleet Premium_

The vulnerability exposure chart on the dashboard can now be filtered by severity (CVSS score), giving Security Engineers more control over what the chart shows. We can also set the default severity filter for the whole organization by configuring [`cvss_min`/`cvss_max` in GitOps](https://fleetdm.com/docs/configuration/yaml-files#features).

GitHub issue: [#47326](https://github.com/fleetdm/fleet/issues/47326)

### Require SSO for Fleet Desktop

_Available in Fleet Premium_

IT admins can now require end users to sign in with SSO before they can access self-service in Fleet Desktop for macOS, Windows, Linux, or iOS/iPadOS hosts. In **Organization settings > Fleet Desktop**, turn on the new "End user authentication" setting, or set [`fleet_desktop.sso_enabled` in GitOps](https://fleetdm.com/docs/configuration/yaml-files#fleet-desktop), to add an extra layer of authentication in front of Fleet Desktop.

GitHub issue: [#47116](https://github.com/fleetdm/fleet/issues/47116)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.92.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-09-11">
<meta name="articleTitle" value="Fleet 4.92.0 | Android commands, Windows Autopilot hosts, custom FileVault, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.92.0-1600x900@2x.png">
