# 💭 Why this way?

At Fleet, we rarely label ideas as drafts or theories.  Everything is [always in draft](https://about.gitlab.com/handbook/values/#everything-is-in-draft) and subject to change in future iterations.

To increase clarity and encourage teams to make decisions quickly, leaders and [DRIs](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) sometimes need to explicitly mention when they are voicing an [opinion](https://blog.codinghorror.com/strong-opinions-weakly-held/) or a [decision](https://about.gitlab.com/handbook/values/#disagree-commit-and-disagree).  When an _opinion_ is voiced, there's space for near-term debate.  When a _decision_ is voiced, team commitment is required.

Any past decision is open to questioning in a future iteration, as long as you act in accordance with it until it is changed. When you want to reopen a conversation about a past decision, communicate with the [DRI (directly responsible individual)](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) who can change the decision instead of someone who can't.  Show your argument is informed by previous conversations, and assume the original decision was made [with the best intent](https://about.gitlab.com/handbook/values/#assume-positive-intent).

Here are some of Fleet's decisions about the best way to work, and the reasoning for them.

<img width="384" alt="image" src="https://github.com/fleetdm/fleet/assets/618009/234d3072-96eb-43b7-8b9e-b97af17ef72e">


## Why open source?

Fleet's source code, website, documentation, company handbook, and internal tools are [public](https://github.com/fleetdm/fleet) and accessible to everyone, including engineers, executives, and end users. (Even [paid features](https://fleetdm.com/pricing) are source-available.)

Meanwhile, the [company behind Fleet](https://twitter.com/fleetctl) is built on the [open-core](https://www.heavybit.com/library/video/commercial-open-source-business-strategies) business model.  Openness is one of our core [values](https://fleetdm.com/handbook/company#values), and everything we do is [public by default](https://handbook.gitlab.com/handbook/values/#public-by-default).  Even the [company handbook](https://fleetdm.com/handbook) is open to the world.

Is open-source collaboration _all that_?  Is it any good?

Here are some of the reasons we build in the open:

- **Transparency.** You are not dealing with a black box.  Anyone can read the code and [confirm](https://github.com/signalapp/Signal-Android/issues/11101#issuecomment-814476405) it does what it's supposed to.  When it comes to security and device management, great power should come with great openness.
- **Modifiability.** You are not stuck.  Anybody can make changes to the code at any time. Plugins and configuration settings you need may already exist.  If not, you can add them.
- **Community.** You are not alone.  Open source contributors are real people who love solving real problems and sharing solutions.  All bugs and feature requests are public.  You can [search](https://github.com/fleetdm/fleet) and weigh in on any of them with your GitHub account, or [in Slack](https://fleetdm.com/slack).
- **Less waste.** You are not redundant.  Contributing back to open source [benefits everybody](https://fleetdm.com/handbook/company): Instead of other organizations and individuals wasting time rediscovering bug fixes and reinventing the same new features in a vacuum, everybody can just upgrade.
- **Perspective.**  You are not siloed.  [Anyone can contribute](https://about.gitlab.com/company/mission).  That means startups, enterprises, and humans all over the world push fixes, add features, and influence the roadmap.  Instead of [waiting months](http://selmiak.bplaced.net/games/pc/index.php?lang=eng&game=Loom&page=Audio-Drama--Game-Script#:~:text=I%20need%20to%20see%20at%20least%20eight%20hours%20ahead.%20EIGHT%20hours.) to discover rare edge cases, or last-minute gaps in "enterprise-readiness", or how that cool new unsupported networking protocol your CISO wants to use isn't supported yet, you get to take advantage of the investment from the last contributor who had the same problem.
- **Sustainability.** You are not the only contributor.  Open-source software is public and highly visible.  Projects like osquery and Fleet have an incentive to be proactive about code reviews, semantic versioning, release notes, vulnerabilities, documentation, and other [secure development best practices](https://github.com/osquery/osquery/blob/master/ASSURANCE.md#security-implemented-in-development-lifecycle-processes).  For example, anybody in the community can suggest and review changes, but only maintainers with appropriate subject matter expertise can merge them.
- **Accessibility.** You are smart and cool enough.  Open source isn't just [the Free Software movement](https://www.youtube.com/watch?v=UIDb6VBO9os) anymore.  Today, there are many other reasons to contribute and opportunities to contribute, even if you don't [yet know how](https://www.youtube.com/playlist?list=PL4nf6riqo7srdUHdhRSoABvES81Oygyp3) to write code.  (For example, try clicking "Edit this page" to make an improvement to this page of Fleet's handbook.)
- **More timeless.** You are not doomed to disappear forever when you change jobs.  Why should your code?  In most jobs, most of the work you do becomes inaccessible when you quit.  But [open source is forever](https://twitter.com/mikermcneil/status/1476799587423772674).


## Why handbook-first strategy?

The Fleet handbook provides team members with up-to-date information about how to do things in the company. 

At Fleet, we make changes to the handbook first.  That means, before any change to how we run the business is "live" or "official", it is first changed in the relevant [handbook pages](https://fleetdm.com/handbook) and [issue templates](https://github.com/fleetdm/confidential/tree/main/.github/ISSUE_TEMPLATE).

Making changes to the handbook first [encourages](https://www.youtube.com/watch?v=aZrK8AQM8Ro) a culture of self-reliance, which is essential for daily asynchronous work as part of an all-remote team.  It keeps everyone in sync across the all-remote team in different timezones, avoids miscommunications, and ensures the right people have reviewed every change. 

> The Fleet handbook is inspired by the [GitLab team handbook](https://about.gitlab.com/handbook/about/).  It shares the same [advantages](https://about.gitlab.com/handbook/about/#advantages) and will probably undergo a similar [evolution](https://about.gitlab.com/handbook/ceo/#evolution-of-the-handbook).

To contribute to the handbook, click "Edit this page" and make your [edits in Markdown](https://fleetdm.com/handbook/company).


## Why read documentation?

There are three reasons for visiting [the docs](https://fleetdm.com/docs):
- **Tire-kicking**: "I think this is cool, now is it something that I could ACTUALLY use? Does it ACTUALLY work? What all's in it?  What links can I share with my colleagues to help them see what I'm seeing?"
- **Committed learning**: "I've decided to learn this. I need a curriculum to get me there; with content that makes it as easy as possible, surface-level as possible. I want to learn how Fleet works and how to do all the things."
- **Quick reference**: "Is this thing broken or am I using it right? How do I use this?" Whether they just stumbled in from a search engine, an on-site search, or through the Fleet website navigation, visitors interested in quick reference are interested in getting to the correct answer quickly.  Quick referencers search for REST API pages, the config surface of the Fleet server, agent options, how to build YAML for `fleetctl apply`, the built-in MDM profiles, the table schema, the built-in queries, reference architectures and cost calculators for deploying your own Fleet instance.

> Currently, Fleet only adds new "Quick reference" content to the [docs](https://fleetdm.com/docs). New "Committed learning" docs should be written as [articles](https://fleetdm.com/articles). Why? The docs surface area is large and Fleet iterates on features more quickly than we can update extensive how-tos. We've learned that users are more forgiving when articles are out-of-date. But, reference documentation always needs to be correct.

Everyone [can contribute](https://fleetdm.com/handbook/company#openness) to Fleet's documentation.  Here are a few principles to keep in mind:
 
- **🚪 Start simple.** It's easier to learn when you aren't overwhelmed.  Good documentation pages and sections start _prescriptive, brief, and clear_; ideally with a short example.  You can always hedge and caveat further down the page. This makes the docs more [accessible and outsider-friendly](https://fleetdm.com/handbook/company#purpose).  For example, notice how [this page gets more complicated as you scroll down](https://sailsjs.com/documentation/reference/blueprint-api/destroy), or how [both](https://sailsjs.com/documentation/concepts/models-and-orm/model-settings#?schema) of [these sections](https://sailsjs.com/documentation/concepts/models-and-orm/model-settings#?seldomused-settings) start simple, with caveats pushed down to the end. 
- **🪟 Write it once.** Deduplicate content more than you think you need to. Use links instead. Decorate relevant bits of text, don’t put full URLs. Think of SEO: the link text matters.
- **🚪 Don’t break links.** Whenever practical, maintain a path to documentation. Even if it's an [outdated rough draft](https://kevin.burke.dev/kevin/dont-use-sails-or-waterline/).
- **🔌 Document the rough bits.** Just do it at the bottom of the page/section so that most people can skip over it, but the quick referencer who really needs it can find it. Docs are the requirements. If it doesn’t work as documented, it’s a bug.
- **🪟 Some people read the docs as text on GitHub.** It’s a small audience and we can’t afford to optimize for this use case. But it’s worth remembering. For example, someone from Iran needed help from Mike in 2013 to access content from sailsjs.com unencrypted when https:// was banned in her country.

<!-- 🔌🚪🪟 -->

## Why the emphasis on training?

Investing in people and providing generous, prioritized training, especially up front, helps contributors understand what is going on at Fleet. By making training a prerequisite at Fleet, we can:
- help team members feel confident in the better decisions they make at work. 
- create a culture of helping others, which results in team members feeling more comfortable even if they aren’t familiar with the osquery, security, startup, or IT space. 

Here are a few examples of how Fleet prioritizes training:
- the first 3 days at the company for every new team member are reserved for working on the tasks and training in their onboarding issue.
- during the first 2 weeks at the company, every new fleetie joins a **daily 1:1 meeting** with their manager to check in and see how they're doing, and if they have any questions or blockers.  If the manager is not available for this meeting, the CEO (pending availability) or the Head of Digital Experience will join this short daily meeting with them instead.
- In their first few days, every new fleetie joins:
  - hands-on contributor experience training session with the Head of Digital Experience where they share their screen, check the configuration of their tools, complete any remaining setup, and discuss best practices.
  - a short sightseeing tour with the Head of Digital Experience and (pending availability) Fleet's CEO to show them around and welcome them to the company.


## Why direct responsibility?

Like Apple and GitLab, Fleet uses the concept of [directly responsible individuals (DRIs)](https://about.gitlab.com/handbook/people-group/directly-responsible-individuals/) to know who is responsible for what. 

DRIs help us collaborate efficiently by knowing exactly who is responsible and can make decisions about the work they're doing.  This saves time by eliminating a requirement for consensus decisions or political presenteeism, enables faster decision-making, and ensures a single individual is aware of what to do next.

- **What is a DRI?**: A DRI is a person who is singularly responsible for a given aspect of the open-source project, the product, or the company.  A DRI is responsible for making decisions, accomplishing goals, and getting any resources necessary to make a given area of Fleet successful.  For example, every department maintains its own dedicated [handbook page](https://fleetdm.com/handbook) which is kept up to date with accurate, current information, including the group's [kanban board](https://fleetdm.com/handbook/company/why-this-way#why-make-work-visible), Slack channels, and recurring tasks ("rituals").
- **Change control**: In keeping with Fleet's handbook-first philosophy and value of writing things down, changes are always approved by the DRI [first, before they become real](https://fleetdm.com/handbook/company/why-this-way#why-handbook-first-strategy).  Fleet aims to make picking the right reviewer for your change as easy and automatic as possible. 
- **Picking a reviewer**: In most cases, you won't need to select a particular reviewer for your pull request.  (It will just happen automatically.)  Automatic PR review requests are driven by a combination of [custom repo automation](https://github.com/fleetdm/fleet/pull/12786) and [CODEOWNERS files](https://github.com/search?q=org%3Afleetdm+path%3ACODEOWNERS&type=code).  When in doubt, refer to the roles in the company's [cross-functional product groups](https://fleetdm.com/handbook/company#product-groups), and (to a lesser degree) the job titles and reporting structure indicated by the [company's organizational chart](https://fleetdm.com/handbook/company#org-chart).
- **"Maintained by" photo**: For [handbook pages](https://github.com/fleetdm/fleet/tree/main/handbook) and [articles](https://github.com/fleetdm/fleet/tree/main/articles), the "Maintained by" photo displayed on the website corresponds with the `name="maintainedBy"` tags at the very bottom of the raw markdown source for each page.  This photo should match the DRI who is auto-requested to approve changes.  (It is determined by the person's GitHub profile picture.)
- **Multiple maintainers**: In some cases, multiple subject-matter experts called "maintainers" can merge changes to certain file paths, even though there is already a dedicated DRI configured as the "CODEOWNER".  For examples of this, see the auto-approval flows configured as `sails.config.custom.githubRepoMaintainersByPath` and related configuration in [`website/config/custom.js`](https://github.com/fleetdm/fleet/blob/main/website/config/custom.js).


## Why do we use a wireframe-first approach?

Wireframing (usually as part of what Fleet calls ["drafting"](https://fleetdm.com/handbook/company/development-groups#making-changes)) provides a clear overview of page layout, information architecture, user flow, and functionality. The [wireframe-first approach](https://speakerdeck.com/mikermcneil/i-love-apis?slide=28) extends beyond what users see on their screens. Wireframe-first is also excellent for drafting APIs, config settings, CLI options, and even business processes.

It's design thinking, applied to software development.

Here's why Fleet uses a wireframe-first approach:
- We create a wireframe for every change we make and favor small, iterative changes to deliver value quickly. 
- We can think through the functionality and user experience more deeply by wireframing before committing any code. As a result, our coding decisions are clearer, and our code is cleaner and easier to maintain.
- Content hierarchy, messaging, error states, interactions, URLs, API parameters, and API response data are all considered during the wireframing process (often with several rounds of review). This initial quality assurance means engineers can focus on their code and confidently catch any potential edge-cases or issues along the way.
- Wireframing is accessible to people who understand our users but are not necessarily code-literate. So anyone can contribute a suggestion (at any level of fidelity). At the very least, you'll need a napkin and a pen, although we prefer to use Figma.
- Wireframes can be shown to customers and other users in the community [for feedback](https://www.linkedin.com/feed/update/urn:li:activity:7034272412724555776?commentUrn=urn%3Ali%3Acomment%3A%28activity%3A7034272412724555776%2C7034276934494683136%29&replyUrn=urn%3Ali%3Acomment%3A%28activity%3A7034272412724555776%2C7034539835654569984%29&dashCommentUrn=urn%3Ali%3Afsd_comment%3A%287034276934494683136%2Curn%3Ali%3Aactivity%3A7034272412724555776%29&dashReplyUrn=urn%3Ali%3Afsd_comment%3A%287034539835654569984%2Curn%3Ali%3Aactivity%3A7034272412724555776%29).
- Designing from the "outside, in" gives us the opportunity to obsess over details in the interaction design.  An undefined "what" exposes the results to the chaos of [unplanned extra work](https://fleetdm.com/handbook/company/development-groups#planned-and-unplanned-changes) and context shifting for engineers.  This way, every engineer doesn't have to personally spend the time to get and stay up to speed with: 
  - the latest reactions from users
  - all of the motivations and discussions from the previous rounds of wireframe revisions that were thrown away
  - how the UI has evolved in previous releases to better serve the people using and building it
- Wireframing is important for both maintaining the quality of our work and outlining what work needs to be done.
- With Figma, thanks to its powerful component and auto-layout features, we can create high-fidelity wireframes - fast. We can iterate quickly without costing more work and less [sunk-cost fallacy](https://dictionary.cambridge.org/dictionary/english/sunk-cost-fallacy).
- But wireframes don't have to be high fidelity.  It is OK to communicate ideas for changes using ugly, marked-up screenshots, a photo of a piece of paper.  Fleet's [drafting process](https://fleetdm.com/handbook/company/development-groups#making-changes) helps turn these rough wireframes into product changes that can be implemented quickly with minimal UX and technical debt.
- Wireframes created to describe individual changes are disposable and may have slight stylistic inconsistencies.  Fleet's user interface styleguide in Figma is the source of truth for overarching design decisions like spacing, typography, and colors.
- While the "wireframe first" practice is [still sometimes misunderstood](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall), today many modern high-performing teams now use a [wireframe-first methodology](https://speakerdeck.com/mikermcneil/i-love-apis), including [startups](https://www.forbes.com/sites/danwoods/2015/10/19/dont-get-ubered-apis-hold-key-to-digital-transformation/?sh=50112fea182c#:~:text=One%20recommendation%20that,deep%20experience) and [publicly-traded companies](https://about.gitlab.com/handbook/product-development-flow/#validation-phase-3-design).


## Why do we use one repo?

At Fleet, we keep everything in one repo ([`fleetdm/fleet`](https://github.com/fleetdm/fleet)). Here's why:

- One repo is easier to manage. It has less surface area for keeping content up to date and reduces the risk of things getting lost and forgotten.
- Our work is more visible and accessible to the community when all project pieces are available in one repo. 
- One repo pools GitHub stars and more accurately reflects Fleet’s presence.
- One repo means one set of automations and labels to manage, resulting in a consistent GitHub experience that is easier to keep organized.


The only exceptions are:
1. **Other open-source projects:** When we contribute to open-source projects owned by other people and organizations, we contribute to those outside repositories.  For example, Fleet contributes to [osquery](https://github.com/osquery/osquery/commits/master), [Sails.js](https://github.com/balderdashy/sails/commits/master), and [other open-source projects](https://github.com/orgs/fleetdm/sponsoring).
2. **Non-public matters:** Since GitHub does not allow non-public issues inside public repos, we have to use separate repositories to track non-public issues.  Sometimes it is also useful to contribute files to a non-public repository, such as when they mention customer relationships that are under non-disclosure agreements.  When we work on something non-public, we contribute to the repository with the appropriate [level of confidentiality](https://fleetdm.com/handbook/company#levels-of-confidentiality):
   - _Confidential:_ [`fleetdm/confidential`](https://github.com/fleetdm/confidential)
   - _Classified (¶¶):_ [`fleetdm/classified`](https://github.com/fleetdm/classified)
3. **GitHub Actions:** Since GitHub requires GitHub Actions to live in dedicated repositories in order to submit them to the marketplace, Fleet uses a separate repo for publishing [GitHub Actions designed for other people to deploy and use (and/or fork)](https://github.com/fleetdm/fleet-mdm-gitops).
4. **Software vulnerabilities:** Since GitHub only allows one latest release per repository, we currently maintain two repositories to host our CVE/CPE database releases: 
  - _vulnerabilities:_ [`fleetdm/vulnerabilities`](https://github.com/fleetdm/vulnerabilities)
  - _nvd:_ [`fleetdm/nvd`](https://github.com/fleetdm/nvd)
5. **Terraform modules:** Since terraform clones the entire repo once per each tagged version of a module, a full `terraform init` using the terraform AWS modules was taking up over 11 GB of space when the modules resided in [fleetdm/fleet](https://github.com/fleetdm/fleet).  Moved to [fleetdm/fleet-terraform](https://github.com/fleetdm/fleet-terraform)


Besides the exceptions above, Fleet does not use any other repositories.  Other GitHub repositories in `fleetdm` should be archived and made private.


> _**Tip:** Did you know that you can [search through issues from all public and non-public Fleet repos](https://github.com/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+) at the same time?_
> 
> _**Tip:** In addition to the built-in search available for the public handbook on fleetdm.com, you can also [search any public AND non-public content, including issue templates, at the same time](https://github.com/search?q=org%3Afleetdm+path%3A.github%2FISSUE_TEMPLATE+path%3Ahandbook%2F+path%3Adocs%2F+foo&type=code)._


## Why not continuously generate REST API reference docs from javadoc-style code comments?

Here are a few of the drawbacks that we have experienced when generating docs via tools like Swagger or OpenAPI, and some of the advantages of doing it by hand with Markdown.

- Markdown gives us more control over how the docs are compiled, what annotations we can include, and how we [present the information to the end-user](https://x.com/wesleytodd/status/1769810305448616185?s=46&t=4_cwTxqV5IXDLBvCm8KI6Q). 
- Markdown is more accessible. Anyone can edit Fleet's docs directly from our website without needing coding experience. 
- A single Markdown file reduces the amount of surface area to manage that comes from spreading code comments across multiple files throughout the codebase. (see ["Why do we use one repo?"](#why-do-we-use-one-repo)).
- Autogenerated docs can become just as outdated as handmade docs, except since they are siloed, they require more skills to edit.
- When docs live at separate repo paths from source code, we are able to automate approval processes that allow contributors to make small improvements and notes, directly from the website.  This [leads to more contributions](https://github.com/balderdashy/sails-docs/network/members), since it lowers the barrier of entry for [becoming a contributor](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Committing-Changes.md#committing-changes).
- Autogenerated docs are typically hosted on a subdomain. This means we have less control over a user's journey through our website and lose the SEO benefits of self-hosted documentation.
- Autogenerating docs from code comments is not always the best way to make sure reference docs accurately reflect the API.
- As the Fleet REST API, documentation, and tools mature, a more declarative format such as OpenAPI might become the source of truth, but only after investing in a format and processes to make it continually accurate as well as visible, accessible, and modifiable for all contributors.


## Why group Slack channels?

Groups (`g-*`) are organized around goals. Connecting people with the same goals helps them produce better results by fostering freer communication. Some groups align with teams in the org chart.  Other groups, such as [product groups](https://fleetdm.com/handbook/company/development-groups), are cross-functional, with some group members who do not report to the same manager.

Every group at Fleet maintains their own Slack channel, which all group members join and keep unmuted.  Everyone else at Fleet is encouraged to mute these channels, using them only as needed.  Each channel has a directly responsible individual responsible for keeping up with all new messages, even if they aren't explicitly mentioned (`@`).


## Why make work visible?

Work is tracked in [GitHub issues](https://github.com/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+).

Every department organizes their work into [team-based kanban boards](https://app.zenhub.com/workspaces/-g-digital-experience-63f3dc3cc931f6247fcf55a9/board?sprints=none).  This provides a consistent framework for how every team works, plans, and requests things from each other.

1. **Intake:** Give people from anywhere in the world the ability to [request something](https://github.com/fleetdm/confidential/issues/new/choose) from a particular team, and give that team the ability to see and [respond quickly](https://fleetdm.com/handbook/company#results) to new requests.
2. **Planning:** Give the team's manager and other team members a way to plan the [next three-week iteration](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence) of what the team is working on.  Provide a world (the kanban board) where the team has clarity, and the appropriate [DRI](https://fleetdm.com/handbook/company#why-direct-responsibility) can confidently [prioritize and plan changes](https://fleetdm.com/handbook/company/development-groups#planned-and-unplanned-changes) with enough context to make the right decisions.
3. **Shared to-do list:** What should I work on next? Who needs help? What important work is blocked? Is that bug fix merged yet? When will it be released? When will that new feature ship? What did I do yesterday?


## Why agile?

Releasing software [🟢iteratively](https://fleetdm.com/handbook/company#results) gets changes and improvements into the hands of users faster and generally results in [🔵software that works](https://fleetdm.com/handbook/company#objectivity). This makes contributors fitter, happier, and more productive.

We apply the [twelve principles of agile](https://agilemanifesto.org) to Fleet's [development process](https://fleetdm.com/handbook/company/product-groups#making-changes):

1. Our highest priority is to [🔴satisfy the customer](https://fleetdm.com/handbook/company#empathy) through early and continuous delivery of valuable software.
2. Welcome changing requirements, even late in development. Agile processes harness change for the customer's competitive advantage.
3. Deliver working software frequently, from a couple of weeks to a couple of months, with a preference to the shorter timescale.
4. Business people and developers must [work together daily](https://fleetdm.com/handbook/company/product-groups) throughout the project.
5. Build projects around motivated individuals. Give them the environment and support they need, and trust them to get the job done.
6. The most efficient and effective method of conveying information to and within a development team is [face-to-face conversation](https://fleetdm.com/handbook/communications#meetings).
7. Working software is the primary measure of progress.
8. Agile processes promote sustainable development. The sponsors, developers, and users should be able to maintain a constant pace indefinitely.
9. Continuous attention to technical excellence and good design enhances agility.
10. Simplicity--the art of maximizing the amount of work not done--is essential.
11. The best architectures, requirements, and designs emerge from self-organizing teams [with an effective process](https://fleetdm.com/handbook/company/product-groups#making-changes).
12. At regular intervals, the team reflects on how to become more effective, then tunes and adjusts its behavior accordingly.


### Why scrum?

Scrum is an agile framework for software development that helps teams deliver high quality software faster. It emphasizes teamwork, collaboration, and continuous improvement to achieve business objectives. Here are some of the key reasons why [we use scrum at Fleet](https://fleetdm.com/handbook/engineering#scrum)): 
- Improved collaboration and communication: Scrum emphasizes teamwork and collaboration, which leads to better communication between team members and stakeholders. This helps ensure that everyone is aligned and working towards the same goals.
- Flexibility and adaptability: Scrum allows teams to respond quickly to changing requirements and market conditions. By working in short sprints, teams can continuously adapt to new information and feedback, and adjust their approach as needed.
- Continuous improvement: Scrum encourages teams to reflect on their processes and identify areas for improvement. The regular sprint retrospective meetings provide a forum for the team to discuss what went well and what could be improved, and to make changes to their processes accordingly.
- Faster delivery of working software: Scrum helps teams deliver working software faster by breaking down the development process into manageable chunks that can be completed within a sprint. Stakeholders can see progress and provide feedback more quickly, which helps ensure the final product meets their needs.
- Higher quality software: Scrum includes regular testing and quality assurance activities, which help ensure that the software being developed is of high quality and meets the required standards.


### Why lean software development?

[Lean software development](https://en.wikipedia.org/wiki/Lean_software_development) is an iterative and incremental approach to software development that aims to eliminate waste and deliver value to customers quickly. It is based on the principles of [lean manufacturing](https://en.wikipedia.org/wiki/Lean_manufacturing) and emphasizes continuous improvement, collaboration, and customer focus.

Lean development can be summarized by its seven principles:
1. Eliminate waste: Eliminate anything that doesn't add value to the customer, such as unnecessary features, extra processing, and waiting times.
2. Amplify learning: Share knowledge and expertise across the team to continuously improve the process and increase efficiency.
3. Decide as late as possible: Defer major decisions and commitments until the last possible moment to enable more informed and optimal decisions.
4. Deliver as fast as possible: Deliver value to customers as quickly as possible to ensure their needs are met and to receive feedback for continuous improvement.
5. Empower the team: Respect and empower the team, including customers, stakeholders, and developers, by providing a supportive environment and clear communication.
6. Build integrity in: Build quality into the software by continuously testing, reviewing, and improving the code throughout the development process.
7. Optimize the whole: Optimize the entire process and focus on the system's overall performance rather than just individual parts to ensure the most efficient and effective use of resources.


## Why a three-week cadence?

The Fleet product is released every three weeks. By syncing the whole company to this schedule, we can:

- Keep all team members (especially those who aren't directly involved with the core product) aware of the current version of Fleet and when the next release is shipping.
- Align project planning and milestones across all teams, which helps us schedule our content calendar and manage company-wide goals.

## Why spend so much energy responding to every potential production incident?
At Fleet, every 5xx response, timed-out request, and failed scheduled job is a P1 outage.

As soon as the outage is detected in any production environment (including fleetdm.com, hosted customer environments, TUF, and others), we create an outage issue _immediately_: before we know for sure whether any real users are affected, and even before we know what the error message says.

Then, we determine impact quickly. We take a close look right away, even if we think it might not matter.  If there is any chance of it affecting even one user, we keep digging.  We reach out to affected users to acknowledge their problem, provide them with a workaround or some other way to make their day less painful.

Finally, we determine the root cause, and we make sure a resolution for it is filed as a bug, with attention from the appropriate contributors.  Then, we close the issue.

Why bother with all that?  And why do it in this particular order?
- **Avoids slip-ups.** When every outage is visible, information about production problems flows more freely.  It is harder to accidentally overlook a production issue, or accidentally deprioritize a major issue.  You never know whether an error like this is a real issue until you take a close look.  Even if you think it probably isn't.
- **Faster diagnosis.** Prioritizing outages gets everyone on the same page about exactly what errors are happening as soon as possible, and gets subject matter experts involved faster.
- **Helps us measure.** When outage issues are created as soon as the outage occured, the time of issue creation and closing give us useful metrics about the outage, with minimal extra effort.
- **Better customer experience.** Understanding the impact of every production issue means we can reach out to affected users ASAP and acknowledge their challenge, showing them that Fleet takes quality and stability seriously.  This kind of customer support is rare and memorable.
- **It helps us prevent future outages.** By finding outages sooner, we incentivize ourselves to fix the root cause sooner.  And by fixing bugs sooner, we prevent them from stacking and bleeding into one another, and we prevent ourselves from implementing future fixes and improvements on top of shaky foundations.  This makes contributions less risky and reduces the number of outages.


## Why fix small inconsistencies quickly?

Fixing small inconsistencies quickly is worth it. When we tolerate and overlook little things, we send contributors mixed messages, like "[broken windows](https://en.wikipedia.org/wiki/Broken_windows_theory)". (Are these things actually important?) Since new contributors join the team all the time, we also prevent learning the right way to do things through osmosis, since pattern matching from a bad pattern creates even more inconsistencies. Plus these things add up over time, creating problems for both users and contributors.


## Why make it obvious when stuff breaks?

At Fleet, we detect and fix bugs as quickly as possible.  

Breaking loudly means we can fix the break sooner and improve how fast and certain we are about making future changes.  Especially in an all-remote environment, this provides contributors with discipline around quality and stability of the main branch.  This is ["good annoying"](https://agilehope.blogspot.com/2014/12/diy-build-light-indicator.html).

When contributing to Fleet, for every PR, the person submitting the PR tests it by hand before it is merged, regardless who else tested it, or who else reviewed the code.  Thanks to this, we should normally never end up in a situation where a merged PR causes a broken contributor experience, because the person submitting it would have experienced that broken contributor experience when testing.

If that happens by mistake, first priority is merging a fix, then notifying the contributor who made the mistake so they're aware for future changes.  (This is not about blame; it's about clarity.)  We always prioritize fixing bugs, because bugs are the best early sign of misunderstandings, stale assumptions, and impactful coding mistakes that can fundamentally damage Fleet's long-term development speed, contributor experience, and code base complexity.

> Here is [an example of a deliberate decision to make broken images in Fleet fail more loudly](https://github.com/fleetdm/fleet/issues/12305#issuecomment-1671924257) so that they can't be overlooked, even though this might slow down short-term development.


## Why keep issue templates simple?

At Fleet, we optimize for the person submitting the issue, not the person receiving it.

We avoid making the submitter read anything.  We prompt for as little information as possible.  Why?
- When someone is submitting an issue, they are usually in a hurry.  They [might not even work here](https://fleetdm.com/handbook/company/why-this-way#why-open-source).  Even short issue templates can cause confusion, lead to mistakes, and discourage future submissions.
- The person receiving an issue has more context, and so will have an easier time filling in the gaps and, if necessary, following up to better understand.
- This encourages [everyone to contribute](https://fleetdm.com/handbook/company#openness), and [keep contributing](https://fleetdm.com/handbook/company#results).

For example, here is the [philosophy behind Fleet's bug report template](https://github.com/fleetdm/fleet/pull/13204#issuecomment-1678419117).


## Why spend less?

- **Default to efficiency. Reward richly.** At Fleet, we celebrate success and reward hard work.  But we do everyday things cheap.  And that is very important, because it shapes the kind of people we hire, and the kind of expectations we set for the team about what "comfortable" feels like.
- **Offsites are not rewards.** Day to day, Fleet does not look rich.  Rich !== welcoming.  The company is open, not closed.  Work here means flexible collaboration, accessible people, and clear expectations.  And a rich, exciting future worth working for.  Not a rich, complacent baseline worth coasting for.
- **Minimally viable comfort.**  We stay at La Quintas by the train tracks every single time unless customers are coming into the room and we need more space.  Even then, we accommodate in the spirit of _hospitality_, not to show off how well Fleet is doing.  They'll know how well we're doing by how great the product is, how great the support is, and [how that makes them feel](https://fleetdm.com/handbook/company#purpose).  They'll remember openness, flexibility, accessibility, and clarity in all of their interactions with the brand.  Not the view from our hotel rooms.
- **Everyday efficiency.** Fleet isn't the place you work for the everyday amenities.  Like [Southwest Airlines](https://hbsp.harvard.edu/product/W94C04-PDF-ENG), Fleet is egalitarian and outsider-friendly.  We lift people up, but we remember where we came from.  The company is efficient and friendly, more than it is polished or formal.  Never show off.  Look smart _and_ real.  Make Fleet look easy and welcoming, never slick.  And rarely fancy.


## Why don't we sell like everyone else?

Many companies encourage salespeople to ["spray and pray"](https://www.linkedin.com/posts/amstech_the-rampant-abuse-of-linkedin-connections-activity-7178412289413246978-Ci0I?utm_source=share&utm_medium=member_ios) email blasts, and to do whatever it takes to close deals.  This can sometimes be temporarily effective.  But Fleet takes a [🟠longer-term](https://fleetdm.com/handbook/company#ownership) approach:
- **No spam.**  Fleet is deliberate and thoughtful in the way we do outreach, whether that's for community-building, education, or [🧊 conversation-starting](https://docs.google.com/document/d/1IbucpsZZ0qbJQRPRtm9e2kMcSBDTXjixAMVOWyTu_pA/edit?tab=t.0).
- **Be a helper.**  We focus on [🔴being helpers](https://fleetdm.com/handbook/company#empathy).  Always be depositing value.  This is how we create a virtuous cycle. (That doesn't mean sharing a random article; it means genuinely hearing, doing whatever it takes to fully understand, and offering only advice or links that we would actually want.)  We are genuinely curious and desperate to help, because creating real value for people is the way we win.
- **Engineers first.** We always talk to engineers first, and learn how it's going.  Security and IT engineers are the people closest to the work, and the people best positioned to know what their organizations need.
- **Fewer words.  Fewer pings.**  People are busy.  We don't waste their time.  Avoid dumping work on prospect's plates at all costs.  Light touches, no asks.  Every notification from Fleet is a ping they have to deal with.  We don't overload people with words and links.  We [🟢keep things simple](https://fleetdm.com/handbook/company#results) and [write briefly](http://www.paulgraham.com/writing44.html).
- **Community-first.**  We go to conferences.  We write docs.  We are participants, not sponsors.  We don't write spammy articles and landing pages. We want people who choose Fleet to be successful, whether they are paying customers or not.  We are not pushy.  We are only as commercial as we have to be to help people out.
- **Be genuine.**  No puffery. No impressive-sounding words.  We are [🟣open and outsider friendly](https://fleetdm.com/handbook/company#openness).  We expand acronyms, and insist on using simple language that lets everyone understand and contribute.  We help the people we work with grow in their careers and learn from each other.  We are sincere, curious, and [🔵fair to competitors](https://fleetdm.com/handbook/company#objectivity).
- **Step up.** We look at the [🟠big picture](https://fleetdm.com/handbook/company#ownership).  The goal is for the organization using Fleet to be successful, as well as the individuals who decide to use or buy the product.  There are multiple versions of Fleet, and so many ways to "do" open-source security and IT.  It is in the company's best interest to help engineers pick the right one; even if that's Fleet Free, or another solution altogether.  We think about our customer's needs like they are our own.


## Why does Fleet support query packs?

As originally envisioned by Zach Wasserman and the team when creating osquery, packs are a way to import and export queries into (and out of!) any platform that speaks osquery, whether that's Fleet, [Security Onion](https://securityonionsolutions.com/), an EDR, or even Rapid7. Queries [should be portable](https://github.com/fleetdm/fleet/blob/f711e60de47c69ab8be5bc13cf73fedf88adc338/README.md#lighter-than-air) to minimize lock-in to particular tools.

The "Packs" section of the UI that began in `kolide/fleet` c. 2017 was an early attempt to  segment and target formations of hosts that share certain characteristics.  This came with some difficulties with debugging and collaboration, since it could be hard to tell which queries were running on which hosts. It also made it harder to understand what performance impact running all those queries might cause.

Eventually, when working on some related improvements, it became clear that Fleet needed a better way to organize hosts, controls, reports, and configuration that wasn't tied exclusively to data collection in Splunk.  It was time to learn from the original design and come up with a smarter way to group hosts.

The first step was to add a simpler way to schedule queries, and tuck away the legacy feature called "Packs", so that "packs" refer to what they were originally: a portable way to import and export queries. 

Packs will always be supported in Fleet.


## Why does Fleet use sentence case?

Fleet uses sentence case capitalization for all headings, subheadings, button text in the Fleet product, fleetdm.com, the documentation, the handbook, marketing material, direct emails, in Slack, and in every other conceivable situation.

In sentence case, we write and capitalize words as if they were in sentences:

> Ask questions about your servers, containers, and laptops running Linux, Windows, and macOS

As we use sentence case, only the first word is capitalized. But, if a word would normally be capitalized in the sentence (e.g., a proper noun, an acronym, or a stylization) it should remain capitalized. User roles (e.g., "observer" or "maintainer") and features (e.g. "automations") in the Fleet product aren't treated as proper nouns and shouldn't be capitalized. However, words/phrases that follow step numbers in the documentation _should_ be capitalized (e.g. "Step 1: Create"), since the step number is merely acting as a label for the phrase that follows.

The reason for sentence case at Fleet is that everyone capitalizes differently in English, and capitalization conventions have not been taught very consistently in schools.  Sentence case simplifies capitalization rules so that contributors can deliver more natural, even-looking content with a voice that feels similar no matter where you're reading it.


## Why not use superlatives?

A superlative is an adjective or adverb that expresses the degree of a quality, such as "best," "worst," or "most beautiful."

A superlative is a judgment or evaluation, [which only the customer can decide](https://twitter.com/mikermcneil/status/1686837625187930112). 

✅ **Do:**

<blockquote>
Avoid using too many unnecessary words or superlatives, so your writing is shorter and easier to understand. 
</blockquote>

❌ **Don't:**

<blockquote>
 There exists an exceptionally significant rationale that unequivocally warrants refraining from the utilization of an exceptionally vast multitude of gratuitous, superfluous, surplus verbiage, or excessive superlatives when one is tasked with the composition of official documentation that is destined for perusal and comprehension by our distinguished and highly regarded clientele. When the writer in question opts to employ an excessively copious quantity, or even a modicum of superfluous verbiage that, in truth, does not contribute substantively to the essence and signification of the text, it invariably leads to an undue lengthening of the document and an exponentially augmented level of complexity in terms of comprehensibility.
</blockquote>


## Why does Fleet use "MDM on/off" instead of "MDM enrolled/unenrolled"?

MDM should be a capability, not a product category.

In Fleet, the word "enrolled" means "the host shows up in the dashboard and API".

When some tools like Workspace ONE say a host is "enrolled", they mean that data is being collected _and_ enforcement features are activated on that host.

Since Fleet is more than MDM, you can collect logs and health data on any computer.  You can also enforce OS settings on any computer.  But you don't have to enable both: for example, you can build an installer that only collects data, without enabling enforcement features like MDM protocol support and script execution.

That means you can collect logs from Linux servers or Windows factory workstations without enabling remote script execution on those computers, even if you're using script execution on your Macs.


## Why not mention the CEO in Slack threads?

Everyone else who works at Fleet is expected to read (and reply or acknowledge with an emoji reaction) every time they're mentioned in Slack, even deep inside long threads.

Now that the company has grown, the CEO gets mentioned in threads [too often](https://docs.google.com/document/d/1vK-Dy2BVrw7doYUzabOPyCiN4RfolWFgOKMm23l91s0/edit) to keep up with thread replies, even for threads he participates in.

From Mike:

<blockquote purpose="large-quote">
  Staying on top of your Slack mentions (including in threads!) is very important. Please use them. 
But now that the company has grown, in my role as CEO, I get mentioned in Slack very often.

I held on as long as I could.  But due to volume, in late 2022, I made the decision to no longer read Slack threads where I am mentioned.

 What do I still read?
 
 - If you mention me in a top-level channel message, I'll see and read it in 1 business day.
 - If you send me a direct message, I'll see and read that ASAP.

Keep in mind I am often in meetings all day, and may not be able to reply promptly.
  
When in doubt, you can look at my calendar and join whatever meeting I'm in.  If none of that works, and there is an emergency where you need my immediate attention, get help from [Sam Pfluger](https://fleetdm.com/handbook/digital-experience#team).
Thank you so much!" 🙇
</blockquote> 


#### Stubs

The following stubs are included only so that old links continue to work (for backwards compatibility.)

##### Reporting structure
Please see [handbook/company/why-this-way#why-direct-responsibility](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility).

##### Reviewers
Please see [handbook/company/why-this-way#why-direct-responsibility](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility).

##### What is a P1?
Please see [handbook/company/why-this-way#why-spend-so-much-energy-responding-to-every-potential-production-incident](https://fleetdm.com/handbook/company/why-this-way#why-spend-so-much-energy-responding-to-every-potential-production-incident).

<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="💭 Why this way?">
