# Engineering
This handbook page details processes specific to working [with](#team) and [within](#responsibilities) this department

## What we do
The 🚀 Engineering department at Fleet is directly responsible for writing and maintaining the [code](https://github.com/fleetdm/fleet) for Fleet's core product and infrastructure.


## Team
| Role                            | Contributor(s)           |
|:--------------------------------|:-----------------------------------------------------------------------------------------------------------|
| Director of Product Devleopment | [Luke Heath](https://www.linkedin.com/in/lukeheath/) _([@lukeheath](https://github.com/lukeheath))_
| Engineering Managers            | [George Karr](https://www.linkedin.com/in/george-karr-4977b441/) _([@georgekarrv](https://github.com/georgekarrv))_, [Sharon Katz](https://www.linkedin.com/in/sharon-katz-45b1b3a/) _([@sharon-fdm](https://github.com/sharon-fdm))_
| Product Quality Specialists     | [Reed Haynes](https://www.linkedin.com/in/george-karr-4977b441/) _([@xpkoala](https://github.com/xpkoala))_, [Sabrina Coy](https://www.linkedin.com/in/bricoy/) _([@sabrinabuckets](https://github.com/sabrinabuckets))_
| Infrastructure Engineers        | [Robert Fairburn](https://www.linkedin.com/in/robert-fairburn/) _([@rfairburn](https://github.com/rfairburn))_
| Developers                      | _See ["Current product groups"](https://fleetdm.com/handbook/company/product-groups#current-product-groups)_

<!-- 
| Software Engineers              | [Rachel Perkins](https://www.linkedin.com/in/rachelelysia/) _([@rachelelysia](https://github.com/rachelelysia))_, [Lucas Rodriguez](https://www.linkedin.com/in/lukmr/) _([@lucasmrod](https://github.com/lucasmrod))_, [Jacob Shandling](https://www.linkedin.com/in/jacob-shandling/) _([@jacobshandling](https://github.com/jacobshandling))_, [Tim Lee](https://www.linkedin.com/in/mostlikelee/) _([@mostlikelee](https://github.com/mostlikelee))_, [Jahziel Villasana-Espinoza](https://www.linkedin.com/in/jahziel-v/) _([@jahzielv](https://github.com/jahzielv))_, [Victor Lyuboslavsky](https://www.linkedin.com/in/lyuboslavsky/) _([@getvictor](https://github.com/getvictor))_, Sarah Gillespie _([@gillespi314](https://github.com/gillespi314))_, [Martin Angers](https://www.linkedin.com/in/martin-angers-3210305/) _([@mna](https://github.com/mna))_, [Roberto Dip](https://www.linkedin.com/in/roperzh) _([@roperzh](https://github.com/roperzh))_, [Gabe Hernandez](https://www.linkedin.com/in/gabriel-hernandez-gh) _([@ghernandez345](https://github.com/ghernandez345))_, [Marcos Oviedo](https://www.linkedin.com/in/marcosoviedo/) _([@marcosd4h](https://github.com/marcosd4h))_
-->

## Contact us
- Any community member can file a 🦟 ["Bug report"](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=).
  - Any Fleet team member can view the 🦟 ["Bugs" kanban board](https://app.zenhub.com/workspaces/-bugs-647f6d382e171b003416f51a/board) including the status on all reported bugs.
- If urgent, or if you need help submiting an issue, mention a [team member](#team) in the [#help-engineering](https://fleetdm.slack.com/archives/C019WG4GH0A) Slack channel.
- Any Fleet team member can view the dedicated sprint boards managed by this department:
  - 💻 [MDM (#g-mdm)](https://app.zenhub.com/workspaces/-g-mdm-current-sprint-63bc507f6558550011840298/board)
  - 🌟 [Endpoint ops (#g-endpoint-ops)](https://app.zenhub.com/workspaces/-g-endpoint-ops-current-sprint-63bd7e0bf75dba002a2343ac/board)

## Scrum at Fleet
- [Sprint ceremonies](#sprint-ceremonies)
- [Scrum boards](#scrum-boards)
- [Scrum items](#scrum-items)

Fleet [product groups](https://fleetdm.com/handbook/company/development-groups#what-are-product-groups) employ scrum, an agile methodology, as a core practice in software development. This process is designed around sprints, which last three weeks to align with our release cadence.

New tickets are estimated, specified, and prioritized on the roadmap:
- [Roadmap](https://app.zenhub.com/workspaces/-roadmap-ships-in-6-weeks-6192dd66ea2562000faea25c/board)

### Scrum items
Our scrum boards are exclusively composed of four types of scrum items:

1. **User stories**: These are simple and concise descriptions of features or requirements from the user's perspective, marked with the `story` label. They keep our focus on delivering value to our customers. Occasionally, due to ZenHub's ticket sub-task structure, the term 'epic' may be seen. However, we treat these as regular user stories.

2. **Sub-tasks**: These smaller, more manageable tasks contribute to the completion of a larger user story. Sub-tasks are labeled as `~sub-task` and enable us to break down complex tasks into more detailed and easier-to-estimate work units. Sub-tasks are always assigned to exactly one user story.

3. **Timeboxes**: Tasks that are specified to complete within a pre-defined amount of time are marked with the `timebox` label. Timeboxes are research or investigation tasks necessary to move a prioritized user story forward, sometimes called "spikes" in scrum methodology. We use the term "timebox" because it better communicates its purpose. Timeboxes are always assigned to exactly one user story.

4. **Bugs**: Representing errors or flaws that result in incorrect or unexpected outcomes, bugs are marked with the `bug` label. Like user stories and sub-tasks, bugs are documented, prioritized, and addressed during a sprint. Bugs [may be estimated or left unestimated](https://fleetdm.com/handbook/engineering#do-we-estimate-released-bugs-and-outages), as determined by the product group's engineering manager.

> Our sprint boards do not accommodate any other type of ticket. By strictly adhering to these four types of scrum items, we maintain an organized and focused workflow that consistently adds value for our users.

### Sprint ceremonies
Each sprint is marked by five essential ceremonies:

1. **Sprint kickoff**: On the first day of the sprint, the team, along with stakeholders, select items from the backlog to work on. The team then commits to completing these items within the sprint.
2. **Daily standup**: Every day, the team convenes for updates. During this session, each team member shares what they accomplished since the last standup, their plans until the next meeting, and any blockers they are experiencing. Standups should last no longer than fifteen minutes. If additional discussion is necessary, it takes place after the standup with only the required partipants. 
3. **Weekly estimation sessions**: The team estimates backlog items once a week (three times per sprint). These sessions help to schedule work completion and align the roadmap with business needs. They also provide estimated work units for upcoming sprints. The EM is responsible for the point values assigned to each item and ensures they are as realistic as possible.
4. **Sprint demo**: On the last day of each sprint, all engineering teams and stakeholders come together to review the next release. Engineers are allotted 3-10 minutes to showcase features, improvements, and bug fixes they have contributed to the upcoming release. We focus on changes that can be demoed live and avoid overly technical details so the presentation is accessible to everyone. Features should show what is capable and bugs should identify how this might have impacted existing customers and how this resolution fixed that. (These meetings are recorded and posted publicly to YouTube or other platforms, so participants should avoid mentioning customer names.  For example, instead of "Fastly", you can say "a publicly-traded hosting company", or use the [customer's codename](https://fleetdm.com/handbook/customers#customer-codenames).)
5. **Sprint retrospective**: Also held on the last day of the sprint, this meeting encourages discussions among the team and stakeholders around three key areas: what went well, what could have been better, and what the team learned during the sprint.

## Meetings
- [Goals](#goals)
- [Principles](#principles)
- [Sprint ceremonies](#sprint-ceremonies)
- [User story discovery](#user-story-discovery)
- [Eng together](#eng-together)
- [Group weeklies](#group-weeklies)
- [Product development process review](#product-development-process-review)

### Principles
- Support the [Maker Schedule](http://www.paulgraham.com/makersschedule.html) by keeping meetings to a minimum.
- Each individual must have a weekly or biweekly sync 1:1 meeting with their manager. This is key to making sure each individual has a voice within the organization.
- Favor async communication when possible. This is very important to make sure every stakeholder on a project can have a clear understanding of what’s happening or what was decided, without needing to attend every meeting (i.e., if a person is sick or on vacation or just life happened.)
- If an async conversation is not proving to be effective, never hesitate to hop on or schedule a call. Always document the decisions made in a ticket, document, or whatever makes sense for the conversation.

### Eng Together
This meeting is to disseminate engineering-wide announcements, promote cohesion across groups within the engineering team, and connect with engineers (and the "engineering-curious") in other departments. Held monthly for one hour.

#### Participants
Everyone at the company is welcome to attend. All engineers are asked to attend. The subject matter is focused on engineering.

#### Agenda
- Announcements
- Engineering KPIs review
- “Tech talks”
  - At least one engineer from each product group demos or discusses a technical aspect of their recent work.
  - Everyone is welcome to present on a technical topic. Add your name and tech talk subject in the agenda doc included in the Eng Together calendar event.
- Social
  - Structured and/or unstructured social activities

### User story discovery
User story discovery meetings are scheduled as needed to align on large or complicated user stories. Before a discovery meeting is scheduled, the user story must be prioritized for product drafting and go through the design and specification process. When the user story is ready to be estimated, a user story discovery meeting may be scheduled to provide more dedicated, synchronous time for the team to discuss the user story than is available during weekly estimation sessions.

All participants are expected to review the user story and associated designs and specifications before the discovery meeting.

#### Participants
- Product Manager
- Product Designer
- Engineering Manager
- Backend Software Engineer
- Frontend Software Engineer
- Product Quality Specialist

#### Agenda
- Product Manager: Why this story has been prioritized
- Product Designer: Walk through user journey wireframes
- Engineering Manager: Review specifications and any defined sub-tasks
- Software Engineers: Clarifying questions and implementation details
- Product Quality Specialist: Testing plan

### Group weeklies
A chance for deeper, synchronous discussion on topics relevant across product groups like “Frontend weekly”, “Backend weekly”, etc.

#### Participants
Anyone who wishes to participate.

#### Sample agenda (Frontend weekly)
- Discuss common patterns and conventions in the codebase
- Review difficult frontend bugs
- Write engineering-initiated stories

## Engineering-initiated stories
- [Creating an engineering-initiated story](#creating-an-engineering-initiated-story)

Engineering-initiated stories are types of user stories created by engineers to make technical changes to Fleet. Technical changes should improve the user experience or contributor experience. For example, optimizing SQL that improves the response time of an API endpoint improves user experience by reducing latency. A script that generates common boilerplate, or automated tests to cover important business logic, improves the quality of life for contributors, making them happier and more productive, resulting in faster delivery of features to our customers.

It is important to frame engineering-initiated user stories the same way we frame all user stories. Stay focused on how this technical change will drive value for our users.

Engineering-initiated stories follow the [user story drafting process](https://fleetdm.com/handbook/company/development-groups#drafting). Once your user story is created using the [new story template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=story%2C%3Aproduct&projects=&template=story.md&title=), add the `~engineering-initiated` label, assign it to yourself, and work with an EM or PM to progress the story through the drafting process.

> We prefer the term engineering-initiated stories over technical debt because the user story format helps keep us focused on our users.

### Creating an engineering-initiated story
1. Create a [new feature request issue](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=~engineering-initiated&projects=&template=feature-request.md&title=) in GitHub.
2. Ensure it is labeled with `~engineering-initiated` and the relevant product group. Remove any `~customer-request` label.
3. Assign it to yourself. You will own this user story until it is either prioritized or closed.
4. Schedule a time with an EM and/or PM to present your story. Iterate based on feedback.
5. You, your EM or PM can bring this to Feature Fest for consideration. All engineering-initiated changes go through the same [drafting process](https://fleetdm.com/handbook/product#intake) as any other story.

> We aspire to dedicate 20% of each sprint to technical changes, but may allocate less based on customer needs and business priorities.

## Documentation for contributors
Fleet's documentation for contributors can be found in the [Fleet GitHub repo](https://github.com/fleetdm/fleet/tree/main/docs/Contributing).

## Release process
This section outlines the release process at Fleet.

The current release cadence is once every three weeks and is concentrated around Wednesdays.

### Release freeze period
To ensure release quality, Fleet has a freeze period for testing beginning the Tuesday before the release at 9:00 AM Pacific. Effective at the start of the freeze period, new feature work will not be merged into `main`.

Bugs are exempt from the release freeze period.

### Freeze day
To begin the freeze, [open the repo on Merge Freeze](https://www.mergefreeze.com/installations/3704/branches/6847) and click the "Freeze now" button. This will freeze the `main` branch and require any PRs to be manually unfrozen before merging. PRs can be manually unfrozen in Merge Freeze using the PR number.

> Any Fleetie can [unfreeze PRs on Merge Freeze](https://www.mergefreeze.com/installations/3704/branches) if the PR contains documentation changes or bug fixes only. If the PR contains other changes, please confirm with your manager before unfreezing.

#### Check dependencies
Before kicking off release QA, confirm that we are using the latest versions of dependencies we want to keep up-to-date with each release. Currently, those dependencies are:

1. **Go**: Latest minor release

- Check the [version included in Fleet](https://github.com/fleetdm/fleet/blob/main/.github/workflows/build-binaries.yaml#L30).
- Check the [latest minor version of Go](https://go.dev/dl/). For example, if we are using `go1.19.8`, and there is a new minor version `go1.19.9`, we will upgrade.
- If the latest minor version is greater than the version included in Fleet, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current on-call engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer). Add the `~release blocker` label. We must upgrade to the latest minor version before publishing the next release.
- If the latest major version is greater than the version included in Fleet, [create a story](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=story%2C%3Aproduct&projects=&template=story.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current on-call engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer). This will be considered for an upcoming sprint. The release can proceed without upgrading the major version.

> In Go versioning, the number after the first dot is the "major" version, while the number after the second dot is the "minor" version. For example, in Go 1.19.9, "19" is the major version and "9" is the minor version. Major version upgrades are assessed separately by engineering.

2. **macadmins-extension**: Latest release

- Check the [latest version of the macadmins-extension](https://github.com/macadmins/osquery-extension/releases).
- Check the [version included in Fleet](https://github.com/fleetdm/fleet/blob/main/go.mod#L60).
- If the latest stable version of the macadmins-extension is greater than the version included in Fleet, [file a bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&projects=&template=bug-report.md&title=) and assign it to the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals) and the [current on-call engineer](https://fleetdm.com/handbook/engineering#how-to-reach-the-oncall-engineer).
- Add the `~release blocker` label.

>**Note:** Some new versions of the macadmins-extension include updates that require code changes in Fleet. Make sure to note in the bug that the update should be checked for any changes, like new tables, that require code changes in Fleet.

Our goal is to keep these dependencies up-to-date with each release of Fleet. If a release is going out with an old dependency version, it should be treated as a [critical bug](https://fleetdm.com/handbook/engineering#critical-bugs) to make sure it is updated before the release is published.

#### Create release QA issue
Next, create a new GitHub issue using the [Release QA template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=&projects=&template=smoke-tests.md&title=). Add the release version to the title, and assign the quality assurance members of the [MDM](https://fleetdm.com/handbook/company/development-groups#mdm-group) and [Endpoint ops](https://fleetdm.com/handbook/company/product-groups#endpoint-ops-group) product groups.

### Merging during the freeze period
We merge bug fixes and documentation changes during the freeze period, but we do not merge other code changes. This minimizes code churn and helps ensure a stable release. To merge a bug fix, you must first unfreeze the PR in [Merge Freeze](https://app.mergefreeze.com/installations/3704/branches), and click the "Unfreeze 1 pull request" text link. 


It is sometimes necessary to delay the release to allow time to complete partially merged feature work. In these cases, an exception process must be followed before merging during the freeze period. 

1. The engineer requesting the feature work merge exception during freeze notifies their Engineering Manager. 
2. The Engineering Manager notifies the QA lead for the product group and the [release ritual DRI](https://fleetdm.com/handbook/engineering#rituals). 
3. The Engineering Manager, QA lead, and [release ritual DRI](#rituals) must all approve the feature work PR before it is unfrozen and merged.

### Release readiness
After each product group finishes their QA process during the freeze period, the EM @ mentions the release ritual DRI in the #help-qa Slack channel. When all EMs have certified that they are ready for release, the release ritual DRI begins the [release process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md). 

### Release day
Documentation on completing the release process can be found [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md).

## Deploying to dogfood
After each Fleet release, the new release is deployed to Fleet's dogfood (internal) instance.

How to deploy a new release to dogfood:

1. Head to the **Tags** page on the fleetdm/fleet Docker Hub: https://hub.docker.com/r/fleetdm/fleet/tags
2. In the **Filter tags** search bar, type in the latest release (ex. v4.19.0).
3. Locate the tag for the new release and copy the image name. An example image name is "fleetdm/fleet:v4.19.0".
4. Head to the "Deploy Dogfood Environment" action on GitHub: https://github.com/fleetdm/fleet/actions/workflows/dogfood-deploy.yml
5. Select **Run workflow** and paste the image name in the **The image tag wished to be deployed.** field.

> Note that this action will not handle down migrations. Always deploy a newer version than is currently deployed.
>
> Note that "fleetdm/fleet:main" is not a image name, instead use the commit hash in place of "main".

## Milestone release ritual 
Immediately after publishing a new release, we close out the associated GitHub issues and milestones. 

### Update milestone in GitHub

1. **Rename current milestone**: In GitHub, [change the current milestone name](https://github.com/fleetdm/fleet/milestones) from `4.x.x-tentative` to `4.x.x`. `4.37.0-tentative` becomes `4.37.0`.

### ZenHub housekeeping
2. **Update product group boards**: In ZenHub, go to each product group board tracking the current release. Usually, these are [#g-endpoint-ops](https://app.zenhub.com/workspaces/-g-endpoint-ops-current-sprint-63bd7e0bf75dba002a2343ac/board) and [#g-mdm](https://app.zenhub.com/workspaces/-g-mdm-current-sprint-63bc507f6558550011840298/board).

3. **Remove milestone from unfinished items**: If you see any items in columns other than "Ready for release" tagged with the current milestone, remove that milestone tag. These items didn't make it into the release.

4. **Prep release items**: Make sure all items in the "Ready for release" column have the current milestone and sprint tags. If not, select all items in the column and apply the appropriate tags.

5. **Move user stories to drafting board**: Select all items in "Ready for release" that have the `story` label. Apply the `:product` label and remove the `:release` label. These items will move back to the product drafting board.

6. **Confirm and close**: Make sure that all items with the `story` label have left the "Ready for release" column. Select all remaining items in the "Ready for release" column and move them to the "Closed" column. This will close the related GitHub issues.

8. **Confirm and celebrate**: Now, head to the [Drafting](https://app.zenhub.com/workspaces/-drafting-ships-in-6-weeks-6192dd66ea2562000faea25c/board) board. Find all `story` issues with the current milestone (these are the ones you just moved). Move them to the "Confirm and celebrate" column. Product will close the issues during their [confirm and celebrate ritual](https://fleetdm.com/handbook/product#rituals).

9. **Close GitHub milestone**: Visit [GitHub's milestone page](https://github.com/fleetdm/fleet/milestones) and close the current milestone.

10. **Create next milestone**: Create a new milestone for the next versioned release, `4.x.x-tentative`.

11. **Remove the freeze**: [Open the repo in Merge Freeze](https://app.mergefreeze.com/installations/3704/branches/6847) and click the "Unfreeze" button. 

12. Announce that `main` is unfrozen and the milestone has been closed in #help-engineering.

## On-call rotation
- [The rotation](#the-rotation)
- [Responsibilities](#responsibilities)
- [Clearing the plate](#clearing-the-plate)
- [How to reach the on-call engineer](#how-to-reach-the-oncall-engineer)
- [Escalations](#escalations)
- [Handoff](#handoff)

### The rotation
See [the internal Google Doc](https://docs.google.com/document/d/1FNQdu23wc1S9Yo6x5k04uxT2RwT77CIMzLLeEI2U7JA/edit#) for the engineers in the rotation.

Fleet team members can also subscribe to the [shared calendar](https://calendar.google.com/calendar/u/0?cid=Y181MzVkYThiNzMxMGQwN2QzOWEwMzU0MWRkYzc5ZmVhYjk4MmU0NzQ1ZTFjNzkzNmIwMTAxOTllOWRmOTUxZWJhQGdyb3VwLmNhbGVuZGFyLmdvb2dsZS5jb20) for calendar events.

New engineers are added to the on-call rotation by their manager after they have completed onboarding and at least one full release cycle. We aim to alternate the rotation between product groups when possible. 

> The on-call rotation may be adjusted with approval from the EMs of any product groups affected. Any changes should be made before the start of the sprint so that capacity can be planned accordingly.

### Responsibilities
- [Second-line response](#second-line-response)
- [PR reviews](#pr-reviews)
- [Customer success meetings](#customer-success-meetings)
- [Improve documentation](#improve-documentation)

#### Second-line response
The on-call engineer is a second-line responder to questions raised by customers and community members.

The on-call engineer is responsible for the first response to community pull requests. 

Customer Support Engineers are responsible for the first response to Slack messages in the [#fleet channel](https://osquery.slack.com/archives/C01DXJL16D8) of osquery Slack, and other public Slacks. The Customer Success group is responsible for the first response to messages in private customer Slack channels.

We respond within 1-hour (during business hours) for interactions and ask the on-call engineer to address any questions sent their way promptly. When a Customer Support Engineer is unavailable, the on-call engineer may sometimes be asked to take over the first response duties. Note that we do not need to have answers within 1 hour -- we need to at least acknowledge and collect any additional necessary information, while researching/escalating to find answers internally. See [Escalations](#escalations) for more on this.

> Response SLAs help us measure and guarantee the responsiveness that a customer [can expect](https://fleetdm.com/handbook/company#values) from Fleet.  But SLAs aside, when a Fleet customer has an emergency or other time-sensitive situation ongoing, it is Fleet's priority to help them find them a solution quickly.

#### PR reviews
PRs from Fleeties are reviewed by auto-assignment of codeowners, or by selecting the group or reviewer manually.

PRs should remain in draft until they are ready to be reviewed for final approval, this means the feature is complete with tests already added. This helps keep our active list of PRs relevant and focused. It is ok and encouraged to request feedback while a PR is in draft to engage the team.

All PRs from the community are routed through the on-call engineer. For code changes, if the on-call engineer has the knowledge and confidence to review, they should do so. Otherwise, they should request a review from an engineer with the appropriate domain knowledge. It is the on-call engineer's responsibility to monitor community PRs and make sure that they are moved forward (either by review with feedback or merge).

#### Customer success meetings
The on-call engineer is encouraged to attend some of the customer success meetings during the week. Post a message to the #g-endpoint-ops Slack channel requesting invitations to upcoming meetings.

This has a dual purpose of providing more context for how our customers use Fleet. The engineer should actively participate and provide input where appropriate (if not sure, please ask your manager or organizer of the call).

#### Improve documentation
The on-call engineer is asked to read, understand, test, correct, and improve at least one doc page per week. Our goal is to 1, ensure accuracy and verify that our deployment guides and tutorials are up to date and work as expected. And 2, improve the readability, consistency, and simplicity of our documentation – with empathy towards first-time users. See [Writing documentation](https://fleetdm.com/handbook/marketing#writing-documentation) for writing guidelines, and don't hesitate to reach out to [#g-digital-experience](https://fleetdm.slack.com/archives/C01GQUZ91TN) on Slack for writing support. A backlog of documentation improvement needs is kept [here](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3A%22%3Aimprove+documentation%22).

### Clearing the plate
Engineering managers are asked to be aware of the [on-call rotation](https://docs.google.com/document/d/1FNQdu23wc1S9Yo6x5k04uxT2RwT77CIMzLLeEI2U7JA/edit#) and schedule a light workload for engineers while they are on-call. While it varies week to week considerably, the on-call responsibilities can sometimes take up a substantial portion of the engineer's time.

We aspire to clear sprint work for the on-call engineer, but due to capacity or other constraints, sometimes the on-call engineer is required for sprint work. When this is the case, the EM will work with the on-call engineer to take over support requests or @oncall assignment completely when necessary.

The remaining time after fulfilling the responsibilities of on-call is free for the engineer to choose their own path. Please choose something relevant to your work or Fleet's goals to focus on. If unsure, speak with your manager.

Some ideas:

- Do training/learning relevant to your work.
- Improve the Fleet developer experience.
- Hack on a product idea. Note: Experiments are encouraged, but not all experiments will ship! Check in with the product team before shipping user-visible changes.
- Create a blog post (or other content) for fleetdm.com.
- Try out an experimental refactor. 

### How to reach the on-call engineer
Oncall engineers do not need to actively monitor Slack channels, except when called in by the Community or Customer teams. Members of those teams are instructed to `@oncall` in `#help-engineering` to get the attention of the on-call engineer to continue discussing any issues that come up. In some cases, the Community or Customer representative will continue to communicate with the requestor. In others, the on-call engineer will communicate directly (team members should use their judgment and discuss on a case-by-case basis how to best communicate with community members and customers).

### Escalations
When the on-call engineer is unsure of the answer, they should follow this process for escalation.

To achieve quick "first-response" times, you are encouraged to say something like "I don't know the answer and I'm taking it back to the team," or "I think X, but I'm confirming that with the team (or by looking in the code)."

How to escalate:

1. Spend 30 minutes digging into the relevant code ([osquery](https://github.com/osquery/osquery), [Fleet](https://github.com/fleetdm/fleet)) and/or documentation ([osquery](https://osquery.readthedocs.io/en/latest/), [Fleet](https://fleetdm.com/docs)). Even if you don't know the codebase (or even the programming language), you can sometimes find good answers this way. At the least, you'll become more familiar with each project. Try searching the code for relevant keywords, or filenames.

2. Create a new thread in the [#help-engineering channel](https://fleetdm.slack.com/archives/C019WG4GH0A), tagging `@zwass` and provide the information turned up in your research. Please include possibly relevant links (even if you didn't find what you were looking for there). Zach will work with you to craft an appropriate answer or find another team member who can help.

### Handoff
The on-call engineer changes each week on Wednesday.

A Slack reminder should notify the on-call of the handoff. Please do the following:

1. The new on-call engineer should change the `@oncall` alias in Slack to point to them. In the
   search box, type "people" and select "People & user groups." Switch to the "User groups" tab.
   Click `@oncall`. In the right sidebar, click "Edit Members." Remove the former on-call, and add
   yourself.

2. Hand off newer conversations (Slack threads, issues, PRs, etc.). For more recent threads, the former on-call can unsubscribe from the thread, and the new on-call should subscribe. The former on-call should explicitly share each of
   these threads and the new on-call can select "Get notified about new replies" in the "..." menu.
   The former on-call can select "Turn off notifications for replies" in that same menu. It can be
   helpful for the former on-call to remain available for any conversations they were deeply involved
   in, so use your judgment on which threads to hand off. Anything not clearly handed off remains the responsibility of the former on-call engineer.

In the Slack reminder thread, the on-call engineer includes their retrospective. Please answer the following:

1. What were the most common support requests over the week? This can potentially give the new on-call an idea of which documentation to focus their efforts on.

2. Which documentation page did you focus on? What changes were necessary?

3. How did you spend the rest of your on-call week? This is a chance to demo or share what you learned.

## Incident postmortems
At Fleet, we take customer incidents very seriously. After working with customers to resolve issues, we will conduct an internal postmortem to determine any documentation or coding changes to prevent similar incidents from happening in the future. Why? We strive to make Fleet the best osquery management platform globally, and we sincerely believe that starts with sharing lessons learned with the community to become stronger together.

At Fleet, we do postmortem meetings for every production incident, whether it's a customer's environment or on fleetdm.com.

- [Postmortem document](#postmortem-document)
- [Postmortem meeting](#postmortem-meeting)
- [Postmortem action items](#postmortem-action-items)

### Postmortem document
Before running the postmortem meeting, copy this [Postmortem Template](https://docs.google.com/document/d/1Ajp2LfIclWfr4Bm77lnUggkYNQyfjePiWSnBv1b1nwM/edit?usp=sharing) document and populate it with some initial data to enable a productive conversation.

### Postmortem meeting
Invite all stakeholders, typically the team involved and QA representatives.

Follow the document topic by topic. Keep the goal in mind which is to take action items for addressing the root cause and making sure a similar incident will not happen again.

Distinguish between the root cause of the bug, which by that time was solved and released, and the root cause of why this issue reached our customers. These could be different issues. (e.g. the root cause of the bug was a coding issue, but the root causes (plural) of the event may be that the test plan did not cover a specific scenario, a lack of testing, and a lack of metrics to identify the issue quickly).

[Example Finished Document](https://docs.google.com/document/d/1YnETKhH9R7STAY-PaFnPy2qxhNht2EAFfkv-kyEwebQ/edit?usp=share_link)

### Postmortem action items
Each action item will have an owner that will be responsible for creating a Github issue promptly after the meeting. This Github issue should be prioritized with the relevant PM/EM.

## Outages
At Fleet, we consider an outage to be a situation where new features or previously stable features are broken or unusable.

- Occurences of outages are tracked in the [Outages](https://docs.google.com/spreadsheets/d/1a8rUk0pGlCPpPHAV60kCEUBLvavHHXbk_L3BI0ybME4/edit#gid=0) spreadsheet.
- Fleet encourages embracing the inevitability of mistakes and discourages blame games.
- Fleet stresses the critical importance of avoiding outages because they make customers' lives worse instead of better.

## Scaling Fleet
Fleet, as a Go server, scales horizontally very well. It’s not very CPU or memory intensive. However, there are some specific gotchas to be aware of when implementing new features. Visit our [scaling Fleet page](https://fleetdm.com/handbook/engineering/scaling-fleet) for tips on scaling Fleet as efficiently and effectively as possible.

## Load testing
The [load testing page](https://fleetdm.com/handbook/engineering/load-testing) outlines the process we use to load test Fleet, and contains the results of our semi-annual load test.

## Version support
To provide the most accurate and efficient support, Fleet will only target fixes based on the latest released version. In the current version fixes, Fleet will not backport to older releases.

Community version supported for bug fixes: **Latest version only**

Community support for support/troubleshooting: **Current major version**

Premium version supported for bug fixes: **Latest version only**

Premium support for support/troubleshooting: **All versions**

## Reviewing PRs from the community
If you're assigned a community pull request for review, it is important to keep things moving for the contributor. The goal is to not go more than one business day without following up with the contributor.

A PR should be merged if:

- It's a change that is needed and useful.
- The CI is passing.
- Tests are in place.
- Documentation is updated.
- Changes file is created.

For PRs that aren't ready to merge:

- Thank the contributor for their hard work and explain why we can't merge the changes yet.
- Encourage the contributor to reach out in the #fleet channel of osquery Slack to get help from the rest of the community.
- Offer code review and coaching to help get the PR ready to go (see note below).
- Keep an eye out for any updates or responses.

> Sometimes (typically for Fleet customers), a Fleet team member may add tests and make any necessary changes to merge the PR.

If everything is good to go, approve the review.

For PRs that will not be merged:

- Thank the contributor for their effort and explain why the changes won't be merged.
- Close the PR.

### Merging community PRs
When merging a pull request from a community contributor:

- Ensure that the checklist for the submitter is complete.
- Verify that all necessary reviews have been approved.
- Merge the PR.
- Thank and congratulate the contributor.
- Share the merged PR with the team in the #help-promote channel of Fleet Slack to be publicized on social media. Those who contribute to Fleet and are recognized for their contributions often become great champions for the project.

## Changes to tables' schema
Whenever a PR is proposed for making changes to our [tables' schema](https://fleetdm.com/tables/screenlock)(e.g. to schema/tables/screenlock.yml), it also has to be reflected in our osquery_fleet_schema.json file.

The website team will [periodically](https://fleetdm.com/handbook/marketing/website-handbook#rituals) update the json file with the latest changes. If the changes should be deployed sooner, you can generate the new json file yourself by running these commands:
```
cd website
./node_modules/sails/bin/sails.js run generate-merged-schema
```

> When adding a new table, make sure it does not already exist with the same name. If it does, consider changing the new table name or merge the two tables if it makes sense.

> If a table is added to our ChromeOS extension but it does not exist in osquery or if it is a table added by fleetd, add a note that mentions it. As in this [example](https://github.com/fleetdm/fleet/blob/e95e075e77b683167e86d50960e3dc17045e3c44/schema/tables/mdm.yml#L2).

## Quality
- [Human-oriented QA](#human-oriented-qa)
- [Finding bugs](#finding-bugs)
- [Outages](#outages)

### Human-oriented QA
Fleet uses a human-oriented quality assurance (QA) process to make sure the product meets the standards of users and organizations.

Automated tests are important, but they can't catch everything. Many issues are hard to notice until a human looks empathetically at the user experience, whether in the user interface, the REST API, or the command line.

The goal of quality assurance is to identify corrections and improvements before release:
- Bugs
- Edge cases
- Error message UX
- Developer experience using the API/CLI
- Operator experience looking at logs
- API response time latency
- UI comprehensibility
- Simplicity
- Data accuracy
- Perceived data freshness

### Finding bugs
To try Fleet locally for QA purposes, run `fleetctl preview`, which defaults to running the latest stable release.

To target a different version of Fleet, use the `--tag` flag to target any tag in [Docker Hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated), including any git commit hash or branch name. For example, to QA the latest code on the `main` branch of fleetdm/fleet, you can run: `fleetctl preview --tag=main`.

To start a preview without starting the simulated hosts, use the `--no-hosts` flag (e.g., `fleetctl preview --no-hosts`).

For each bug found, please use the [bug report template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=) to create a new bug report issue.

For unreleased bugs in an active sprint, a new bug is created with the `~unreleased bug` label. The `:release` label and associated product group label is added, and the engineer responsible for the feature is assigned. If QA is unsure who the bug should be assigned to, it is assigned to the EM. Fixing the bug becomes part of the story.

### Debugging
You can read our guide to diagnosing issues in Fleet on the [debugging page](https://fleetdm.com/handbook/engineering/debugging).

## Bug process
- [Bug states](#bug-states)
- [Finding bugs](#finding-bugs)
- [Outages](#outages)
- [All bugs](#all-bugs)

All bugs in Fleet are tracked by QA on the [bugs board](https://app.zenhub.com/workspaces/-bugs-647f6d382e171b003416f51a/board) in ZenHub.

### Bug states
The lifecycle stages of a bug at Fleet are:
1. [Inbox](#inbox)
2. [Reproduced](#reproduced)
3. [In product drafting (as needed)](#in-product-drafting-as-needed)
4. [In engineering](#in-engineering)
5. [Awaiting QA](#awaiting-qa)

The above are all the possible states for a bug as envisioned in this process. These states each correspond to a set of GitHub labels, assignees, and boards.

See [Bug states and filters](#bug-states-and-filters) at the end of this document for descriptions of these states and links to each GitHub filter.

#### Inbox
Quickly reproducing bug reports is a [priority for Fleet](https://fleetdm.com/handbook/company/why-this-way#why-make-it-obvious-when-stuff-breaks).

When a new bug is created using the [bug report form](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), it is in the "inbox" state.

At this state, the [bug review DRI](#rituals) (QA) is responsible for going through the inbox and documenting reproduction steps, asking for more reproduction details from the reporter, or asking the product team for more guidance.  QA has **1 business day** to move the bug to the next step (reproduced).

For community-reported bugs, this may require QA to gather more information from the reporter. QA should reach out to the reporter if more information is needed to reproduce the issue. Reporters are encouraged to provide timely follow-up information for each report. At two weeks since last communication QA will ping the reporter for more information on the status of the issue. After four weeks of stale communication QA will close the issue. Reporters are welcome to re-open the closed issue if more investigation is warranted.

Once reproduced, QA documents the reproduction steps in the description and moves it to the reproduced state. If QA or the engineering manager feels the bug report may be expected behavior, or if clarity is required on the intended behavior, it is assigned to the group's product manager. [See on GitHub](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+label%3Abug+label%3A%3Areproduce+sort%3Acreated-asc+).

##### Weekly bug review
QA has weekly check-in with product to go over the inbox items. QA is responsible for proposing “not a bug”, closing due to lack of response (with a nice message), or raising other relevant questions. All requires product agreement

QA may also propose that a reported bug is not actually a bug. A bug is defined as “behavior that is not according to spec or implied by spec.” If agreed that it is not a bug, then it's assigned to the relevant product manager to determine its priority.

#### Reproduced
QA has reproduced the issue successfully. It should now be transferred to engineering.

Remove the “reproduce” label, add the label of the relevant team (e.g. #g-endpoint-ops, #g-mdm, #g-infra, #g-website), and assign it to the relevant engineering manager. (Make your best guess as to which team. The EM will re-assign if they think it belongs to another team.) [See on GitHub](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+label%3Abug+label%3A%3Aproduct%2C%3Arelease+-label%3A%3Areproduce+sort%3Aupdated-asc+).

##### Fast track for Fleeties
Fleeties do not have to wait for QA to reproduce the bug. If you're confident it's reproducible, it's a bug, and the reproduction steps are well-documented, it can be moved directly to the reproduced state.

#### In product drafting (as needed)
If a bug requires input from product, the `:product` label is added, it is assigned to the product group's PM, and the bug is moved to the "Product drafting" column of the [bugs board](https://app.zenhub.com/workspaces/-bugs-647f6d382e171b003416f51a/board). It will stay in this state until product closes the bug, or removes the `:product` label and assigns to an EM.

#### In engineering
A bug is in engineering after it has been reproduced and assigned to an EM. If a bug meets the criteria for a [critical bug](https://fleetdm.com/handbook/engineering#critical-bugs), the `:release` and `~critical bug` labels are added, and it is moved to the "Current release' column of the bugs board. If the bug is a `~critical bug`, the EM follows the [critical bug notification process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#critical-bug-notification-process).

If the bug does not meet the criteria of a critical bug, the EM will determine if there is capacity in the current sprint for this bug. If so, the `:release` label is added, and it is moved to the "Current release' column on the bugs board. If there is no available capacity in the current sprint, the EM will move the bug to the "Sprint backlog" column where it will be prioritized for the next sprint.

When fixing the bug, if the proposed solution requires changes that would affect the user experience (UI, API, or CLI), notify the EM and PM to align on the acceptability of the change.

Engineering teams coordinate on bug fixes with the product team during the joint sprint kick-off review. If one team is at capacity and a bug needs attention, another team can step in to assist by following these steps:

For MDM support on Endpoint ops bugs:
- Remove the `#g-endpoint-ops` label and add `#g-mdm` label.
- Add `~assisting g-endpoint-ops` to clarify the bug’s origin.

For Endpoint ops support on MDM bugs:
- Remove the `#g-mdm` label and add `#g-endpoint-ops` label.
- Add `~assisting g-mdm` to clarify the bug’s origin.


Fleet [always prioritizes bugs](https://fleetdm.com/handbook/product#prioritizing-improvements) into a release within six weeks. If a bug is not prioritized in the current release, and it is not prioritized in the next release, it is removed from the "Sprint backlog" and placed back in the "Product drafting" column with the `:product` label. Product will determine if the bug should be closed as accepted behavior, or if further drafting is necessary.

#### Awaiting QA
Bugs will be verified as fixed by QA when they are placed in the "Awaiting QA" column of the relevant product group's sprint board. If the bug is verified as fixed, it is moved to the "Ready for release" column of the sprint board. Otherwise, the remaining issues are noted in a comment, and it is moved back to the "In progress" column of the sprint board.

### All bugs
- [See on GitHub](https://github.com/fleetdm/fleet/issues?q=is%3Aissue+is%3Aopen+label%3Abug).
- [See on ZenHub](https://app.zenhub.com/workspaces/-bugs-647f6d382e171b003416f51a/board).

#### Bugs opened this week
This filter returns all "bug" issues opened after the specified date. Simply replace the date with a YYYY-MM-DD equal to one week ago. [See on GitHub](https://github.com/fleetdm/fleet/issues?q=is%3Aissue+archived%3Afalse+label%3Abug+created%3A%3E%3DREPLACE_ME_YYYY-MM-DD).

#### Bugs closed this week

This filter returns all "bug" issues closed after the specified date. Simply replace the date with a YYYY-MM-DD equal to one week ago. [See on Github](https://github.com/fleetdm/fleet/issues?q=is%3Aissue+archived%3Afalse+is%3Aclosed+label%3Abug+closed%3A%3E%3DREPLACE_ME_YYYY-MM-DD).

## Release testing
- [Release blockers](#release-blockers)
- [Critical bugs](#critical-bugs)

When a release is in testing, QA should use the Slack channel #help-qa to keep everyone aware of issues found. All bugs found should be reported in the channel after creating the bug first.

When a critical bug is found, the Fleetie who labels the bug as critical is responsible for following the [critical bug notification process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#critical-bug-notification-process) below.

All unreleased bugs are addressed before publishing a release. Released bugs that are not critical may be addressed during the next release per the standard [bug process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#bug-process).

### Release blockers
Product may add the `~release blocker` label to user stories to indicate that the story must be completed to publish the next version of Fleet. Bugs are never labeled as release blockers.

### Critical bugs
A critical bug is a bug with the `~critical bug` label. A critical bug is defined as behavior that:
* Blocks the normal use of a workflow
* Prevents upgrades to Fleet
* Causes irreversible damage, such as data loss
* Introduces a security vulnerability

#### Critical bug notification process
We need to inform customers and the community about critical bugs immediately so they don’t trigger it themselves. When a bug meeting the definition of critical is found, the bug finder is responsible for raising an alarm.
Raising an alarm means pinging @here in the #help-product-design channel with the filed bug.

If the bug finder is not a Fleetie (e.g., a member of the community), then whoever sees the critical bug should raise the alarm. (We would expect this to be Customer success in the community Slack or QA in the bug inbox, though it could be anyone.)
Note that the bug finder here is NOT necessarily the **first** person who sees the bug. If you come across a bug you think is critical, but it has not been escalated, raise the alarm!

Once raised, product confirms whether or not it's critical and defines expected behavior.
When outside of working hours for the product team or if no one from product responds within 1 hour, then fall back to the #help-p1.

Once the critical bug is confirmed, Customer success needs to ping both customers and the community to warn them. If Customer success is not available, the on-call engineer is responsible for doing this.
If a quick fix workaround exists, that should be communicated as well for those who are already upgraded.

When a critical bug is identified, we will then follow the patch release process in [our documentation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#patch-releases).

## Measurement
We track the success of this process by observing the throughput of issues through the system and identifying where buildups (and therefore bottlenecks) are occurring.
The metrics are:
* Number of bugs opened this week
* Total # bugs open
* Bugs in each state (inbox, acknowledged, reproduced)
* Number of bugs closed this week

Each week these are tracked and shared in the weekly KPI sheet by Luke Heath.

### Definitions
In the above process, any reference to "product" refers to: Noah Talerman, Head of Product Design.
In the above process, any reference to "QA" refers to: Reed Haynes, Product Quality Specialist

## Infrastructure
- [Infrastructure links](#infrastructure-links)
- [Best practices](#best-practices)

### Infrastructure links
The following are quick links to infrastructure-related README files in both public and private repos that can be used as a quick reference for infrastructure-related code:

- [Terraform Module](https://github.com/fleetdm/fleet/blob/main/terraform/README.md)
- [Loadtesting](https://github.com/fleetdm/fleet/blob/main/infrastructure/loadtesting/terraform/readme.md)
- [Cloud](https://github.com/fleetdm/confidential/blob/main/infrastructure/cloud/template/README.md)
- [SSO](https://github.com/fleetdm/confidential/blob/main/infrastructure/sso/README.md)
- [VPN](https://github.com/fleetdm/confidential/blob/main/vpn/README.md)

### Best practices for containers
Follow the industry best practices when designing and deploying infrastructure. For containerized infrastructure, Google has created a [reference document](https://cloud.google.com/architecture/best-practices-for-operating-containers) as an ideal reference for these practices.

## Accounts
Engineering is responsible for managing third-party accounts required to support engineering infrastructure.

### Apple developer account
We use the official Fleet Apple developer account to notarize installers we generate for Apple devices. Whenever Apple releases new terms of service, we are unable to notarize new packages until the new terms are accepted.

When this occurs, we will begin receiving the following error message when attempting to notarize packages: "You must first sign the relevant contracts online." To resolve this error, follow the steps below.

1. Visit the [Apple developer account login page](https://appleid.apple.com/account?appId=632&returnUrl=https%3A%2F%2Fdeveloper.apple.com%2Fcontact%2F).

2. Log in using the credentials stored in 1Password under "Apple developer account".

3. Contact the Head of Business Operations to determine which phone number to use for 2FA.

4. Complete the 2FA process to log in.

5. Accept the new terms of service.

## Responsibilities

> work in progress, contributions welcome, please just make only one small change at a time per PR.  See https://fleetdm.com/handbook/company/leadership#vision-for-dept-handbook-pages for info

### Interview a developer candidate

As a hiring manager we want to ensure the interview process follows these steps in order. This process must follow [creating a new position](https://fleetdm.com/handbook/company/leadership#creating-a-new-position) through [receiving job applications](https://fleetdm.com/handbook/company/leadership#receiving-job-applications). Once the position is approved manage this process per candidate in [hiring pipeline](https://drive.google.com/drive/folders/1dLZaor9dQmAxcxyU6prm-MWNd-C-U8_1?usp=drive_link)

1. **Reach out**: If you are not already the primary contact with this candidate send an email or linkedin message introducing yourself and the intent that you would like the start the interview process including the link to the position and asking if they are comfortable with completing a coding exercise.
2. **Deliver code prompt**: After recieving confirmation that they are interested download the zip of the [code challenge](https://github.com/fleetdm/wordgame) and ask them to complete this and send their entry back within 5 business days.
3. **Test code prompt**: Verify the code runs and can complete the challenge correctly. Check the code for good style and tests that match our standards here at Fleet.
4. **Schedule manager interview**: Send the candidate a calendly link for 1hr to talk to you and screen them if they are a good fit for this role and our culture.
5. **Schedule technical interview**: Send the candidate a calendly link for 1hr to talk to a senior engineer on your team where the goal is to understand the thechnical capabilities of the candidate. An additional engineer can optionally join if available.
6. **Schedule DOPD interview**: Send the candidate a calendly link for 30m talk to the Director of Product Development @lukeheath.
7. **Schedule CTO interview**: Send the candidate a calendly link for 30m talk with our CTO @zwass.

If the candidate passes all of these steps then continue with [hiring a new team member](https://fleetdm.com/handbook/company/leadership#hiring-a-new-team-member).


### Renew MDM certificate signing request (CSR) 

The certificate signing request (CSR) certificate expires every year. It needs to be renewed prior to expiring. This is notified to the team by the MDM calendar event [IMPORTANT: Renew MDM CSR certificate](https://calendar.google.com/calendar/event?action=TEMPLATE&tmeid=NHM3YzZja2FoZTA4bm9jZTE3NWFvMTduMTlfMjAyNDA5MTFUMTczMDAwWiBjXzI0YjkwMjZiZmIzZDVkMDk1ZGQzNzA2ZTEzMWQ3ZjE2YTJmNDhjN2E1MDU1NDcxNzA1NjlmMDc4ODNiNmU3MzJAZw&tmsrc=c_24b9026bfb3d5d095dd3706e131d7f16a2f48c7a505547170569f07883b6e732%40group.calendar.google.com&scp=ALL)

Steps to renew the certificate:

1. Visit the [Apple developer account login page](https://appleid.apple.com/account?appId=632&returnUrl=https%3A%2F%2Fdeveloper.apple.com%2Fcontact%2F).
2. Log in using the credentials stored in 1Password under **Apple developer account**.
3. Verify you are using the **Enterprise** subaccount for Fleet Device Management Inc.
4. Generate a new certificate following the instructions in [MicroMDM](https://github.com/micromdm/micromdm/blob/c7e70b94d0cfc7710e5c92be20d4534d9d5a0640/docs/user-guide/quickstart.md?plain=1#L103-L118).
5. Note: `mdmctl` (a micromdm command for MDM vendors) will generate a `VendorPrivateKey.key` and `VendorCertificateRequest.csr` using an appropriate shared email relay and a passphrase (suggested generation method with pwgen available in brew / apt / yum `pwgen -s 32 -1vcy`)
6. Uploading `VendorCertificateRequest.csr` to Apple you will download a corresponding `mdm.cer` file
7. Convert the downloaded cert to PEM with `openssl x509 -inform DER -outform PEM -in mdm.cer -out server.crt.pem`
8. Update the **Config vars** in [Heroku](https://dashboard.heroku.com/apps/production-fleetdm-website/settings):
* Update `sails_custom__mdmVendorCertPem` with the results from step 7 `server.crt.pem`
* Update `sails_custom__mdmVendorKeyPassphrase` with the passphrase used in step 4
* Update `sails_custom__mdmVendorKeyPem` with `VendorPrivateKey.key` from step 4
9. Store updated values in [Confidential 1Password Vault](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=lcvkjobeheaqdgnz33ontpuhxq&i=byyfn2knejwh42a2cbc5war5sa&h=fleetdevicemanagement.1password.com)
10. Verify by logging into a normal apple account (not billing@...) and Generate a new Push Certificate following our [setup MDM](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#step-2-generate-an-apns-certificate) steps and verify the Expiration date is 1 year from today.
11. Adjust calendar event to be between 2-4 weeks before the next expiration.

## Rituals
<rituals :rituals="rituals['handbook/engineering/engineering.rituals.yml']"></rituals>


#### Stubs

##### Scrum boards
Please see 📖[handbook/company/engineering#contact-us](https://fleetdm.com/handbook/company/engineering#contact-us)

<meta name="maintainedBy" value="lukeheath">
<meta name="title" value="🚀 Engineering">
