# The riskiest step in a Mac refresh is finally manageable

*For years, Mac-to-Mac migration was either blocked outright or left entirely to the person at the keyboard. macOS 26.4's Managed Migration Assistant makes it a declarative, auditable decision that IT controls, and that matters more than the feature name suggests.*

Every hardware refresh ends with the one question IT never got to answer: what comes with you from the old Mac? The choices have always been bad. Block migration and frustrate users, or allow it and trust an end user to decide which folders, accounts, keys, and privacy settings land on a corporate machine, with no record of what actually moved.

Apple's Managed Migration Assistant (macOS 26.4) closes that gap, turning migration from an unmanaged user choice into a declarative configuration your device management service delivers during Setup Assistant. Here's why that matters more than the feature name suggests.

## Key takeaways

- **Migration stops being a user decision and becomes organizational policy.** You declare which folders and files are required, which are excluded, which accounts are off the table, and whether system-level privacy settings carry over.
- **You finally get a record of what moved.** A declarative status channel reports progress during the transfer and produces an after-action report: date, time, data transferred, and any files that couldn't migrate.
- **Refreshes get faster and quieter.** Migration is embedded in Setup Assistant, auto-selects the fastest available transport, and runs zero-touch through Automated Device Enrollment, which also makes it easier to finally move the Intel holdouts off old hardware.
- **The prerequisites are specific, so scope them deliberately.** Supervised devices enrolled through Apple Business Manager or Apple School Manager, the destination Mac on macOS 26.4 or later, and the declaration deployed with `await_device_configured`.
- **The right way to operationalize it is config-as-code.** The whole policy is a small declaration. Managed as version-controlled YAML through a GitOps workflow, your migration policy gets peer review, rollback, and an audit trail, and you can verify in real time what actually landed on the new Mac.

<a purpose="cta-button" href="https://fleetdm.com/contact">Get a demo</a>

## Migration has always been the ungoverned step

Declarative device management, zero-touch enrollment, configuration profiles, FileVault enforcement: the modern Apple deployment stack governs almost everything about a new Mac. Almost. The moment a user chose to bring data over from their previous machine, governance stopped and trust took over.

That gap has real consequences. On the security side, an uncontrolled migration can drag personal accounts, stale credentials, SSH keys, and a decade of unmanaged files onto a freshly provisioned corporate device. On the compliance side, you had no answer to a basic auditor's question: what data was transferred to this device, and when? And on the human side, IT's only reliable lever was to disable migration, which is why so many organizations still have users clinging to aging Intel Macs rather than face a manual rebuild of their environment.

Managed Migration Assistant is the first time that final step joins the rest of the managed deployment.

## From user choice to organizational policy

The core shift is simple: you describe the migration you want, and Setup Assistant enforces it.

The declaration lets you specify which subfolders and files inside the user's Home folder are required to migrate, which are excluded, which user accounts aren't offered at all, and whether system-level privacy settings come across. Paths are relative to the in-scope user's Home folder. `Documents/Work/` in `RequiredPaths` enforces the transfer of that project directory, while an exclusion can carve a single subfolder back out of an otherwise-required path. A useful detail for storage-constrained refreshes: when you list required paths, the order sets priority, so the most important data wins if the new Mac runs short on space.

A few boundaries are worth designing around rather than fighting. Hidden files migrate by default unless you exclude them. That includes things like SSH keys, which you may very much want to leave behind. The user's `~/Library` folder is always migrated and can't be excluded. Items in `/Applications` and certain system settings aren't eligible for transfer at all, so your existing app-deployment workflow still owns getting software onto the new machine. And the Restore pane can't be hidden. None of these are dealbreakers; they're just the shape of the box you're designing inside.

The point is that "what comes with you" is now a decision your security and IT teams make once, in policy, instead of a decision a user improvises during onboarding.

## The report you've never had

The capability that should get compliance teams' attention is the quietest one in the documentation.

The declarative device status channel reports status during migration and delivers a report after the transfer completes. That report includes the date, the time, the amount of data transferred, and whether any files could not be migrated. For the first time, "what moved to this device and when" has a documented answer instead of a shrug.

