<meta name="category" value="industry news">
<meta name="articleTitle" value="AWS interviewed 154 executives on AI governance and found approval processes still built for six-month IT projects">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-09-28">
<meta name="description" value="AWS's Reimagine 2026 report found AI governance still runs on six-month approval clocks. Here's how Fleet closes that gap with structural controls.">

# AWS interviewed 154 executives on AI governance and found approval processes still built for six-month IT projects

*AWS's Reimagine 2026 report spent nine months asking executives how they govern AI agents, and found most of them running approval processes designed for a different era of software delivery. Its own fix isn't more policy. It's controls an agent can't talk its way around.*

## Key takeaways

- **The mismatch shows up in AWS's own interviews, not just a hunch.** Across 154 executives at 128 organizations in 23 industries, AWS's Reimagine 2026 report found approval processes built for six-month IT programs still governing AI work that finishes in days.
- **A slow approval doesn't stop AI use, it hides it.** When a two-week experiment waits a month for sign-off, the report found some teams stop asking, and one CIO interviewed described the resulting shadow AI as running at a far larger scale than the shadow IT they'd already spent years managing.
- **AWS's own recommendation is structural, not procedural.** The report argues for building controls outside the agent, since an agent can misinterpret whatever rules live in its own prompt or policy document, and for expanding an agent's autonomy only after its reliability is proven, what the report calls treating it like probation.
- **Fleet already shows what an agent can reach, not just what a policy says it's allowed to.** fleetd's `ai_tools` table inventories every AI tool running on a device, and its `risk_flags` column flags shell execution, filesystem writes, and disabled guardrails per tool, so risk gets classified against real capability instead of a self-report.
- **Governance-as-code is how you produce the evidence AWS says most teams can't.** When reports and policies live in Git as YAML and change through pull requests, an agent's proposed change arrives as a diff with an author, a reviewer, and a timestamp attached.
- **A narrow approval gate is fast enough that nobody routes around it.** The alternative to both a month-long review and no review at all is a typed set of agent capabilities with a human sign-off on anything that changes state, the same pattern AWS's report points toward.

<a purpose="cta-button" href="https://fleetdm.com/visibility-and-reporting">See what Fleet can tell you</a>

AWS's Reimagine 2026 report is built from confidential interviews, 45 to 60 minutes each, with 154 executives across 128 organizations in 23 industries, conducted over nine months. The pattern that turns up across those conversations isn't a technology gap. It's a process one: a number of organizations interviewed still route AI agent work through approval processes designed for six-month IT programs, applied to work that a team can finish in days.

That gap doesn't make AI development slower. It makes it invisible.

## Why a slow approval process backfires

The report's own framing is direct: if a two-week experiment waits a month for approval, some teams stop asking for permission. Policy built to catch risk ends up pushing the work underground instead, and it stays there, unreviewed, until something goes wrong.

One CIO interviewed for the report put a number on what that looks like in practice: having already spent years managing shadow IT, they're now facing shadow AI at a scale roughly ten times larger. That's one executive's estimate from one conversation, not a report-wide statistic, but it matches the mechanism the report describes elsewhere: friction in the approved path doesn't reduce adoption, it reroutes it somewhere nobody's watching.

## The fix AWS points to isn't more policy

It would be easy to read that finding as an argument for stricter sign-off, and the report explicitly argues against that. Its recommendation is to build controls outside the AI agent, on the reasoning that an agent can misinterpret or route around rules embedded in its own prompt or policy document, in a way a control built into the surrounding system can't. It also recommends starting new agent deployments with mandatory human approval and expanding autonomy only after the agent's reliability is proven, treating early deployment like probation rather than a one-time review.

That's a description of a technical boundary, not a governance document. It requires knowing, concretely, what an agent can reach on a given device or system, and having a mechanism that reviews and approves what it does before that action lands.

## Seeing what an agent can reach, not what a policy claims

Fleet's agent already answers the first half of that question. fleetd's `ai_tools` table returns one row per AI tool running on a device, across macOS, Windows, and Linux, whether that's an MCP server, an agent CLI, an IDE plugin, or a desktop app, and its `risk_flags` column describes what each one has been granted:

```sql
SELECT type, name, risk_flags, path
FROM ai_tools
WHERE risk_flags != '';
```

`mcp_shell_exec` and `mcp_fs_write` mean an MCP server can execute shell commands or write to the filesystem. `bypass_permissions` and `skip_permissions_runtime` mean someone disabled the guardrails a platform ships by default. That's a risk classification built from what's configured on the device, the input AWS's report says most organizations' review processes are missing when they sort AI projects by risk on paper instead.

## Producing the evidence a policy document can't

The other half of AWS's recommendation, proving reliability before expanding autonomy, needs a record of what an agent actually did and who approved it. That's where most governance breaks down in practice: an agent proposes a change, someone approves it verbally or in a chat thread, and six months later nobody can reconstruct why.

Because Fleet's reports and policies live in Git as YAML and deploy through the same GitOps workflow as the rest of a team's configuration, an agent's proposed change to a report or policy arrives as a pull request rather than a console edit. That diff carries an author, a reviewer, a timestamp, and a revert path, which is the evidence a "treat it like probation" period requires. Without it, probation is just a word in a report.

## The gate has to be faster than the workaround

AWS's report describes teams stopping asking for permission the moment approval takes longer than the work itself. The way to hold a human approval gate without triggering that same workaround is to make the gate narrow: a typed set of capabilities an agent can request, reviewed against what it can reach, with sign-off fast enough that going around it isn't the easier option. That's a smaller ask than a month-long review cycle, and a more durable one than a policy an agent was never going to read.

## See it live

- **Get a demo** to see AI tool inventory and risk flags across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the ground truth piece** on what agentic security needs from your device data: [fleetdm.com/articles/agentic-security-needs-ground-truth](https://fleetdm.com/articles/agentic-security-needs-ground-truth)

## Sources

- AWS Executive Insights, [Announcing the new AWS Reimagine Report on AI](https://aws.amazon.com/blogs/enterprise-strategy/announcing-the-new-aws-reimagine-report-on-ai/).
- Help Net Security, [AI tests the limits of enterprise security governance](https://www.helpnetsecurity.com/2026/09/28/ai-agent-security-governance-aws-report/).
