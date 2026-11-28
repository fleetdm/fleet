# Don't put users on hold: show Nudge when the update is ready

*Nudge is great at getting people to update macOS. It's less great when the person who clicks **Update** then waits 40 minutes for a download. Here's how to fix that by letting declarative device management do the waiting.*

## Key takeaways

- **Ask for an update only when it's ready to install.** Nudge appears after macOS has downloaded and prepared the update, so clicking **Update** leads to a quick restart instead of a long wait.
- **Each tool does the job it's best at.** Declarative device management (DDM) downloads and prepares updates quietly in the background. Nudge handles the reminders, deferrals, and deadlines people respond to.
- **One status item and one activation do the gating.** A simple activation watches `softwareupdate.install-state` and installs Nudge's settings only when the value is `prepared`.
- **Fleet manages the whole setup as code.** Since Fleet 4.91.0, you can attach a custom activation to a declaration in GitOps, review it in a pull request, and roll it back the same way.
- **Plan for the Macs the gate can't open.** Unsupervised Macs don't report the install state, and major upgrades aren't staged on their own, so those Macs need a fallback.
- **The time adds up fast.** Saving 30 minutes per update, eight or so times a year, across thousands of people is real time back.

<a purpose="cta-button" href="https://fleetdm.com/guides/show-nudge-only-after-macos-prepares-an-update">Read the step-by-step guide</a>

