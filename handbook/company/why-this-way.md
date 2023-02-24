# Why this way?

At Fleet, we rarely label ideas as drafts or theories.  Everything is [always in draft](https://about.gitlab.com/handbook/values/#everything-is-in-draft) and subject to change in future iterations.

To increase clarity and encourage teams to make decisions quickly, leaders and [DRIs](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) sometimes need to explicitly mention when they are voicing an [opinion](https://blog.codinghorror.com/strong-opinions-weakly-held/) or a [decision](https://about.gitlab.com/handbook/values/#disagree-commit-and-disagree).  When an _opinion_ is voiced, there's space for near-term debate.  When a _decision_ is voiced, team commitment is required.

Any past decision is open to questioning in a future iteration, as long as you act in accordance with it until it is changed. When you want to reopen a conversation about a past decision, communicate with the [DRI (directly responsible individual)](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) who can change the decision instead of someone who can't.  Show your argument is informed by previous conversations, and assume the original decision was made [with the best intent](https://about.gitlab.com/handbook/values/#assume-positive-intent).

Here are some of Fleet's decisions about the best way to work, and the reasoning for them.

## Why open source?

Fleet's source code, website, documentation, company handbook, and internal tools are [public](https://github.com/fleetdm/fleet) and accessible to everyone, including engineers, executives, and end users. (Even [paid features](https://fleetdm.com/pricing) are source-available.)

Meanwhile, the [company behind Fleet](https://twitter.com/fleetctl) is built on the [open-core](https://www.heavybit.com/library/video/commercial-open-source-business-strategies) business model.  Openness is one of our core [values](https://fleetdm.com/handbook/company#values), and everything we do is public by [default](https://about.gitlab.com/handbook/values/#public-by-default).  Even the [company handbook](https://fleetdm.com/handbook) is open to the world.

Is open-source collaboration _really_ worth all that?  Is it any good?

Here are some of the reasons we build in the open:

- **Transparency.** You are not dealing with a black box.  Anyone can read the code and [confirm](https://github.com/signalapp/Signal-Android/issues/11101#issuecomment-814476405) it does what it's supposed to.  When it comes to security and device management, great power should come with great openness.
- **Modifiability.** You are not stuck.  Anybody can make changes to the code at any time. You can build on existing ideas or start something brand new.  Every contribution benefits the project as a whole.  Plugins and configuration settings you need may already exist.  If not, you can add them.
- **Community.** You are not alone.  Open source contributors are real people who love solving real problems and sharing solutions. As we gain experience and our careers grow, so does [the community](https://chat.osquery.io).  As we learn, we get better at helping each other, which makes it easier to get started with the project, which drives even more adoption, and so on.
- **Less waste.** You are not redundant.  Contributing back to open source [benefits everybody](https://fleetdm.com/handbook/company): Instead of other organizations and individuals wasting time rediscovering bug fixes and reinventing the same new features in a vacuum, everybody can just upgrade to the latest version of Fleet and take advantage of all those improvements automatically.
- **Perspective.**  You are not siloed.  [Anyone can contribute](https://about.gitlab.com/company/mission).  That means startups, enterprises, and humans all over the world push fixes, add features, and influence the roadmap.  Diversity of thought accelerates the cycle time for stability and innovation.  Instead of [waiting months](http://selmiak.bplaced.net/games/pc/index.php?lang=eng&game=Loom&page=Audio-Drama--Game-Script#:~:text=I%20need%20to%20see%20at%20least%20eight%20hours%20ahead.%20EIGHT%20hours.) to discover rare edge cases, or last-minute gaps in "enterprise-readiness", or how that cool new unsupported networking protocol your CISO wants to use isn't supported yet, you get to take advantage of the investment from the last contributor who had the same problem.  It's like [seeing around corners](https://thefutureorganization.com/how-leaders-can-see-around-corners/).
- **Sustainability.** You are not the only contributor.  Open-source software is public and highly visible.  Mistakes are more obvious, which activates the community to discover (and fix) vulnerabilities and bugs more quickly.  Open-source projects like osquery and Fleet have an incentive to be proactive and thoughtful about responsible disclosure, code reviews, strict semantic versioning, release notes, documentation, and other [secure development best practices](https://github.com/osquery/osquery/blob/master/ASSURANCE.md#security-implemented-in-development-lifecycle-processes).  For example, anybody in the community can suggest and review changes, but only maintainers with appropriate subject matter expertise can merge them.
- **Accessibility.** You are smart and cool enough.  Open source isn't just [the Free Software movement](https://www.youtube.com/watch?v=UIDb6VBO9os) anymore.  Today, there are many other reasons to contribute and opportunities to contribute, even if you don't [yet know how](https://www.youtube.com/playlist?list=PL4nf6riqo7srdUHdhRSoABvES81Oygyp3) to write code.  (For example, try clicking "Edit this page" to make an improvement to this page of Fleet's handbook.)  Since 2020, Fleet has given visibility into over 1.65 million servers and workstations at Fortune 1000 companies like [Comcast](https://www.youtube.com/watch?v=J9V83Qsf3lg), [Twilio](https://fleetdm.com/podcasts/the-future-of-device-management-ep2), [Uber](https://fleetdm.com/podcasts/the-future-of-device-management-ep3), [Atlassian](https://www.youtube.com/watch?v=qflUfLQCnwY), and [Wayfair](https://fleetdm.com/device-management/fleet-user-stories-wayfair).  But did you know that during that time, Fleet inspired one 9-year-old kid to learn coding, when almost no one else believed she could do it?
- **More timeless.** You are not doomed to disappear forever when you change jobs.  Why should your code?  In most jobs, most of the work you do becomes inaccessible when you quit.  But [open source is forever](https://twitter.com/mikermcneil/status/1476799587423772674).


## Why handbook-first strategy?

The Fleet handbook provides team members with up-to-date information about how to do things in the company. 

At Fleet, we make changes to the handbook first.  That means, before any change to how we run the business is "live" oror "official", it is first changed in the relevant [handbook pages](https://fleetdm.com/handbook) and [issue templates](https://github.com/fleetdm/confidential/tree/main/.github/ISSUE_TEMPLATES).

Making changes to the handbook first [encourages](https://www.youtube.com/watch?v=aZrK8AQM8Ro) a culture of self-reliance, which is essential for daily asynchronous work as part of an all-remote team.  It keeps everyone in sync across the all-remote team in different timezones, avoids miscommunications, and ensures the right people have reviewed every change. 

> The Fleet handbook is inspired by the [GitLab team handbook](https://about.gitlab.com/handbook/about/).  It shares the same [advantages](https://about.gitlab.com/handbook/about/#advantages) and will probably undergo a similar [evolution](https://about.gitlab.com/handbook/ceo/#evolution-of-the-handbook).

To contribute to the handbook, click "Edit this page" and make your [edits in Markdown](handbook/company/handbook.md).


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

A DRI is a person who is singularly responsble for a given aspect of the open-source project, the product, or the company.  A DRI is responsible for making decisions, accomplishing goals, and getting any resources necessary to make a given area of Fleet successful.

For example, every department maintains its own dedicated [handbook page](https://fleetdm.com/handbook), with a single DRI, and which is kept up to date with accurate, current information, including the group's kanban board, Slack channels, and recurring tasks ("rituals").

DRIs help us collaborate efficiently by knowing exactly who is responsible and can make decisions about the work they're doing.  This saves time by eliminating a requirement for consensus decisions or political presenteeism, enables faster decision-making, and ensures a single individual is aware of what to do next.

### Reporting structure
In addition to Fleet's [organizational chart](https://fleetdm.com/handbook/company#org-chart), the company also organizes [cross-functional product groups](https://fleetdm.com/handbook/company#product-groups) to allow for faster collaboration and fewer roundtrips.


### Reviewers
Fleet aims to make picking the right reviewer for your change as easy and automatic as possible.  In many cases, you won't need to select a particular reviewer for your pull request.  (It will just happen automatically.)

To check out the right person to review a given piece of content or source code path, consider:
1. The [CODEOWNERS](https://github.com/fleetdm/fleet/blob/main/CODEOWNERS) files of the fleetdm/fleet and fleetdm/confidential repositories.
2. The  `name="maintainedBy"` tags at the very bottom of the raw markdown source for [every handbook page](https://github.com/fleetdm/fleet/tree/main/handbook) and [individual article](https://github.com/fleetdm/fleet/tree/main/articles).
3. The job titles and reporting structure indicated by the [company's organizational chart](https://fleetdm.com/handbook/company#org-chart) and the roles in our [cross-functional product groups](https://fleetdm.com/handbook/company#product-groups).

> In some cases, multiple subject-matter experts can merge changes to files even though there is a dedicated DRI configured as the "CODEOWNER".  For examples of this, see the auto-approval flows configured as `sails.config.custom.githubRepoDRIByPath` and `sails.config.custom.confidentialGithubRepoDRIByPath` in [`website/config/custom.js`](https://github.com/fleetdm/fleet/blob/main/website/config/custom.js).


## Why do we use a wireframe-first approach?

Wireframing (or "drafting," as we often refer to it at Fleet) provides a clear overview of page layout, information architecture, user flow, and functionality. The wireframe-first approach extends beyond what users see on their screens. Wireframe-first is also excellent for drafting APIs, config settings, CLI options, and even business processes. 

Here's why we use a wireframe-first approach at Fleet.

- We create a wireframe for every change we make and favor small, iterative changes to deliver value quickly. 
- We can think through the functionality and user experience more deeply by wireframing before committing any code. As a result, our coding decisions are clearer, and our code is cleaner and easier to maintain.
- Content hierarchy, messaging, error states, interactions, URLs, API parameters, and API response data are all considered during the wireframing process (often with several rounds of review). This initial quality assurance means engineers can focus on their code and confidently catch any potential edge-cases or issues along the way.
- Wireframing is accessible to people who understand our users but are not necessarily code-literate. So anyone can contribute a suggestion (at any level of fidelity). At the very least, you'll need a napkin and a pen, although we prefer to use Figma.
- Wireframes can be shown to customers and other users in the community [for feedback](https://www.linkedin.com/feed/update/urn:li:activity:7034272412724555776?commentUrn=urn%3Ali%3Acomment%3A%28activity%3A7034272412724555776%2C7034276934494683136%29&replyUrn=urn%3Ali%3Acomment%3A%28activity%3A7034272412724555776%2C7034539835654569984%29&dashCommentUrn=urn%3Ali%3Afsd_comment%3A%287034276934494683136%2Curn%3Ali%3Aactivity%3A7034272412724555776%29&dashReplyUrn=urn%3Ali%3Afsd_comment%3A%287034539835654569984%2Curn%3Ali%3Aactivity%3A7034272412724555776%29).
- Designing from the "outside, in" gives us the opportunity to obsess over details in the interaction design.  An undefined "what" exposes the results to the chaos of unplanned extra work and context shifting for engineers.  This way, every engineer doesn't have to personally spend the time to get and stay up to speed with: 
  - the latest reactions from users
  - all of the motivations and discussions from the previous rounds of wireframe revisions that were thrown away
  - how the UI has evolved in previous releases to better serve the people using and building it
- Wireframing is important for both maintaining the quality of our work and outlining what work needs to be done.
- With Figma, thanks to its powerful component and auto-layout features, we can create high-fidelity wireframes - fast. We can iterate quickly without costing more work and less [sunk-cost fallacy](https://dictionary.cambridge.org/dictionary/english/sunk-cost-fallacy).
- But wireframes don't have to be high fidelity.  It is OK to communicate ideas for changes using ugly, marked-up screenshots, a photo of a piece of paper.  Fleet's [drafting process](https://fleetdm.com/handbook/company/development-groups#making-changes) helps turn these rough wireframes into product changes that can be implemented quickly with minimal UX and technical debt.
- Wireframes created to describe individual changes are disposable and may have slight stylistic inconsistencies.  Fleet's user interface styleguide in Figma is the source of truth for overarching design decisions like spacing, typography, and colors.

> Got a question about creating wireframes or the drafting process?  Mention Noah Talerman or Luke Heath in `#help-product`.


## Why do we use one repo?
At Fleet, we keep everything in one repo ([`fleetdm/fleet`](https://github.com/fleetdm/fleet)). Here's why:

- One repo is easier to manage. It has less surface area for keeping content up to date and reduces the risk of things getting lost and forgotten.
- Our work is more visible and accessible to the community when all project pieces are available in one repo. 
- One repo pools GitHub stars and more accurately reflects Fleet’s presence.
- One repo means one set of automations and labels to manage, resulting in a consistent GitHub experience that is easier to keep organized.

The only exception ([`fleetdm/confidential`](https://github.com/fleetdm/confidential)) is when we're working on something confidential since GitHub does not allow confidential issues inside public repos.

> Tip: Did you know that you can [search through issues from both repos](https://github.com/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+) at the same time?  In addition to the built-in search in the handbook on fleetdm.com, you can also search for any content from the handbook, documentation, or issue templates from either repo [using GitHub search](https://github.com/search?q=org%3Afleetdm+path%3A.github%2FISSUE_TEMPLATE+path%3Ahandbook%2F+path%3Adocs%2F+foo&type=code).


## Why not continuously generate REST API reference docs from javadoc-style code comments?
Here are a few of the drawbacks that we have experienced when generating docs via tools like Swagger or OpenAPI, and some of the advantages of doing it by hand with Markdown.

- Markdown gives us more control over how the docs are compiled, what annotations we can include, and how we present the information to the end-user. 
- Markdown is more accessible. Anyone can edit Fleet's docs directly from our website without needing coding experience. 
- A single Markdown file reduces the amount of surface area to manage that comes from spreading code comments across multiple files throughout the codebase. (see ["Why do we use one repo?"](#why-do-we-use-one-repo)).
- Autogenerated docs can become just as outdated as handmade docs, except since they are siloed, they require more skills to edit.
- When docs live at separate repo paths from source code, we are able to automate approval processes that allow contributors to make small improvements and notes, directly from the website.  This [leads to more contributions](https://github.com/balderdashy/sails-docs/network/members), since it lowers the barrier of entry for [becoming a contributor](https://fleetdm.com/docs/contributing/committing-changes).
- Autogenerated docs are typically hosted on a subdomain. This means we have less control over a user's journey through our website and lose the SEO benefits of self-hosted documentation.
- Autogenerating docs from code comments is not always the best way to make sure reference docs accurately reflect the API.
- As the Fleet REST API, documentation, and tools mature, a more declarative format such as OpenAPI might become the source of truth, but only after investing in a format and processes to make it continually accurate as well as visible, accessible, and modifiable for all contributors.


## Why group Slack channels?
Groups (`g-*`) are organized around goals. Connecting people with the same goals helps them produce better results by fostering freer communication. Some groups align with teams in the org chart.  Other groups, such as [product groups](https://fleetdm.com/handbook/company/development-groups), are cross-functional, with some group members who do not report to the same manager.

Every group at Fleet maintains their own Slack channel, which all group members join and keep unmuted.  Everyone else at Fleet is encouraged to mute these channels, using them only as needed.  Each channel has a directly responsible individual responsible for keeping up with all new messages, even if they aren't explicitly mentioned (`@`).


## Why organize work in team-based kanban boards?
It's helpful to have a consistent framework for how every team works, plans, and requests things from each other. Fleet's kanban boards are that framework, and they cover three goals:

1. **Intake:** Give people from anywhere in the world the ability to request something from a particular team (i.e., add it to their backlog).
2. **Planning:** Give the team's manager and other team members a way to plan the next three-week iteration of what the team is working on in a world (the board) where the team has ownership and feels confident making changes.
3. **Shared to-do list:** What should I work on next? Who needs help? What important work is blocked? Is that bug fix merged yet? When will it be released? When will that new feature ship? What did I do yesterday?


## Why scrum?
Releasing software iteratively gets changes and improvements into the hands of users faster and generally results in software that works. This makes contributors fitter, happier, and more productive. See [the agile manifesto](https://agilemanifesto.org/) for more information.

> TODO: expand


## Why a three-week cadence?
The Fleet product is released every three weeks. By syncing the whole company to this schedule, we can:

- keep all team members (especially those who aren't directly involved with the core product) aware of the current version of Fleet and when the next release is shipping.
- align project planning and milestones across all teams, which helps us schedule our content calendar and manage company-wide goals.



<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="Why this way?">
