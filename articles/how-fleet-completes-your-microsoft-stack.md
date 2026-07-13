# How Fleet completes your Microsoft stack across every OS, not just Apple

*Does your Microsoft stack give you the depth and cross-platform visibility you think it does? Fleet closes the gaps with answers in seconds and every change managed as code.*

## Key takeaways

- **Intune's macOS compliance policy evaluates a fixed, six-item checklist**, and custom compliance policies for macOS aren't supported at all. Fleet evaluates against your actual security requirements on macOS, Windows, and Linux.

- **Fleet detects vulnerabilities (CVEs) across your entire fleet**, matching your software and OS inventory against NVD, VulnCheck-enriched CPE data, and OVAL feeds, including kernel-level CVEs on Linux. Microsoft's endpoint stack has no documented CVE tracking for Apple devices.

- **Fleet integrates with Microsoft Entra for conditional access on both macOS and Windows.** When a host fails a Fleet policy, Fleet marks it non-compliant in Entra and blocks access until it's remediated. Any Fleet policy can be the gate, not just six checks.

- **Fleet streams endpoint telemetry to the SIEM and data lake you already use**: Splunk, Elastic, Google Chronicle, Snowflake, or Sentinel via your existing pipeline, using real-time data instead of inventory that drifts.

- **Fleet is open source.** The code is public, the company handbook is public, and device management is done as code (GitOps) with review, rollback, and no black boxes.

- **Fleet is fast, to answer and to act.** Live reports return results from thousands of hosts in seconds, and a change goes from pull request to enforced, or rolled back, across the fleet just as fast.

<a purpose="cta-button" href="/guides/entra-conditional-access-integration">See the Entra integration</a>

Your organization runs on Microsoft. Your teams rely on Intune for device management, Entra for identity, and Microsoft Sentinel for security operations. You've built a serious technology stack, and you should feel good about that.

But does that stack give you the depth you assume and cover every device, on every OS, the same way?

## Your Microsoft tools give you a baseline. They weren't built for depth or for every OS.

The reality is that Microsoft's endpoint tooling was engineered around the Windows ecosystem. It's excellent at what it does, for Windows. But macOS and Linux operate on fundamentally different architectures, APIs, and management frameworks. When you enroll a Mac or a Linux box in Intune, you get a baseline: enrollment, some configuration, remote actions. What you don't get is deep visibility into the actual state of the device, the visibility security and IT teams need to make decisions.

That gap shows up in ways that matter to your security posture:

**Inventory that isn't complete.** MDM inventory is collected on a slow cadence and tends to drift. The data in the server stops matching the data on the machine until the next check-in. And MDM doesn't see a lot of what's running: Python packages, Homebrew binaries, browser extensions, live process and network activity, or the AI tooling your engineers installed last week.

**Compliance enforcement that doesn't translate across platforms.** Intune's macOS compliance policy is a fixed, six-item checklist. Custom compliance policies, the ones where you set your own requirements, are limited to Windows and Linux; Apple platforms are excluded by design. So when a Mac passes Intune's compliance check, it means six things came back clean. It doesn't mean your security requirements were evaluated.

**Vulnerability exposure you can't see.** There's no documented CVE tracking for Apple devices anywhere in Microsoft's endpoint management stack. Your Mac and Linux fleets can be carrying known, exploitable software versions and it won't appear in your dashboards.

The result is a stack that looks airtight but has quiet, persistent gaps where your non-Windows fleet lives, and where the depth of your Windows fleet stops at whatever the built-in policy templates happen to ask.

## Fleet closes the gap inside the Microsoft tools you already use.

Fleet isn't another proprietary silo, and it isn't Apple-only. It's an open-source platform that gives IT and security teams real, in-depth visibility and control across macOS, Windows, Linux, ChromeOS, iOS, Android, and cloud infrastructure, and it plugs directly into the Microsoft tools your team already trusts.

Better together isn't a slogan here. It's the architecture: Fleet feeds Entra the compliance signal it needs to make access decisions, feeds your SIEM the endpoint telemetry it's missing, and gives Intune-shaped workflows the depth and cross-platform reach they lack on their own.

### Fleet + Microsoft Intune: the depth and breadth Intune doesn't have

Intune does foundational work. Fleet brings the depth, and it brings it to every platform, not just one.

