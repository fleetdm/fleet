# Why AI-powered device management requires GitOps
## Device management is about to change. AI is why. GitOps is how.

The slow part is gone.

I’ve written before about why device management needs to move to GitOps. The case for configuration as code - auditability, rollbacks, code review, a single source of truth - stands on its own. But something has changed in the last two years that makes that case even more urgent, and it’s not about the benefits of version control anymore.

It’s about what becomes possible once your config is in a repo.

The thing that made GitOps-based device management feel like homework - knowing the YAML schema, remembering whether the right key is `labels_include_any` or `labels_any_include`, hunting down the right osquery table, copying an example from the docs and manually adapting it - that part is gone. AI ate it.

The barrier was never philosophical. Everyone understood why config in a repo was better than config in a GUI. The barrier was ergonomic. Writing YAML is fine in theory and tedious in practice, and tedium loses to the GUI console every single time.

That trade-off just inverted.

## The loop that now closes
Point an AI at a real MDM configuration repo and ask it to add a policy that checks whether full-disk encryption is enabled across the Finance department. In seconds, you have a pull request.

Not a snippet to figure out where to paste. An actual PR. The policy lands in the right directory because that’s where the other policies live. It’s scoped to Finance using the label pattern that already exists in the repo. It sets the severity. It links back to the original Slack thread for the paper trail. None of that was in the prompt - the AI inferred it from the structure of the repo, because the structure is legible. Text, in a hierarchy, with conventions an AI can read and pattern-match against.

Try the same thing against a GUI-based MDM. The loop doesn’t close. There’s no schema to learn from, no examples to pattern-match against, no place for the output to land that doesn’t require a human to manually translate it into clicks. The AI has nothing to grab onto.

This is the line: without GitOps, there is no AI-accelerated device management. It doesn’t matter which AI you prefer. If your source of truth is a GUI, the whole value proposition evaporates. And if your MDM doesn’t support a real config-as-code workflow, that’s a signal about your MDM.

## The safety model holds
When people hear “AI writes your config,” the first instinct is fear. What if it gets it wrong? What if something bad deploys because nobody caught a mistake?

That instinct is right. But the GitOps workflow already handles it.

The AI writes the diff. The diff lands in a pull request. A human reviews the PR before it touches a single device. The AI is a very fast junior engineer who can only submit pull requests - it can’t merge, can’t deploy, can’t change anything on any machine. That’s the same trust model you’d apply to a new human engineer, and it’s the right one. You get the speed of AI authoring and the safety of human review at the same time, and the only reason both are possible simultaneously is that your config is text in a repo that supports the review workflow.

## Where this goes next
Here’s what I think the next few years look like. I’m genuinely uncertain about some of it - but the direction feels clear.

### From syntax help to intent translation. 
Right now, the workflow is mostly “I know what I want, I just need help writing it.” That’s the early version. The next version is “I need our macOS fleet to pass SOC 2 by end of quarter” - and the AI pulls the relevant controls, maps them to your existing policy structure, identifies the gaps, and opens a PR for each one. You describe the outcome. The AI does the translation all the way down. That only works if your existing config is legible to the AI. Which means text, in a repo, with consistent conventions.

### Drift detection that talks back. 
Your MDM already tells you when a device falls out of compliance. The natural extension is that it doesn’t just alert - it proposes the remediation. Device X failed the encryption check. Here’s the PR that would fix it. Want me to open one? The human still reviews. The human still merges. But the triage loop - figure out what broke, figure out the fix, write it, get it reviewed - compresses from hours to minutes.

### Natural language queries. 
osquery is a query language for your device state. Writing good osquery currently requires schema knowledge, an understanding of what’s available, and enough SQL fluency to avoid killing performance. That skill floor is dropping fast. “Show me devices in Finance with Gatekeeper disabled that haven’t checked in for 72 hours” is something a non-technical person can type. The AI turns it into a query. You get a real answer against real device state. The gap between “I have a security question” and “I have an answer” has historically been gated by technical fluency. That gate is going away - which means security teams and compliance analysts who couldn’t self-serve before soon will be able to.

### Autonomous remediation, eventually. 
This is the part where the trust model gets complicated. Everything above keeps a human in the loop at the merge step. I think that’s right for now.

But I don’t think it stays that way forever. Some fixes are low-risk, well-understood, and time-sensitive enough that waiting for a human to review a PR is the wrong trade-off. Renewing a certificate before it expires. Patching a known critical CVE on devices already approved for auto-update.

The question isn’t whether autonomous remediation ever makes sense. It’s which cases, under what conditions, with what audit trail, and who bears responsibility when something goes wrong. Those are governance questions, not technology questions. The tooling already exists. The organizational frameworks for deciding when it’s safe are still catching up.

The GitOps model handles this cleanly when you’re ready. You define the automation rules as config. You review changes to those rules in PRs. The automation acts within the boundaries humans set. Still reviewable. Still auditable. Still legible.

Not legible if it’s all in a GUI.

## What this means for the work
If AI gets good at authoring configuration, a reasonable question is: what do the people who used to author configuration do next?

The work that goes away is the transcription layer - translating a requirement into YAML, looking up the schema, remembering the syntax. That was always the least interesting part. It was the overhead between having a good idea and getting it into production.

What doesn’t go away is judgment. Knowing which policies actually matter. Understanding your org’s specific risk tolerance. Catching the AI when it’s technically correct but contextually wrong. Recognizing when a proposed change has implications the PR description didn’t mention. Designing the repo structure and conventions that make the AI’s output consistent and reviewable in the first place. Those are still human problems.

Maybe the shape of this work shifts toward more architecture and less implementation. More review and less authoring. More judgment calls and fewer syntax lookups. Maybe that’s better - I’m genuinely not sure. I know people who find real satisfaction in the careful work of writing a well-crafted policy. I also know people who’ve been doing YAML transcription for years when what they actually wanted was to do the harder thinking. If this changes that ratio, it seems like a net positive. But the transition is real, and not everyone lands somewhere better automatically.

The orgs that navigate this well will treat the freed-up capacity as an opportunity to do harder work, not as an opportunity to reduce headcount. Whether that actually happens isn’t a technology question.

## The question
Picture your own org. Your own MDM. Your own team.

Could you do this on Monday?

If your config is in a GUI, the answer is no - not because the AI isn’t capable, but because there’s no surface for it to work against.

Getting your config into a repo isn’t a nice-to-have anymore. It’s the prerequisite for everything that’s about to happen in this space. The orgs that have been building GitOps discipline are positioned to move when the next capability lands. The orgs with config in a GUI are going to be doing a migration while everyone else is already running.

The slow part is gone. The question is whether you’re set up to benefit from that.



<meta name="articleTitle" value="Why AI-powered device management requires GitOps">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2026-04-17">
<meta name="category" value="articles">
<meta name="description" value="Device management is about to change. AI is why. GitOps is how.">
