# Why this way?

## Why do we use a wireframe-first approach?

Wireframing (or "drafting," as we often refer to it at Fleet) provides a clear overview of page layout, information architecture, user flow, and functionality. The wireframe-first approach extends beyond what users see on their screens. Wireframe-first is also excellent for drafting APIs, config settings, CLI options, and even business processes. 

Here's why we use a wireframe-first approach at Fleet.

- We create a wireframe for every change we make and favor small, iterative changes to deliver value quickly. 
- We can think through the functionality and user experience more deeply by wireframing before committing any code. As a result, our coding decisions are clearer, and our code is cleaner and easier to maintain.
- Content hierarchy, messaging, error states, interactions, URLs, API parameters, and API response data are all considered during the wireframing process (often with several rounds of review). This initial quality assurance means engineers can focus on their code and confidently catch any potential edge-cases or issues along the way.
- Wireframing is accessible to people who understand our users but are not necessarily code-literate. So anyone can contribute a suggestion (at any level of fidelity). At the very least, you'll need a napkin and a pen, although we prefer to use Figma.
- Designing from the "outside, in" gives us the opportunity to obsess over details in the interaction design.  An undefined "what" exposes the results to the chaos of unplanned extra work and context shifting for engineers.  This way, every engineer doesn't have to personally spend the time to get and stay up to speed with: 
  - the latest reactions from users
  - all of the motivations and discussions from the previous rounds of wireframe revisions that were thrown away
  - how the UI has evolved in previous releases to better serve the people using and building it
- Wireframing is important for both maintaining the quality of our work and outlining what work needs to be done.
- With Figma, thanks to its powerful component and auto-layout features, we can create high-fidelity wireframes - fast. We can iterate quickly without costing more work and less [sunk-cost fallacy](https://dictionary.cambridge.org/dictionary/english/sunk-cost-fallacy).

## Why do we use one repo?
At Fleet, we keep everything in one repo. The only exception is when we're working on something confidential since GitHub does not allow confidential issues inside public repos. Here's why:

- One repo is easier to manage. It has less surface area for keeping content up to date and reduces the risk of things getting lost and forgotten.
- Our work is more visible and accessible to the community when all project pieces are available in one repo. 
- One repo pools GitHub stars and more accurately reflects Fleet’s presence.
- One repo means one set of automations and labels to manage, resulting in a consistent GitHub experience that is easier to keep organized.

## Why organize work in team-based kanban boards?
It's helpful to have a consistent framework for how every team works, plans, and requests things from each other. Fleet's kanban boards are that framework, and they cover three goals:

1. **Intake:** Give people from anywhere in the world the ability to request something from a particular team (i.e., add it to their backlog).
2. **Planning:** Give the team's manager and other team members a way to plan the next three-week iteration of what the team is working on in a world (the board) where the team has ownership and feels confident making changes.
3. **Shared to-do list:** What should I work on next? Who needs help? What important work is blocked? Is that bug fix merged yet? When will it be released? When will that new feature ship? What did I do yesterday?

## Why a three-week cadence?
The Fleet product is released every three weeks. By syncing the whole company to this schedule, we can:

- keep all team members (especially those who aren't directly involved with the core product) aware of the current version of Fleet and when the next release is shipping.
- align project planning and milestones across all teams, which helps us schedule our content calendar and manage company-wide goals.

## Why use agile methodology?
Releasing software iteratively gets changes and improvements into the hands of users faster and generally results in software that works. This makes contributors fitter, happier, and more productive. See [the agile manifesto](https://agilemanifesto.org/) for more information.

## Why the emphasis on training?
Investing in people and providing generous, prioritized training, especially up front, helps contributors understand what is going on at Fleet. By making training a prerequisite at Fleet, we can:
- help team members feel confident in the better decisions they make at work. 
- create a culture of helping others, which results in team members feeling more comfortable even if they aren’t familiar with the osquery, security, startup, or IT space. 


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

## Why handbook-first strategy?
The Fleet handbook provides team members with up-to-date information about how to do things in the company. By adopting the handbook-first strategy, we can encourage a culture of self-service and self-learning, which is essential for daily a-synchronous work as part of an all-remote team.

This strategy was inspired by GitLab, which uses it with great effect. Check out this [short three-minute video](https://www.youtube.com/watch?v=aZrK8AQM8Ro) about their take on the handbook-first approach.

## Why direct responsibility?
We use the concept of [directly responsible individuals](https://fleetdm.com/handbook/people#directly-responsible-individuals) (DRIs) to know who is responsible for what. Every group maintains its own dedicated [handbook page](https://fleetdm.com/handbook), which is kept up to date with accurate, current information, including the group's [kanban board](https://github.com/orgs/fleetdm/projects?type=beta), Slack channels, and recurring tasks ("rituals").

## Why group Slack channels?
Groups are organized around goals. Connecting people with the same goals helps them produce better results by fostering freer communication. While groups sometimes align with the organization chart, some groups consist of people who do not report to the same manager. For example, [product groups](https://fleetdm.com/handbook/product) like `#g-agent` include engineers, not just the product manager.

Every group at Fleet maintains specific Slack channels, which all group members join and keep unmuted.  Everyone else at Fleet is encouraged to mute these channels, using them only as needed.  Each channel has a directly responsible individual responsible for keeping up with all new messages, even if they aren't explicitly mentioned (`@`).

## Why does Fleet value Openness?
A lot of thought went into making Openness one of our values.

It might be easier to keep company information confidential. You don’t have to think so hard about what should or shouldn’t be public. Declassifying information later carries risk. Anything you release affects how people see your company.

But we prefer to focus on the benefits.

Sharing information helps other organizations, which means helping more end users.

Everything we release is free marketing. Publicizing information is marketing itself. We’re an open-source company after all.

Sure, people might point out our mistakes. But that just helps us grow. And that’s what open source is all about: creating and improving tools together as a community.

For more thoughts on Fleet's values, visit the [Values section of the Company page in our handbook](https://fleetdm.com/handbook/company#openness).

<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="Why this way?">
