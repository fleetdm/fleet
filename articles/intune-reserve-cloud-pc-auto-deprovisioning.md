# Windows 365 Reserve auto-deprovisioning: Know what a Cloud PC held before it disappears

*Intune's new preview setting deletes Reserve Cloud PCs that nobody has connected to for a set number of days. It doesn't keep a record of what those short-lived machines ran before they went away.*

## Key takeaways

- **Idle Cloud PCs get cleaned up without a ticket.** A new Intune preview setting removes Windows 365 Reserve Cloud PCs that go unused past a number of days you choose, so they stop spending the 10 days of Reserve access each user gets per year.
- **A deletion timer only acts on inactivity.** It doesn't look at what a Cloud PC ran or who signed in to it, and deprovisioning deletes the Cloud PC with all of its non-backed-up data, so that history goes with it.
- **Reserve Cloud PCs are short-lived by design.** IT provisions one when a user's device is lost, broken, or caught up in an outage, and each user gets at most 10 days a year, so many of these hosts exist for only a few days.
- **A shorter window saves more days, with a tradeoff.** A tighter idle limit returns unused access days sooner but raises the chance of deleting a Cloud PC that still holds someone's unsaved work, so choose the number deliberately.
- **Inventory is what makes the timer trustworthy.** If you can list accounts, software, and settings on every Windows host, physical or virtual, you can check what the timer is about to remove and what it will leave behind.

<a purpose="cta-button" href="https://fleetdm.com/contact">Talk to us about Windows visibility</a>

Microsoft is adding an auto-deprovisioning setting for Windows 365 Reserve Cloud PCs in Intune. It's in preview, and it removes a Reserve Cloud PC once it has gone unused for a set number of days.

That's a sensible setting for machines that only exist to cover a gap. The open question is what a deletion timer does and doesn't cover, because "idle" and "safe" are different properties.

## What the setting does

Reserve Cloud PCs are meant as a fallback when a user's primary device is lost, broken, or unavailable. IT provisions one only when it's needed, and from that moment the user's 10 days of access keep counting until the Cloud PC is deprovisioned, whether anyone connects or not.

The direct payoff is that idle Cloud PCs stop spending access days. Once the user's laptop is back, a forgotten Reserve Cloud PC burns days the user may need for the next incident, and removing it on a schedule closes that gap without anyone remembering to do it. Any days left over stay with the user for the next time IT provisions one.

## What a timer can't see

The setting acts on one signal: nobody has connected to the machine. It doesn't evaluate what is on the machine or what happened on it. A Reserve Cloud PC used during an outage or a security incident can hold local files, user profiles, and logs that someone may need later, and deprovisioning deletes all of it, with no snapshot or grace period when an admin or user does it.

Treat the deletion as the end of the record. If you want to know later which software a Reserve Cloud PC ran or who signed in to it, that information has to be collected somewhere else before the timer fires.

## Picking the window deliberately

There is no universally right number of days. A short window favors hygiene and cost. A long window favors availability for the person who finally needs the fallback machine. Decide with data: how often do Reserve Cloud PCs actually get used, and how long after provisioning does the first use usually happen? If you can't answer that, the setting will quietly make the call for you.

## Checking what the timer leaves behind

This is where a cross-platform inventory helps. If Fleet's agent runs on your Windows hosts, including virtual ones, you can build a report of local accounts, installed software, and OS version, then compare Reserve Cloud PCs against the rest of your fleet. Fleet keeps a host's last reported data after the host stops checking in, so that record is still there after Intune deletes the Cloud PC.

Whether Fleet's agent is deployed on a given Cloud PC depends on how you provision them, so confirm that before you rely on the data. A host that never reports in is its own finding.

## Treat deletion as the last step

The timer is a good backstop. It's a poor substitute for knowing what you have. Use it to remove machines that are truly unused, and use inventory to find the ones that were unused and also carrying stale access before the timer got to them.

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about reporting on Windows hosts.
- **Join a GitOps training session:** [Sign up for a workshop](https://fleetdm.com/gitops-workshop).

## Sources

- HTMD, coverage of Intune auto-deprovisioning for idle Windows 365 Reserve Cloud PCs (preview).

<meta name="articleTitle" value="Windows 365 Reserve auto-deprovisioning: Know what a Cloud PC held before it disappears">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-09">
<meta name="description" value="Intune can now delete idle Windows 365 Reserve Cloud PCs. Learn what the timer removes, what it keeps, and how to inventory each Cloud PC first.">