At MacDevOpsYUL 2026, Wes Whetstone from Stripe gave a seven-minute lightning talk called [But you called me?](https://www.youtube.com/watch?v=SggxxPCDCSI). Wes _famously_ moved 10,000 Macs to Fleet, and the talk describes a small change with a big effect on how those Macs get updated.

We liked the idea so much that we rolled it out on Fleet's own Macs. This article walks through the problem, Wes's approach, and what we learned putting it into production. If you want to set it up yourself, the [companion guide](https://fleetdm.com/guides/show-nudge-only-after-macos-prepares-an-update) has the steps.

## "But you called me?"

The talk's title comes from a message Wes received from a senior leader at Stripe. Nudge had asked them to update macOS. They clicked **Update**, and Software Update started a download that took around 40 minutes. Their reaction: you called me, so why are you putting me on hold?

It's a fair point. Nudge exists to get people to act, and it works. But when the action it asks for starts with a long download, the person who did the right thing pays for it. Multiply that by every Mac and every update, and the cost is significant.

## Why Nudge can't fix it alone

Nudge can download updates ahead of time, but as Wes found, that only works on Intel Macs. On Apple silicon, installing an update needs a volume owner's credentials, so Nudge can't stage the update for the user.

Declarative device management (DDM) doesn't have that problem. With DDM, macOS downloads and prepares the update itself, in the background, with nothing for the user to do. The Mac also reports the update's progress back to your MDM, so you can tell when it's ready to install. That makes DDM the right tool for the slow part of an update.

## Let each tool do what it's good at

Where DDM falls short is getting people to restart. Its reminders are polite and easy to ignore, while Nudge is persistent by design.

So Wes blended the two:

1. DDM downloads and prepares the update.
2. Software Update's own notifications are turned off (`Notifications: false` in the `com.apple.configuration.softwareupdate.settings` declaration), so Nudge is the only thing users see.
3. Nudge appears only once the update is ready to install.

Wes also shared a warning from experience. If you send an enforcement declaration to a large fleet all at once, every Mac starts downloading the update when it receives it. On a small network with a lot of Macs, that hurts, and a caching server isn't always an easy fix. Wes suggests rolling it out in waves.

## The trigger: a status item and an activation

The clever part is how Nudge knows the update is ready. DDM reports a status item called `softwareupdate.install-state`. Its values are `none`, `downloading`, `prepared`, `installing`, and `failed`. When it's `prepared`, the update is downloaded and ready to install.

DDM activations can include a predicate that's evaluated on the Mac, so an activation can say "install this configuration only when `@status(softwareupdate.install-state) == 'prepared'`." Nudge reads its settings from a configuration profile, not a declaration, so Wes wraps that profile in a legacy profile declaration (`com.apple.configuration.legacy`). When the update is prepared, the activation installs the legacy declaration. The Mac downloads Nudge's profile, and Nudge starts enforcing the new version. Once the update installs, the install state changes, and the profile comes off again.

Fleet added support for custom activations in [Fleet 4.91.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.91.0). In GitOps, you add an `activation:` key next to the declaration's `path:`. When the predicate is false, Fleet shows the declaration as **Verified** with a note that the predicate evaluated to false, so a Mac with nothing staged doesn't look like a failure.

## What we changed for Fleet's own Macs

Fleet runs its own Macs with Fleet, and everything is in [our GitOps repository](https://github.com/fleetdm/fleet/tree/main/it-and-security). Putting Wes's approach into production taught us a few things worth sharing.

### The files

- [`Nudge settings.json`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/declaration-profiles/Nudge%20settings.json) is the legacy profile declaration.
- [`Nudge activation.json`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/declaration-profiles/Nudge%20activation.json) is the activation with the `prepared` predicate.
- [`nudge-preferences.mobileconfig`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-preferences.mobileconfig) holds Nudge's settings. The Mac downloads it when the activation fires.
- [`nudge-background-task.mobileconfig`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-background-task.mobileconfig) keeps Nudge's launch agent a managed login item at all times.
- [`nudge-preferences-ungated.mobileconfig`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-preferences-ungated.mobileconfig) is the fallback for Macs the gate can't open.
- [`Software Update settings.json`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/declaration-profiles/Software%20Update%20settings.json) turns off Software Update's notifications and keeps automatic downloads on.
- [`workstations.yml`](https://github.com/fleetdm/fleet/blob/main/it-and-security/fleets/workstations.yml) ties it all together.

The full change, with the reasoning behind each decision, is in [pull request #51546](https://github.com/fleetdm/fleet/pull/51546).

### Keep the login item out of the gated profile

Our original Nudge profile also held the `com.apple.servicemanagement` rule that keeps Nudge's launch agent from being turned off. If that rule lived in the gated profile, it would be installed and removed with every update. Users would see a "Background Items Added" notification each time and could turn Nudge off in between. We moved it to its own profile that's always installed.

### Host the profile somewhere immutable

A legacy profile declaration points at a URL. Fleet doesn't serve mobileconfig files, so we host ours on GitHub, pinned to a commit. Macs only download the profile again when the declaration changes, so a URL that points at `main` would mean settings changes never reach Macs. With a pinned commit, every change gets a new URL. The trade-off is a two-step edit: commit the profile, then update the URL to that commit.

Apple describes `ProfileURL` as expected to be on the MDM server, and the Mac presents its device identity certificate when it fetches the file. On macOS 27, a legacy declaration can reference a DDM asset instead (`ProfileAssetReference`), which is Apple's supported way to host the profile elsewhere and includes a hash check. If you host the file yourself, Fleet's guide to [securing externally hosted DDM assets](https://fleetdm.com/guides/securing-externally-hosted-ddm-assets) covers verifying those requests.

### Give the hosted profile its own identifier

The same GitOps run that adds the declaration removes the old Nudge profile. MDM commands and DDM are separate channels, so if the hosted profile reused the old identifier, the removal could delete it after the declaration installed it. A new identifier avoids that race.

### Plan for the Macs the gate can't open

`softwareupdate.install-state` is only available on supervised Macs. On macOS, that means Macs enrolled with Automated Device Enrollment (ADE). Macs enrolled another way never report `prepared`, so they'd never see Nudge.

We also set a deadline for Macs on macOS 15 to upgrade to a newer major version. Nothing in our setup makes macOS stage a major upgrade by itself, so those Macs might never reach `prepared` either.

Both groups get the same Nudge settings without the gate, through two labels: [macOS 15 and older hosts](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/all/labels/macos-15-and-older-hosts.yml) and [Macs not enrolled through ADE](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/all/labels/macs-not-enrolled-through-ade.yml). The declaration excludes those labels, and the ungated profile targets them.

### Download without enforcing

Wes uses a software update enforcement declaration to download and prepare updates. On Fleet's workstations, we haven't set `macos_updates` yet, so our setup relies on automatic downloads (`AutomaticActions.Download: AlwaysOn`) instead. If your Macs don't reach `prepared` that way, Fleet's [OS update settings](https://fleetdm.com/guides/enforce-os-updates) send the enforcement declaration for you. Keep Wes's warning about simultaneous downloads in mind.

## The payoff

Wes did the math for Stripe. Before the change, an update took 30 to 40 minutes from click to done. After, the restart takes 5 to 10. Apple ships around eight macOS updates a year. Across 12,000 to 13,000 people, saving 30 minutes each time adds up quickly, and employers like getting time back.

The bigger win is trust. When IT asks for something, it should be quick and worth it. Wes's approach makes the update request match that promise: by the time Nudge asks, the hard part is done.

Thank you, Wes, for sharing it.

## See it live

- Follow the [step-by-step guide](https://fleetdm.com/guides/show-nudge-only-after-macos-prepares-an-update) to set this up in your own fleet.
- **Watch the talk:** [Wes Whetstone, "But you called me?"](https://www.youtube.com/watch?v=SggxxPCDCSI) at MacDevOpsYUL 2026.
- **Get a demo:** [talk to us](https://fleetdm.com/contact) about managing macOS updates with Fleet.

<meta name="articleTitle" value="Don't put users on hold: show Nudge when the update is ready">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-10-20">
<meta name="description" value="Show Nudge only after macOS has downloaded and prepared an update, so users restart instead of waiting. Inspired by Wes Whetstone's MacDevOpsYUL talk.">
