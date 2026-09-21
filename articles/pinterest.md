# Pinterest’s IT team manages 13,000 devices and engineered human error out of the job

## The challenge

Pinterest's IT Platform Engineering team manages endpoints and applications that the whole company depends on. They are engineers, but their MDM didn't let them work like engineers.

For eight years, everything ran on Omnissa Workspace ONE. The team was never happy or unhappy with it, but the platform was slow and the UI a little clunky. There was no native GitOps functionality, and when they tried to build their own workflow around it, the inflexible API got in the way. Worst of all, getting technical support was painful.

<div purpose="attribution-quote">

*Support responses could take hours or, more often, several days.*

**Dan Hutchings**

IT Platform Engineer, Pinterest
</div>

Without peer reviewed changes, a configuration update made in the wrong order could create more work for the team to manually undo. Support gaps could turn small problems into big emergencies, and waiting days to get assistance was out of the question.

A few years into their partnership, Broadcom's late-2023 acquisition of VMware put the product's future in doubt. For Pinterest, it was a priority to replace their MDM.

## Why Fleet

The Pinterest security team built an internal zero-trust system that relied on osquery for device telemetry. When the team moved that data source to Fleet, they got a close look at how the company operated: open, collaborative, responsive support, and willing to build what customers needed.

Those values fit well with Pinterest's engineering culture. Much of their macOS and Windows tooling was already open source, so moving to an open MDM felt like a natural decision.

<div purpose="attribution-quote">

*A lot of our tech stack is already open source, so Fleet fit right in. A lot of the time we don't even need to contact support. We can read the code or point AI at it, and answer the question ourselves.*

**Dustin Davis**

Sr. Manager, IT Platform Engineering, Pinterest
</div>

GitOps sealed the deal. Managing MDM as code was a top priority on the team's roadmap for years, and Fleet made it a real possibility. Before, GitOps meant copy-pasting peer-reviewed XML into a clunky UI which was human error prone. Today, Security and other teams contribute to how devices are managed just by opening a pull request. GitOps is native and their favorite feature of Fleet.

The API told the same story. Where their previous MDM had poorly documented API’s with support staff who were frequently unsure of how to use them, Fleet's just worked, especially at scale.

<div purpose="attribution-quote">

*The API docs are accurate, reliable, have great examples of expected output, and we never hit a wall. We can push one config profile to 6,500 Macs and just hit deploy. On the old platform that took days, batched by a few hundred until completion or sometimes a thousand followed by a slow trickle.*

**Dan Hutchings**

IT Platform Engineer, Pinterest
</div>

## The solution

**Device management as code.** Everything runs through GitOps now, with Munki handling software and a pull-request requiring a peer review on every change. Security defines the posture; IT turns that into config management code and a great user experience. Nothing ships without review, which is the whole point.

**Zero trust, built to be preventative.** Pinterest's internal system enforces access rules: if a device is out of posture, the user can't reach Okta or any other core internal resources. Rather than let people trip those rules, the IT team designs around them. If running an SSH server would get someone blocked, they prevent the SSH server from being run in the first place, so security stays tight and employees don't hit a wall.

**Onboarding that runs on rails.** A homegrown IT operations dashboard ties together Okta, Slack, Google Workspace, and Fleet, checking certificate health and pushing new certs, with onboarding and offboarding automated against Workday.

**Less privilege, less babysitting.** This one was a deliberate choice with a real payoff. In the old console, dozens of IT ops people held admin accounts that constantly had to be managed as staff came and went.

<div purpose="attribution-quote">

*We deliberately didn't recreate all those admin accounts in Fleet. We used to spend real time babysitting permissions through constant turnover. That busy work is just gone. We have better roles-based access control, and the peer review process ensures that everyone gets what they need quickly from our team.*

**Dustin Davis**

Sr. Manager, IT Platform Engineering, Pinterest
</div>

## The results

The clearest win is the team's transition to a GitOps workflow.

<div purpose="attribution-quote">

