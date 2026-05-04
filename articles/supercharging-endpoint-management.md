# Supercharging endpoint management with AI assistants and event-driven automation

The engineering-driven stack is ready. Now the question is who gets to use it. With AI assistants and event-driven automation, the answer is everyone on your team.

### Links to article series:

- Part 1: [The confidence gap: why IT leaders are abandoning legacy endpoint management](https://fleetdm.com/articles/the-confidence-gap)
- Part 2: [Rethinking endpoint management: Fleet, osquery and Infrastructure as Code](https://fleetdm.com/articles/rethinking-endpoint-management)
- Part 3: Supercharging endpoint management with AI assistants and event-driven automation

## Where we left off

In [Part 1](https://fleetdm.com/articles/the-confidence-gap), we covered the confidence gap: the void between what legacy MDM dashboards report and what is actually happening on your devices. In [Part 2](https://fleetdm.com/articles/rethinking-endpoint-management), we laid out the modern stack that closes that gap: real-time device visibility, Fleet for orchestration, and Infrastructure as Code (IaC) for rigor.

That stack is powerful. It is also, on its face, intimidating. Writing live queries against hundreds of device state tables. Authoring YAML configuration profiles. Opening pull requests. Running CI pipelines. For an IT team whose day job is answering tickets and keeping laptops patched, this reads like a DevOps job description.

So the real question is: how do you give every IT admin, help desk engineer, and compliance analyst the leverage of a senior platform engineer - without first sending them through a six-month GitOps bootcamp?

The answer is a shift already underway at forward-thinking IT organizations. AI assistants handle the translation from intent to code. Event-driven automation handles the translation from "something happened" to "the right thing got done." Fleet sits in the middle as the control plane that makes both possible.

## Why this moment is different

Two things changed in the last 18 months that matter for IT.

First, AI coding assistants got good enough to write production-quality configuration when pointed at structured, well-documented systems. Not good enough to replace engineers, but good enough to lower the skill floor for anyone writing queries, YAML, or a CI pipeline. The key phrase is structured and well-documented. AI works well against code, schemas, and text files. It falls apart against GUI click paths, proprietary APIs, and black-box dashboards.

Second, the tools IT teams already use - Slack, Microsoft Teams, GitHub, Okta, Workday - all speak webhooks and APIs. Events from one system can trigger actions in another without a human in the middle, provided the middle system has an open API. Fleet is API-first, so every action you can take in the UI is also available as an API call.

Put these together and the shape of modern IT work changes. Changes get proposed in chat. AI writes the YAML. A human reviews the pull request. CI deploys. Events from your IdP or HRIS trigger automated workflows. The IT team stops being a bottleneck and starts being a reviewer of high-leverage changes.

## AI assistants: from intent to code

Here is the practical picture. An IT admin opens Slack and types:

"Make sure every workstation in the Finance team is on macOS 15.2 or later. Give people a 3-day grace period before enforcing."

An AI assistant connected to your Fleet GitOps repo reads the existing structure - the directory conventions, the label schema, how other policies are written - and proposes a pull request. The PR contains a new policy file, scoped to the Finance team label, with the grace period set. It lands in the right directory because that is where the other policies live. It sets the severity correctly. Its back to the Slack thread for context.

The human reviews the PR. Edits anything that is off. Approves. Merges. Fleet applies the change across devices within minutes.

None of the structural decisions were in the prompt. The AI inferred them from the repo, because the repo is legible. Text, in a hierarchy, with conventions a model can pattern-match against.

### Example 1: Patching a CVE on a Friday afternoon

A CVE drops. It affects a popular browser extension your employees use. Your CISO wants to know by end-of-day whether you are exposed and, if you are, how fast you can be clean.

**Old IT workflow:** An admin logs into the MDM console. Opens a search. Realizes the console does not index browser extensions. Writes a one-off script. Deploys it via a custom command. Waits hours for results to trickle in. Manually compiles a list. Opens another console to push a remediation. Sends a status email at 9 PM.

**New IT workflow with Fleet + AI:** The admin types into Slack:

"Find every managed device with the affected browser extension installed. For any device on a vulnerable version, schedule a forced update within 24 hours."

The AI assistant generates a live query against Fleet's browser extension inventory, opens a PR with both the query and a remediation policy, and includes an AI-written description of what the change does and why. A senior engineer reviews it in five minutes. Fleet runs the query live, identifies the exposed devices in under 30 seconds, and the remediation rolls out with a grace period. The CISO gets a dashboard link, not a status email.

This is exactly the scenario Fleet was built for: live queries against all devices in seconds, remediation expressed as code, and a version-controlled paper trail of the entire response.

### Example 2: Closing the last mile on a patch rollout

Part 1 of this series mentioned a pattern Mike McNeil calls "the last mile" - the gap between "we rolled out the patch" and "every single device actually received it." Legacy tools leave 25% of the workforce exposed for days because the failures are buried in logs nobody reads.

With Fleet, every host exposes its full MDM command history, script exit codes, and configuration status. An AI assistant can tail this data and write a summary:

"3 devices in the Engineering team failed to apply the latest FileVault profile. 2 failed because the user had not rebooted. 1 failed with exit code 70, which means disk full. Open a ticket for the disk-full device and send a Slack nudge to the other two?"

The admin says yes. The automation opens a Jira ticket with the exact device ID and error code, posts a direct message to the two users with a short explanation, and re-triggers the profile push the next time those devices check in. The last mile closes in minutes, not days.

## What Fleet looks like through an AI lens

Fleet's design choices make this work. The product is API-first, so the AI has one consistent surface to write against rather than six per-OS APIs. Configuration lives in YAML files in a git repo, which is exactly the format AI models handle best. Device data is exposed through a stable, documented schema of hundreds of tables, so the model can reason about which data is available and how to ask for it. The open-source codebase means there is no black box the AI has to guess about.

This is why Fleet's position is that without GitOps, there is no AI-accelerated device management. Without structured, legible inputs, the AI has nothing to grab onto.

## Event-driven automation: from trigger to action

AI assistants change how changes get authored. Event-driven automation changes when actions happen.

The pattern is simple. An event fires in one system. A webhook hits Fleet's API. Fleet runs a policy, a query, or a workflow. The result either completes the task or routes it to a human for review.

Three examples from real IT workflows:

### Onboarding a new hire

A record gets created in Workday. The HRIS fires a webhook to an automation runner. The runner calls Fleet's API to pre-provision a device profile tied to the new hire's team label. Fleet then applies team-specific configuration - software, restrictions, Wi-Fi profiles - the moment that device enrolls. The new hire's laptop arrives configured on day one, with zero tickets and zero manual clicks.

### Offboarding a departure

The same HRIS event fires a departure webhook. Fleet receives the API call, locks the device, triggers a secure wipe once the device is online, and posts a completion message to a shared Slack channel for the IT lead to verify. The device never sits unmanaged in a closet for three weeks.

### Compliance drift

A Fleet policy detects that a device has drifted out of CIS benchmark compliance. Fleet fires a webhook to your ticketing system and opens an incident scoped to the device owner. If the drift is auto-remediable - say, a setting that got toggled - an automation runs a remediation script and re-checks the policy. If it is not, the ticket routes to a human with the exact check that failed and the current device state attached.

None of this requires writing new management software. It requires a control plane whose actions are available via API, and an event bus connecting your IT tools together. Fleet provides the first. Your existing stack provides the second.

## Putting the pieces together: the new IT workflow

Here is the full picture of what modern, engineering-driven device management looks like when AI assistants and event-driven automation are layered onto the stack from Part 2:

1. **Intent is entered in plain English.** An engineer, admin, or even a compliance analyst describes what they need in Slack, Microsoft Teams, or a chat interface.
2. **AI translates to code.** An AI assistant reads the Fleet GitOps repo and opens a pull request with the right YAML, query, or script.
3. **Humans review.** A senior engineer approves, edits, or rejects. The pull request history becomes the audit trail.
4. **CI deploys.** `fleetctl` applies the change via Fleet's API. Devices update in minutes.
5. **Events trigger follow-through.** Webhooks from HRIS, IdP, ticketing, and Fleet itself drive automated workflows. Compliance drift, new hires, departures, and CVEs all route through the same stack.
6. **Fleet reports the ground truth.** Live queries confirm that intent matches reality on every device.

The common thread: every step is visible, version-controlled, and reversible. You see every change, undo any error, and repeat every success.

## Why this matters for IT leaders

Three things change for IT organizations that adopt this model.

**IT stops being a bottleneck.** The team's role shifts from executing tickets to reviewing high-leverage changes. The ratio of work shipped per engineer goes up without anyone working more hours.

**The skill floor drops.** Junior admins can ship production configuration on day one because the AI handles the syntax and the GitOps workflow handles the safety. Senior engineers spend their time on architecture, not on writing the 400th query of the year.

**Audits become boring.** Every change is a git commit with an author, a timestamp, a PR description, and a review trail. Questions from security, finance, or the board get answered in minutes using live Fleet reports rather than weeks of log archaeology.

This is also how IT stops being seen as a cost center. When changes ship in minutes instead of weeks, when audits are painless, and when new hires are productive on day one, the business sees IT as leverage - not overhead.

## Closing the series

Legacy MDM asked IT leaders to trust a black box. The confidence gap was the cost of that trust.

The modern stack - live device visibility, Fleet for orchestration, IaC for rigor, AI for leverage, and events for follow-through - replaces blind trust with transparent, auditable, automated control.

You do not need to rebuild your team to get there. You need an open, API-first control plane that AI models can read and event systems can call. That is what Fleet was designed to be.

If you are an IT or Security leader looking to close the confidence gap, consolidate your tool sprawl, and give your team the leverage of modern engineering practices, [get a demo](https://fleetdm.com/contact) or [join a GitOps workshop](https://fleetdm.com/gitops-workshop). The tools are ready. The practices are proven. The only question is how quickly you move.

<meta name="articleTitle" value="Supercharging endpoint management with AI assistants and event-driven automation">
<meta name="authorFullName" value="Ashish Kuthiala, CMO, Fleet Device Management">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-04-22">
<meta name="description" value="Part 3 of 3 - Article series on supercharging modern endpoint management.">
