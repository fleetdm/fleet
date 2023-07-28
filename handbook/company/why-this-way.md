# Why this way?

At Fleet, we rarely label ideas as drafts or theories.  Everything is [always in draft](https://about.gitlab.com/handbook/values/#everything-is-in-draft) and subject to change in future iterations.

To increase clarity and encourage teams to make decisions quickly, leaders and [DRIs](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) sometimes need to explicitly mention when they are voicing an [opinion](https://blog.codinghorror.com/strong-opinions-weakly-held/) or a [decision](https://about.gitlab.com/handbook/values/#disagree-commit-and-disagree).  When an _opinion_ is voiced, there's space for near-term debate.  When a _decision_ is voiced, team commitment is required.

Any past decision is open to questioning in a future iteration, as long as you act in accordance with it until it is changed. When you want to reopen a conversation about a past decision, communicate with the [DRI (directly responsible individual)](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) who can change the decision instead of someone who can't.  Show your argument is informed by previous conversations, and assume the original decision was made [with the best intent](https://about.gitlab.com/handbook/values/#assume-positive-intent).

Here are some of Fleet's decisions about the best way to work, and the reasoning for them.

## Why open source?

Fleet's source code, website, documentation, company handbook, and internal tools are [public](https://github.com/fleetdm/fleet) and accessible to everyone, including engineers, executives, and end users. (Even [paid features](https://fleetdm.com/pricing) are source-available.)

Meanwhile, the [company behind Fleet](https://twitter.com/fleetctl) is built on the [open-core](https://www.heavybit.com/library/video/commercial-open-source-business-strategies) business model.  Openness is one of our core [values](https://fleetdm.com/handbook/company#values), and everything we do is [public by default](https://handbook.gitlab.com/handbook/values/#public-by-default).  Even the [company handbook](https://fleetdm.com/handbook) is open to the world.

Is open-source collaboration _really_ worth all that?  Is it any good?

Here are some of the reasons we build in the open:

- **Transparency.** You are not dealing with a black box.  Anyone can read the code and [confirm](https://github.com/signalapp/Signal-Android/issues/11101#issuecomment-814476405) it does what it's supposed to.  When it comes to security and device management, great power should come with great openness.
- **Modifiability.** You are not stuck.  Anybody can make changes to the code at any time. You can build on existing ideas or start something brand new.  Every contribution benefits the project as a whole.  Plugins and configuration settings you need may already exist.  If not, you can add them.
- **Community.** You are not alone.  Open source contributors are real people who love solving real problems and sharing solutions. As we gain experience and our careers grow, so does [the community](https://chat.osquery.io).  As we learn, we get better at helping each other, which makes it easier to get started with the project, which drives even more adoption, and so on.
- **Less waste.** You are not redundant.  Contributing back to open source [benefits everybody](https://fleetdm.com/handbook/company): Instead of other organizations and individuals wasting time rediscovering bug fixes and reinventing the same new features in a vacuum, everybody can just upgrade to the latest version of Fleet and take advantage of all those improvements automatically.
- **Perspective.**  You are not siloed.  [Anyone can contribute](https://about.gitlab.com/company/mission).  That means startups, enterprises, and humans all over the world push fixes, add features, and influence the roadmap.  Diversity of thought accelerates the cycle time for stability and innovation.  Instead of [waiting months](http://selmiak.bplaced.net/games/pc/index.php?lang=eng&game=Loom&page=Audio-Drama--Game-Script#:~:text=I%20need%20to%20see%20at%20least%20eight%20hours%20ahead.%20EIGHT%20hours.) to discover rare edge cases, or last-minute gaps in "enterprise-readiness", or how that cool new unsupported networking protocol your CISO wants to use isn't supported yet, you get to take advantage of the investment from the last contributor who had the same problem.  It's like [seeing around corners](https://thefutureorganization.com/how-leaders-can-see-around-corners/).
- **Sustainability.** You are not the only contributor.  Open-source software is public and highly visible.  Mistakes are more obvious, which activates the community to discover (and fix) vulnerabilities and bugs more quickly.  Open-source projects like osquery and Fleet have an incentive to be proactive and thoughtful about responsible disclosure, code reviews, strict semantic versioning, release notes, documentation, and other [secure development best practices](https://github.com/osquery/osquery/blob/master/ASSURANCE.md#security-implemented-in-development-lifecycle-processes).  For example, anybody in the community can suggest and review changes, but only maintainers with appropriate subject matter expertise can merge them.
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

Everyone [can contribute](https://fleetdm.com/handbook/company#openness) to Fleet's documentation.  Here are a few principles to keep in mind:
 
- **🚪 Start simple.** It's easier to learn when you aren't overwhelmed.  Good documentation pages and sections start _prescriptive, brief, and clear_; ideally with a short example.  You can always hedge and caveat further down the page. This makes the docs more [accessible and outsider-friendly](https://fleetdm.com/handbook/company#purpose).  For example, notice how [this page gets more complicated as you scroll down](https://sailsjs.com/documentation/reference/blueprint-api/destroy), or how [both](https://sailsjs.com/documentation/concepts/models-and-orm/model-settings#?schema) of [these sections](https://sailsjs.com/documentation/concepts/models-and-orm/model-settings#?seldomused-settings) start simple, with caveats pushed down to the end. 

<!-- 🔌🚪🪟 -->


## Why the emphasis on training?
Investing in people and providing generous, prioritized training, especially up front, helps contributors understand what is going on at Fleet. By making training a prerequisite at Fleet, we can:
- help team members feel confident in the better decisions they make at work. 
- create a culture of helping others, which results in team members feeling more comfortable even if they aren’t familiar with the osquery, security, startup, or IT space. 

Here are a few examples of how Fleet prioritizes training:
- the first 3 days at the company for every new team member are reserved for working on the tasks and training in their onboarding issue.
- during the first 2 weeks at the company, every new fleetie joins a **daily 1:1 meeting** with their manager to check in and see how they're doing, and if they have any questions or blockers.  If the manager is not available for this meeting, the CEO (pending availability) or Charlie will join this short daily meeting with them instead.
- In their first few days, every new fleetie joins:
  - hands-on contributor experience training session with Charlie where they share their screen, check the configuration of their tools, complete any remaining setup, and discuss best practices.
  - a short sightseeing tour with Charlie and (pending availability) Fleet's CEO to show them around and welcome them to the company.


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

> _**Note:** The only exception to the wireframe-first policy is for temporary pages and experiments not listed in the navigation or sitemap, and which are housed behind /imagine ☁️🪟. You can read more about marketing's [experimentation process](https://fleetdm.com/handbook/marketing#experimentation)._


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


Besides the exceptions above, Fleet does not use any other repositories.  Other GitHub repositories in `fleetdm` should be archived and made private.


> _**Tip:** Did you know that you can [search through issues from all public and non-public Fleet repos](https://github.com/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+) at the same time?_
> 
> _**Tip:** In addition to the built-in search available for the public handbook on fleetdm.com, you can also [search any public AND non-public content, including issue templates, at the same time](https://github.com/search?q=org%3Afleetdm+path%3A.github%2FISSUE_TEMPLATE+path%3Ahandbook%2F+path%3Adocs%2F+foo&type=code)._



## Why not continuously generate REST API reference docs from javadoc-style code comments?
Here are a few of the drawbacks that we have experienced when generating docs via tools like Swagger or OpenAPI, and some of the advantages of doing it by hand with Markdown.

- Markdown gives us more control over how the docs are compiled, what annotations we can include, and how we present the information to the end-user. 
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

Every department organizes their work into [team-based kanban boards](https://app.zenhub.com/workspaces/-g-business-operations-63f3dc3cc931f6247fcf55a9/board?sprints=none).  This provides a consistent framework for how every team works, plans, and requests things from each other.

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
6. The most efficient and effective method of conveying information to and within a development team is [face-to-face conversation](https://fleetdm.com/handbook/business-operations#meetings).
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

At Fleet, we consider every 5xx error, timeout, or errored scheduled job a P1 incident.  We create an outage issue for it, no matter the environment, as soon as the issue is detected, even before we understand.  We always determine impact quickly, reach out to affected users to acknowledge their problem, and determine the root cause. Why?

- It helps us learn.
- You never know whether an error like this is a real issue until you take a close look.  Even if you think it probably isn't.
- It incentivizes us to fix the root cause sooner.
- It keeps the number of errors low.
- It ensures the team understands exactly what errors are happening.
- It helps us fix bugs sooner, preventing them from stacking and bleeding into one another and making fixes harder.
- It gets everyone on the same page about what an issue is.
- It prevents stoppage of information about bugs and problems.  Every outage is visible.
- It allows us to reach out to affected users ASAP and acknowledge their challenge, showing them that Fleet takes quality and stability seriously.

### What is a P1?
Every 5xx error, timeout, or failed scheduled job is a P1.  

That means:
1. It gets a postmortem issue created within the production issue response time SLA, even before we know the impact, the root cause, or even what the error message says.
2. It gets a close look right away, even if we think it might not matter.  If there is any chance of it affecting even one user, we keep digging.
3. Including a situation where a user has to wait longer than 5 seconds during signup on fleetdm.com  (or any time we breach an agreed upon response time guarantee)
4. Including when a scheduled job fails and we aren't sure yet whether or not any real users are affected.

## Why don't we sell like everyone else?

Many companies encourage salespeople to "spray and pray" email blasts, and to do whatever it takes to close deals.  This can sometimes be temporarily effective.  But Fleet takes a [🟠longer-term](https://fleetdm.com/handbook/company#ownership) approach:
- **No spam.**  Fleet is deliberate and thoughtful in the way we do outreach, whether that's for community-building, education, or [🧊 conversation-starting](https://github.com/fleetdm/confidential/blob/main/cold-outbound-strategy.md).
- **Be a helper.**  We focus on [🔴being helpers](https://fleetdm.com/handbook/company#empathy).  Always be depositing value.  This is how we create a virtuous cycle. (That doesn't mean sharing a random article; it means genuinely hearing, doing whatever it takes to fully understand, and offering only advice or links that we would actually want.)  We are genuinely curious and desperate to help, because creating real value for poeple is the way we win.
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

As we use sentence case, only the first word is capitalized. But, if a word would normally be capitalized in the sentence (e.g., a proper noun, an acronym, or a stylization) it should remain capitalized. User roles (e.g., "observer" or "maintainer") and features (e.g. "automations") in the Fleet product aren't treated as proper nouns and shouldn't be capitalized.

The reason for sentence case at Fleet is that everyone capitalizes differently in English, and capitalization conventions have not been taught very consistently in schools.  Sentence case simplifies capitalization rules so that contributors can deliver more natural, even-looking content with a voice that feels similar no matter where you're reading it.

## Why does Fleet use "MDM on/off" instead of "MDM enrolled/unenrolled"?

Fleet is more than an MDM (mobile device management) solution.

With Fleet, you can secure and investigate Macs, Windows servers, Chromebooks, and more by installing the fleetd agent (or chrome extension for Chromebooks). When we use the word "enroll" in Fleet, we want this to mean anytime one of these hosts shows up in Fleet and the user can see that sweet telemetry.

Fleet also has MDM features that allow IT admins to enforce OS settings, OS updates, and more. When we use the phrase "MDM on" in Fleet, it means a host has these features activated.

Workspace ONE and other MDM solutions use "enroll" to mean both telemetry is being collecting and enforcement features are activated.

Since Fleet is more than MDM, you can collect telemetry on your Windows servers and you can enforce OS settings on your Macs. Or you can collect telemetry for both without enforcing OS settings.




#### Stubs
The following stubs are included only so that old links continue to work (for backwards compatibility.)

##### Reporting structure
Please see [handbook/company/why-this-way#why-direct-responsibility](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility).

##### Reviewers
Please see [handbook/company/why-this-way#why-direct-responsibility](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility).


<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="Why this way?">
