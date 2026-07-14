# How Fleet completes your Microsoft stack across every OS, not just Apple

*Does your Microsoft stack give you the depth and cross-platform visibility you think it does? Fleet closes the gaps with answers in seconds and every change managed as code.*

## Key takeaways

- **Intune's macOS compliance policy evaluates a fixed, six-item checklist**, and custom compliance policies for macOS aren't supported at all. Fleet checks the requirements you actually set, on macOS, Windows, and Linux.

- **Fleet detects vulnerabilities (CVEs) across your entire fleet**, matching your software and OS inventory against NVD, VulnCheck-enriched CPE data, and OVAL feeds, including kernel-level CVEs on Linux. Nothing in Microsoft's endpoint tooling documents CVE detection for Apple devices.

- **Fleet integrates with Microsoft Entra for conditional access on both macOS and Windows.** When a host fails a Fleet policy, Fleet marks it non-compliant in Entra and blocks access until it's remediated. Any Fleet policy can be the gate, not just six checks.

- **Fleet streams endpoint telemetry to the SIEM and data lake you already use**: Splunk, Elastic, Google Chronicle, Snowflake, or Sentinel via your existing pipeline, using real-time data instead of inventory that drifts.

- **Fleet is open source.** The code is public, the company handbook is public, and device management is done as code (GitOps) with review, rollback, and no black boxes.

- **Fleet is fast, to answer and to act.** Live reports return results from thousands of hosts in seconds, and a change goes from pull request to enforced, or rolled back, across the fleet just as fast.

<a purpose="cta-button" href="/guides/entra-conditional-access-integration">See the Entra integration</a>

If your company standardized on Microsoft, the shape of your stack is familiar: Intune manages the devices, Entra owns identity, and Sentinel anchors security operations. Consolidating on one vendor was a defensible call, and the stack does a lot.

The question worth sitting with is narrower: does it watch your Macs and Linux machines as closely as it watches Windows? And even on Windows, does "compliant" mean what your security team needs it to mean?

## Microsoft's endpoint tools go deep on Windows and shallow everywhere else

Intune, Entra, and Sentinel grew up managing Windows, and on Windows it shows: rich configuration, mature policy tooling, deep telemetry. macOS and Linux are different operating systems with different APIs and different management surfaces, and Microsoft's tooling treats them more like guests than residents. Enroll a Mac or a Linux host in Intune and you can push profiles, trigger some remote actions, and read basic inventory. You can't interrogate the actual state of the device, and that state is what IT and security teams make decisions on.

In practice, the shortfall surfaces in three places:

**Stale, shallow inventory.** MDM inventory refreshes on a slow check-in cycle, so what the server shows and what's on the machine drift apart between check-ins. And plenty never shows up at all: Python packages, Homebrew binaries, browser extensions, live process and network activity, or the AI tooling your engineers installed last week.

**Compliance checks that thin out off Windows.** On macOS, Intune's compliance policy evaluates exactly six settings, and custom compliance policies (the kind where you define your own requirements) only exist for Windows and Linux. A Mac that Intune calls compliant has cleared six checks. Whether it meets your security requirements is a separate question, and Intune never asks it.

**No vulnerability picture off Windows.** Nothing in Microsoft's endpoint management documentation describes CVE tracking for Apple devices. A Mac or Linux host running a known, exploitable software version simply doesn't register in your dashboards.

None of this announces itself, which is the problem: the consoles report green while your non-Windows fleet operates outside their field of view, and even Windows depth ends where the built-in policy templates do.

## Where Fleet fits: inside the stack, not beside it

Fleet isn't another proprietary silo, and it isn't Apple-only. It's an open-source platform that gives IT and security teams real, in-depth visibility and control across macOS, Windows, Linux, ChromeOS, iOS, Android, and cloud infrastructure, and it plugs directly into the Microsoft tools your team already runs.

The integration is structural, not a partnership slide: Fleet feeds Entra the compliance signal it needs to make access decisions, feeds your SIEM the endpoint telemetry it's missing, and gives Intune-shaped workflows the depth and cross-platform reach they lack on their own.

### Fleet + Microsoft Intune: the depth and breadth Intune doesn't have