*The chance for a small human mistake to cause a big, time-based problem has dropped altogether. If you copy the wrong profile and paste on top of one that shouldn't change, you've got a real mess. That's impossible to do now with Fleet.*

**Ashley Smith**

IT Platform Engineer, Pinterest
</div>

When something does need fixing, the team isn't at a vendor's mercy anymore. Instead of filing a ticket and waiting days, they can chat with Fleet’s support team in Slack and get responses within an hour. They can also open a pull request against Fleet's repo to suggest a fix to the engineering team directly.

The migration itself was substantial but controlled: roughly 6,500 Macs moved over about six months (paced around end-of-year change freezes and some auth and VPN checks), and the smaller Windows fleet in about a month. What the team reports now is refreshingly dull.

<div purpose="attribution-quote">

*Most of what we report is boring and static, and that's the point. The dashboards to show leadership our device posture rarely moves off green. About 98% of our Macs are up to date and in compliance within 30 days or less.*

**Dustin Davis**

Sr. Manager, IT Platform Engineering, Pinterest
</div>

And because security and IT now work in the same system toward the same posture, the relationship between the two teams got tighter. Slack alerts fire to both the moment any key metric drifts toward a warning status. They work together to address required updates quickly, and everyone is happy.

## Looking ahead

Pinterest isn't shy about where Fleet still has room to grow, which is part of why they trust it.

<div purpose="attribution-quote">

*Fleet's great, and it checks the boxes for us. Android continues to be developed and needs more maturity. But the open source foundation and GitOps are real, meaningful benefits that our team realized quickly.*

**Dustin Davis**

Sr. Manager, IT Platform Engineering, Pinterest
</div>

For Dan, the lasting value is something a lot of IT teams struggle to earn.

<div purpose="attribution-quote">

*My favorite thing about Fleet is the transparency. We can show a user exactly how we govern their device, and that earns a lot of trust with the business. The Fleet team is open and collaborative in a way we just hadn't seen before.*

**Dan Hutchings**

IT Platform Engineer, Pinterest
</div>

<meta name="category" value="case study">
<meta name="articleTitle" value="Pinterest’s IT team manages 13,000 devices and engineered human error out of the job">
<meta name="description" value="Pinterest manages 13,000 devices as code with Fleet GitOps: 98% of Macs in compliance and config pushes that took days now land immediately.">

<meta name="publishedOn" value="2026-09-17">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="pinterest-logo-40x40@2x.png">
<meta name="quoteAuthorImageFilename" value="dustin-davis-120x120@2x.png">
<meta name="quoteAuthorName" value="Dustin Davis">
<meta name="quoteAuthorJobTitle" value="Sr. Manager, IT Platform Engineering, Pinterest">
<meta name="quoteContent" value="“Most of what we report is boring and static, and that's the point. The dashboards to show leadership our device posture rarely moves off green. About 98% of our Macs are up to date and in compliance within 30 days or less.”">

<meta name="companyName" value="Pinterest">
<meta name="companyInfo" value="Pinterest is a visual discovery platform where hundreds of millions of people find ideas and inspiration to create a life they love. Its IT Platform Engineering team is chartered to give employees the technology and services to do inspired work from anywhere. Learn more at pinterest.com.">
<meta name="companyInfoLineTwo" value="Pinterest's IT Platform Engineering team supports endpoints across macOS, Windows, iOS, Android, and Linux.">

<meta name="summaryChallenge" value="They are engineers, but their MDM didn't let them work like engineers. For eight years, everything ran on Omnissa Workspace ONE. There was no native GitOps functionality, and when they tried to build their own workflow around it, the inflexible API got in the way. Worst of all, getting technical support was painful.">
<meta name="summarySolution" value="~6,500 Macs plus ~200 Windows devices, all managed as code via GitOps, and wired into Pinterest's internal zero-trust system.">
<meta name="summaryKeyResults" value="Devices are managed like a software development project, using peer-reviewed pull requests to engineer routine human error out of the job; 98% of Macs in compliance with security policies; Config pushes to 13,000+ devices that used to take days now landing immediately">
