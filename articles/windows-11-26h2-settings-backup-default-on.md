# Windows settings backup is on by default in 26H2. Here's what to check next.

*Windows settings backup is now on by default for eligible devices. Whether yours qualify depends on a policy you may never have set.*

## Key takeaways

- **The default flipped on October 1.** Windows 11 version 26H2 shipped on September 29, and on October 1 Microsoft began enabling Windows settings backup and restore by default for eligible managed devices. A device that was quiet last month can be sending data to Microsoft's cloud this month.
- **What leaves the device is narrow, and it still leaves.** Microsoft describes the backup as Windows settings and the list of installed Microsoft Store apps. That is not a file backup, but it is configuration data about your users, and it now moves without a ticket.
- **"Not Configured" is now a decision.** The default applies only when nobody set the backup policy. A policy you never touched is the one that changed meaning.
- **Eligibility has more than one condition.** Reports describe a device on 26H2, outside Digital Markets Act regions and sovereign or restricted clouds, joined to Microsoft Entra ID, with the policy left unset. Each condition can differ from one device to the next.
- **Intune and Group Policy settings still win.** An explicit enable or disable overrides the default, so the fix for an unwanted default is a deliberate setting, not a hope.
- **Explicit settings still win, and Fleet can set them.** An explicit enable or disable overrides the default. Fleet can deploy that setting as a Windows configuration profile, report which devices picked up 26H2, and trigger an automation when one is missing the setting.

<a purpose="cta-button" href="https://fleetdm.com/device-management">See how Fleet reports Windows device state</a>

Microsoft released Windows 11 version 26H2 on September 29. Two days later, on October 1, Windows settings backup and restore became a default instead of an opt-in feature for eligible organizational devices. [BleepingComputer](https://www.bleepingcomputer.com/news/microsoft/microsoft-enables-windows-settings-backup-by-default-for-orgs/) and [Petri](https://petri.com/microsoft-enables-windows-settings-backup-by-default/) both cover the change.

For IT teams, the useful question is not whether backup is good. It's which of your devices will start doing it, and whether that matches what your organization decided.

## What the default does

Microsoft says the backup covers Windows settings and the list of Microsoft Store apps, so users can restore them after a reset, a replacement, or a reimage. Microsoft's [message center post MC1483538](https://mc.merill.net/message/MC1483538) describes the change for eligible devices. Files aren't part of that description.

That's a smaller footprint than a full device backup. It is still data about how your users configure their machines, stored in a Microsoft cloud service, and a compliance reviewer will ask whether you knew.

## The conditions that decide it

Based on the reporting, the default switches on only when a device meets all of these:

1. It runs Windows 11 version 26H2 or later.
2. It's outside regions regulated by the EU Digital Markets Act.
3. It isn't in a sovereign or restricted cloud environment.
4. The backup policy is still Not Configured.
5. It is joined to Microsoft Entra ID, or hybrid joined.

Devices that started on Windows 11 version 26H1 aren't exempt either. Microsoft says they get the same default-on treatment when they move to a later supported feature release.

## What your management tools tell you

Intune and Group Policy can both set the policy explicitly, and an explicit setting takes precedence over the default. That's your opt-out, or your deliberate opt-in.

None of these tools answers a different question on its own: which devices are on 26H2 right now, and which of them are running without an explicit setting? A policy assigned to a group covers only the devices in that group. A device that hasn't checked in yet hasn't received anything. A machine that moved to 26H2 earlier than your rollout plan expected is the one most likely to surprise you.

## Checking the device itself

Fleet's agent reports the exact Windows build and configuration from each host. That lets you build a list of devices on 26H2 and compare it against the ones where your policy is present. Reports can run on a schedule, so the list stays current as devices update.

Where the backup policy is stored on a device is something to confirm on a test machine before you write a check. Set the policy through Intune or Group Policy on one device, find where the value lands, and then query for that location across your fleet. A device with no value there, on 26H2, is a candidate for the new default.

Once you know what to look for, a Fleet policy can keep asking. Policy automations can send a webhook, open a ticket, or run a script when a host starts failing, so a 26H2 device without an explicit setting gets flagged to the right team instead of waiting for someone to check a dashboard. And because policies and profiles can live in Git as [YAML](https://fleetdm.com/docs/configuration/yaml-files), updating the check for the next feature release is a pull request someone reviews, not a console edit someone has to remember.

## Where this leaves you

The easy response is to decide on purpose. If you want settings backup, set it, so the behavior is yours instead of a default. If you don't, set the opposite, and confirm it landed on every device that matters.

Either way, a default changed under a setting you never configured. The teams that find out from a device report this week will have a much shorter conversation than the ones that find out during a review.

## See it live

- **Get a demo** to see Windows device state reported across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet handles device management:** [fleetdm.com/device-management](https://fleetdm.com/device-management)
## More from Fleet

- [WSL containers are now on Windows. Can you see them?](https://fleetdm.com/industry-news/wsl-containers-registry-allow-list-visibility)
- [Creating Windows CSPs](https://fleetdm.com/guides/creating-windows-csps)

## Sources

- BleepingComputer, [Microsoft enables Windows settings backup by default for orgs](https://www.bleepingcomputer.com/news/microsoft/microsoft-enables-windows-settings-backup-by-default-for-orgs/).
- Petri, [Microsoft enables Windows settings backup by default in Windows 11 version 26H2](https://petri.com/microsoft-enables-windows-settings-backup-by-default/).
- heise online, [Windows settings backup becomes default in Windows 11 26H2](https://www.heise.de/en/news/Windows-settings-backup-becomes-default-in-Windows-11-26H2-11472934.html).
- Microsoft 365 message center, [MC1483538](https://mc.merill.net/message/MC1483538).

<meta name="articleTitle" value="Windows settings backup is on by default in 26H2. Here's what to check next.">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-02">
<meta name="description" value="Windows 11 26H2 turns settings backup on by default for eligible devices. See which conditions apply and how to check which of your PCs picked it up.">
