<meta name="articleTitle" value="Windows 365 Reserve Cloud PCs now expire on a timer, but a timer isn't an access review">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-09">
<meta name="description" value="Intune can now remove idle Windows 365 Reserve Cloud PCs after set days. Here is what that timer covers and what IT still has to check on each host.">

# Windows 365 Reserve Cloud PCs now expire on a timer, but a timer isn't an access review

*Intune's new preview setting deletes idle Reserve Cloud PCs for you. It doesn't tell you what those machines could reach before they went away.*

## Key takeaways

- **Idle Cloud PCs get cleaned up without a ticket.** A new Intune preview setting removes Windows 365 Reserve Cloud PCs that go unused past a number of days you choose, which cuts license cost and the number of desktops nobody is watching.
- **A deletion timer only acts on inactivity.** It doesn't know which idle machines hold local admin accounts, cached credentials, or software that should have been patched, so the days before the timer fires still count as exposure.
- **Reserve machines are easy to forget.** They exist for the day a primary device fails, so they tend to sit unused and unreviewed, which is the profile of a host with stale access.
- **Shortening the window shrinks the risk, with a tradeoff.** A tighter idle limit leaves fewer forgotten desktops but raises the chance of deleting one someone needs, so the number is a decision, not a default.
- **Inventory is what makes the timer trustworthy.** If you can list accounts, software, and settings on every Windows host, physical or virtual, you can check what the timer is about to remove and what it will leave behind.

<a purpose="cta-button" href="https://fleetdm.com/contact">Talk to us about Windows visibility</a>

Microsoft is adding an auto-deprovisioning setting for Windows 365 Reserve Cloud PCs in Intune. It's in preview, and it removes a Reserve Cloud PC once it has gone unused for a set number of days.

That's a sensible default for a category of machine built to wait. The open question is what a deletion timer does and doesn't cover, because "idle" and "safe" are different properties.

## What the setting does

Reserve Cloud PCs are meant as a fallback when a user's primary device is lost, broken, or unavailable. Most of them never get used, which means they sit provisioned, licensed, and assigned to a person. According to the preview, an admin can now set a number of days, and Intune removes any Reserve Cloud PC that has gone unused for that long.

The direct payoff is cost and tidiness. An unused virtual desktop still bills, and it still has a user attached to it. Removing it on a schedule closes that gap without anyone remembering to do it.

## What a timer can't see

The setting acts on one signal: the machine hasn't been used. It doesn't evaluate what is on the machine during the idle window. A Reserve Cloud PC that was imaged months ago can carry local accounts, installed software, and configuration that drifted from your current baseline, and all of it is reachable until the day the timer runs out.

Treat the idle window as exposure time. If you set it to a long period to avoid deleting machines people need, you've also chosen how long an unreviewed desktop can stay on your network.

## Picking the window deliberately

There is no universally right number of days. A short window favors hygiene and cost. A long window favors availability for the person who finally needs the fallback machine. Decide with data: how often do Reserve Cloud PCs actually get used, and how long after provisioning does the first use usually happen? If you can't answer that, the setting will quietly make the call for you.

## Checking what the timer leaves behind

This is where a cross-platform inventory helps. If Fleet's agent runs on your Windows hosts, including virtual ones, you can build a report of local accounts, installed software, and OS version, then compare Reserve Cloud PCs against the rest of your fleet. That gives you a list of idle machines that also look out of policy, which is a better place to start than a day count alone.

Whether Fleet's agent is deployed on a given Cloud PC depends on how you provision them, so confirm that before you rely on the data. A host that never reports in is its own finding.

## Treat deletion as the last step

The timer is a good backstop. It's a poor substitute for knowing what you have. Use it to remove machines that are truly unused, and use inventory to find the ones that were unused and also carrying stale access before the timer got to them.

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about reporting on Windows hosts.
- **Join a GitOps training session:** [Sign up for a workshop](https://fleetdm.com/gitops-workshop).

## Sources

- HTMD, coverage of Intune auto-deprovisioning for idle Windows 365 Reserve Cloud PCs (preview).
