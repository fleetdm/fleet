Most organizations have adopted CIS benchmarks. Far fewer can prove, at any given moment, that their entire fleet actually meets them.

That gap, between adopting a standard and continuously verifying it, is where most CIS compliance programs quietly break down. The benchmark documents are excellent. The Center for Internet Security publishes detailed, consensus-driven configuration guidance that represents the collective judgment of security experts worldwide. Adopting them is the right decision.

But adoption is a policy statement. Verification is an operational practice. And the tools most organizations use to manage devices were never built to close the distance between the two.

## Adopting a benchmark is not the same as meeting it

Here is the problem with how most endpoint tools handle CIS benchmarks.

A traditional MDM can push a configuration profile to a device and record that the profile was delivered. What it usually cannot tell you is whether the setting is actually in effect right now. Profiles get overridden. Users change settings. Updates reset defaults. Configurations drift. The MDM reports that it sent the instruction, not that the device is in the state the instruction intended.

This is the difference between knowing a configuration was pushed and knowing it is actually in place. For a compliance program, that difference is everything. An auditor does not want to see that you intended a device to be compliant. They want evidence that it is.

Fleet closes this gap because it is built on osquery. Instead of trusting that a pushed configuration took effect, Fleet queries the device directly and reads its actual state. When a CIS benchmark says disk encryption must be enabled, Fleet does not check whether an encryption profile was sent. It checks whether encryption is on. Each benchmark becomes a policy, and each policy is a query that returns a clear pass or fail based on what the device actually reports.

Fleet provides out-of-the-box CIS benchmark policies for macOS and Windows, available in Fleet Premium and Ultimate, covering the full set of benchmarks that can be automated. Some CIS controls are not automatable by design and still require manual review, and Fleet is explicit about which ones those are rather than implying coverage it does not have. For everything that can be checked programmatically, Fleet checks the live state of the device, not a record of intent.

## Continuous evaluation turns compliance into a real-time metric

Most CIS compliance work happens in bursts. A team runs an assessment, generates a report, remediates what it can before a deadline, and then moves on until the next cycle. Between those cycles, the actual compliance state of the fleet is unknown. Devices drift, new machines enroll, and the gap between the last report and current reality widens every day.

This is backwards. The frameworks that reference CIS benchmarks almost always call for continuous monitoring, not periodic snapshots. A point-in-time assessment tells you the fleet was compliant on the day you checked. It tells you nothing about today.

Fleet evaluates policies continuously. CIS benchmark policies run on a schedule against every enrolled device, and the results are always current in the dashboard. Benchmark compliance becomes a live operational metric rather than an audit artifact you reconstruct under deadline pressure. You can see, at any moment, what percentage of your macOS and Windows fleet passes each benchmark, which specific devices fail, and exactly what the failure is.

The practical shift is in timing. A device that drifts out of compliance shows up the day it drifts, not at the next quarterly assessment, so you are fixing one machine while the exposure is small rather than reconstructing a quarter of accumulated gaps under deadline pressure. Fleet can also trigger automated remediation when a policy fails, running a script to bring the device back into the intended state, so common failures self-heal without a manual queue. And the evidence an auditor asks for is already in the dashboard, current as of today, instead of being assembled by hand the week before the audit.

This is not theoretical. Faire uses Fleet to monitor its Macs against CIS benchmarks and to take remediation action when issues are found, which improves their device security posture without adding a manual reconciliation burden.

There is a compliance benefit beyond the operational one. A program that monitors benchmark compliance continuously and remediates in real time is demonstrating exactly the kind of ongoing control that frameworks expect to see. The continuous practice is not just easier to run. It is stronger evidence.

## One compliance view across the platforms you manage

CIS publishes benchmarks for macOS, Windows, and major Linux distributions. In most organizations, each platform is assessed by a different tool, on a different schedule, using a different methodology. The Mac team checks Macs one way. The Windows team checks Windows another way. Linux, if it is checked at all, is checked by a third process. When it is time to report fleet-wide compliance, someone has to reconcile three inconsistent data sets into a single number, and that reconciliation is slow, manual, and hard to defend.

Fleet gives both IT and security a single compliance view across every platform it manages, evaluated through one consistent osquery-based methodology. Fleet's built-in CIS benchmark library covers macOS and Windows, and both run as policies through the same engine and report into the same dashboard. There is no separate Mac tool and Windows tool to reconcile. The macOS compliance number and the Windows compliance number are produced the same way, from live device state, in one place.

For Linux, the same continuous policy model applies, but the coverage is different and it is worth being precise about how. Fleet does not ship a pre-built CIS Linux benchmark library today. Because every policy is simply an osquery query, teams can author their own CIS-aligned Linux checks and manage them alongside macOS and Windows in the same view, using the same evaluation logic and the same evidence trail. So the unified compliance view spans Linux, but the ready-made CIS benchmark content does not yet, and closing that last gap is work the team takes on rather than something Fleet hands you out of the box.

That consistency is still valuable. Where the built-in benchmarks apply, every platform is measured the same way, from the same kind of live data, so the compliance story you tell an auditor is coherent rather than assembled from tools that each define and measure compliance differently.

## What changes when CIS compliance becomes continuous

When benchmark compliance is verified from live device state and evaluated continuously, the practical changes are concrete:

**You measure what is true, not what was intended.** Compliance reflects the actual configuration of each device right now, not a record that a profile was once delivered. The number you report is the number on the fleet.

**Gaps get fixed when they happen.** Configuration drift surfaces immediately instead of accumulating until the next assessment. Failed checks can trigger automated remediation, so common issues resolve without a manual queue.

**Audit preparation stops being a project.** Evidence is maintained continuously rather than reconstructed before a deadline. When an auditor asks for proof, the data already exists and is already current.

**One methodology spans the fleet.** macOS and Windows are measured the same way through Fleet's built-in benchmarks, and Linux fits the same model through custom policies, so the compliance picture is consistent rather than reconciled from separate tools.

## The bottom line

CIS benchmarks have always been worth adopting. The hard part was never the standard. It was proving, continuously and credibly, that a real fleet of real devices actually meets it.

Fleet does that justice. By reading live device state through osquery rather than trusting that a configuration was pushed, by evaluating benchmark policies continuously rather than in periodic bursts, and by measuring every managed platform through one consistent methodology, Fleet turns CIS compliance from a recurring audit scramble into an ongoing operational practice.

The benchmark tells you how devices should be configured. Fleet tells you, at any moment, whether they actually are.

*See how Fleet implements CIS benchmarks in the [CIS benchmarks guide](https://fleetdm.com/guides/cis-benchmarks), or [talk to Fleet](https://fleetdm.com/contact) about continuous configuration compliance across your fleet.*

<meta name="articleTitle" value="Benchmarks without the burden: continuous CIS compliance">
<meta name="authorFullName" value="Dhruv Majumdar">
<meta name="authorGitHubUsername" value="drvcodenta">
<meta name="category" value="security">
<meta name="publishedOn" value="2026-06-11">
<meta name="description" value="Adopting CIS benchmarks isn't the same as meeting them. Fleet verifies CIS compliance continuously from live device state across macOS and Windows.">
