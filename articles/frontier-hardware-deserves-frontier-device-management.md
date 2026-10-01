# Frontier hardware deserves frontier device management

*AI supercomputers now sit on desks, run agents on their owners' behalf, and hold the models and data your company cares about most. They belong in your device management, not beside it.*

## Key takeaways

- **The workstation is where AI runs now.** Hardware like NVIDIA DGX Spark puts model weights, training data, and inference on the desk instead of in someone else's data center. The device that holds all of that is the one you most need to see.

- **Everyone has an executive assistant, and it lives on the device.** AI agents read, write, and act for the people who run them, with access to code, credentials, and internal systems. Management has to reach the place where that access is exercised.

- **The vendor tooling assumes you'll build the rest.** NVIDIA's manageability guide ships sensible collectors and an update controller, then leaves targeting, access control, evidence, and vulnerability detection to your team.

- **SSH and a spreadsheet don't scale to a fleet of these.** A handful of AI workstations turns into bastion hosts, service account keys, per-device sudo rules, and JSON files nobody diffs. An agent that checks in outbound makes device state something you query.

- **Frontier devices belong in the same fleet as everything else.** A DGX Spark should inherit your workstation baseline, patch cadence, and encryption stance, then add only what's specific to it. A separate silo gets you the opposite.

- **Honesty about the gaps is part of the job.** Fleet covers most of what these devices need on Linux, but not all of it, and you should know where the edges are before you roll out.

<a purpose="cta-button" href="https://fleetdm.com/guides/manage-dgx-spark-with-fleet">Read the DGX Spark guide</a>

A few years ago, the most capable computer most employees touched was a laptop that mostly ran a browser. The real work, and the real data, lived in SaaS apps and cloud consoles, each with its own admin panel. Device management followed that shape. Keep the laptop patched and encrypted, and let the SaaS vendors handle the rest.

That shape is changing. NVIDIA's DGX Spark packs a Grace Blackwell GB10 chip and 128 GB of unified memory into a box that fits next to a monitor, and partners ship systems built on the same chip. Teams buy them to run and fine-tune models locally, so the models, the data, and the agents stay on the device. If your device management treats that box as someone else's problem, the most capable computer in the building is also the least managed one.

## The workstation moved back to the desk

For most of the last decade, "workstation" meant a laptop with a nice screen. The compute that mattered was rented by the hour somewhere else. Local AI reverses that. A single DGX Spark holds model weights your company paid to train or license, fine-tuning datasets pulled from internal systems, vector stores built from your documents, and containers pulled from NVIDIA's registry.

None of that shows up in a SaaS admin console, because none of it is SaaS. It's files on a disk, packages in a repository, and processes on a GPU. The only place to see it is the device itself.

NVIDIA says as much: it classifies DGX Spark as a workstation, not a data center system. That classification matters. It means these boxes belong to the team that manages workstations, with the same expectations for inventory, patching, encryption, and audit that every laptop already meets.

## Everyone has an executive assistant now

Executive assistants used to be reserved for the C-suite. A good one reads your email, manages your calendar, drafts replies, and acts on your behalf. They also got a background check, an onboarding, and access that someone deliberately scoped.

Today every engineer, analyst, and designer can have an assistant. It's an AI agent, and increasingly it runs on the person's own workstation. It's wired into repositories, ticketing systems, and internal tools through MCP servers, and it runs with the credentials of the person who launched it. On a DGX Spark, that assistant has enough local compute to do real work without ever calling out to a hosted model.

