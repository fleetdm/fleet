# How Campus runs IT for higher education with a team of one, Fleet, and AI

## The challenge

Campus set out to make a quality education more accessible and affordable. Today, it's an accredited, technology-driven college offering two-year, live online associate degrees, with an affordable tuition model that allows eligible students to pay nothing out of pocket after financial aid. That puts Campus in an unusual position. It has to meet the compliance requirements of a college while also managing the systems and processes around federal financial aid.

For a team of one, the old setup was creating more work than it was saving. Jamf was almost entirely ClickOps, which made it difficult to see what was changing, who was changing it, and why.

<div purpose="attribution-quote">

*I was really tired of ClickOps. With Jamf, the only way to know what was going on was to read through a full audit log every day. And anyone on the team could make a change without approval.*

**Robbie Trencheny**

Head of Infrastructure and IT, Campus
</div>

The pricing was another problem. Jamf's education rates made the basics affordable, but the security features Campus needed required significant upgrades, largely offsetting the education discount.

<div purpose="attribution-quote">

*It was a few dollars a host for the basics, and ten times that once you wanted SSO and the advanced security features.*

**Robbie Trencheny**

Head of Infrastructure and IT, Campus
</div>

Then there was the cost of getting something wrong. When Robbie accidentally disabled content caching on the wrong hosts, fixing it meant pulling the setting out of a security profile, creating a new profile, and reassigning everything manually. What should have been a simple change turned into 30 minutes of careful clicking and review.

With the Jamf renewal approaching and a wave of compliance work ahead, Robbie decided it was time for a different approach.

## Why Fleet

There wasn't a bake-off. Robbie had known about Fleet since 2023, and when the Jamf renewal came up, he already knew where he wanted to go.

What gave him confidence was that he could see the product evolving in the open. Fleet's GitHub repository gave him a clear view of what was happening, from open issues for bugs and feature requests to merged pull requests showing what was shipping in each release and a public roadmap showing where Fleet was headed.

<div purpose="attribution-quote">

*There was no POC, and no other vendor to evaluate. I know exactly where Fleet is going because it's all out in the open.*

**Robbie Trencheny**

Head of Infrastructure and IT, Campus
</div>

The biggest draw was GitOps. As an infrastructure engineer, Robbie wanted device management to work like the rest of his stack: version-controlled, peer-reviewable, and driven by code instead of clicks. And as a team of one and the budget owner, he didn't have a long internal approval process to navigate.

The switch itself was straightforward. Robbie credits Fleet's sales and support team with making the transition an easy one.

## The solution

The implementation meant treating device management like the rest of the infrastructure. Today, every change goes through a GitHub pull request and an automated CodeRabbit review, while Robbie uses AI to write, troubleshoot, and translate the work. For a team of one, that changes what's possible.

Instead of rebuilding his Jamf setup by hand, Robbie used AI to translate his existing profiles and configurations into Fleet.

<div purpose="attribution-quote">

*When our Fleet software page was loading slowly, Claude pulled the Google Cloud logs and our Fleet logs from Datadog, worked out what was happening, and opened a bug report on Fleet's repo for me.*

**Robbie Trencheny**

Head of Infrastructure and IT, Campus
</div>

The same philosophy carries into onboarding. New hires get the software and identity groups they need without Robbie having to manually work through the process. A new hire in Campus's HR system triggers a device shipment through its inventory system. The workflow then prestages the right group in Fleet, and when the laptop first powers on, it notifies the user's manager and HR in Slack.

<div purpose="attribution-quote">

*Someone's up and running on their laptop in about 20 minutes, with the right software and the right identity groups. In Jamf that was a day or two of lost productivity.*

**Robbie Trencheny**

Head of Infrastructure and IT, Campus
</div>

That approach is also becoming the foundation for compliance. Campus has a long list of requirements ahead, including SOC 2, FERPA, GLBA, GDPR, and NIST 800-171, driven in part by its federal financial-aid funding. Rather than manage those controls as another set of checklists, Robbie wants to manage them through Fleet GitOps.

## The results

The migration took two weeks. Robbie moved engineering first, then the rest of the team, with a firm deadline and clear communication throughout.

The bigger difference was what came after. With Jamf, a bad configuration profile could take down a server, and limited visibility made even routine changes feel risky.

<div purpose="attribution-quote">

*In Jamf, a bad config push could actually cripple the server. Then I was at the mercy of Jamf support to get back up and running. Now I own my own destiny. We run the stack in our self-hosted environment.*

**Robbie Trencheny**

Head of Infrastructure and IT, Campus
</div>

GitOps gave Robbie a different way to work. Every change is reviewable and traceable, so he can move quickly without worrying about what a wrong click might break. And for the people he supports, the best outcome is that nothing feels different at all.

<div purpose="attribution-quote">

*Honestly, the users don't notice. That's the goal. It just works, and I can move faster because AI is doing so much of the development for us.*

**Robbie Trencheny**

Head of Infrastructure and IT, Campus
</div>

With Fleet, Campus has:

<div purpose="checklist">

Migrated off Jamf in two weeks

Cut new-hire setup from two days to about 20 minutes

Reduced a 30-minute configuration task to about two minutes with AI and GitOps

600 macOS and Windows devices managed as code through Fleet GitOps

A path toward automating hundreds of NIST 800-171 controls
</div>

<meta name="category" value="case study">
<meta name="articleTitle" value="How Campus runs IT for higher education with a team of one, Fleet, and AI">
<meta name="description" value="Campus migrated off Jamf in two weeks and now runs 600 devices as code with a one-person IT team, Fleet GitOps, and AI.">

<meta name="publishedOn" value="2026-08-24">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="campus-logo-182x40@2x.png">
<meta name="quoteAuthorImageFilename" value="robbie-trencheny-120x120@2x.png">
<meta name="quoteAuthorName" value="Robbie Trencheny">
<meta name="quoteAuthorJobTitle" value="Head of Infrastructure and IT, Campus">
<meta name="quoteContent" value="“Honestly, the users don't notice. That's the goal. It just works, and I can move faster because AI is doing so much of the development for us.”">

<meta name="companyName" value="Campus">
<meta name="companyInfo" value="Campus is an accredited, two-year online college built around making a world-class education more accessible and affordable. Students can earn associate degrees in business, IT, and healthcare through live online classes, with tuition priced low enough that eligible students can attend with little or no out-of-pocket tuition after financial aid.">
<meta name="companyInfoLineTwo" value="A one-person IT team, now onboarding team member number two, manages 600 macOS and Windows devices as code through Fleet GitOps.">

<meta name="summaryChallenge" value="Jamf was almost entirely ClickOps, so it was hard to see what was changing, who changed it, and why, and anyone on the team could push a change without approval. The security features Campus needed cost ten times the base rate, and a single wrong click could take 30 minutes to unwind.">
<meta name="summarySolution" value="Campus manages devices as code with Fleet GitOps. Every change goes through a GitHub pull request and an automated CodeRabbit review, and Robbie uses AI to write, troubleshoot, and translate configuration. Onboarding is wired into the HR and inventory systems, and compliance controls are managed the same way.">
<meta name="summaryKeyResults" value="Migrated off Jamf in two weeks; Cut new-hire setup from two days to about 20 minutes; Reduced a 30-minute configuration task to about two minutes with AI and GitOps; 600 macOS and Windows devices managed as code through Fleet GitOps; A path toward automating hundreds of NIST 800-171 controls">
