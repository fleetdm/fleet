# From 120 days to 30: Project Zero's disclosure clock is closing the gap on your patch cycle

*Google Project Zero now publishes bug details 90 days after reporting them, whether or not the vendor has shipped a fix. By 2028 that window is 30 days. How fast can you name every affected device once the details are public?*

## Key takeaways

- **The 30-day grace period is already gone.** Since November 13, Project Zero publishes technical details at day 90 with no extra soak time, and the deadline drops to 60 days in February 2027 and 30 days in January 2028.
- **Public details can now arrive before a patch.** A vendor that misses the deadline doesn't get an extension, so some disclosures will describe bugs that have no fix yet.
- **AI is the stated reason.** Project Zero says AI-assisted research finds and weaponizes bugs faster, and that its own team now finds several vulnerabilities a week instead of a few a year.
- **Your first job on disclosure day is scoping.** Before any patch decision, you need a current list of which devices run the affected software and version.
- **A version you can query beats a version you assume.** Software inventory and live reports answer the scoping question in minutes, on macOS, Windows, and Linux.
- **Mitigations need the same visibility as patches.** When the fix is a configuration change or a disabled feature, you have to confirm it landed on each device.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software inventory in Fleet</a>

Since 2021, Project Zero has run a "90 + 30" policy. Vendors got 90 days to fix a reported bug, then 30 more days for the patch to roll out before Project Zero published the technical details. That second window was the quiet buffer most IT teams never thought about. It gave a patch time to reach devices before anyone outside the vendor knew exactly what it fixed.

That buffer is gone. On November 13, Project Zero moved to what it calls "Full 90": details go public 90 days after the report, fixed or not. The clock keeps shrinking from there. So the question for IT and security teams isn't whether Google's policy is fair to vendors. It's what your own response looks like when the details land on day 90.

## What changed and when

Project Zero's new policy, announced by Natalie Silvanovich, sets three dates:

| Date | Disclosure deadline |
|---|---|
| November 13, 2026 | 90 days after report, no soak period |
| February 2, 2027 | 60 days after report |
| January 3, 2028 | 30 days after report |

Most special cases are gone, including the grace period and the variant policy. Severity doesn't set the date, and neither does the vendor's own severity rating. Bugs exploited in the wild still follow Google's separate policy for those, and hardware bugs may still get extra time in rare cases. Project Zero will keep tracking whether a bug is fixed, but fix status no longer moves the disclosure date.

## Why Project Zero says the clock had to shrink

The reasoning is about AI on both sides of the disclosure. Project Zero says large language models help attackers find bugs faster, which raises the odds that someone else finds the same bug during an embargo. They also shorten the time from a published patch to a working exploit. Project Zero now treats zero time between patch and exploitation as "a reasonable, though imperfect, model of attacker behavior."

That's also why soak time went away. If a patch already shows an attacker where the bug is, holding back the write-up for 30 days doesn't protect anyone.

Volume is the other factor. Silvanovich wrote that Project Zero now finds several vulnerabilities a week, up from a few a year, and that the parts of the policy built on vendor negotiation were becoming unmanageable. Critics have pointed out that the announcement doesn't measure how much faster attackers have actually become. The direction is still clear, and Project Zero isn't the only research group adjusting to it.

## Disclosure day is a scoping problem first

When a Project Zero write-up goes public, the vendor's patch timeline is out of your hands. Your own timeline starts with one question: which of our devices run the affected software, at which version?

Many teams answer that question late. They wait for the vendor advisory, then for their scanner's vulnerability feed to update, then for the next scheduled scan. Under the old policy, that lag was partly covered by the 30-day soak window. Under Full 90, and later Full 30, the lag is exposure.

The faster path is to treat the version check as something you can run the moment you read the advisory. If the write-up names a product and an affected version range, you should be able to list every device in that range without waiting on anyone.

## Answer the question from the device

Fleet's agent collects installed software from macOS, Windows, and Linux devices, including apps, packages, and browser extensions. Search the **Software** page for the product named in the disclosure to see which versions are installed and how many hosts run each one.

For anything the inventory page doesn't make obvious, run a live report against the devices directly. For example, to find every Linux host with a specific package below a fixed version:

```sql
SELECT name, version FROM deb_packages WHERE name = 'example-package'
UNION ALL
SELECT name, version FROM rpm_packages WHERE name = 'example-package';
```

Swap in the package name from the advisory. On Windows, the `programs` table plays the same role, and on macOS, the `apps` table does. Live reports return results from online devices in seconds, so the scoping question doesn't wait for a scan schedule.

Fleet also matches installed software against published vulnerability data and shows affected hosts by CVE. That's the right long-term view, but feeds take time to catch up to a fresh disclosure. A direct version check works the same day.

## When the fix isn't a patch

Project Zero expects more vendors to respond with non-patch fixes, such as temporarily disabling a code path or a feature. It's also asking vendors to protect users while a patch is pending. That means some disclosures will come with a mitigation instead of an update.

A mitigation is only useful if it's actually in place. Write it as a Fleet policy, a yes-or-no check against each device, so you can see which hosts pass and which don't. If the mitigation is a configuration profile or a script, deploy it and keep the policy running until every host passes. The same check then confirms the real patch when it ships.

## Shorter clocks reward teams that already know

Project Zero's deadlines will keep moving toward 30 days, and other researchers are likely to follow the same logic. You can't control when details go public. You can control how long it takes to answer "are we affected, and where?" Teams that can answer from the device, on the day of disclosure, lose the least ground when the clock runs out.

## See it live

- **Learn more:** Read how [Fleet processes vulnerabilities](https://fleetdm.com/guides/vulnerability-processing) across macOS, Windows, and Linux.
- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about scoping exposure across your devices.

## Sources

- Google Project Zero, "Policy and Disclosure: 2026 Edition."
- Help Net Security, "Google Project Zero will disclose bugs after 90 days, fixed or not."
- BleepingComputer, "Google Project Zero cuts bug disclosures to 30 days amid AI threats."

<meta name="articleTitle" value="From 120 days to 30: Project Zero's disclosure clock is closing the gap on your patch cycle">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-16">
<meta name="description" value="Project Zero now publishes bug details 90 days after reporting, fixed or not, dropping to 30 by 2028. Scope your exposure from the device on day one.">