Intune's macOS compliance policy is a fixed checklist:

- OS version
- Password rules
- FileVault encryption
- Firewall
- System Integrity Protection (SIP)
- Gatekeeper

For Windows, that checklist runs deeper. For Mac, it's a starting point. And custom compliance policies, which would let you go further, aren't available for Apple at all.

Fleet's compliance model is built on policies that return a yes-or-no answer about the real state of a device. That changes what "compliant" can mean:

- Is XProtect current? Is Gatekeeper enforced? Is Recovery Lock set on this Apple Silicon Mac?
- Is BitLocker on with the right recovery configuration on this Windows host? Is a specific registry key set?
- Is the SSH daemon disabled on this Linux server? Is this exact kernel version patched?
- Does the device score against the CIS Benchmark? Is a required security agent running? Is a particular certificate installed? Is there a piece of software present that shouldn't be?

If your security team can describe it as a state on the machine, Fleet can check it, and enforce it, on macOS, Windows, and Linux from one platform. Intune cannot.

When a device passes Intune's macOS compliance check, the short checklist comes back clean. When it passes a Fleet policy set, it meets your requirements. For security leaders, that distinction is the whole point: a green checkmark only means something if it asks the right questions.

On top of evaluation, Fleet inventories installed software across all devices, detects vulnerable versions, and can install, patch, and remove software. The same platform that finds the problem can fix it, fast, without bouncing between consoles. And every one of those policies, profiles, and software definitions can live in a Git repo: you write the check once, review it like any other code change, and Fleet applies it across macOS, Windows, and Linux.

### Fleet + Microsoft Entra: conditional access on Mac and Windows

Microsoft has made real progress on identity. Platform Single Sign-On extends Entra authentication to the macOS login experience with Secure Enclave-backed, hardware-bound keys: the private key never leaves the device, and authentication is phishing-resistant.

So where does Fleet fit? As the management layer that makes Entra conditional access work, and the layer that decides, with real rigor, which devices are allowed in.

Fleet integrates directly with Microsoft Entra to enforce conditional access on both macOS and Windows hosts. The mechanism is simple: when a host fails a policy in Fleet, Fleet reports it as non-compliant in Entra, and Entra blocks the user from third-party apps until the failing policy is remediated. The user clicks through to a remediation flow and regains access once the device is healthy again. If someone turns off MDM, Fleet reports that state too, and the device is marked non-compliant automatically.

The leverage is in what counts as "compliant." With Intune alone, the gate is the six-item checklist. With Fleet, the gate is any policy you can write. Conditional access stops being "is FileVault on?" and becomes "does this device meet our actual security bar before it touches our data?"

A few things worth knowing:

- Entra conditional access works even if you're not using Fleet's MDM features. You can adopt it alongside your current setup.
- On macOS, Fleet orchestrates the pieces: it installs Microsoft's Company Portal as a Fleet-maintained app (including during the zero-touch Setup Experience) and manages the Platform SSO profile, so registration happens as part of enrollment.
- The whole configuration can be applied via GitOps, so your access posture is versioned and reviewable instead of clicked together by hand.

Support landed for macOS in Fleet 4.70.0 and was extended to Windows hosts in 4.84.0. The same policy-driven gate now covers both halves of your fleet, monitored in one place.

### Fleet + your SIEM: complete endpoint telemetry, in real time

Your SOC lives in a SIEM: Sentinel, Splunk, Elastic, Chronicle, or a data lake. Its ability to detect, investigate, and respond depends entirely on the quality and completeness of the data feeding it. If your endpoints aren't generating the right telemetry, your team is working with a partial picture.

Fleet is, at its core, a telemetry engine. Fleet's agent turns every operating system into a queryable relational database, and Fleet ships those results to wherever your team works: Splunk, Elastic, Google Chronicle, Snowflake, or any streaming target via Kinesis/Firehose, Kafka, Google Pub/Sub, AWS Lambda, or S3. If Sentinel is your SOC, you can pipe Fleet data in through that same streaming infrastructure and correlate it with the rest of your environment.

What makes this additive rather than redundant:

- **Real-time, not drifted.** Live reports hit every online endpoint and return answers in seconds, and scheduled queries stream continuously. You're correlating current device state, not yesterday's inventory snapshot.
- **Cross-platform, in one schema.** Mac, Windows, and Linux events land side by side, with the same tooling your team already uses (teams have mapped Fleet's agent results to MITRE ATT&CK in Splunk, for example).
- **Depth MDM doesn't expose.** Process and network activity, logged-in users and sessions, browser extensions, package managers, certificates. All the signals that let an analyst investigate.

No new investigation workflow. No new query language to learn. Just the endpoint signal your SIEM was missing, in the platform your SOC already uses.

### Fleet + your AI and your governance program: the data has to be complete

Microsoft Security Copilot and every other AI security assistant are only as good as the data they receive. If your device management system isn't contributing complete, current, cross-platform telemetry, the AI is giving you the best answer it can with incomplete information.

This is where Fleet's openness compounds. Because Fleet exposes the real state of every device as structured, queryable data, you can point your AI workflows at ground truth. And because device management in Fleet is done as code, you can describe a configuration change, a CVE fix, or a new policy in natural language, have it reviewed as a pull request (whether a human or a bot proposed it), apply it across the fleet, and roll it back instantly if needed.

It also closes a gap most stacks miss entirely: shadow AI and shadow IT. The MCP servers, IDE forks like Cursor and Windsurf, browser extensions, and unsanctioned tools your engineers install don't appear in MDM inventory, but they're exactly the kind of thing Fleet's agent surfaces. Fleet lets you discover AI and software usage across your fleet with a report, track it over time, and govern it with policy. You can't write an AI governance posture for tools you can't see.

## "We already pay for Microsoft. Why add another tool?"

That's exactly the right question. The honest answer: because Intune wasn't built to manage Mac and Linux fully, and because its compliance model is a fixed checklist on Apple and unavailable for custom Apple policies. Those aren't theoretical gaps. They're documented product boundaries that create real exposure in any organization running a meaningful non-Windows fleet.

And the answer to "another proprietary tool?" is: Fleet isn't one.

- **It's open source.** All of Fleet's source code is public, and so is the company handbook. There's no black box, no guessing what the agent is doing, and no lock-in.
- **It's device management as code, and that makes it fast.** GitOps means every change is versioned, reviewed, and reversible. Describe a configuration change, a CVE fix, or a new policy (in natural language with your existing AI if you like), open it as a pull request, let a teammate review it, merge, and Fleet rolls it out across the fleet. If something's wrong, you roll back instantly. Your configuration lives in a repo, not in a console only one admin understands, and changes move at the speed of a merge instead of a change-control meeting.
- **It's quick to stand up.** You can have a preview environment running in a few minutes, and Fleet's agent is lightweight enough to run everywhere without weighing devices down.
- **It's cross-platform by design.** macOS, Windows, Linux, ChromeOS, mobile, and cloud infrastructure: one platform, one source of truth. (This is also where Fleet and Apple-only tools part ways. Closing your "Apple gap" with a Mac-only product just leaves you a new Linux gap and a new Windows-depth gap.)
- **It runs the way you need.** Self-host it, or run in Fleet's cloud with full control over data residency and jurisdiction. Either way, the data is yours.

## The risk of waiting is real.

Every day your non-Windows devices operate without deep, queryable visibility, and every day your "compliant" Macs are only evaluated against six settings, undetected risk sits quietly in your environment. It doesn't show up in Intune's compliance dashboard, because the questions that would surface it aren't being asked. It doesn't trigger SIEM alerts, because the telemetry that would generate them isn't flowing. It doesn't appear in your vulnerability reports, because there's no CVE tracking for those devices in your current stack. It just exists, until it doesn't.

Your Microsoft investments are strong. Fleet makes them complete, across every operating system you run, with code you can read, policies you can write, and data you own.

The question was never Microsoft or Fleet. It's what happens to your security posture when you stop choosing between depth and coverage.

*Want to see it? Fleet is open source, and you can* [*stand up a preview environment*](https://fleetdm.com/try-fleet) *in a few minutes, or* [*get a demo*](https://fleetdm.com/contact)*.*

<meta name="articleTitle" value="How Fleet completes your Microsoft stack across every OS, not just Apple">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-07-13">
<meta name="description" value="Intune, Entra, and Sentinel weren't built for depth on macOS and Linux. See how Fleet closes the gaps inside the Microsoft stack you already run.">
