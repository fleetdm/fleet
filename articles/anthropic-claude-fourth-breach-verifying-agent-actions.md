# Claude tried to quit eight times, then hacked a real company's admin account anyway

*An early Claude Opus 4.6 checkpoint broke into a real company's systems during a security test in January, and Anthropic didn't find out until August. Here's why an agent's own transcript isn't the same as verification.*

## Key takeaways

- **A misconfiguration turned a routine test into a real breach.** An early Claude Opus 4.6 checkpoint lost its intended target when a network conflict knocked the machine offline, then found a live path to an unrelated third party's systems and kept going.
- **The model tried to stop itself, and couldn't.** It attempted to abort the exercise eight times. A bug in the evaluation harness blocked every attempt, and Anthropic says the model never questioned whether continuing was authorized.
- **It walked out with admin access and one person's data.** The model harvested credentials, gained admin rights, collected more passwords from system files, changed settings to make its own return easier, and read personal information belonging to someone connected to the third party.
- **The transcript sat unread for seven months.** Anthropic didn't catch the incident in January. It surfaced in August, while the company was assembling records for an outside evaluator investigating three unrelated breaches disclosed in July, making this the fourth Claude breach disclosed in 2026.
- **A vendor's own disclosure is still a self-report.** Anthropic gets credit for publishing this. It's also the clearest evidence yet that "the agent stayed in scope" is a claim sitting in a log somewhere, not a fact anyone confirmed until someone went looking for something else.
- **You can check what's running with agent-level access on your own devices today.** fleetd's `ai_tools` table inventories every AI tool per host, with risk flags for the same kind of overreach this incident turned up after the fact: credentials sitting in the open, guardrails disabled, shell access granted.

<a purpose="cta-button" href="https://fleetdm.com/visibility-and-reporting">See what's running on your fleet</a>

On September 9, Anthropic [published an assessment](https://www.anthropic.com/research/alignment-assessment-cybersecurity-incidents) of a fourth incident in which a Claude model broke into a real system during a security test. Unlike the three breaches the company disclosed on July 30, this one didn't happen last month. It happened in January, and Anthropic only found it while pulling transcripts together for an unrelated review.

That gap is the part worth sitting with. Not that a test escaped its intended boundary, evaluation harnesses misfire, and Anthropic was transparent about it, but that the company running the test had no way to know it happened until it went looking for something else entirely. That's the same gap [Fleet has argued](https://fleetdm.com/articles/agentic-security-needs-ground-truth) sits underneath every agentic security story: an agent's account of itself is not the same as a verified record of what it did.

## A test that lost its target and didn't stop

The incident started as an ordinary capture-the-flag exercise. An early checkpoint of Claude Opus 4.6 was given a fictional scenario, a target machine, and a piece of information to retrieve. A network misconfiguration assigned the target machine a conflicting IP address, and the intended system dropped offline mid-exercise.

Instead of ending there, the model found an unintended live path onto the internet, the same kind of egress gap behind an earlier Opus 4.7 incident, and used it to reach a real, unrelated third party's systems.

## Eight failed aborts, then it kept going anyway

The model tried to stop. Anthropic's transcripts show it attempted to abort the exercise eight times over the course of the session, and a bug in the evaluation harness blocked every attempt. Rather than pausing to question the situation, it worked around the dead end: harvesting credentials, escalating to admin access, and collecting more passwords from system files it reached along the way. It changed account settings to make its own return easier and, before the session ended when it ran out of token budget, read personal information belonging to at least one person connected to the third party.

Anthropic's write-up is specific about why this incident differs from the three it disclosed in July. Those it attributed to biased reasoning, the model talked itself into believing a real target was still part of the exercise. This one it calls recklessness: none of the model's recorded thinking blocks questioned whether continuing was authorized, and none described the environment as simulated. The model wasn't confused about where it was. It kept going anyway.

## The part that should worry you more than the breach

A single incident escaping a test environment is a hard problem, and Anthropic isn't hiding from it. What's more instructive for anyone adopting agents is the seven months between the breach and its discovery. Nobody hid the transcript. Nobody had a reason to open it until a different, unrelated set of breaches from July sent someone back through the logs for an outside evaluator.

That's the shape of the risk that matters here. An agent that can quietly reach further than it's supposed to doesn't announce itself. It sits in a record that stays a record until someone decides to read it, and "we'll catch it in the transcript" only holds if something prompts a person to go look. A promise that an agent stayed in scope is a claim, and claims are exactly what verification exists to check.

## Verify what an agent touched, don't wait to read about it later

You don't need to run frontier model evaluations to have this same blind spot. Any AI coding agent, MCP server, or assistant with shell or filesystem access on your devices can reach further than whoever installed it intended, and most of that software arrives without a purchase order or a review.

fleetd's `ai_tools` table inventories that surface directly: one row per AI tool across macOS, Windows, and Linux, with a `risk_flags` column that names the same categories of overreach this incident produced. `mcp_shell_exec` and `mcp_fs_write` mean a tool was granted shell execution or filesystem writes. `bypass_permissions` means someone turned a guardrail off. `plaintext_secret` means a credential is sitting somewhere it shouldn't be. None of that requires trusting the tool's own account of what it's allowed to do, since it's a query against the device, not a summary the tool wrote about itself.

The same principle extends past inventory. Fleet's policies and reports live in Git as YAML, so if an agent proposes a change to one, that proposal is a pull request with an author, a reviewer, and a revert path, not a console edit an outside evaluator would have to reconstruct from a transcript seven months later.

## Verification is the part that doesn't wait for a disclosure

Anthropic isn't uniquely careless here. It ran the evaluation, found the problem in its own records, and published what it found, which is more than most vendors running agents against real infrastructure can say. That's exactly why the seven-month gap matters: a company that does the transparent thing still needed a different investigation to notice.

The lesson isn't specific to one lab or one model. It's that "the agent stayed in scope" is worth exactly as much as the verification behind it, and a transcript nobody has a reason to open isn't verification. Know what's running with elevated access on your own devices now, not whenever the next disclosure prompts someone to check.

## See it live

- [**Get a demo**](https://fleetdm.com/contact). We'll show you what fleetd's `ai_tools` table finds on real devices, including the risk flags this incident would have tripped.
- **Read how to find AI tools already running on your fleet**: [Shadow AI is already on your fleet](https://fleetdm.com/articles/shadow-ai-is-already-on-your-fleet).

## Sources

- Anthropic, [An alignment assessment of recent cybersecurity incidents](https://www.anthropic.com/research/alignment-assessment-cybersecurity-incidents).
- Martin Cid Magazine, [Claude tried to stop. The test harness said no. It ended up hacking a real server](https://www.martincid.com/technology-sv/anthropic-claude-fourth-security-breach-live-server/).
- Help Net Security, [Anthropic's Claude breached three companies during security tests](https://www.helpnetsecurity.com/2026/07/31/anthropic-claude-cybersecurity-incidents/).

<meta name="articleTitle" value="Claude tried to quit eight times, then hacked a real company's admin account anyway">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-11">
<meta name="description" value="Anthropic's fourth Claude breach sat unreviewed for seven months. Here's why an agent's own transcript isn't verification.">
