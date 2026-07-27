# Thumbtack migrates more than 90% of Macs with no IT intervention

## The Challenge

Thumbtack helps homeowners care for and improve their homes by connecting them with local service professionals. The platform links more than 300,000 service businesses to homeowners across the United States, and runs on the engineering practices that define the rest of modern software: infrastructure as code, CI/CD, Git-based workflows, and code review before anything ships.

Apple MDM ran in a different world. Configuration changes happened in a dashboard. One person clicked, one change went out, no review. Feedback came back through a ticket queue. If a configuration was wrong, there was no second set of eyes to catch it before it hit every Mac in the company, and rolling it back was neither fast nor straightforward.

That risk was not theoretical. While updating the macOS nudge profile, the team got the version number right but accidentally set the date field to the past. Instead of giving employees the standard two-week window to upgrade, every Mac in the company was immediately forced to update.


<div purpose="attribution-quote">

*Before Fleet, a misconfigured update could immediately hit every Mac in the company and be very hard to roll back safely. Now every change is reviewed before it ships, and if something’s wrong, we can revert it.*

**Adam Anklewicz**

Manager, IT Systems Engineering, Thumbtack
</div>

Engineering moved at engineering speed. IT moved cautiously, because the cost of getting something wrong was high. The gap was a tax on velocity, and on confidence.

## Why Fleet

Thumbtack evaluated several vendors. Fleet was the one that matched the way Thumbtack’s engineers already work.

With Fleet, device configuration is managed in code, reviewed before it ships, and reverted from version control if something is wrong. The only actions that still run through a manual interface are the ones that should: blocking a device, running a one-time script, executing a query. Everything that touches configuration goes through review.


<div purpose="attribution-quote">

*With Fleet, every change gets a second set of eyes on it before it’s deployed. That alone has prevented mistakes that would have been very expensive to fix.*

**Adam Anklewicz**

Manager, IT Systems Engineering, Thumbtack
</div>

Fleet’s API-first architecture meant configuration changes could be triggered, tested, and deployed inside the same CI/CD workflows engineering already runs. Open-source code and a GitOps model gave the team a full audit history of every change, who made it, and why.

Fleet’s support model removed another tax: instead of waiting 24 hours between ticket replies, the Thumbtack team could reach the Fleet team directly in a shared Slack channel.

<div purpose="attribution-quote">

*With our previous MDM vendor, every reply took 24 hours. With Fleet, I post in our joint Slack channel and within minutes I get a reply and we can have a conversation. Their support is huge - it’s a big selling point for us.*

**Adam Anklewicz**

Manager, IT Systems Engineering, Thumbtack
</div>

In addition, Fleet’s community engagement with the MacAdmins community helped Thumbtack gain confidence that other enterprise customers had had great experiences with the company and its software.

<div purpose="attribution-quote">

*All of Fleet’s issues being public on GitHub is huge because I can just search to see if anyone else is having the same problem.*

**Adam Anklewicz**

Manager, IT Systems Engineering, Thumbtack
</div>

## The outcome

Thumbtack migrated more than 90% of its Mac fleet to Fleet with no manual IT intervention. Direct Slack access to the Fleet team kept the migration unblocked. The team now ships endpoint changes the same way engineering ships product: fast, reviewed, and reversible.

With Fleet, Thumbtack has:

- Device configuration managed in Git, reviewed before it deploys, and revertable in minutes
- More than 90% of Macs migrated with no IT-driven enrollment work
- A clear audit history of every configuration change, who made it, and why
- Endpoint changes triggered, tested, and deployed inside the same CI/CD pipelines engineering already runs
- A direct Slack channel to Fleet’s support team, replacing 24-hour ticket cycles

What changed is not just the tooling. It is the cost of moving. The IT team used to ship cautiously because a single mistake could hit every Mac at once. Now every change goes through review, every change can be reverted, and the team keeps pace with engineering without flying blind.

## Looking ahead

Thumbtack continues to pull more endpoint operations into the same workflows the rest of the company runs on - deeper CI/CD integration, broader policy coverage, and automation that removes IT as a manual dependency anywhere a workflow can be code instead.

For an engineering-driven company, the role of Fleet is clear: not a dashboard to click through, but a layer that runs on the same review, version control, and automation as the rest of the stack. Endpoint management finally moves at the speed of the company.


<meta name="category" value="case study">
<meta name="articleTitle" value="Thumbtack migrates more than 90% of Macs with no IT intervention">
<meta name="description" value="Thumbtack migrated more than 90% of Macs to Fleet with no IT intervention and now manages devices with GitOps and fast Slack support.">


<meta name="publishedOn" value="2026-03-13">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="thumbtack-logo-197x40@2x.png">
<meta name="quoteAuthorImageFilename" value="adam-anklewicz-120x120@2x.png">
<meta name="quoteAuthorName" value="Adam Anklewicz">
<meta name="quoteAuthorJobTitle" value="Manager, IT Systems Engineering, Thumbtack">
<meta name="quoteContent" value="With Fleet, every change gets a second set of eyes on it before it's deployed. That alone has prevented mistakes that would have been very expensive to fix.">

<meta name="companyName" value="Thumbtack">
<meta name="companyInfo" value="Thumbtack helps homeowners care for and improve their homes by connecting them with local service professionals. Through its platform, people get guidance on what projects to do, when to do them, and who to hire from a community of more than 300,000 service businesses across the United States.">

<meta name="summaryChallenge" value="Configuration changes happened in a dashboard. One person clicked, one change went out, no review. Feedback came through a ticket queue. If a configuration was wrong, there was no second set of eyes to catch it before it hit every Mac in the company, and rolling it back was neither fast nor straightforward.">
<meta name="summarySolution" value="Thumbtack chose Fleet to bring code review, version control, and rollback to device management. Fleet’s API-first architecture meant configuration changes could be triggered, tested, and deployed inside the same CI/CD workflows engineering already runs.">
<meta name="summaryKeyResults" value="Device configuration managed in Git, reviewed before it deploys, and revertable in minutes.; A clear audit history of every configuration change, who made it, and why.; Endpoint changes triggered, tested, and deployed inside the same CI/CD pipelines engineering already runs.; A direct Slack channel to Fleet’s support team, replacing 24-hour ticket cycles.; More than 90% of Macs migrated with no IT-driven enrollment work.;">
