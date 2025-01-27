# 🛩️ Product groups

This page covers what all contributors (fleeties or not) need to know in order to contribute changes to [the core product](https://fleetdm.com/docs).

When creating software, handoffs between teams or contributors are one of the most common sources of miscommunication and waste.  Like [GitLab](https://docs.google.com/document/d/1RxqS2nR5K0vN6DbgaBw7SEgpPLi0Kr9jXNGzpORT-OY/edit#heading=h.7sfw1n9c1i2t), Fleet uses product groups to minimize handoffs and maximize iteration and efficiency in the way we build the product.

> - Write down philosophies and show how the pieces of the development process fit together on this "🛩️ Product groups" page.
> - Use the dedicated [departmental](https://fleetdm.com/handbook/company#org-chart) handbook pages for [🚀 Engineering](https://fleetdm.com/handbook/engineering) and [🦢 Product Design](https://fleetdm.com/handbook/product) to keep track of specific, rote responsibilities and recurring rituals designed to be read and used only by people within those departments.


## Product roadmap

Fleet team members can read [Fleet's high-level product goals for the current quarter](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit?usp=sharing) (confidential Google Sheet).


## What are product groups?

Fleet organizes product development efforts into separate, cross-functional product groups that include product designers, developers, and quality engineers.  These product groups are organized by business goal, and designed to operate in parallel.

Security, performance, stability, scalability, database migrations, release compatibility, usage documentation (such as REST API and configuration reference), contributor experience, and support escalation are the responsibility of every product group.

At Fleet, [anyone can contribute](https://fleetdm.com/handbook/company#openness), even across product groups.

> Ideas expressed in wireframes, like code contributions, [are welcome from everyone](https://chat.osquery.io/c/fleet), inside or outside the company.


## Current product groups

| Product group                            | Goal _(value for customers and/or community)_                                                                          | Capacity |
|:-----------------------------------------|:-----------------------------------------------------------------------------------------------------------------------|:---------|
| [MDM](#mdm-group)                        | Increase and exceed maturity in the [device management](https://fleetdm.com/device-management) product category.       | 104      |
| [Orchestration](#orchestration-group)    | Increase and exceed maturity in the [orchestration](https://fleetdm.com/orchestration) product category.               | 104      |
| [Software](#software-group)              | Increase and exceed maturity in the [software management](https://fleetdm.com/software-management) product category.   | 104      |

\* The number of [estimated story points](https://fleetdm.com/handbook/company/communications#estimation-points) this group can take on per-sprint under ideal circumstances, used as a baseline number for planning and prioritizing user stories for drafting. In reality, capacity will vary as engineers are on-call, out-of-office, filling in for other product groups, etc.


### MDM group

The goal of the MDM group is to increase and exceed [Fleet's product maturity goals](https://fleetdm.com/device-management) in the "MDM" product category.

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Product Designer                  | [Marko Lisica](https://www.linkedin.com/in/markolisica/) _([@marko-lisica](https://github.com/marko-lisica))_
| Engineering Manager               | [George Karr](https://www.linkedin.com/in/george-karr-4977b441/) _([@georgekarrv](https://github.com/georgekarrv))_
| Product Manager                   | [Noah Talerman](https://www.linkedin.com/in/noah-talerman/) _([@noahtalerman](https://github.com/@noahtalerman))_
| Quality Assurance                 | [Gabe Lopez](https://www.linkedin.com/in/gabelopez/) _([@PezHub](https://github.com/PezHub))_
| Developer                         | [Martin Angers](https://www.linkedin.com/in/martin-angers-3210305/) _([@mna](https://github.com/mna))_, Sarah Gillespie _([@gillespi314](https://github.com/gillespi314))_, [Gabe Hernandez](https://www.linkedin.com/in/gabriel-hernandez-gh) _([@ghernandez345](https://github.com/ghernandez345))_, [Victor Lyuboslavsky](https://www.linkedin.com/in/lyuboslavsky/) _([@getvictor](https://github.com/getvictor))_

> The [Slack channel](https://fleetdm.slack.com/archives/C03C41L5YEL), [kanban release board](https://app.zenhub.com/workspaces/-g-mdm-current-sprint-63bc507f6558550011840298/board), and [GitHub label](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3A%23g-mdm) for this product group is `#g-mdm`.


### Orchestration group

The goal of the orchestration group is to increase and exceed [Fleet's product maturity goals in the orchestration category](https://fleetdm.com/orchestration).

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Product Designer                  | [Rachael Shaw](https://www.linkedin.com/in/rachaelcshaw/) _([@rachaelshaw](https://github.com/rachaelshaw))_
| Engineering Manager               | [Sharon Katz](https://www.linkedin.com/in/sharon-katz-45b1b3a/) _([@sharon-fdm](https://github.com/sharon-fdm))_
| Product Manager                   | [Noah Talerman](https://www.linkedin.com/in/noah-talerman/) _([@noahtalerman](https://github.com/@noahtalerman))_
| Quality Assurance                 | [Reed Haynes](https://www.linkedin.com/in/reed-haynes-633a69a3/) _([@xpkoala](https://github.com/xpkoala))_
| Developer                         | [Dante Catalfamo](https://www.linkedin.com/in/dante-catalfamo-a6330412b/) _([@dantecatalfamo](https://github.com/dantecatalfamo))_, [Scott Gress](https://www.linkedin.com/in/scottgress/) _([@sgress454](https://github.com/sgress454))_, [Lucas Rodriguez](https://www.linkedin.com/in/lukmr/) _([@lucasmrod](https://github.com/lucasmrod))_, [Jacob Shandling](https://www.linkedin.com/in/jacob-shandling/) _([@jacobshandling](https://github.com/jacobshandling))_

> The [Slack channel](https://fleetdm.slack.com/archives/C084F4MKYSJ), [kanban release board](https://app.zenhub.com/workspaces/g-orchestration-current-sprint-677307385e8685000f163867/board), and [GitHub label](https://github.com/fleetdm/fleet/labels/%23g-orchestration) for this product group is `#g-orchestration`.


### Software group

The goal of the software group is to increase and exceed [Fleet's product maturity goals in the software management category](https://fleetdm.com/software-management).

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Product Designer                  | [Eugene Kuo](https://www.linkedin.com/in/eugkuo/) _([@eugkuo](https://github.com/eugkuo))_
| Engineering Manager               | [Tim Lee](https://www.linkedin.com/in/mostlikelee/) _([@mostlikelee](https://github.com/mostlikelee))_
| Product Manager                   | [Noah Talerman](https://www.linkedin.com/in/noah-talerman/) _([@noahtalerman](https://github.com/@noahtalerman))_
| Quality Assurance                 | [Janis Watts](https://www.linkedin.com/in/janis-watts-b080ab94/) _([@jmwatts](https://github.com/jmwatts))_
| Developer                         | [Ian Littman](https://www.linkedin.com/in/ian-littman/) _([@iansltx](https://github.com/iansltx))_, [Rachel Perkins](https://www.linkedin.com/in/rachelelysia/) _([@rachelelysia](https://github.com/rachelelysia))_, [Konstantin Sykulev](https://www.linkedin.com/in/konstantins/) _([@ksykulev](https://github.com/ksykulev))_, [Jahziel Villasana-Espinoza](https://www.linkedin.com/in/jahziel-v/) _([@jahzielv](https://github.com/jahzielv))_

> The [Slack channel](https://fleetdm.slack.com/archives/C086V2QK76X), [kanban release board](https://app.zenhub.com/workspaces/g-software-67685f6ff1830a000f347a73/board), and [GitHub label][(https://github.com/fleetdm/fleet/labels?q=%23g-software](https://github.com/fleetdm/fleet/labels/%23g-software)) for this product group is `#g-software`.

## Making changes

Fleet's highest product ambition is to create experiences that users want.

To deliver on this mission, we need a clear, repeatable process for turning an idea into a set of cohesively-designed changes in the product. We also need to allow [open source contributions](https://fleetdm.com/handbook/company#open-source) at any point in the process from the wider Fleet community - these won't necessarily follow this process.

> Learn more about Fleet's philosophy and process for making interface changes to the product, and [why we use a wireframe-first approach](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

To make a change to Fleet:
- First, [write it down](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=~feature+fest%2C%3Aproduct&projects=&template=feature-request.md&title=)
  - For every customer/prospect requests, file a new GitHub issue. Whether the request is the same as an existing request will be determined by the Head of Product Design and a subject matter expert (SME) in the next step.
- Then, it will be looked at by Fleet's [Head of Product Design](https://fleetdm.com/handbook/product-design#team) and a SME [unpack the "why"](https://fleetdm.com/handbook/product-design#inbox-review).
  - For customer/prospect requests to be looked at, they must have a Gong snippet.
- Then, it will be [prioritized](https://fleetdm.com/handbook/company/product-groups#feature-fest) and written up as one or more user stories.
- Then, it will be [drafted](https://fleetdm.com/handbook/company/product-groups#drafting) (planned).
- Next, it will be [implemented](https://fleetdm.com/handbook/company/product-groups#implementing) and [released](https://fleetdm.com/handbook/engineering#release-process).

> Occasionally, a contributor outside of the [product groups](https://fleetdm.com/handbook/product-groups#current-product-groups) (open source contributor, member of the Customer Success team, etc.) will implement a change that was prioritized and drafted. On the user story for these changes, add the product group label (e.g. `#g-endpoint-ops`, `#g-mdm`), the `:release` label, and notify the product group's Engineer Manager to make sure the changes go through testing (QA) before release.

### Planned and unplanned changes

Most changes to Fleet are planned changes. They are [prioritized](https://fleetdm.com/handbook/product), defined, designed, revised, estimated, and scheduled into a release sprint _prior to starting implementation_.  The process of going from a prioritized goal to an estimated, scheduled, committed user story with a target release is called "drafting", or "the drafting phase".

Occasionally, changes are unplanned.  Like a patch for an unexpected bug, or a hotfix for a security issue.  Or if an open source contributor suggests an unplanned change in the form of a pull request.  These unplanned changes are sometimes OK to merge as-is.  But if they change the user interface, the CLI usage, or the REST API, then they need to go through drafting and reconsideration before merging.

> But wait, [isn't this "waterfall"?](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall) Waterfall is something else.  Between 2015-2023, GitLab and The Sails Company independently developed and coevolved similar delivery processes.  (What we call "drafting" and "implementation" at Fleet, is called "the validation phase" and "the build phase" at GitLab.)


### Experimental features

When a new feature is introduced it may be labeled as experimental. Experimental features are undergoing a rapid [incremental improvement and iteration process](https://fleetdm.com/handbook/company/why-this-way#why-lean-software-development) where new learnings may requires breaking changes. When we introduce experimental features, it is important that any API endpoints or configuration surface that may change in the future be clearly labeled as experimental.

1. Apply the `~experimental` label to all associated user stories.
2. Set the optional `isExperimental` property to "yes" in [pricing-features-table.yml](https://github.com/fleetdm/fleet/blob/main/handbook/company/pricing-features-table.yml).
3. Make sure all API endpoints and configuration surface documentation includes the following message:

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.


### Breaking changes

For product changes that cause breaking API or configuration changes or major impact for users (or even just the _impression_ of major impact!), the company plans migration thoughtfully. If the feature was released as stable (not experimental), the product group and E-group:

1. **Written:** Write a migration guide.
2. **Tested:** Test the migration thoroughly as engineers.
3. **Gamed out:** Pretend we are one or two key customers and try it out as a role play.
4. **Adapt:** If it becomes clear that the plan is insufficient, fix it.
5. **Communicate:** Create a plan for how to proactively communicate the change to customers.

All of the steps above happen prior to any breaking changes to stable features being prioritized for implementation.


#### API changes

To maintain consistency, ensure perspective, and provide a single pair of eyes in the design of Fleet's REST API and API documentation, there is a single Directly Responsible Individual (DRI). The API design DRI will review and approve any alterations at the pull request stage, instead of making it a prerequisite during drafting of the story. You may tag the DRI in a GitHub issue with draft API specs in place to receive a review and feedback prior to implementation. Receiving a pre-review from the DRI is encouraged if the API changes introduce new endpoints, or substantially change existing endpoints.

No API changes are merged without accompanying API documentation and approval from the DRI. The DRI is responsible for ensuring that the API design remains consistent and adequately addresses both standard and edge-case scenarios. The DRI is also the code owner of the API documentation Markdown file. The DRI is committed to reviewing PRs within one business day. In instances where the DRI is unavailable, the Head of Product will act as the substitute code owner and reviewer.


#### Changes to tables' schema

Whenever a PR is proposed for making changes to our [tables' schema](https://fleetdm.com/tables/screenlock)(e.g. to schema/tables/screenlock.yml), it also has to be reflected in our osquery_fleet_schema.json file.

The website team will [periodically](https://fleetdm.com/handbook/marketing/website-handbook#rituals) update the json file with the latest changes. If the changes should be deployed sooner, you can generate the new json file yourself by running these commands:
```
cd website
./node_modules/sails/bin/sails.js run generate-merged-schema
```

> When adding a new table, make sure it does not already exist with the same name. If it does, consider changing the new table name or merge the two tables if it makes sense.

> If a table is added to our ChromeOS extension but it does not exist in osquery or if it is a table added by fleetd, add a note that mentions it, as in this [example](https://github.com/fleetdm/fleet/blob/e95e075e77b683167e86d50960e3dc17045e3c44/schema/tables/mdm.yml#L2).


### Drafting

"Drafting" is the art of defining a change, designing and shepherding it through the drafting process until it is ready for implementation.

The goal of drafting is to deliver software that works every time with less total effort and investment, without making contribution any less fun.  By researching and iterating [prior to development](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach), we design better product features, crystallize fewer bad, preemptive naming decisions, and achieve better throughput: getting more done in less time.

> Fleet's drafting process is focused first and foremost on product development, but it can be used for any kind of change that benefits from planning or a "dry run".  For example, imagine you work for a business who has decided to swap out one of your payroll or device management vendors.  You will probably need to plan and execute changes to a number of complicated onboarding/offboarding processes.


#### Drafting process

The DRI for defining and drafting issues for a product group is the product manager, with close involvement from the designer and engineering manager.  But drafting is a team effort, and all contributors participate.

A user story is considered ready for implementation once:
- [ ] User story [issue created](https://github.com/fleetdm/fleet/issues/new/choose)
- [ ] [Product group](https://fleetdm.com/handbook/company/product-groups) label added (e.g. `#g-mdm`, `#g-orchestration`, `#g-software`)
- [ ] Changes [specified](https://fleetdm.com/handbook/company/development-groups#drafting) and [designed](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach)
- [ ] [Designs revised and settled](#design-reviews)
- [ ] Reviewed and approved during [weekly user story review](#user-story-reviews)
- [ ] [All checklists are complete](#defining-done)
- [ ] [Estimated](https://fleetdm.com/handbook/company/why-this-way#why-scrum)
- [ ] [Scheduled](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence) for development

> All user stories intended for the next sprint are estimated by the last estimation session before the sprint begins. This makes sure contributors have adequate time to complete the current sprint and provide accurate estimates for the next sprint.


#### Writing a good user story

Good user stories are short, with clear, unambiguous language.
- What screen are they looking at?  (`As an observer on the host details page…`)
- What do they want to do? (`As an observer on the host details page, I want to run a permitted query.`)
- Don't get hung up on the "so that I can ________" clause.  It is helpful, but optional.
- Example: "As an admin I would like to be asked for confirmation before deleting a user so that I do not accidentally delete a user."


#### Is it actually a story?

User stories are small and independently valuable.
- Is it small enough? Will this task be likely to fit in 1 sprint when estimated?
- Is it valuable enough? Will this task drive business value when released, independent of other tasks?


#### Defining "done"

To successfully deliver a user story, the people working on it need to know what "done" means. 

Every user story has a product and engineering checklist that is completed before the user story is estimated. This populates the user story with the requirements, wireframes, and test plan necessary for the product group to effectively specify, estimate, implement, and test the change. The Product Designer is the DRI for completing the product checklist, and the Engineering Manager (EM) is the DRI for completing the engineering checklist.

When the Product Designer has completed the product checklist, it is moved to the "User story review" column of the drafting board and reviewed during the [weekly user story review](https://fleetdm.com/handbook/company/product-groups#user-story-reviews) rituals.

When a user story completes the review process, it is moved to the "Ready for spec" column on the drafting board and assigned to the product group's EM. The EM is responsible for completing the engineering checklist and finalizing the test plan with the QA Engineer before moving to the "Ready to estimate" column.


#### Providing context

User story issues contain an optional section called "Context".

This section is optional and hidden by default.  It can be included or omitted, as time allows.  As Fleet grows as an all-remote company with more asynchronous processes across timezones, we will rely on this section more and more.

Here are some examples of questions that might be helpful to answer:
- Why does this change matter more than you might think?
- What else should a contributor keep in mind when working on this change?
- Why create this user story?  Why should Fleet work on it?
- Why now?  Why prioritize this user story today?
- What is the business case?  How does this contribute to reaching Fleet's strategic goals?
- What's the problem?
- What is the current situation? Why does the current situation hurt?
- Who are the affected users?
- What are they doing right now to resolve this issue? Why is this so bad?

These questions are helpful for the product team when considering what to prioritize.  (The act of writing the answers is a lot of the value!)  But these answers can also be helpful when users or contributors (including our future selves) have questions about how best to estimate, iterate, or refine.


#### Initiate an air guitar session

Anyone in the product group can initiate an air guitar session.

1. Initiate: Create a user story and add the `~air-guitar` label to indicate that it is going through the air guitar process. Air guitar issues are always intended to be designed right away. If they can't be, the requestor is notified via at-mention in the issue (that person is either the CSM or AE).

> An air guitar session may be used to design features that won't be shipped.

2. Prioritize: Bring the user story to [feature fest](https://fleetdm.com/handbook/product#rituals). If the user story is prioritized, proceed through the regular steps of specifying and designing as outlined in the drafting process. However, keep in mind that these are conceptual and may or may not proceed to engineering.

> An air guitar session may be needed before the next feature fest. In this case, the Product Designer will prioritize the user story.

3. Review: Conduct an air guitar meeting where the idea or feature is discussed. Involve roles like the product manager, designer, and a sampling of engineers to provide various perspectives.

4. Feedback: Collect internal feedback and iterate on the design. Optionally, conduct customer interviews or gather external feedback.

5. Document: Summarize the learnings, decisions, and next steps in the user story issue.

6. Decide: Assign the issue to the Head of Product Design to determine an outcome:
  1. Move forward with the formal drafting process leading to engineering.
  2. Keep it open for future consideration.
  3. Discard if it is invalidated through the process.

Air guitar sessions are timeboxed to ensure they are fast and focused. Documentation from this process may inform future user stories and can be invaluable when revisiting the idea at a later stage. While the air guitar process is exploratory in nature, it should be thorough enough to provide meaningful insights and data for future decision-making.


### Implementing


#### Developing from wireframes

Please read carefully and [pay special attention](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) to UI wireframes.

Designs have usually gone through multiple rounds of revisions, but they could easily still be overlooking complexities or edge cases! When you think you've discovered a blocker, here's how to proceed:

**For implementation concerns...**

Communicate. Leave a comment [mentioning the appropriate PM](https://fleetdm.com/handbook/company/product-groups#current-product-groups) so they can update the user story and estimation to reflect your new understanding of the issue.

**For all other concerns...**

At Fleet, we prioritize [iteration](https://fleetdm.com/handbook/company#results). So before raising the alarm, think through the following:

+ Would addressing this add design work and/or delay shipping the feature?
+ Will this hurt the first-time user experience if we ship as-is?
+ Is this change a "one-way door"?

After these considerations, if you still think you've found a blocker, alert the [appropriate PM](https://fleetdm.com/handbook/company/product-groups#current-product-groups) so that the user story can be brought back for [expedited drafting](https://fleetdm.com/handbook/product#expedited-drafting). Otherwise, make a [feature request](https://fleetdm.com/handbook/product#intake).


#### Sub-tasks

The simplest way to manage work is to use a single user story issue, then pass it around between contributors/asignees as seldom as possible.  But on a case-by-case basis, for particular user stories and teams, it can sometimes be worthwhile to invest additional overhead in creating separate **unestimated sub-task** issues ("sub-tasks").

A user story is estimated to fit within 1 sprint and drives business value when released, independent of other stories.  Sub-tasks are not.

Sub-tasks:
- can be created by anyone
- add extra management overhead and should be used sparingly
- do NOT have nested sub-tasks
- will NOT necessarily, in isolation, deliver any business value
- are always attached to exactly ONE top-level "user story" (which does drive business value)
- are included as links in the parent user story's "definition of done" checklist
- are NOT the best place to post GitHub comments (instead, concentrate conversation in the top-level "user story" issue)
- will NOT be looked at or QA'd by quality assurance


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


## Release testing

When a release is in testing, QA should use the Slack channel #help-qa to keep everyone aware of issues found. All bugs found should be reported in the channel after creating the bug first.

When a critical bug is found, the Fleetie who labels the bug as critical is responsible for following the [critical bug notification process](https://fleetdm.com/handbook/engineering#notify-community-members-about-a-critical-bug) below.

All unreleased bugs are addressed before publishing a release. Released bugs that are not critical may be addressed during the next release per the standard [bug process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#bug-process).

 - **Release blockers:** Product may add the `~release blocker` label to user stories to indicate that the story must be completed to publish the next version of Fleet. Bugs are never labeled as release blockers.

- **Critical bugs:** A critical bug is a bug with the `~critical bug` label. A critical bug is defined as behavior that:
  - Blocks the normal use of a workflow
  - Prevents upgrades to Fleet
  - Causes irreversible damage, such as data loss
  - Introduces a security vulnerability


### Notify the community about a critical bug

We inform customers and the community about critical bugs immediately so they don’t trigger it themselves. When a bug meeting the definition of critical is found, the bug finder is responsible for raising an alarm. Raising an alarm means pinging @here in the `#g-mdm` or `#g-endpoint-ops` channel with the filed bug.

If the bug finder is not a Fleetie (e.g., a member of the community), then whoever sees the critical bug should raise the alarm. Note that the bug finder here is NOT necessarily the **first** person who sees the bug. If you come across a bug you think is critical, but it has not been escalated, raise the alarm!

Once raised, product design confirms whether or not it's critical and defines expected behavior. When outside of working hours for the product design team or if no one from product design responds within 1 hour, then fall back to the #help-p1 channel.

Once the critical bug is confirmed, a [priority label](https://fleetdm.com/handbook/company/communications#high-priority-user-stories-and-bugs) is applied and the priority response process begins. Customer Success notifies impacted customers and the community if community features are impacted. If Customer Success is not available, the on-call engineer or infrastructure on-call engineer is responsible for this. If a quick fix workaround exists, that should be communicated as well for those who are already upgraded.

The relevant release page on GitHub is updated to indicate that the release contains a critical bug, as shown on the [fleet-v4.45.0 release page](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.45.0).

When a critical bug is identified, we will then follow the patch release process in [our documentation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#patch-releases).

> After a critical bug is fixed, [an incident postmortem](https://fleetdm.com/handbook/engineering#perform-an-incident-postmortem) is scheduled by the EM of the product group that fixed the bug.


## Feature fest

To stay in-sync with our customers' needs, Fleet accepts feature requests from customers and community members on a sprint-by-sprint basis. 

Features that meet a [criteria for prioritization](#criteria-for-prioritization) are prioritized at the 🎁🗣 Feature Fest meeting. 

Anyone in the company is invited to submit requests or simply listen in on the 🎁🗣 Feature Fest meeting. Folks from the wider community can also [request an invite](https://fleetdm.com/contact).

### Making a request

To make a feature request or advocate for a feature request from a customer or community member, [create an issue](https://github.com/fleetdm/fleet/issues/new/choose) using the feature request template. If you found that an issue already exists, add the `:product` label to it.

New requests are reviewed daily by the Head of Product Design and a former IT admin during the ["Unpacking the why"](https://fleetdm.com/handbook/product-design#unpacking-the-why) call. If the request meets the [criteria for prioritization](#criteria-for-prioritization), the request will be added to the upcoming feature fest (`~feature fest` label). If it doesn't, the request will be put to the side and the requester will be notified.


### Criteria for prioritization

To prioritize a new feature, it must meet one of these criteria:

1. Bug
2. Small UX improvement that isn't quite a bug but it's so small that it's worthwhile
3. Contributes to Fleet's [quarterly key results (KRs)](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit?gid=1846478041#gid=1846478041&range=A1)
4. High priority customer request (customer request, workflow blocking, etc.)
5. Prospect request in an order form 

If an issue has the `~feature fest` label, then it's a new feature request that will be weighed at the next 🎁🗣 Feature Fest meeting.

If an issue has the `~customer request` label, then it's a feature request that's already been prioritized. It will have one or more user stories that will be worked on in the current quarter.

If an issue has the `:product` and `story` label, then it's a user story that is currently in progress ([drafting](https://fleetdm.com/handbook/company/development-groups#drafting)). The user story will include a link to the original feature request issue.


### How feature requests are prioritized

Prioritization of new feature requests happens at the 🎁🗣 Feature Fest meeting.

Before the 🎁🗣 Feature Fest meeting, the [Customer renewals DRI](https://fleetdm.com/handbook/company/communications#directly-responsible-individuals-dris) adds customer requests to the 🎁🗣 Feature Fest board (`~feature fest` label) that are a high priority.

Before the meeting, the Feature prioritization DRI adds requests from Fleet's roadmap that contribute to Fleet's [quarterly key results (KRs)](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit?gid=1846478041#gid=1846478041&range=A1).

At the **🎁🗣 Feature Fest** meeting, the Feature prioritization DRI weighs all requests in the inbox. When the team weighs a request, it is immediately prioritized or put to the side (not prioritized).

- A request is _prioritized_ when the Feature prioritization DRI decides it is a priority.
- A request is _put to the side_ when the business perceives competing priorities as more pressing in the immediate moment.

If a feature is not prioritized during a 🎁🗣 Feature Fest meeting, it only means the feature has been rejected _at that time_. Requestors will be notified by the Feature prioritization DRI, and they can add their request back to the feature fest board (`~feature fest` label) to bring it back to a future meeting.


### After the feature is accepted

After the 🎁🗣 Feature fest meeting, the feature prioritization DRI will clear the 🎁 Feature fest board as follows:
- Prioritized features: Remove the `~feature fest` label, add the `~customer request` label, create a new user story with the `:product` label, add a link from the original request to the user story, notify the requester, and move the user story to the "Ready" column in the drafting board. The user story will then be assigned to a [Product Designer](https://fleetdm.com/handbook/company/product-groups#current-product-groups) during the "Design sprint kick-off" ritual.
- Put to the side features: Remove `~feature fest` label and notify the requestor.

> The product team's commitment to the requester is that the prioritized user story will be delivered or the requester will be notified within 1 business day of the decision to de-prioritize the story.

A story may be de-prioritized when its relative priority falls below new requests and there is not enough room in the upcoming engineering sprint. Since Fleet does not maintain a feature backlog, a story is only prioritized if it seems like it can be shipped in the upcoming 3 week engineering sprint. The relative priority of a story and engineering capacity may change over the course of a design sprint.
  - This may be because new higher-priority work (bugs or stories) was prioritized and/or the work in the current engineering sprint took longer than expected.

Just as when a feature request is not accepted in the 🎁🗣 Feature Fest meeting, whenever a feature is de-prioritized after it has been accepted, it only means that the feature has been _de-prioritized at this time_. It is up to the requester to bring the request back again at another 🎁🗣 Feature Fest meeting.


## Quality
The goal of quality assurance is to identify corrections and optimizations before release by verifying;
- Bugs
- Fixes for bugs
- Edge cases
- Error messages
- Developer experience (using the API/CLI)
- Operator experience (looking at logs)
- API response time latency
- UI comprehensibility
- Simplicity, data accuracy, and perceived data freshness

Fleet uses a human-oriented quality assurance (QA) process to make sure the product meets the standards of users and organizations. Automated tests are important, but they can't catch everything. Many issues are hard to notice until a human looks empathetically at the user experience, whether in the user interface, the REST API, or the command line.

You can read our guide to diagnosing issues in Fleet on the [debugging page](https://fleetdm.com/handbook/engineering/debugging). All bugs in Fleet are tracked by QA as [GitHub issues with the "bug" label](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3Abug).

- **Bug states:** The lifecycle stages of a bug at Fleet correspond to a set of GitHub labels, assignees, and boards.
  - [Inbox](https://fleetdm.com/handbook/company/product-groups#inbox)
  - [Reproduced](https://fleetdm.com/handbook/company/product-groups#reproduced)
  - [In product drafting (as needed)](https://fleetdm.com/handbook/company/product-groups#in-product-drafting-as-needed)
  - [In engineering](https://fleetdm.com/handbook/company/product-groups#in-engineering)
  - [Awaiting QA](https://fleetdm.com/handbook/company/product-groups#awaiting-qa)


### All bugs

- [See on GitHub](https://github.com/fleetdm/fleet/issues?q=is%3Aissue+is%3Aopen+label%3Abug).

- **Bugs opened this week:** This filter returns all "bug" issues opened after the specified date. Simply replace the date with a YYYY-MM-DD equal to one week ago. [See on GitHub](https://github.com/fleetdm/fleet/issues?q=is%3Aissue+archived%3Afalse+label%3Abug+created%3A%3E%3DREPLACE_ME_YYYY-MM-DD).

- **Bugs closed this week:** This filter returns all "bug" issues closed after the specified date. Simply replace the date with a YYYY-MM-DD equal to one week ago. [See on Github](https://github.com/fleetdm/fleet/issues?q=is%3Aissue+archived%3Afalse+is%3Aclosed+label%3Abug+closed%3A%3E%3DREPLACE_ME_YYYY-MM-DD).


#### Inbox

Quickly reproducing bug reports is a [priority for Fleet](https://fleetdm.com/handbook/company/why-this-way#why-make-it-obvious-when-stuff-breaks). When a new bug is created using the [bug report form](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), it is in the "inbox" state.

At this state, the bug review DRI (QA) is responsible for going through the inbox and documenting reproduction steps, asking for more reproduction details from the reporter, or asking the product team for more guidance.  QA has **1 business day** to move the bug to the next step (reproduced).

For community-reported bugs, this may require QA to gather more information from the reporter. QA should reach out to the reporter if more information is needed to reproduce the issue. Reporters are encouraged to provide timely follow-up information for each report. At two weeks since last communication QA will ping the reporter for more information on the status of the issue. After four weeks of stale communication QA will close the issue. Reporters are welcome to re-open the closed issue if more investigation is warranted.

Once reproduced, QA documents the reproduction steps in the description and moves it to the reproduced state. If QA or the engineering manager feels the bug report may be expected behavior, or if clarity is required on the intended behavior, it is assigned to the group's product manager. [See on GitHub](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+label%3Abug+label%3A%3Areproduce+sort%3Acreated-asc+).


#### Reproduced

QA has reproduced the issue successfully. It should now be transferred to engineering.

Remove the “reproduce” label, add the following labels:

1. The relevant product group (e.g. `#g-endpoint-ops`, `#g-mdm`, `#g-digital-experience`).
3. The `~released bug` label if the bug is in a published version of Fleet, or `~unreleased bug` if it is not yet published.
2. The `:incoming` label indicates to the EM that it is a new bug.
3. The `:release` label will place the bug on the team's release board.

Once the bug is properly labeled, assign it to the [relevant engineering manager](https://fleetdm.com/handbook/company/product-groups#current-product-groups). (Make your best guess as to which team. The EM will re-assign if they think it belongs to another team.) [See on GitHub](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+label%3Abug+label%3A%3Aproduct%2C%3Arelease+-label%3A%3Areproduce+sort%3Aupdated-asc+).

> **Fast track for Fleeties:** Fleeties do not have to wait for QA to reproduce the bug. If you're confident it's reproducible, it's a bug, and the reproduction steps are well-documented, it can be moved directly to the reproduced state.


#### In product drafting (as needed)

If a bug requires input from product the `:product` label is added, the `:release` label is removed, and the PM is assigned to the issue. It will stay in this state until product closes the bug, or removes the `:product` label and assigns to an EM.


#### In engineering

A bug is in engineering after it has been reproduced and assigned to an EM. If a bug meets the criteria for a [critical bug](https://fleetdm.com/handbook/engineering#critical-bugs), the `~critical bug` label is added, and the EM follows the [critical bug notification process](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#critical-bug-notification-process).

During daily standup, the EM will filter the board to only `:incoming` bugs and review with the team. The EM will remove the `:incoming` label, prioritize the bug in the "Ready" coulmn, unassign themselves, and assign an engineer or leave it unassigned for the first available engineer.

When fixing the bug, if the proposed solution requires changes that would affect the user experience (UI, API, or CLI), notify the EM and PM to align on the acceptability of the change.


#### Awaiting QA

Bugs will be verified as fixed by QA when they are placed in the "Awaiting QA" column of the relevant product group's sprint board. If the bug is verified as fixed, it is moved to the "Ready for release" column of the sprint board. Otherwise, the remaining issues are noted in a comment, and it is moved back to the "In progress" column of the sprint board.


## How to reach the developer on-call

Oncall engineers do not need to actively monitor Slack channels, except when called in by the Community or Customer teams. Members of those teams are instructed to `@oncall` in `#help-engineering` to get the attention of the on-call engineer to continue discussing any issues that come up. In some cases, the Community or Customer representative will continue to communicate with the requestor. In others, the on-call engineer will communicate directly (team members should use their judgment and discuss on a case-by-case basis how to best communicate with community members and customers).


### The developer on-call rotation

See [the internal Google Doc](https://docs.google.com/document/d/1FNQdu23wc1S9Yo6x5k04uxT2RwT77CIMzLLeEI2U7JA/edit#) for the engineers in the rotation.

Fleet team members can also subscribe to the [shared calendar](https://calendar.google.com/calendar/u/0?cid=Y181MzVkYThiNzMxMGQwN2QzOWEwMzU0MWRkYzc5ZmVhYjk4MmU0NzQ1ZTFjNzkzNmIwMTAxOTllOWRmOTUxZWJhQGdyb3VwLmNhbGVuZGFyLmdvb2dsZS5jb20) for calendar events.

New developers are added to the on-call rotation by their manager after they have completed onboarding and at least one full release cycle. We aim to alternate the rotation between product groups when possible.

> The on-call rotation may be adjusted with approval from the EMs of any product groups affected. Any changes should be made before the start of the sprint so that capacity can be planned accordingly.


### Developer on-call responsibilities

- **Second-line response**
The on-call developer is a second-line responder to questions raised by customers and community members.

The on-call developer is responsible for the first response to community pull requests.

Customer Support Engineers are responsible for the first response to Slack messages in the [#fleet channel](https://osquery.slack.com/archives/C01DXJL16D8) of osquery Slack, and other public Slacks. The Customer Success group is responsible for the first response to messages in private customer Slack channels.

We respond within 1-hour (during business hours) for interactions and ask the on-call developer to address any questions sent their way promptly. When a Customer Support Engineer is unavailable, the on-call developer may sometimes be asked to take over the first response duties. Note that we do not need to have answers within 1 hour -- we need to at least acknowledge and collect any additional necessary information, while researching/escalating to find answers internally. See [Escalations](#escalations) for more on this.

> Response SLAs help us measure and guarantee the responsiveness that a customer [can expect](https://fleetdm.com/handbook/company#values) from Fleet.  But SLAs aside, when a Fleet customer has an emergency or other time-sensitive situation ongoing, it is Fleet's priority to help them find them a solution quickly.

- **PR reviews**
PRs from Fleeties are reviewed by auto-assignment of codeowners, or by selecting the group or reviewer manually.

PRs should remain in draft until they are ready to be reviewed for final approval, this means the feature is complete with tests already added. This helps keep our active list of PRs relevant and focused. It is ok and encouraged to request feedback while a PR is in draft to engage the team.

All PRs from the community are routed through the on-call developer. For code changes, if the on-call developer has the knowledge and confidence to review, they should do so. Otherwise, they should request a review from an developer with the appropriate domain knowledge. It is the on-call developer's responsibility to monitor community PRs and make sure that they are moved forward (either by review with feedback or merge).

- **Customer success meetings**
The on-call developer is encouraged to attend some of the customer success meetings during the week. Post a message to the #g-customer-success Slack channel requesting invitations to upcoming meetings.

This has a dual purpose of providing more context for how our customers use Fleet. The developer should actively participate and provide input where appropriate (if not sure, please ask your manager or organizer of the call).

- **Documentation for contributors**
Fleet's documentation for contributors can be found in the [Fleet GitHub repo](https://github.com/fleetdm/fleet/tree/main/docs/Contributing).

The on-call developer is asked to read, understand, test, correct, and improve at least one doc page per week. Our goal is to 1, ensure accuracy and verify that our deployment guides and tutorials are up to date and work as expected. And 2, improve the readability, consistency, and simplicity of our documentation – with empathy towards first-time users. See [Writing documentation](https://fleetdm.com/handbook/marketing#writing-documentation) for writing guidelines, and don't hesitate to reach out to [#g-digital-experience](https://fleetdm.slack.com/archives/C01GQUZ91TN) on Slack for writing support. A backlog of documentation improvement needs is kept [here](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3A%22%3Aimprove+documentation%22).


### Escalations

When the on-call developer is unsure of the answer, they should follow this process for escalation.

To achieve quick "first-response" times, you are encouraged to say something like "I don't know the answer and I'm taking it back to the team," or "I think X, but I'm confirming that with the team (or by looking in the code)."

How to escalate:

1. Spend 30 minutes digging into the relevant code ([osquery](https://github.com/osquery/osquery), [Fleet](https://github.com/fleetdm/fleet)) and/or documentation ([osquery](https://osquery.readthedocs.io/en/latest/), [Fleet](https://fleetdm.com/docs)). Even if you don't know the codebase (or even the programming language), you can sometimes find good answers this way. At the least, you'll become more familiar with each project. Try searching the code for relevant keywords, or filenames.

2. Create a new thread in the [#help-engineering channel](https://fleetdm.slack.com/archives/C019WG4GH0A), tagging `@lukeheath` and provide the information turned up in your research. Please include possibly relevant links (even if you didn't find what you were looking for there). Luke will work with you to craft an appropriate answer or find another team member who can help.


### Changing of the guard

The on-call developer changes each week on Wednesday.

A Slack reminder should notify the on-call of the handoff. Please do the following:

1. The new on-call developer should change the `@oncall` alias in Slack to point to them. In the search box, type "people" and select "People & user groups." Switch to the "User groups" tab. Click `@oncall`. In the right sidebar, click "Edit Members." Remove the former on-call, and add yourself.

2. Hand off newer conversations (Slack threads, issues, PRs, etc.). For more recent threads, the former on-call can unsubscribe from the thread, and the new on-call should subscribe. The former on-call should explicitly share each of these threads and the new on-call can select "Get notified about new replies" in the "..." menu. The former on-call can select "Turn off notifications for replies" in that same menu. It can be helpful for the former on-call to remain available for any conversations they were deeply involved in, so use your judgment on which threads to hand off. Anything not clearly handed off remains the responsibility of the former on-call developer.

In the Slack reminder thread, the on-call developer includes their retrospective. Please answer the following:

1. What were the most common support requests over the week? This can potentially give the new on-call an idea of which documentation to focus their efforts on.

2. Which documentation page did you focus on? What changes were necessary?

3. How did you spend the rest of your on-call week? This is a chance to demo or share what you learned.


## Wireframes

- Showing these principles and ideas, to help remember the pros and cons and conceptualize the above visually.
- Figma: [⚗️ Fleet product project](https://www.figma.com/files/project/17318630/%E2%9A%97%EF%B8%8F-Fleet-product?fuid=1234929285759903870)

We have certain design conventions that we include in Fleet. We will document more of these over time.

**Design system**

The 🧩 ["Design System"](https://www.figma.com/file/8oXlYXpgCV1Sn4ek7OworP/%F0%9F%A7%A9-Design-System-(current)?type=design&mode=design&t=BytcobQwypszkxf5-1) component library in Figma is the source of truth for components. Components in the product (documented in [Storybook](https://fleetdm.com/storybook/)) should match the style of components defined in the Figma library. If the frontend component is inconsistent with one in the Figma library, treat that as a [bug](https://fleetdm.com/handbook/engineering#finding-bugs). As new components are being created, or existing components are being updated, ensure ensure updates are applied to both the Figma Library and Storybook and guidelines are documented in Figma.

**Table empty states**

Use `---`, with color `$ui-fleet-black-50` as the default UI for empty columns.

**Images**

Simple icons (aka any images used in the icon [design system component](https://www.figma.com/design/8oXlYXpgCV1Sn4ek7OworP/%F0%9F%A7%A9-Design-system-(current)?node-id=12-2&t=iO2vXbQ9Sc1kFVEJ-1)) are exported as SVGs. All other images are exported as PNGs, following the [Fleet website image](https://github.com/fleetdm/fleet/tree/main/website/assets/images) naming conventions.

**Form behavior**

Pressing the return or enter key with an open form will cause the form to be submitted.

**Internal links**

For text links that navigates the user to a different page within the Fleet UI, use the `$core-blue` color and `xs-bold` styling. You'll also want to make sure to use the underline style for when the user hovers over these links.

**External links**

For a link that navigates the user to an external site (e.g., fleetdm.com/docs), use the `$core-blue` color and `xs-bold` styling for the link's text. Also, place the link-out icon to the right of the link's text.

When including an external link, specify a [redirect on fleetdm.com](https://github.com/fleetdm/fleet/blob/7b751fa50a9a7f81112c5e65334ab05fa2e9e216/website/config/routes.js#L491-L518) rather than the original link. That way, if the URL changes, we can fix it immediately via a PR to the website and users will not need to upgrade to benefit from the fix. Once the design is settled, make a PR for the redirect as part of preparing the story for development.

**Tooltips**

All tooltips change the cursor to a question mark on hover. All tooltips have a solid background color.

There are two types of tooltips. The two types of tooltips have some unique styles:

1. Tooltips for text (column headers, input titles, inline text) appear when hovering over any text with a dashed underline. These tooltips use left-aligned text.

2. Tooltips for buttons, bubbles, table row elements, and other non-text elements appear when hovering over the element. These tooltips use center-aligned text. These tooltips include a centered arrow.

**Bold text**

For copy in the Fleet UI and Fleet documentation, use bold text when referencing UI elements such as buttons, links, column names, form field names, page names, and section names. For an example, check out the bold text in the docs [here](https://fleetdm.com/docs/using-fleet/mdm-disk-encryption#step-1-enforce-disk-encryption).

This way, if a user is scanning instructions, the bold text tells them what UI element they should look for.

In the docs, if a UI element is part of a section title (already bold) use double quotes. For an example, see this section title [here](https://fleetdm.com/docs/get-started/faq#what-happened-to-the-schedule-page).

**Copy in parentheses (additional information)**

When writing copy, consider whether additional information is necessary before adding it as a new sentence or in parentheses. If the information is needed, use parentheses with an incomplete sentence to keep the copy shorter.

**Writing the time**

When writing the time in the UI using "am" and "pm" abbreviations, write them **without space** between time and abbreviation, with **no punctuation**, and use **lowercase** letters (e.g. Working hours are 8am to 5pm).

**Writing error messages**

When writing error messages in the UI or CLI, follow these rules:
- If the solution to the error isn't obvious, write a message with the **error** followed by the **solution**. For example, "No hosts targeted. Make sure you provide a valid hostname, UUID, osquery host ID, or node key."
- If the solution is obvious when additional info is provided, write a message with the **error** followed by **additional info**. For example, "You don’t have permission to run the script. Only users with the maintainer role and above can run scripts."

**Fleetctl commands with `--hosts` or `--host` flag**

When designing CLI experience for commands that target hosts (e.g. `fleetctl query` or `fleetctl mdm run-command` when including the `--hosts` or `--host` flag), if a non-existing host is specified, use a single error message such as: `Error: No hosts targeted. Make sure you provide a valid hostname, UUID, osquery host ID, or node key.`

When writing copy for CLI help pages use the following descriptions:
```
$ fleetctl <command with --hosts/--host flag> -h

OPTIONS
--hosts     Hosts specified by hostname, uuid, osquery_host_id or node_key that you want to target.
--host      Host specified by hostname, uuid, osquery_host_id or node_key that you want to target.
```


## Meetings


### User story discovery

User story discovery meetings are scheduled as needed to align on large or complicated user stories. Before a discovery meeting is scheduled, the user story must be prioritized for product drafting and go through the design and specification process. When the user story is ready to be estimated, a user story discovery meeting may be scheduled to provide more dedicated, synchronous time for the team to discuss the user story than is available during weekly estimation sessions.

All participants are expected to review the user story and associated designs and specifications before the discovery meeting.

**Participants:**
- Product Manager
- Product Designer
- Engineering Manager
- Backend Software Engineer
- Frontend Software Engineer
- Product Quality Specialist

**Agenda:**
- Product Manager: Why this story has been prioritized
- Product Designer: Walk through user journey wireframes
- Engineering Manager: Review specifications and any defined sub-tasks
- Software Engineers: Clarifying questions and implementation details
- Product Quality Specialist: Testing plan

### Design reviews

Design reviews are conducted daily between the [Head of Product Design](https://fleetdm.com/handbook/product-design#team)(HPD) and contributors (most often Product Designers) proposing changes to Fleet's interfaces, such as the graphical user interface (GUI) or REST API.  This fast cadence shortens the feedback loop, makes progress visible, and encourages early feedback. This helps Fleet stay intentional about how the product is designed and minimize common issues like UI inconsistencies or accidental breaking changes to the API. If the HPD can't make it, a Product Designer from a product group attends to give feedback.

Anyone at Fleet can attend as a shadow. Shadows are asked to leave feedback/comments in the agenda doc without interrupting the meeting. This helps the team iterate and move designs to ready for spec faster. 

> In addition to design reviews, Fleeties or community member can provide feedback asynchronously at any time by finding the GitHub issue (user story) associated with the designs and @ mentioning the assigned Product Designer in the comment section.

Product Designers or other contributors come prepared to this meeting with their proposed changes in a GitHub issue.  Usually these are in the form of Figma wireframes, a pull request to the API docs showing changes, or a demo of a prototype.  

After the meeting, the contributor applies revisions and attends again the next day or as soon as possible for another go-round.  The Head of Product Design is responsible for looping in the right engineers, community members, and other subject-matter experts to iterate on and refine upcoming product changes in the best interest of the business.

Here are some tips for making this meeting effective:
- Say the user story out loud to remind participants of what it is.
- Avoid explaining or showing multiple ways it could work.  Show the one way you think it should work and let your work speak for itself.
- Make clear whether we're in "final review" or "feedback" mode:
  - Final review: contributor is 70% sure the design is 100% done.
  — Feedback: the design is not ready for final review, but contributor would like to get early feedback.
- For follow-ups, repeat the user story, but show only what has changed or been added since the last review.
- Bring 1 key engineer who has been helping out with the user story, when possible and helpful.
- Read Fleet's [best practices for meetings](https://fleetdm.com/handbook/company/communications#meetings).


### User story reviews

User story reviews [happen weekly](https://fleetdm.com/handbook/product-design#rituals) between the [Head of Product Design](https://fleetdm.com/handbook/product-design#team) and the each product group's Product Designer, Engineering Manager (EM), and Quality Assurance (QA) Engineer. During the call, the Product Designer presents all user stories that have completed product design in the past week and are in the "In review" column. The Product Designer is the DRI for completing all product checklist items before bringing to review.

The purpose of the review is to familiarize the EM and QA Engineer with the user story, and provide an opportunity to ask questions, clarify requirements, and highlight potential implementation issues. The first draft of the test plan produced by the Product Designer is reviewed and revised as needed during the call. The QA Engineer is the DRI for finalizing the test plan.

The purpose of the user story review is to align product, engineering, and QA on functionality and implementation details. Wireframe reviews occur daily during [design reviews](https://fleetdm.com/handbook/company/product-groups#design-reviews) where contributors are welcome to join and provide design feedback in the agenda document. However, sometimes there are design changes needed if a gap is discovered or an implementation issue is raised during user story review. If there are design changes, the user story is moved back to the "In progress" column for additional drafting. If there are no design changes, the story is assigned to the EM to [complete the drafting process](#defining-done) before bringing to estimation.


### Group weeklies

A chance for deeper, synchronous discussion on topics relevant across product groups like “Frontend weekly”, “Backend weekly”, etc.

**Participants:** Anyone who wishes to participate.

**Sample agenda from frontend weekly**
- Discuss common patterns and conventions in the codebase
- Review difficult frontend bugs
- Write engineering-initiated stories


### Eng Together

This meeting is to disseminate engineering-wide announcements, promote cohesion across groups within the engineering team, and connect with engineers (and the "engineering-curious") in other departments. Held monthly for one hour.

**Participants:** Everyone at the company is welcome to attend. All engineers are asked to attend. The subject matter is focused on engineering.

**Agenda:**
- Announcements
- Engineering KPIs review
- “Tech talks”
  - At least one member from each product group demos or discusses a technical subject relevant to engineering at Fleet.
  - Everyone is welcome to present on a technical topic. Add your name and tech talk subject in the agenda doc included in the Eng Together calendar event.
- Social
  - Structured and/or unstructured social activities


### New customer promise(s)

The Account Executive (AE) schedules this meeting before Fleet commits to one or more new customer promises. It's meant to streamline communication and encourage getting the best product decisions.

If the buyer (aka the "Santa") hasn't reviewed the price in the first order form or we don't have a date attatched to the promise(s), then we're not ready for this call.

**Participants:** AE, SC, and Head of Product Design.  (+ temporarily: CRO)

**Agenda:**
- Discuss new promises from an order form with promises
- Kick off 1 business day SLA for Head of Product Design to process this and work with CTO to deliver a revised order form back to the AE.


## Development best practices

- Remember the user.  What would you do if you saw that error message? [🔴](https://fleetdm.com/handbook/company#empathy)
- Communicate any blockers ASAP in your group Slack channel or standup. [🟠](https://fleetdm.com/handbook/company#ownership)
- Think fast and iterate.  [🟢](https://fleetdm.com/handbook/company#results)
- If it probably works, assume it's still broken.  Assume it's your fault.  [🔵](https://fleetdm.com/handbook/company#objectivity)
- Speak up and have short toes.  Write things down to make them complete. [🟣](https://fleetdm.com/handbook/company#openness)


## Product design conventions

Behind every [wireframe at Fleet](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach), there are 3 foundational design principles:

- **Use-case first.** Taking advantage of top-level features vs. per-platform options allows us to take advantage of similarities and avoid having two different ways to configure the same thing.
Start off cross-platform for every option, setting, and feature. If we **prove** it's impossible, _then_ work backward making it platform-specific.

- **Bridge the gap.** Implement enough help text, links, guides, gifs, etc that a reasonably persistent human being can figure it out just by trying to use the UI.
   Even if that means we have fewer features or slightly lower granularity (we can iterate and add more granularity later), make it easy enough to understand. Whether they're experienced Mac admins people or career Windows folks (even if someone has never used a Windows tool) they should _"get it"_.

- **Control the noise.** Bring the needs surface level, tuck away things you don't need by default (when possible, given time). For example, hide Windows controls if there are no Windows devices (based on number of Windows hosts).


## Scrum at Fleet

Fleet product groups employ scrum, an agile methodology, as a core practice in software development. This process is designed around sprints, which last three weeks to align with our release cadence.

New tickets are estimated, specified, and prioritized on the roadmap:
- [Roadmap](https://app.zenhub.com/workspaces/-roadmap-ships-in-6-weeks-6192dd66ea2562000faea25c/board)


### Scrum items

Our scrum boards are exclusively composed of four types of scrum items:

1. **User stories**: These are simple and concise descriptions of features or requirements from the user's perspective, marked with the `story` label. They keep our focus on delivering value to our customers. Occasionally, due to ZenHub's ticket sub-task structure, the term "epic" may be seen. However, we treat these as regular user stories.

2. **Sub-tasks**: These smaller, more manageable tasks contribute to the completion of a larger user story. Sub-tasks are labeled as `~sub-task` and enable us to break down complex tasks into more detailed and easier-to-estimate work units. Sub-tasks are always assigned to exactly one user story.

3. **Timeboxes**: Tasks that are specified to complete within a pre-defined amount of time are marked with the `~timebox` label. Timeboxes are research or investigation tasks necessary to move a prioritized user story forward, sometimes called "spikes" in scrum methodology. We use the term "timebox" because it better communicates its purpose. Timeboxes are always assigned to exactly one user story.

4. **Bugs**: Representing errors or flaws that result in incorrect or unexpected outcomes, bugs are marked with the `bug` label. Like user stories and sub-tasks, bugs are documented, prioritized, and addressed during a sprint. Bugs [may be estimated or left unestimated](https://fleetdm.com/handbook/engineering#do-we-estimate-released-bugs-and-outages), as determined by the product group's engineering manager.

> Our sprint boards do not accommodate any other type of ticket. By strictly adhering to these four types of scrum items, we maintain an organized and focused workflow that consistently adds value for our users.


## Sprints

Sprints align with Fleet's [3-week release cycle](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence).

On the first day of each release, all estimated issues are moved into the relevant section of the new "Release" board, which has a kanban view per group.

Sprints are managed in [Zenhub](https://fleetdm.com/handbook/company/why-this-way#why-make-work-visible). To plan capacity for a sprint, [create a "Sprint" issue](https://github.com/fleetdm/confidential/issues/new/choose), replace the fake constants with real numbers, and attach the appropriate labels for your product group.


### Sprint numbering

Sprints are numbered according to the release version. For example, for the sprint ending on June 30th, 2023, on which date we expect to release Fleet v4.34, the sprint is called the 4.34 sprint.


### Sprint ceremonies

Each sprint is marked by five essential ceremonies:

1. **Sprint kickoff**: On the first day of the sprint, the team, along with stakeholders, select items from the backlog to work on. The team then commits to completing these items within the sprint.
2. **Daily standup**: Every day, the team convenes for updates. During this session, each team member shares what they accomplished since the last standup, their plans until the next meeting, and any blockers they are experiencing. Standups should last no longer than fifteen minutes. If additional discussion is necessary, it takes place after the standup with only the required partipants.
3. **Weekly estimation sessions**: The team estimates backlog items once a week (three times per sprint). These sessions help to schedule work completion and align the roadmap with business needs. They also provide estimated work units for upcoming sprints. The EM is responsible for the point values assigned to each item and ensures they are as realistic as possible.
4. **Sprint demo**: On the last day of each sprint, all engineering teams and stakeholders come together to review the next release. Engineers are allotted 3-10 minutes to showcase features, improvements, and bug fixes they have contributed to the upcoming release. We focus on changes that can be demoed live and avoid overly technical details so the presentation is accessible to everyone. Features should show what is capable and bugs should identify how this might have impacted existing customers and how this resolution fixed that. (These meetings are recorded and posted publicly to YouTube or other platforms, so participants should avoid mentioning customer names.  For example, instead of "Fastly", you can say "a publicly-traded hosting company", or use the [customer's codename](https://fleetdm.com/handbook/customers#customer-codenames).)
5. **Sprint retrospective**: Also held on the last day of the sprint, this meeting encourages discussions among the team and stakeholders around three key areas: what went well, what could have been better, and what the team learned during the sprint.


## Outside contributions

[Anyone can contribute](https://fleetdm.com/handbook/company#openness) at Fleet, from inside or outside the company.  Since contributors from the wider community don't receive a paycheck from Fleet, they work on whatever they want.

Many open source contributions that start as a small, seemingly innocuous pull request come with lots of additional [unplanned work](https://fleetdm.com/handbook/company/development-groups#planned-and-unplanned-changes) down the road: unforseen side effects, documentation, testing, potential breaking changes, database migrations, [and more](https://fleetdm.com/handbook/company/development-groups#defining-done).

Thus, to ensure consistency, completeness, and secure development practices, no matter where a contribution comes from, Fleet will still follow the standard process for [prioritizing](#prioritizing-improvements) and [drafting](https://fleetdm.com/handbook/company/development-groups#drafting) a feature when it comes from the community.


#### Stubs
The following stubs are included only to make links backward compatible

##### Endpoint ops group
Please see [handbook/company/product-groups/orchestration](https://fleetdm.com/handbook/company/product-groups#orchestration)

##### Air guitar
Please see [handbook/company/initiate-an-air-guitar-session](https://fleetdm.com/handbook/company/product-groups#initiate-an-air-guitar-session)

##### High priority user stories and bugs
Please see [handbook/company/communications/high-priority-user-stories-and-bugs](https://fleetdm.com/handbook/company/communications#high-priority-user-stories-and-bugs)

<meta name="maintainedBy" value="lukeheath">
<meta name="title" value="🛩️ Product groups">
