# macOS 27 ships September 14, and it leaves every Intel Mac behind.

*Apple confirmed [macOS 27 Golden Gate](https://www.apple.com/os/macos/), arriving September 14 alongside iOS 27 and iPadOS 27, runs only on Apple silicon. Here's how to find every Intel Mac in your fleet before that cutoff, instead of after.*

## Key takeaways

- **September 14 is a hard cutoff, not a recommendation.** macOS 27 Golden Gate installs only on Apple silicon. Any Intel Mac still in service stops receiving new macOS security updates the moment this release ships.
- **"We don't have many Intel Macs left" is a guess until someone checks.** Trade-ins, remote hires, and BYOD devices routinely outlive the spreadsheet that was supposed to track them.
- **Fleet already knows the chip in every Mac you manage.** Hardware inventory reports architecture, model, and OS version for every enrolled host, so the Intel Macs in your fleet are a query away, not a manual audit.
- **A real list turns into a refresh plan, then a retirement plan.** Replace the Intel Macs on it, then wipe and dispose of them properly, trade-in, donation, or certified e-waste, instead of leaving them in a drawer.
- **The same inventory becomes an ongoing check.** A Fleet policy that fails on Intel architecture keeps the list current as new hosts enroll, so the gap doesn't quietly reopen after the initial cleanup.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See hardware inventory in Fleet</a>

Apple confirmed that macOS 27 Golden Gate ships September 14 alongside iOS 27 and iPadOS 27, and that it runs only on Apple silicon. There's no Intel build this time. Every Mac still running an Intel chip on that date stops being eligible for macOS's newest security updates, full stop, regardless of how well it's held up otherwise.

That's a real deadline for any team still running a mixed fleet, but the deadline isn't the hard part. The hard part is knowing which machines it applies to.

## The gap between the inventory spreadsheet and the actual fleet

Most IT teams believe they know roughly how many Intel Macs are left. Few can name them. A Mac bought four years ago for a departed employee and reassigned to someone else, a remote hire's personal device enrolled through BYOD, a lab machine nobody's opened a ticket about in a year, all of these can quietly outlast the asset spreadsheet that was supposed to track them.

That gap is invisible right up until it isn't. An Intel Mac running past September 14 doesn't announce itself. It just stops getting the next patch, and stays that way until whoever's using it reports something going wrong, or until an attacker gets there first.

## Turning "probably a handful" into a list with names on it

Fleet already collects hardware and chip architecture from every enrolled Mac, the same way it collects installed software and OS version. Search that inventory for Intel-based hosts and the result isn't a guess: it's every Mac still on Intel silicon, along with who's using it, what team it's on, and what OS it's currently running, so a hardware refresh plan can start from a real list instead of a rounding error.

That's the difference between "we should look into replacing a few Intel Macs" and a purchase order with a device count and a set of names attached. Finance and procurement can plan around a number. They can't plan around a hunch.

## Making the check permanent, not a one-time cleanup

Running that search once clears the current backlog. It doesn't stop a new Intel Mac from showing up later, whether through a device transfer, a reissued laptop from storage, or an acquisition bringing in a fleet nobody's audited yet. A Fleet policy that flags any host still running Intel architecture keeps that list current automatically, on a schedule, instead of depending on someone remembering to re-run the search every quarter.

Because policies live in Git as YAML and deploy through the same GitOps workflow as everything else Fleet manages, updating that policy as the fleet changes is a reviewable pull request. The Intel Mac count stays visible on its own, not buried in a spreadsheet someone updates when they remember to.

## See it live

- **Get a demo** to see your own fleet's hardware and chip architecture broken out by host: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)
## Sources

- Apple, [macOS 27 Golden Gate](https://www.apple.com/os/macos/).
- IBM, [One in four malicious breaches are AI-enabled, costing companies $6 million on average](https://newsroom.ibm.com/2026-07-29-ibm-study-one-in-four-malicious-breaches-are-ai-enabled,-costing-companies-6-million-on-average).
- PCI Security Standards Council, [PCI DSS v4.x resource hub](https://blog.pcisecuritystandards.org/pci-dss-v4-0-resource-hub).
- HHS, [The Security Rule](https://www.hhs.gov/hipaa/for-professionals/security/index.html).
- AICPA & CIMA, [System and Organization Controls: SOC suite of services](https://www.aicpa-cima.com/resources/landing/system-and-organization-controls-soc-suite-of-services).
- ISO, [ISO/IEC 27001:2022](https://www.iso.org/standard/27001).
<meta name="articleTitle" value="macOS 27 ships September 14, and it leaves every Intel Mac behind.">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-10">
<meta name="description" value="macOS 27 runs only on Apple silicon. See how to find every Intel Mac in your fleet before security updates stop on September 14.">