That changes migration from a trust exercise into an auditable event. It gives you the troubleshooting trail to explain a partial transfer, and it gives auditors the evidence that data handling during refresh follows a defined, recorded process. In regulated environments, the difference between "we have a policy" and "we have a policy and a record of it being applied" is the entire conversation.

## Faster refreshes, fewer tickets

The governance story is the headline, but the operational story is what gets this adopted.

Migration Assistant picks the fastest available transport on its own: a direct Wi-Fi connection, infrastructure Wi-Fi, Ethernet, or Thunderbolt. It keeps checking for a faster option mid-transfer. Combined with embedding in Setup Assistant and zero-touch enrollment through Automated Device Enrollment, a managed migration becomes part of the same hands-off provisioning flow as the rest of the device.

That has a second-order benefit worth naming: it lowers the cost of a refresh enough to finally move people off old hardware. A supported, governed migration path is a far easier sell to a reluctant user than "we'll wipe your machine and you'll rebuild your setup from scratch." Independent testing has found the feature working with source Macs going back several macOS versions earlier than the official macOS 15 baseline, useful context if your stragglers are exactly the ones you most want to retire.

## The caveat to solve before you roll this out

There's one prerequisite that will determine whether this works in your environment, and it's an access problem, not a technical one.

To start the migration on the source Mac, the user has to authenticate with local administrator credentials. In environments where your users are standard users, the fix is to pair this with just-in-time privilege elevation so a standard user can briefly launch Migration Assistant without holding standing admin rights. Tools like [SAP Privileges](https://github.com/SAP/macOS-enterprise-privileges) (open source) solve this. Where those aren't options, the `authorizationdb` can be adjusted to let standard users launch Migration Assistant, then reset afterward. Either way, this is the planning step that separates a smooth rollout from a stalled one. Decide how a standard user gets temporary rights before you publish the policy, not after the first refresh fails at the login prompt.

The rest of the prerequisites are straightforward but specific: the destination Mac needs macOS 26.4 or later, devices must be supervised and enrolled through Apple Business Manager or Apple School Manager and assigned to a device management service, and the declaration has to be delivered with `await_device_configured` set so it's in place before the user reaches the transfer step.

## Make the migration policy code, not a console click

A migration policy is too important to live as a setting someone toggled in a web console six months ago and can't quite remember the reason for.

The declaration itself is small: a `ShouldDoManagedMigration` flag, a `ShouldMigrateSecurityPrivacySettings` flag, and the `RequiredPaths` and `ExcludedPaths` arrays. That compactness is exactly why it belongs in version control. Managed through a GitOps workflow, your migration policy gets reviewed in a pull request before it ships, carries a history of who changed what and why, and can be rolled back the moment a refresh goes sideways. The policy that decides what corporate data moves onto every new Mac should be as auditable and reversible as any other piece of your infrastructure.

This is where [Fleet](https://fleetdm.com) fits the workflow. Fleet delivers Apple's declarative configurations from version-controlled YAML, applies them through CI/CD with drift correction, and pairs that with real-time reporting from Fleet's agent so you can confirm what actually landed on the new device rather than trusting that the declaration took.

## The stakes

Migration was the last unmanaged step in an otherwise governed deployment, and "unmanaged" on the step that decides what data lands on a new corporate device was never a comfortable place to be. Managed Migration Assistant doesn't just make refreshes smoother, it brings the migration decision under the same policy, audit, and version control as everything else you deploy. The organizations that treat it as a governance capability, not just a convenience feature, are the ones who'll get the audit trail and the clean baseline at the same time.

## See it live

The fastest way to see this in detail is to read Fleet's [Managed Migration Assistant guide](https://fleetdm.com/guides/managed-migration-assistant-mac-to-mac-migration-with-fleet) and adapt the example declaration to your environment. If you'd like a hand getting there, two good next steps:

- [**Get a demo**](https://fleetdm.com/contact)**.** We'll walk through how managed migration could work in your environment.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop)**.** Managing your migration policy as code is exactly what our hands-on workshop covers: declarations in Git, reviewed in pull requests, deployed through CI.

<meta name="articleTitle" value="The riskiest step in a Mac refresh is finally manageable">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-06-26">
<meta name="description" value="macOS 26.4's Managed Migration Assistant turns Mac-to-Mac migration into a declarative, auditable policy that IT controls.">