Intune handles enrollment, profiles, and baseline configuration competently. What it can't tell you is whether a device is in the state your security team requires, and on Apple hardware it can't even ask.

On macOS, the whole of Intune's compliance vocabulary is:

- OS version
- Password rules
- FileVault encryption
- Firewall
- System Integrity Protection (SIP)
- Gatekeeper

Windows gets a longer list plus custom compliance scripts to extend it. Apple devices get those six items, full stop.

Fleet's compliance model is built on policies that return a yes-or-no answer about the real state of a device. That changes what "compliant" can mean:

- Is XProtect current? Is Gatekeeper enforced? Is Recovery Lock set on this Apple Silicon Mac?
- Is BitLocker on with the right recovery configuration on this Windows host? Is a specific registry key set?
- Is the SSH daemon disabled on this Linux server? Is this exact kernel version patched?
- Does the device score against the CIS Benchmark? Is a required security agent running? Is a required certificate present? Is there a piece of software installed that shouldn't be?

If your security team can describe it as a state on the machine, Fleet can check it, and enforce it, on macOS, Windows, and Linux from one platform. Intune cannot.

That's the practical difference between the two models. Intune's checkmark certifies that six settings look right. A Fleet policy suite certifies whatever your team wrote into it, so the green light carries the meaning your auditors and your CISO assume it does.

On top of evaluation, Fleet inventories installed software across all devices, detects vulnerable versions, and can install, patch, and remove software. The same platform that finds the problem can fix it, fast, without bouncing between consoles. And every one of those policies, profiles, and software definitions can live in a Git repo: you write the check once, review it like any other code change, and Fleet applies it across macOS, Windows, and Linux.

### Fleet + Microsoft Entra: conditional access on Mac and Windows

Credit where it's due: Entra identity on the Mac has gotten meaningfully better. Platform Single Sign-On brings Entra sign-in to the macOS login window, backed by keys generated in the Secure Enclave, so credentials are hardware-bound and resistant to phishing.

Fleet's role is different. It supplies the device-trust signal that conditional access depends on, and it decides, with real rigor, which devices count as healthy.

Fleet integrates directly with Microsoft Entra to enforce conditional access on both macOS and Windows hosts. The mechanism is simple: when a host fails a policy in Fleet, Fleet reports it as non-compliant in Entra, and Entra blocks the user from third-party apps until the failing policy is remediated. The user clicks through to a remediation flow and regains access once the device is healthy again. If someone turns off MDM, Fleet reports that state too, and the device is marked non-compliant automatically.

The leverage is in what counts as "compliant." With Intune alone, the gate is the fixed checklist. With Fleet, the gate is any policy you can write. Conditional access stops being "is FileVault on?" and becomes "does this device meet our security bar before it touches our data?"

A few things worth knowing:

- Entra conditional access works even if you're not using Fleet's MDM features. You can adopt it alongside your current setup.
- On macOS, Fleet orchestrates the pieces: it installs Microsoft's Company Portal as a Fleet-maintained app (including during the zero-touch Setup Experience) and manages the Platform SSO profile, so registration happens as part of enrollment.
- The whole configuration can be applied via GitOps, so your access posture is versioned and reviewable instead of clicked together by hand.

Support landed for macOS in Fleet 4.70.0 and was extended to Windows hosts in 4.84.0. The same policy-driven gate now covers both halves of your fleet, monitored in one place.

### Fleet + your SIEM: complete endpoint telemetry, in real time

A SOC is only as good as what reaches it. Whether your team works in Sentinel, Splunk, Elastic, Chronicle, or a data lake, detection and response run on the telemetry your endpoints send, and endpoints that send little leave your analysts investigating on guesswork.

Fleet is, at its core, a telemetry engine. Fleet's agent turns every operating system into a queryable relational database, and Fleet ships those results to wherever your team works: Splunk, Elastic, Google Chronicle, Snowflake, or any streaming target via Kinesis/Firehose, Kafka, Google Pub/Sub, AWS Lambda, or S3. If Sentinel is your SOC, you can pipe Fleet data in through that same streaming infrastructure and correlate it with the rest of your environment.

What makes this additive rather than redundant:

- **Real-time, not drifted.** Live reports hit every online endpoint and return answers in seconds, and scheduled queries stream continuously. You're correlating current device state, not yesterday's inventory snapshot.
- **Cross-platform, in one schema.** macOS, Windows, and Linux report through the same tables, so one detection rule can cover all three (teams have mapped Fleet's agent results to MITRE ATT&CK in Splunk, for example).
- **Depth MDM doesn't expose.** Process and network activity, logged-in users and sessions, browser extensions, package managers, certificates. All the signals that let an analyst investigate.

Your analysts keep their console, their queries, and their muscle memory. What changes is that the device-level signal they've been working without starts arriving.

### Fleet + your AI and your governance program: the data has to be complete

AI security tooling, Microsoft Security Copilot included, reasons over whatever telemetry it's given. Hand it a partial view of your fleet and it will produce confident summaries of a partial view.

This is where Fleet's openness compounds. Because Fleet exposes the real state of every device as structured, queryable data, you can point your AI workflows at ground truth. And because device management in Fleet is done as code, you can describe a configuration change, a CVE fix, or a new policy in natural language, have it reviewed as a pull request (whether a human or a bot proposed it), apply it across the fleet, and roll it back instantly if needed.

It also closes a gap most stacks miss entirely: shadow AI and shadow IT. The MCP servers, IDE forks like Cursor and Windsurf, browser extensions, and unsanctioned tools your engineers install don't appear in MDM inventory, but they're exactly the kind of thing Fleet's agent surfaces. Fleet lets you discover AI and software usage across your fleet with a report, track it over time, and govern it with policy. You can't write an AI governance posture for tools you can't see.

## Doesn't our Microsoft licensing already cover this?

It's the objection every budget owner should raise, so here's the direct answer: the E5 line item doesn't buy full management of Macs and Linux machines, and the compliance model it does include stops at a fixed list on Apple hardware. Microsoft's own documentation draws those boundaries. In any organization with a meaningful non-Windows fleet, they translate into risk nobody is measuring.

As for "another proprietary tool?": Fleet isn't one.

- **It's open source.** All of Fleet's source code is public, and so is the company handbook. There's no black box, no guessing what the agent is doing, and no lock-in.
- **It's device management as code, and that makes it fast.** GitOps means every change is versioned, reviewed, and reversible. Describe a configuration change, a CVE fix, or a new policy (in natural language with your existing AI if you like), open it as a pull request, let a teammate review it, merge, and Fleet rolls it out across the fleet. If something's wrong, you roll back instantly. Your configuration lives in a repo, not in a console only one admin understands, and changes move at the speed of a merge instead of a change-control meeting.
- **It's quick to stand up.** You can have a preview environment running in a few minutes, and Fleet's agent is lightweight enough to run everywhere without weighing devices down.
- **It's cross-platform by design.** macOS, Windows, Linux, ChromeOS, mobile, and cloud infrastructure: one platform, one source of truth. (This is also where Fleet and Apple-only tools part ways. Closing your "Apple gap" with a Mac-only product just leaves you a new Linux gap and a new Windows-depth gap.)
- **It runs the way you need.** Self-host it, or run in Fleet's cloud with full control over data residency and jurisdiction. Either way, the data is yours.

## The risk of not looking

The gaps described here don't announce themselves. A Mac that has drifted out of your security baseline still reads compliant in Intune, because only six things were checked. A vulnerable package on a Linux host raises no alert, because no telemetry describing it ever left the machine. A known CVE on an Apple device stays out of your reports, because nothing in the stack is looking for it. Absence of signal looks like safety, and that's the most expensive kind of quiet.

Keep the Microsoft stack; it's earning its keep. Add Fleet where the stack goes quiet: every operating system you run, checks you define yourself, code you can read, and data you own.

This was never a Microsoft-or-Fleet decision. It's a decision about whether "compliant" in your environment means what your leadership thinks it means.

*Want to see it? Fleet is open source, and you can* [*stand up a preview environment*](https://fleetdm.com/try-fleet) *in a few minutes, or* [*get a demo*](https://fleetdm.com/contact)*.*

<meta name="articleTitle" value="How Fleet completes your Microsoft stack across every OS, not just Apple">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-07-13">
<meta name="description" value="Intune, Entra, and Sentinel weren't built for depth on macOS and Linux. See how Fleet closes the gaps inside the Microsoft stack you already run.">
