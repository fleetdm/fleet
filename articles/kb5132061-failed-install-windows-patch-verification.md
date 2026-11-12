# Error 0x800F0993 leaves 726 Windows fixes uninstalled. Would your patch report notice?

*Some Windows 11 machines can't install KB5132061, November's cumulative update, and Microsoft hasn't explained why. The machines that fail are the ones a deployment report is most likely to count as done.*

## Key takeaways

- **A sent update and an installed update are different facts.** KB5132061 closes 726 Windows flaws, but some machines fail to install it with errors 0x800F0993 and 0x800F0989, so a report that tracks deployment can overstate coverage.
- **There's no fix yet, so the gap stays open.** Microsoft hasn't published a cause or a workaround, which means failed machines will keep missing these fixes until something changes.
- **Retries don't clear a persistent failure.** A machine that fails the same update every night looks busy, not broken, unless someone checks the result of each attempt.
- **The device can tell you what it's running.** Windows records each update attempt and its result, and the OS build number shows which cumulative update is actually in place.
- **Measure patching by state, not by intent.** Reports and policies that read the device directly give IT and security the same answer, and they keep working when the next update misbehaves.

<a purpose="cta-button" href="https://fleetdm.com/contact">See how Fleet reports Windows patch state</a>

Microsoft's November Patch Tuesday was large even by recent standards, and KB5132061 is the Windows 11 cumulative update at the center of it. The Register reports that some users can't install it. The failures show up as 0x800F0993 and 0x800F0989, and Microsoft hasn't offered an explanation or a fix.

A failed install is annoying for one user. Across a fleet, it raises a harder question: how many of your machines are reported as patched when they aren't?

## What's failing

KB5132061 rolls up fixes for 726 Windows vulnerabilities into one cumulative update. On affected machines, the install fails and one of the two error codes appears in the update history. As of publication, Microsoft hasn't listed the issue with a cause or a workaround.

Error 0x800F0993 has appeared before. Earlier this year Microsoft tied it to some devices that were upgraded to Windows 11 24H2 or 25H2 from older releases and then couldn't install cumulative updates. Whether the same cause is behind the KB5132061 failures isn't confirmed, so treat that history as a lead for troubleshooting, not a diagnosis.

## Why deployment status overstates coverage

Most patching workflows track what was sent: the update was approved, assigned to a group, and offered to devices. That's useful for operations, but it answers "did we ask?" rather than "did it happen?"

When a cumulative update fails to install, the device keeps running its old build. Depending on the tool and how it reports, the update can still look in progress or complete from the console. Meanwhile the vulnerability scanner and the patching dashboard can disagree, and security and IT end up arguing about whose number is right.

A failed install is functionally the same as an update that was never sent. The device is still exposed to every flaw the update was supposed to close.

## Read the result from the device

Windows keeps its own record. Each update attempt lands in the update history with a result, and the OS build number changes only when a cumulative update installs. Both are readable on the device.

If Fleet's agent runs on your Windows hosts, a report like this lists every attempt at KB5132061 and how it ended:

```sql
SELECT date, title, result_code, hresult
FROM windows_update_history
WHERE title LIKE '%KB5132061%';
```

Hosts that show repeated failures are the ones to look at first. A policy can turn the same idea into a pass or fail check, so hosts without the update show up as failing until it actually installs:

```sql
SELECT 1 FROM patches WHERE hotfix_id = 'KB5132061';
```

The `patches` table reads Windows' installed hotfix list, which can lag or miss entries on some systems. Pair the policy with the OS version on the host details page, which includes the build and revision, before you act on the results. Neither query has been tested against KB5132061 itself, so validate them on a known-good machine first.

## Vulnerability data follows the build

Fleet maps Windows vulnerabilities from Microsoft's MSRC security bulletins to each host's OS version. A host that failed KB5132061 stays on its old build, so the CVEs that update fixes should still be listed for it in Fleet. That gives security a host-level view of the same gap the update history shows IT, from the same data.

## Patched means installed

KB5132061 will eventually get a fix or a workaround from Microsoft. The habit worth keeping is the one this issue exposes: measure patching by what the device reports, so a failed install surfaces as a failure instead of hiding inside a deployment count.

## See it live

- **Learn more:** Read how [Fleet processes vulnerabilities](https://fleetdm.com/guides/vulnerability-processing) across Windows, macOS, and Linux.
- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about Windows patch reporting.

## Sources

- The Register, coverage of KB5132061 install failures with errors 0x800F0993 and 0x800F0989.
- BleepingComputer, coverage of Microsoft's earlier notice about upgraded Windows 11 devices failing to install cumulative updates with error 0x800F0993.

<meta name="articleTitle" value="Error 0x800F0993 leaves 726 Windows fixes uninstalled. Would your patch report notice?">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-12">
<meta name="description" value="Some Windows 11 machines can't install KB5132061, and Microsoft has no fix yet. Check update results and build numbers on each device instead.">