That's the point. When the assistant, the data it reads, and the actions it takes all live on one device, that device is where access control has to be visible. You need to know what's installed, what's running, which packages came from which repositories, and whether the box still matches the baseline you approved. We've written about finding [shadow AI on your fleet](https://fleetdm.com/articles/shadow-ai-is-already-on-your-fleet). AI workstations are the same problem with more horsepower behind it.

## The vendor tooling assumes you'll build the rest

NVIDIA publishes a thoughtful manageability guide for DGX Spark. It includes tools for device identity, hardware and firmware inventory, driver inventory, health diagnostics, reset reasons, and controlled updates. Each tool writes one bounded JSON document to stdout. The guide is clear about the model: your orchestrator opens SSH, runs the tool, and parses the output.

The guide is also clear about what it doesn't cover. Targeting, scheduling, identity, role-based access, evidence retention, and ticket linkage are all listed as enterprise responsibilities. Vulnerability detection doesn't appear in the model at all. That's a reasonable choice for a hardware vendor that can't assume what you already run. It still leaves your team with a to-do list.

The hardware itself has details that trip up generic Ubuntu tooling. DGX OS moved its kernel from 6.11 to 6.17 to 7.0 within a single major version, so a policy that matches a kernel line breaks on the next update. The NVIDIA driver ships as a pre-built kernel module package, not through DKMS, so scripts written for generic Ubuntu driver installs find nothing. A stock unit trusts several NVIDIA package repositories plus a Canonical PPA. It ships with an unencrypted root filesystem. And some of the obvious GPU health signals, like memory utilization and ECC counters, return "Not Supported" or "N/A" on GB10 because the CPU and GPU share memory.

None of this is a flaw in the device. It's the kind of detail you only learn by managing one, and it's why "we'll SSH in when we need to" stops working at the second or third unit.

## SSH doesn't scale, and state should be a query

The SSH model works for one box on one desk. At fleet scale it becomes infrastructure: a bastion to maintain, service account keys to rotate across sites, a sudo rule for each tool on each device, a place to store every JSON blob, and a script to diff today's blob against last month's. Every one of those is something your team builds and owns.

Fleet inverts the model. Fleet's agent checks in outbound over TLS, so there's no inbound SSH path to secure. It reports device state continuously into a database you can query. Most of what NVIDIA's collectors produce becomes a report you schedule once. Drift stops being a diff of two files and becomes a policy that fails. When a policy fails, an automation can open a Jira or Zendesk ticket, call a webhook, or run a remediation script.

For the things only a vendor binary can do, like reading firmware through `fwupdmgr` or running `nvidia-bug-report.sh`, Fleet runs scripts. NVIDIA's own tools run unmodified, and every run lands in the activity feed with who ran it, where, the exit code, and the output. File carving pulls diagnostics bundles off the device when an incident needs them.

## Frontier devices belong in the same fleet

It's tempting to give a new class of hardware its own management silo. Resist it. A DGX Spark is a Linux workstation, and it should inherit everything your other Linux workstations already have: the baseline, the patch cadence, the encryption stance, the agent configuration, and the people who own them.

In Fleet, that means enrolling the Spark into your existing workstations fleet and adding a dynamic label that matches its hardware model. DGX-specific policies and reports are scoped to that label, so they only grade the devices they apply to. If someone reimages a Spark onto a different OS, it stops matching the label and stops being graded against a baseline that no longer fits.

Everything lives in Git. The approved kernel list, the firmware allow-list, the health policies, and the update scripts are YAML and shell files, reviewed in a pull request and deployed by CI. A new baseline is a merge. A bad baseline is a revert. When an auditor asks who approved the change and what it did on each device, the answer is the Git history plus the activity feed, not a reconstruction from memory.

The same platform also brings vulnerability detection. Fleet matches Ubuntu packages against Canonical's OVAL data, which accounts for backported fixes. That gives a DGX Spark the CVE coverage the vendor model leaves out.

## Be honest about the edges

Fleet covers most of what an AI workstation needs, but some gaps are worth knowing before you roll out:

- **No configuration profiles for Linux.** Configuration enforcement on Linux is policies plus remediation scripts, a detect-and-fix loop rather than a declarative setting.
- **No deadline-based OS updates for Linux.** You can enforce updates with a policy, an automation, and a scheduled batch script run, but there's no end-user countdown like there is on macOS. On a headless box under a desk, that costs you little.
- **Disk encryption has to start at install.** Fleet can escrow LUKS2 keys on Ubuntu, Kubuntu, and Fedora, but escrow runs through Fleet Desktop and needs a user at the keyboard. Encryption can't be added after the OS is installed, so plan it at provisioning.
- **Vulnerability matching is noisier for NVIDIA's pieces.** The DGX kernel is a custom `-nvidia` variant, and NVIDIA's packages come from NVIDIA's repositories, so both fall back to NVD matching instead of OVAL.
- **Fleet doesn't roll back.** Detection is fast and remediation is scriptable, but rollback for DGX OS stays with NVIDIA's mechanism and your rollout rings.
- **It's an agent.** Some regulated environments avoid endpoint agents. Fleet's agent is open source, Fleet Desktop shows users what's collected, and Fleet can run on-premises or air-gapped. If that's still not enough, NVIDIA's SSH model remains a reasonable fallback.

If you already run Canonical Landscape for patching, you don't have to choose. Landscape can keep handling OS updates from a local repository while Fleet reports across your DGX Sparks, Macs, Windows PCs, and everything else.

## The desk is the new data center

The most capable computers in your organization are moving from racks to desks, and they're running assistants that act with real access. Treating them as a side project for whichever team bought them leaves your most valuable data on your least managed hardware. Give frontier hardware the same fleet, baseline, and change control as every other workstation, and then add what's specific to it.

## See it live

The [DGX Spark guide](https://fleetdm.com/guides/manage-dgx-spark-with-fleet) walks through enrollment, labels, reports, policies, diagnostics, and controlled updates, all tested on real hardware.

- [**Get a demo**](https://fleetdm.com/contact) to see Fleet manage Linux workstations alongside macOS and Windows.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop) to learn how to manage your baseline as code.

<meta name="articleTitle" value="Frontier hardware deserves frontier device management">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-09-29">
<meta name="description" value="AI workstations like NVIDIA DGX Spark hold your models, data, and agents. Here's why they belong in the same fleet as every other device.">
