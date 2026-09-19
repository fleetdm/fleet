# It took Microsoft days to undo the Patch Tuesday update that broke Remote Desktop on Windows Server

*September's cumulative updates left Windows Server 2019, 2022, and 2025 hosts freezing hours after boot, with Remote Desktop sessions hanging or refusing to connect. Here's how to find every server still running the broken update, or missing the fix, before a support ticket finds it for you.*

## Key takeaways

- **The break wasn't immediate.** Servers ran fine for hours after installing September's cumulative update before Remote Desktop Services started hanging, so a clean reboot right after patching told you nothing about whether a host was affected.
- **Three server versions, three separate KBs.** KB5122876 (Windows Server 2019), KB5122882 (Windows Server 2022), and KB5122871 (Windows Server 2025) all carry the same underlying bug.
- **The fix isn't a normal update.** Microsoft shipped a Known Issue Rollback (KIR), which disables the broken code path through a registry override rather than a new patch, and on servers it has to be deployed through Group Policy by hand.
- **You can't roll back what you can't find.** Fleet already reports the exact OS build and installed updates on every host, so identifying which servers still carry the unpatched KBs is a query, not a memory exercise.
- **A registry check tells you if the rollback took.** Because the KIR works by setting a registry value rather than replacing a file, confirming it applied is its own separate check, worth running after you deploy the Group Policy object.
- **This kind of gap will happen again.** A saved Fleet policy answers "who still needs this fix" every time a host enrolls or changes, instead of a one-time sweep that goes stale the next time Patch Tuesday causes a surprise.

<a purpose="cta-button" href="https://fleetdm.com/vulnerability-management">See patch visibility in Fleet</a>

Microsoft's September 2026 Patch Tuesday cumulative updates broke Remote Desktop Services on Windows Server 2019, 2022, and 2025. Affected hosts ran normally for a few hours after the update installed, then Remote Desktop sessions began hanging at "Connecting…," existing sessions stopped disconnecting cleanly, and in some reports the only recovery was a hard reset. Microsoft acknowledged the issue and shipped a Known Issue Rollback on September 11, about three days after the updates went out.

Three days sounds fast, until you consider what "fixed" means here. A Known Issue Rollback isn't a patch that reaches servers through the usual update pipeline. It's a registry override that admins deploy themselves, one Group Policy object at a time. That gap between "Microsoft published a fix" and "every affected server has it" is exactly the kind of gap that's easy to lose track of.

## What broke, and why a quick reboot didn't catch it

The bug traces to a deadlock between the RDP server component and the Local Session Manager. A developer investigating on Windows Server 2022 found the service hanging in `RDPServerBase!WdLib_Close` with no timeout set, which lets the deadlock accumulate until the whole session host stops accepting new connections. Because the freeze built up gradually rather than failing at boot, a server could pass a post-patch smoke test and still fail hours later once real session traffic hit it.

That delay is what makes this worth checking for directly rather than trusting that "no incidents yet" means "not affected." Three cumulative updates, all released September 8, 2026, carry the bug: [KB5122876](https://support.microsoft.com/en-us/servicing/os/windows-10/2026/09/kb5122876-windows-10-1809-security-update) for Windows Server 2019, [KB5122882](https://support.microsoft.com/en-us/servicing/os/windows-server/2026/09/kb5122882-windows-server-2022-security-update) for Windows Server 2022, and [KB5122871](https://support.microsoft.com/en-us/servicing/os/windows-server/2026/09/kb5122871-windows-server-2025-security-update) for Windows Server 2025. Microsoft's advisory extends the same known issue back to Windows Server 2012 and 2012 R2.

## Finding every server still carrying the broken update

Fleet already reports the exact OS build and the hotfixes installed on every enrolled host, so this doesn't require asking around for who remembers which boxes run Remote Desktop Services. A live query against the hotfix inventory picks out any host still on one of the three affected builds:

```sql
SELECT csname, hotfix_id, installed_on
FROM patches
WHERE hotfix_id IN ('KB5122876', 'KB5122882', 'KB5122871');
```

That list is your rollout target. It also doubles as a heads-up for servers you didn't know were still running Windows Server 2019 or 2012 R2, since those tend to be the hosts nobody remembers to check during a routine patch review.

## Confirming the rollback landed

Because a Known Issue Rollback works by writing a registry value instead of replacing a binary, "I deployed the GPO" and "the fix is active on this host" aren't the same claim. Published details on this rollback point to a value under `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Policies\Microsoft\FeatureManagement\Overrides`. A registry query against affected hosts turns "I think the GPO applied everywhere" into a checked fact instead of an assumption carried over from the deployment ticket:

```sql
SELECT path, data
FROM registry
WHERE path LIKE 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Policies\Microsoft\FeatureManagement\Overrides\%';
```

Confirm the specific override value against Microsoft's own Known Issue Rollback documentation for your KB before treating a match as proof, since Microsoft can add or renumber override entries between advisories.

## Turning this into a policy instead of a one-off sweep

A single query answers "who's exposed today." A saved Fleet policy answers it every day going forward, for every host that enrolls or drifts, without anyone re-running the hunt the next time a Patch Tuesday update causes an unplanned outage. Because Fleet policies live in Git as YAML and deploy through the same GitOps workflow as everything else, updating the target KBs when Microsoft ships next month's cumulative update is a reviewable pull request, not a console setting someone has to remember to change.

This won't be the last cumulative update that breaks something before it fixes something else. What changes is whether finding the affected servers takes a query or a support ticket.

## See it live

- **Get a demo** to see OS build and patch status reported across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore vulnerability management** in Fleet: [fleetdm.com/vulnerability-management](https://fleetdm.com/vulnerability-management)

## Sources

- BleepingComputer, [September Windows Server updates break Remote Desktop Services](https://www.bleepingcomputer.com/news/microsoft/september-windows-server-updates-break-remote-desktop-services/).
- BleepingComputer, [Microsoft: September updates cause RDS failures on Windows Server](https://www.bleepingcomputer.com/news/microsoft/microsoft-september-updates-cause-rds-failures-on-windows-server/).
- it-connect.tech, [September 2026 RDS bug on Windows Server: how to fix it now](https://www.it-connect.tech/september-2026-rds-bug-on-windows-server-how-to-fix-it-now/).

<meta name="articleTitle" value="It took Microsoft days to undo the Patch Tuesday update that broke Remote Desktop on Windows Server">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-14">
<meta name="description" value="September's Patch Tuesday broke Remote Desktop on Windows Server. See how to find every affected host and confirm Microsoft's rollback took.">
