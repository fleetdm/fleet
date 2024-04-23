# 🛩️ Product groups
This page covers what all contributors (fleeties or not) need to know in order to contribute changes to [the core product](https://fleetdm.com/docs).

When creating software, handoffs between teams or contributors are one of the most common sources of miscommunication and waste.  Like [GitLab](https://docs.google.com/document/d/1RxqS2nR5K0vN6DbgaBw7SEgpPLi0Kr9jXNGzpORT-OY/edit#heading=h.7sfw1n9c1i2t), Fleet uses product groups to minimize handoffs and maximize iteration and efficiency in the way we build the product.

> - Write down philosophies and show how the pieces of the development process fit together on this "🛩️ Product groups" page.
> - Use the dedicated [departmental](https://fleetdm.com/handbook/company#org-chart) handbook pages for [🚀 Engineering](https://fleetdm.com/handbook/engineering) and [🦢 Product Design](https://fleetdm.com/handbook/product) to keep track of specific, rote responsibilities and recurring rituals designed to be read and used only by people within those departments.

## Product roadmap
Fleet team members can read [Fleet's high-level product goals and planned releases for the current quarter and the next quarter](https://docs.google.com/document/d/11XEb__EJoGQJE9hXwaLrN45_5_k1NCi-zlJKH-OlKKk/edit#heading=h.33k3ii7z7ubc) (confidential Google Doc).

## What are product groups?
Fleet organizes product development efforts into separate, cross-functional product groups that include product designers, developers, and quality engineers.  These product groups are organized by business goal, and designed to operate in parallel.

Security, performance, stability, scalability, database migrations, release compatibility, usage documentation (such as REST API and configuration reference), contributor experience, and support escalation are the responsibility of every product group.

At Fleet, [anyone can contribute](https://fleetdm.com/handbook/company#openness), even across product groups.

> Ideas expressed in wireframes, like code contributions, [are welcome from everyone](https://chat.osquery.io/c/fleet), inside or outside the company.

## Current product groups

| Product group             | Goal _(value for customers and/or community)_                       | Capacity\* |
|:--------------------------|:--------------------------------------------------------------------|:-----------------|
| [Endpoint ops](#endpoint-ops-group)                     | Increase and exceed maturity in the "Endpoint operations" category.             | 130       |
| [MDM](#mdm-group)                                       | Reach maturity in the "MDM" product category.           | 130       |

\* The number of estimated story points this group can take on per-sprint under ideal circumstances, used as a baseline number for planning and prioritizing user stories for drafting. In reality, capacity will vary as engineers are on-call, out-of-office, filling in for other product groups, etc.

> _**What happened to "CX"?**  The customer experience (CX) group at Fleet is now [`#g-endpoint-ops`](#endpoint-ops-group)._
>
> _Why?  Making users and customers happier and more successful is the goal of _every_ product group.  This includes simpler usage, lovable design + help text + error messages, fixed bugs, responding quickly to incidents, using Fleet's brand standards, more successful customer onboarding, features that drive more win-win meetings with contributors and Fleet's sales team, and "whole product solutions", including professional services, design partnerships, and training._

### Endpoint ops group
The goal of the endpoint ops group is to increase and exceed [Fleet's product maturity goals in the endpoint operations category](https://drive.google.com/file/d/11yQ_2WG7TbRErUpMBKWu_hQ5wRIZyQhr/view?usp=sharing).

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Product Designer                  | [Rachael Shaw](https://www.linkedin.com/in/rachaelcshaw/) _([@rachaelshaw](https://github.com/rachaelshaw))_
| Engineering Manager               | [Sharon Katz](https://www.linkedin.com/in/sharon-katz-45b1b3a/) _([@sharon-fdm](https://github.com/sharon-fdm))_
| Product Manager                   | [Noah Talerman](https://www.linkedin.com/in/noah-talerman/) _([@noahtalerman](https://github.com/@noahtalerman))_
| Quality Assurance                 | [Reed Haynes](https://www.linkedin.com/in/reed-haynes-633a69a3/) _([@xpkoala](https://github.com/xpkoala))_
| Developer                         | [Jacob Shandling](https://www.linkedin.com/in/jacob-shandling/) _([@jacobshandling](https://github.com/jacobshandling))_, [Lucas Rodriguez](https://www.linkedin.com/in/lukmr/) _([@lucasmrod](https://github.com/lucasmrod))_, [Rachel Perkins](https://www.linkedin.com/in/rachelelysia/) _([@rachelelysia](https://github.com/rachelelysia))_, [Eric Shaw](https://www.linkedin.com/in/eric-shaw-1423831a9/) _([@eashaw](https://github.com/eashaw))_, [Tim Lee](https://www.linkedin.com/in/mostlikelee/) _([@mostlikelee](https://github.com/mostlikelee))_, [Victor Lyuboslavsky](https://www.linkedin.com/in/lyuboslavsky/) _([@getvictor](https://github.com/getvictor))_

> The [Slack channel](https://fleetdm.slack.com/archives/C01EZVBHFHU), [kanban release board](https://app.zenhub.com/workspaces/-g-endpoint-ops-current-sprint-63bd7e0bf75dba002a2343ac/board), and [GitHub label](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3A%23g-endpoint-ops) for this product group is `#g-endpoint-ops`.

### MDM group
The goal of the MDM group is to increase and exceed [Fleet's product maturity goals](https://drive.google.com/file/d/11yQ_2WG7TbRErUpMBKWu_hQ5wRIZyQhr/view?usp=sharing) in the "MDM" product category.

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Product Designer                  | [Marko Lisica](https://www.linkedin.com/in/markolisica/) _([@marko-lisica](https://github.com/marko-lisica))_
| Engineering Manager               | [George Karr](https://www.linkedin.com/in/george-karr-4977b441/) _([@georgekarrv](https://github.com/georgekarrv))_
| Product Manager                   | [Noah Talerman](https://www.linkedin.com/in/noah-talerman/) _([@noahtalerman](https://github.com/@noahtalerman))_
| Quality Assurance                 | [Position open](https://www.fleetdm.com/jobs/)
| Developer                         | [Gabe Hernandez](https://www.linkedin.com/in/gabriel-hernandez-gh) _([@ghernandez345](https://github.com/ghernandez345))_, [Roberto Dip](https://www.linkedin.com/in/roperzh) _([@roperzh](https://github.com/roperzh))_, Sarah Gillespie _([@gillespi314](https://github.com/gillespi314))_, [Martin Angers](https://www.linkedin.com/in/martin-angers-3210305/) _([@mna](https://github.com/mna))_, [Jahziel Villasana-Espinoza](https://www.linkedin.com/in/jahziel-v/) _([@jahzielv](https://github.com/jahzielv))_, [Dante Catalfamo](https://www.linkedin.com/in/dante-catalfamo-a6330412b/) _([@dantecatalfamo](https://github.com/dantecatalfamo))_

> The [Slack channel](https://fleetdm.slack.com/archives/C03C41L5YEL), [kanban release board](https://app.zenhub.com/workspaces/-g-mdm-current-sprint-63bc507f6558550011840298/board), and [GitHub label](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3A%23g-mdm) for this product group is `#g-mdm`.

## Making changes
Fleet's highest product ambition is to create experiences that users want.

To deliver on this mission, we need a clear, repeatable process for turning an idea into a set of cohesively-designed changes in the product. We also need to allow [open source contributions](https://fleetdm.com/handbook/company#open-source) at any point in the process from the wider Fleet community - these won't necessarily follow this process.

> Learn more about Fleet's philosophy and process for making interface changes to the product, and [why we use a wireframe-first approach](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

To make a change to Fleet:
- First, [get it prioritized](https://fleetdm.com/handbook/product-design).
- Then, it will be [drafted](https://fleetdm.com/handbook/company/product-groups#drafting) (planned).
- Next, it will be [implemented](https://fleetdm.com/handbook/company/product-groups#implementing) and [released](https://fleetdm.com/handbook/engineering#release-process).

### Planned and unplanned changes
Most changes to Fleet are planned changes. They are [prioritized](https://fleetdm.com/handbook/product), defined, designed, revised, estimated, and scheduled into a release sprint _prior to starting implementation_.  The process of going from a prioritized goal to an estimated, scheduled, committed user story with a target release is called "drafting", or "the drafting phase".

Occasionally, changes are unplanned.  Like a patch for an unexpected bug, or a hotfix for a security issue.  Or if an open source contributor suggests an unplanned change in the form of a pull request.  These unplanned changes are sometimes OK to merge as-is.  But if they change the user interface, the CLI usage, or the REST API, then they need to go through drafting and reconsideration before merging.

> But wait, [isn't this "waterfall"?](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall) Waterfall is something else.  Between 2015-2023, GitLab and The Sails Company independently developed and coevolved similar delivery processes.  (What we call "drafting" and "implementation" at Fleet, is called "the validation phase" and "the build phase" at GitLab.)

### Breaking changes
For product changes that cause breaking API or configuration changes or major impact for users (or even just the _impression_ of major impact!), the company plans migration thoughtfully.  That means the product department and E-group:

1. **Written:** Write a migration guide, even if that's just a Google Doc
2. **Tested:** Test out the migration ourselves, first-hand, as an engineer.
3. **Gamed out:** We pretend we are one or two key customers and try it out as a role play.
4. **Adapt:** If it becomes clear that the plan is insufficient, then fix it.
5. **Communicate:** Develop a plan for how to proactively communicate the change to customers.

That all happens prior to work getting prioritized for the change.

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
- [ ] [Product group](https://fleetdm.com/handbook/company/product-groups) label added (e.g. `#g-endpoint-ops`, `#g-mdm`)
- [ ] Changes [specified](https://fleetdm.com/handbook/company/development-groups#drafting) and [designed](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach)
- [ ] [Designs revised and settled](#design-reviews)
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

Since the goal of a user story is to implement certain changes to the product, the "definition of done" is written and maintained by the product manager.  But ultimately, this "definition of done" involves everyone in the product group.  We all collectively rely on accuracy of estimations, astuteness of designs, and cohesiveness of changes envisioned in order to deliver on time and without fuss.

Things to consider when writing the "definition of done" for a user story:
- **Design changes:** Does this story include changes to the user interface, or to how the CLI is used?  If so, those designs [will need to reviewed and revised](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) prior to estimation and before code is written.
- **Database schema migrations:** Does this story require changes to the database schema and need schema migrations?  If so, those migrations will need to be written as part of the changes, and additional quality assurance will be required.
- **Out-of-date docs:** How should [Fleet's documentation](https://fleetdm.com/docs) and [articles](https://fleetdm.com/articles) be updated to reflect the changes included in this user story?
  - **REST API:** If the Fleet API is changing, then the [REST API docs](https://fleetdm.com/docs/using-fleet/rest-api) will need to be updated.
  - **Configuration changes:** If this user story includes any changes to the way Fleet is configured, then the server configuration reference will need to be updated.
  - **Telemetry schema:** If osquery-compatible tables are changing as part of this user story, then the [telemetry data model reference](https://fleetdm.com/tables) will need to be updated.
  - **Other content:** What keywords should we [search for](https://github.com/fleetdm/fleet/search?q=path%3A%2Fdocs%2F+path%3A%2Farticles%2F+path%3A%2Fschema+sso&type=) to locate doc pages and articles that need updates?  List these and any other aspects/gotchas the product group should make sure are covered by the documentation.
- **Changes to paid features or tiers:** Does this user story add or change any paid features, or modify features' tiers? If so, describe the changes that should be made to the [pricing page](https://fleetdm.com/pricing), and make sure that code for any non-free features lives in the `ee/` directory.
- **Semantic versioning:** Does this change introduce breaking changes to Fleet's REST API or CLI usage?  If so, then we need to either figure out a crafty way to maintain backwards compatibility, or discuss a major version release with the CTO (`#help-engineering` and mention `@lukeheath`).
- **Scope transparency:** Does this change the scope of access that Fleet has on end user devices?  If so, describe this user story so that it includes the edits necessary to the [transparency guide](https://fleetdm.com/transparency).
- **Measurement?:** User stories are small changes that are best served by being released as quickly as possible in order to get real world feedback, whether quantitative or qualitative.  The norm is NOT to prioritize additional analytics or measurement work.  Is it especially important for the change described by this user story to come with extra investment in measuring usage, adoption, and success?  If so, describe what measurements we need to implement, along with the current state of any existing, related measurements.
- **QA:** Changes are tested by hand prior to submitting pull requests. In addition, quality assurance will do an extra QA check prior to considering this story "done".  Any special QA notes?
- **Follow-through:** Is there anything in particular that we should inform others (people who aren't in this product group) about after this user story is released?  For example: communication to specific customers, tips on how best to highlight this in a release post, gotchas, etc.

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

2. Prioritize: Bring the user story to [feature fest](https://fleetdm.com/handbook/product#rituals). If the user story is prioritized, proceed through the regular steps of specifying and designing as outlined in the drafting process. However, keep in mind that these are conceptual and may or may not proceed to engineering.

> An air guitar session may be needed before the next feature fest. In this case, the product group PM will prioritize the user story. 

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

Once the critical bug is confirmed, a [priority label](https://fleetdm.com/handbook/company/product-groups#high-priority-user-stories-and-bugs) is applied and the priority response process begins. Customer Success notifies impacted customers and the community if community features are impacted. If Customer Success is not available, the on-call engineer or infrastructure on-call engineer is responsible for this. If a quick fix workaround exists, that should be communicated as well for those who are already upgraded.

The relevant release page on GitHub is updated to indicate that the release contains a critical bug, as shown on the [fleet-v4.45.0 release page](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.45.0).

When a critical bug is identified, we will then follow the patch release process in [our documentation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Releasing-Fleet.md#patch-releases).

> After a critical bug is fixed, [an incident postmortem](https://fleetdm.com/handbook/engineering#preform-an-incident-postmortem) is scheduled by the EM of the product group that fixed the bug.

## Feature fest
To stay in-sync with our customers' needs, Fleet accepts feature requests from customers and community members on a sprint-by-sprint basis in the regular 🎁🗣 Feature Fest meeting. Anyone in the company is invited to submit requests or simply listen in on the 🎁🗣 Feature Fest meeting. Folks from the wider community can also [request an invite](https://fleetdm.com/contact). 

### Making a request
To make a feature request or advocate for a feature request from a customer or community member, [create an issue](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=~customer+request&projects=&template=feature-request.md&title=) and attend the next scheduled 🎁🗣 Feature Fest meeting. 

Requests are weighed from top to bottom while prioritizing attendee requests. This means that if the individual that added a feature request is not in attendance, the feature request will be discussed towards the end of the call if there's time.

To be acceptable for consideration, a request must:
- Have a clear proposed change
- Have a well-articulated underlying user need
- Specify the requestor (either internal stakeholder or customer or community user)

To help the product team, other pieces of information can be optionally included:
- How would they solve the problem without any changes if pressed?
- How does this change fit into the requester's overall usage of Fleet?
- What other potential changes to the product have you considered?

To ensure your request appears on the ["Feature Fest" board](https://app.zenhub.com/workspaces/-feature-fest-651b2962605ba29209324c57/board):
- Add the `~feature fest` label to your issue
- Add the relevant customer label (if applicable) 

To maximize your chances of having a feature accepted, requesters can visit the [🗣 Product office hours](#rituals) meeting to get feedback on requests prior to being accepted. 

### How feature requests are evaluated
Digestion of these new product ideas (requests) happens at the **🎁🗣 Feature Fest** meeting.

At the **🎁🗣 Feature Fest** meeting, the DRI (Head of Product) weighs all requests on the board. When the team weighs a request, it is immediately prioritized or put to the side.

Product Managers prioritize all potential product improvements worked on by Fleeties. Anyone (Fleeties, customers, and community members) are invited to suggest improvements.

- A _request is prioritized_ when the DRI decides it is a priority. When this happens, the team sets the request to be estimated within five business days.
- A _request is put to the side_ when the business perceives competing priorities as more pressing in the immediate moment.

If a feature is not prioritized during a 🎁🗣 Feature Fest meeting, it only means the feature has been rejected _at that time_. Requestors will be notified by the Head of Product, and they can resubmit their request at a future meeting.

Requests are weighed by:
- The completeness of the request (see [making a request](#making-a-request))
- How urgent the need is for the customer
- How much impact the request will have. This may be a wide impact across many customers and/or high impact on one
- How well the request fits within Fleet's product vision and roadmap
- Whether the feature seems like it can be designed, estimated, and developed in 6 weeks, given its individual complexity and when combined with other work already accepted

### Customer feature requests 
The product team's goal is to prioritize 16 customer feature requests at Feature Fest, then take them from settled to shipped. The customer success team is responsible for providing the Head of Product a live count during the Feature Fest meeting. Product Operations is responsible for monitoring this KPI and raising alarms throughout the design and engineering sprints. 
> Customer stories should be estimated at 1-3 points each to count as 1 request. If a feature request spans across multiple customers, it will be counted as the number of customers involved. 

### After the feature is accepted
After the 🎁🗣 Feature Fest meeting, Product Operations will clear the Feature Fest board as follows:
**Prioritized features:** Remove `feature fest` label, add `:product` label, and assign the group Product Manager. 
**Put to the side features:** Remove `feature fest` label and close the issue.

Group Product Managers will then develop user stories for the prioritized features. 

> The product team's commitment to the requester is that a prioritized feature will be delivered within 6 weeks or the requester will be notified within 1 business day of the decision to de-prioritize the feature. 

Potential reasons for why a feature may be de-prioritized include:
- The work was not designed in time. Since Fleet's engineering sprints are 3 weeks each, this means that a prioritized feature has 3 weeks to be designed, approved, and estimated in order to make it to the engineering sprint. At the prioritization meeting, the perceived design complexity of proposed features will inevitably be different from the actual complexity. 
  - This may be because other higher-priority design work took longer than expected or the work itself was more complex than expected
- The was designed but was not selected for the sprint. When a new sprint starts, it is populated with bugs, features, and technical tasks. Depending on the size and quantity of non-feature work, certain features may not be selected for the sprint.

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

Engineering teams coordinate on bug fixes with the product team during the joint sprint kick-off review. If one team is at capacity and a bug needs attention, another team can step in to assist by following these steps:

For MDM support on Endpoint ops bugs:
- Remove the `#g-endpoint-ops` label and add `#g-mdm` label.
- Add `~assisting g-endpoint-ops` to clarify the bug’s origin.

For Endpoint ops support on MDM bugs:
- Remove the `#g-mdm` label and add `#g-endpoint-ops` label.
- Add `~assisting g-mdm` to clarify the bug’s origin.

Fleet [always prioritizes bugs](https://fleetdm.com/handbook/product#prioritizing-improvements). 

#### Awaiting QA
Bugs will be verified as fixed by QA when they are placed in the "Awaiting QA" column of the relevant product group's sprint board. If the bug is verified as fixed, it is moved to the "Ready for release" column of the sprint board. Otherwise, the remaining issues are noted in a comment, and it is moved back to the "In progress" column of the sprint board.

## High priority user stories and bugs
All issues are treated as standard priority by default. Some issues are assigned a priority label to indicate urgency for the business.

1. Emergency: `P0`
- Examples: Customer outage, confirmed security vulnerability ([critical bug](https://fleetdm.com/handbook/company/product-groups#release-testing)), a new feature is needed to address an immediate business emergency.
- Response: Immediately stop other work to swarm the issue. Work 24/7 in shifts until resolved.
- Impact: Significant impact. May void current sprint.

2. Critical: `P1`
- Examples: A supported workflow is broken ([critical bug](https://fleetdm.com/handbook/company/product-groups#release-testing)), a potential security vulnerability, a new feature is required to address an immediate critical business need.
- Response: Issue brought to next standup for estimation and immediately brought into the sprint. Necessary team members are assigned as their top priority.
- Impact: High impact. Does not void sprint, but reduces overall velocity and requires deprioritizing other work.

3. Urgent: `P2`
- Examples: A supported workflow is not functioning as intended, a newly drafted feature has an associated urgent business need.
- Response: Issue is prioritized at the top of the next sprint. If opporunity cost of waiting for the next sprint is too high, it may be considered for current sprint.
- Impact: Low to medium impact. If prioritized into current sprint, may reduce overall velocity and require deprioritizing other work.

Add as much context as possible to the issue description and assign labels to help the team understand the problem and what is driving the urgency. All issues with a `P0`, `P1`, or `P2` label should be assigned to the [DRI for what goes in a release](https://fleetdm.com/handbook/company/communications#directly-responsible-individuals-dris). For immediate action, follow up on Slack or by phone.

Once the release DRI is aware of the issue, they will adjust the labels as needed and assign to the PM and EM of the appropriate product group. If they disagree with the priority label applied to the issue, they will contact the requestor to discuss further.

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
The on-call developer is encouraged to attend some of the customer success meetings during the week. Post a message to the #g-endpoint-ops Slack channel requesting invitations to upcoming meetings.

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

1. The new on-call developer should change the `@oncall` alias in Slack to point to them. In the
   search box, type "people" and select "People & user groups." Switch to the "User groups" tab.
   Click `@oncall`. In the right sidebar, click "Edit Members." Remove the former on-call, and add
   yourself.

2. Hand off newer conversations (Slack threads, issues, PRs, etc.). For more recent threads, the former on-call can unsubscribe from the thread, and the new on-call should subscribe. The former on-call should explicitly share each of
   these threads and the new on-call can select "Get notified about new replies" in the "..." menu.
   The former on-call can select "Turn off notifications for replies" in that same menu. It can be
   helpful for the former on-call to remain available for any conversations they were deeply involved
   in, so use your judgment on which threads to hand off. Anything not clearly handed off remains the responsibility of the former on-call developer.

In the Slack reminder thread, the on-call developer includes their retrospective. Please answer the following:

1. What were the most common support requests over the week? This can potentially give the new on-call an idea of which documentation to focus their efforts on.

2. Which documentation page did you focus on? What changes were necessary?

3. How did you spend the rest of your on-call week? This is a chance to demo or share what you learned.

## Wireframes 
- Showing these principles and ideas, to help remember the pros and cons and conceptualize the above visually.
- Figma: [⚗️ Fleet product project](https://www.figma.com/files/project/17318630/%E2%9A%97%EF%B8%8F-Fleet-product?fuid=1234929285759903870)

We have certain design conventions that we include in Fleet. We will document more of these over time.

**Figma component library**

Use the 🧩 ["Design System (current)"](https://www.figma.com/file/8oXlYXpgCV1Sn4ek7OworP/%F0%9F%A7%A9-Design-System-(current)?type=design&mode=design&t=BytcobQwypszkxf5-1) Figma library as a source of truth for components. Components in the product ([Storybook](https://fleetdm.com/storybook/)) should match the style of components defined in the Figma library. If the frontend component is inconsistent with one in the Figma library, treat that as a [bug](https://fleetdm.com/handbook/engineering#finding-bugs).

**Table empty states**

Use `---`, with color `$ui-fleet-black-50` as the default UI for empty columns.

**Form behavior**

Pressing the return or enter key with an open form will cause the form to be submitted.

**Internal links**

For text links that navigates the user to a different page within the Fleet UI, use the `$core-blue` color and `xs-bold` styling. You'll also want to make sure to use the underline style for when the user hovers over these links.

**External links**

For a link that navigates the user to an external site (e.g., fleetdm.com/docs), use the `$core-blue` color and `xs-bold` styling for the link's text. Also, place the link-out icon to the right of the link's text.

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

### Design consultation
Design consultations are scheduled as needed with the relevant participants, typically product designers and frontend engineers. It is an opportunity to collaborate and discuss design, implementation, and story requirements. The meeting is scheduled as needed by the product designer or frontend engineer when a user story is in the "Prioritized" column on the [drafting board](https://app.zenhub.com/workspaces/-drafting-ships-in-6-weeks-6192dd66ea2562000faea25c/board). 

**Participants:**
- Product Designer
- Software Engineers (UI/UX)

**Sample agenda**
- Review user story requirements
- Review wireframes
- Discuss design input 
- Discuss implementation details

### Design reviews
Design reviews are conducted daily between the [Head of Product Design](https://fleetdm.com/handbook/product-design#team) and contributors proposing changes to Fleet's interfaces, such as the graphical user interface (GUI) or REST API.  This fast cadence shortens the feedback loop, makes progress visible, and encourages early feedback.  This helps Fleet stay intentional about how the product is designed and minimize common issues like UI inconsistencies or accidental breaking changes to the API.

Product designers or other contributors come prepared to this meeting with their proposed changes in a GitHub issue.  Usually these are in the form of Figma wireframes, a pull request to the API docs showing changes, or a demo of a prototype.  The Head of Product Design and other participants review the changes quickly and give feedback, and then the contributor applies revisions and attends again the next day or as soon as possible for another go-round.  The Head of Product Design is responsible for looping in the right engineers, community members, and other subject-matter experts to iterate on and refine upcoming product changes in the best interest of the business.

Here are some tips for making this meeting effective:
- Bring 1 key engineer who has been helping out with the user story, when possible and helpful.
- Say the user story out loud to remind participants of what it is.
- At the beginning of describing your change, indicate whether you are 70% sure you are 100% done, or are looking for early feedback.
- Avoid explaining or showing multiple ways it could work.  Show the one way you think it should work and let your work speak for itself.
- For follow-ups, repeat the user story, but show only what has changed or been added since the last review.
- Read Fleet's [best practices for meetings](https://fleetdm.com/handbook/company/communications#meetings).

> To allow for asynchronous participation, instead of attending, contributors can alternatively choose to add an agenda item to the "Product design review" meeting with a GitHub link.  Then, the Head of Product Design will review during the meeting and provide feedback.  Every "Product design review" is recorded and automatically transcribed to a Google Doc so that it is searchable by every Fleet team member.

### Weekly bug review
QA has weekly check-in with product to go over the inbox items. QA is responsible for proposing “not a bug”, closing due to lack of response (with a nice message), or raising other relevant questions. All requires product agreement

QA may also propose that a reported bug is not actually a bug. A bug is defined as “behavior that is not according to spec or implied by spec.” If agreed that it is not a bug, then it's assigned to the relevant product manager to determine its priority.

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

3. **Timeboxes**: Tasks that are specified to complete within a pre-defined amount of time are marked with the `timebox` label. Timeboxes are research or investigation tasks necessary to move a prioritized user story forward, sometimes called "spikes" in scrum methodology. We use the term "timebox" because it better communicates its purpose. Timeboxes are always assigned to exactly one user story.

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
Please see [handbook/company/product-groups#endpoint-ops-group](https://fleetdm.com/handbook/company/product-groups#endpoint-ops-group)

##### Air guitar
Please see [handbook/company/initiate-an-air-guitar-session](https://fleetdm.com/handbook/company/product-groups#initiate-an-air-guitar-session)


<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="🛩️ Product groups">
