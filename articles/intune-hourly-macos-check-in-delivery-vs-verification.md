# An hourly Intune check-in tells you the Mac got the change, not that the change stuck

*Microsoft cut the Intune management agent's routine macOS check-in from about eight hours to one. That closes the delivery gap. Whether the change landed is a separate question.*

## Key takeaways

- **Intune's macOS agent now checks in about every hour.** With service release 2611, Microsoft moved the regular check-in for the Intune management agent from about eight hours to one, and tightened the schedule during onboarding.
- **Apps and scripts reach Macs sooner, with no admin action.** Microsoft says the change helps newly assigned apps, scripts, and dynamic group updates arrive faster. The service controls the cadence.
- **Microsoft frames it around newly enrolled Macs.** The announcement names newly enrolled devices, and some Intune docs still describe the eight-hour cycle. Test what your existing Macs do before you change your expectations.
- **Delivery is not the same as applied.** Microsoft notes that Intune reports can lag the device, and that Intune doesn't evaluate the payload of custom Apple configuration profiles. A faster check-in doesn't change either.
- **The answer has to come from the Mac.** Fleet's agent reads state directly from each host, verifies macOS configuration profiles after install, and keeps checking policies on a schedule you set.
- **Faster checks are worth more than faster pushes.** An hourly delivery is useful. An hourly answer about which Macs are out of line, with an automation attached, is what lets a team act on it.

<a purpose="cta-button" href="https://fleetdm.com/device-management">See how Fleet verifies Mac settings</a>

Microsoft's Intune [service release 2611](https://learn.microsoft.com/en-us/intune/whats-new/), for the week of November 16, changes how often the Intune management agent checks in on macOS. During onboarding, the agent now checks every three minutes for 15 minutes, then every 15 minutes for two hours. After that, the regular cadence moves from about eight hours to about one. [HTMD](https://www.anoopcnair.com/microsoft-intune-increases-macos-agent-check-in/) covered the change on November 18.

For Mac admins who have waited most of a workday for a script to run, that's a real improvement. It's also worth being precise about what it fixes. A check-in is how a change gets to a Mac. It isn't a record of what the Mac did with it.

## What changed and what didn't

Microsoft's own wording is specific. "Service-controlled hourly check-ins help deliver newly assigned apps, scripts, and dynamic-group updates sooner, without admin or user action." That covers the work the Intune management agent does on macOS, such as shell scripts and apps.

Two details are worth checking in your own tenant. The announcement describes the change for "newly enrolled macOS devices," then says the regular cadence changes "after enrollment." It doesn't say outright whether Macs enrolled before release 2611 get the hourly schedule. Microsoft's [policy refresh table](https://learn.microsoft.com/en-us/intune/device-configuration/troubleshoot-device-profiles#policy-refresh-intervals) now lists "around every hour" for newly enrolled macOS devices. Its [shell scripts article](https://learn.microsoft.com/en-us/intune/device-management/tools/run-shell-scripts-macos) still describes an eight-hour check-in. Until the docs agree, assume your existing Macs may behave differently from new ones, and test it.

## Delivered is not the same as applied

Faster delivery shortens the wait between assigning a change and the Mac receiving it. It doesn't settle whether the change did what you meant.

Microsoft says as much in its own documentation. It can take more time for Intune reports to reflect a policy's latest status on the device. For custom macOS configuration profiles, Microsoft writes that Intune "doesn't evaluate the payload of Apple Configuration files" and "merely serves as the delivery mechanism." If two profiles conflict, Apple decides which setting applies.

There's also everything that happens after delivery. A script can exit cleanly and leave the setting it was meant to fix unchanged. An app can install and then be removed by the user. A setting can be correct on Monday and different on Thursday. An hourly check-in delivers the next change sooner. It doesn't watch the last one.

## Checking state on the Mac itself

Fleet approaches the question from the device side. Fleet's agent reads state directly on each Mac, so a report shows what's true on that host, not what was sent to it.

For macOS configuration profiles, Fleet doesn't stop at "sent." After a profile installs, Fleet's agent checks that it's present on the host. The profile moves from "Verifying" to "Verified" within about an hour by default, and that interval is [configurable](https://fleetdm.com/docs/configuration/fleet-server-configuration#osquery-detail-update-interval). Declarative device management (DDM) declarations are verified from the Mac's own status report. Fleet's [OS settings guide](https://fleetdm.com/guides/custom-os-settings) covers each status.

Profile verification confirms the profile is installed, not that every setting inside it won a conflict. For settings that matter, a policy checks the outcome directly. This one, from Fleet's built-in policy library, passes only when FileVault is on:

```sql
SELECT 1 FROM disk_encryption WHERE filevault_status = 'on' LIMIT 1;
```

Fleet runs policies every hour by default, and you can [change that interval](https://fleetdm.com/docs/configuration/fleet-server-configuration#osquery-policy-update-interval). When you need an answer right now, a live report returns results from online hosts as they respond. When a Mac starts failing a policy, an automation can open a ticket, send a webhook, or run a [script](https://fleetdm.com/guides/scripts) to fix it. Fleet shows each script's output and exit code per host, so a script that ran but didn't help is visible too.

## Fast delivery needs a fast answer

Microsoft's change is a welcome one for Intune shops, and it narrows a gap Mac admins have complained about for years. It moves the bottleneck rather than removing it. When delivery takes an hour instead of eight, the slow part becomes finding out which Macs didn't end up where you intended.

That answer comes from the device. An hourly check that reports state, paired with a policy and an automation, turns a faster push into a fleet you can trust.

## See it live

- **Get a demo** to see verified macOS settings across your own Macs: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet delivers and verifies OS settings:** [fleetdm.com/guides/custom-os-settings](https://fleetdm.com/guides/custom-os-settings)

## Sources

- Microsoft Learn, [What's new in Microsoft Intune](https://learn.microsoft.com/en-us/intune/whats-new/), week of November 16, 2026 (service release 2611).
- Microsoft Learn, [Questions with policies and profiles in Microsoft Intune](https://learn.microsoft.com/en-us/intune/device-configuration/troubleshoot-device-profiles).
- Microsoft Learn, [Use shell scripts on macOS devices in Microsoft Intune](https://learn.microsoft.com/en-us/intune/device-management/tools/run-shell-scripts-macos).
- HTMD, [Microsoft Intune increases macOS agent check-in frequency for faster app and script delivery](https://www.anoopcnair.com/microsoft-intune-increases-macos-agent-check-in/).

<meta name="articleTitle" value="An hourly Intune check-in tells you the Mac got the change, not that the change stuck">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-19">
<meta name="description" value="Intune's macOS agent now checks in about hourly, so changes reach Macs faster. Here's how to confirm they were applied and stayed applied.">
