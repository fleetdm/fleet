# Product groups

## Background
When creating software, handoffs between teams or contributors are one of the most common sources of miscommunication and waste.  Like [GitLab](https://docs.google.com/document/d/1RxqS2nR5K0vN6DbgaBw7SEgpPLi0Kr9jXNGzpORT-OY/edit#heading=h.7sfw1n9c1i2t), Fleet uses product groups to minimize handoffs and maximize iteration and efficiency in the way we build the product.

## What are product groups?
Fleet organizes product development efforts into separate, cross-functional product groups that include members from Design, Engineering, Quality, and Product.  These product groups are organized by business goal, and designed to operate in parallel.

Security, performance, stability, scalability, database migrations, release compatibility, usage documentation (such as REST API and configuration reference), contributor experience, and support escalation are the responsibility of every product group.

At Fleet, [anyone can contribute](https://fleetdm.com/handbook/company#openness), even across product groups.

> Ideas expressed in wireframes, like code contributions, [are welcome from everyone](https://chat.osquery.io/c/fleet), inside or outside the company.

## Current product groups

| Product group             | Goal _(value for customers and/or community)_                       |
|:--------------------------|:--------------------------------------------------------------------|
| [MDM](#mdm-group)                                       | Reach maturity in the "MDM" product category.
| [Customer experience (CX)](#customer-experience-group)  | Make customers happier and more successful.
| [Infrastructure](#infrastructure-group)                 | Provide and support reliable and secure infrastructure.
| [Website](#website-group)                               | Make the website wonderful.


### MDM group

The goal of the MDM group is to reach [product maturity](https://drive.google.com/file/d/11yQ_2WG7TbRErUpMBKWu_hQ5wRIZyQhr/view?usp=sharing) in the "MDM" product category, across macOS and Windows.

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Designer                          | Marko Lisica
| Engineering manager               | George Karr
| Quality assurance                 | Sabrina Coy
| Product manager                   | Noah Talerman
| Software engineers (developers)   | Gabe Hernandez, Roberto Dip, Sarah Gillespie, Marcos Oviedo

> The Slack channel, kanban release board, and label for this product group is `#g-mdm`.


### Customer experience group

The goal of the customer experience (CX) group is to make users and customers happier and more successful.  This includes simpler usage, more successful customer onboarding, features that drive more win-win meetings with contributors and Fleet's sales team (such as initiatives like out-of-the-box CIS compliance for customers), and "whole product solutions", including professional services, design partnerships, and training.

> _**Note:** If a user story involves only changes to fleetdm.com, without changing the core product, then that user story is prioritized, drafted, implemented, and shipped by the [website group](https://fleetdm.com/handbook/company/development-groups#website-group)._


| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Designer                          | Rachael Shaw
| Engineering manager               | Sharon Katz
| Quality assurance                 | Reed Haynes
| Product manager                   | Mo Zhu
| Software engineers (developers)   | Jacob Shandling, Juan Fernandez, Lucas Rodriguez, Rachel Perkins, Eric Shaw, Martin Angers

> The Slack channel, kanban release board, and label for this product group is `#g-cx`.

### Infrastructure group

The goal of the infrastructure group is to provide and support reliable and secure infrastructure for Fleet and Fleet's customers. This includes AWS provisioning, monitoring, and management, 24-hour on-call support, as well as initiatives to streamline customer deployments, enhance customer onboarding experiences, and develop infrastructure solutions that align with and support Fleet's overall business goals.

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Engineering manager               | Luke Heath                
| Product manager                   | Luke Heath               
| Infrastructure engineers          | Robert Fairburn, Zach Winnerman

> The Slack channel, kanban release board, and label for this product group is `#g-infra`.


### Website group

The goal of the website group is to make visitors on Fleet's website get what they want and what they need.  This includes making the website more navigable, more beautiful, simpler, and easier to understand.

> _**Note:** If a user story involves **both** changes to the core product **and** to fleetdm.com, then that user story is prioritized, drafted, implemented, and shipped by the [CX group](https://fleetdm.com/handbook/company/development-groups#customer-experience-group)._

| Responsibility                    | Human(s)                  |
|:----------------------------------|:--------------------------|
| Designer                          | Michael Thomas
| Engineering manager               | Mike McNeil
| Quality assurance                 | Michael Thomas
| Product manager                   | Michael Thomas
| Software engineers (developers)   | Eric Shaw

> The Slack channel, kanban release board, and label for this product group is `#g-website`.


## Making changes

Fleet's highest product ambition is to create experiences that users want.

To deliver on this mission, we need a clear, repeatable process for turning an idea into a set of cohesively-designed changes in the product. We also need to allow [open source contributions](https://fleetdm.com/handbook/company#open-source) at any point in the process from the wider Fleet community - these won't necessarily follow this process.

To make a change to Fleet:
- First, [get it prioritized](https://fleetdm.com/handbook/product).
- Then, it will be [drafted](https://fleetdm.com/handbook/company/development-groups#drafting) (planned).
- Next, it will be [implemented](https://fleetdm.com/handbook/company/development-groups#implementing) and [released](https://fleetdm.com/handbook/engineering#release-process).

### Planned and unplanned changes
Most changes to Fleet are planned changes. They are [prioritized](https://fleetdm.com/handbook/product), defined, designed, revised, estimated, and scheduled into a release sprint _prior to starting implementation_.  The process of going from a prioritized goal to an estimated, scheduled, committed user story with a target release is called "drafting", or "the drafting phase".

Occasionally, changes are unplanned.  Like a patch for an unexpected bug, or a hotfix for a security issue.  Or if an open source contributor suggests an unplanned change in the form of a pull request.  These unplanned changes are sometimes OK to merge as-is.  But if they change the user interface, the CLI usage, or the REST API, then they need to go through drafting and reconsideration before merging.

> But wait, [isn't this "waterfall"?](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall) Waterfall is something else.  Between 2015-2023, GitLab and The Sails Company independently developed and coevolved similar delivery processes.  (What we call "drafting" and "implementation" at Fleet, is called "the validation phase" and "the build phase" at GitLab.)

### Drafting
"Drafting" is the art of defining a change, designing and shepherding it through the drafting process until it is ready for implementation.

The goal of drafting is to deliver software that works every time with less total effort and investment, without making contribution any less fun.  By researching and iterating [prior to development](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach), we design better product features, crystallize fewer bad, preemptive naming decisions, and achieve better throughput: getting more done in less time. 

> Fleet's drafting process is focused first and foremost on product development, but it can be used for any kind of change that benefits from planning or a "dry run".  For example, imagine you work for a business who has decided to swap out one of your payroll or device management vendors.  You will probably need to plan and execute changes to a number of complicated onboarding/offboarding processes.

#### Drafting process

The DRI for defining and drafting issues for a product group is the product manager, with close involvement from the designer and engineering manager.  But drafting is a team effort, and all contributors participate.

A user story is considered ready for implementation once:
- [ ] User story [issue created](https://github.com/fleetdm/fleet/issues/new/choose)
- [ ] [Product group](https://fleetdm.com/handbook/company/product-groups) label added (e.g. `#g-cx`, `#g-mdm`)
- [ ] Changes [specified](https://fleetdm.com/handbook/company/development-groups#drafting) and [designed](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach)
- [ ] [Designs revised and approved](#design-reviews)
- [ ] [Estimated](https://fleetdm.com/handbook/company/why-this-way#why-scrum)
- [ ] [Scheduled](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence) for development

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
- **Design changes** Does this story include changes to the user interface, or to how the CLI is used?  If so, those designs [will need to reviewed and revised](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) prior to estimation and before code is written.
- **Database schema migrations** Does this story require changes to the database schema and need schema migrations?  If so, those migrations will need to be written as part of the changes, and additional quality assurance will be required.
- **Out-of-date docs** How should [Fleet's documentation](https://fleetdm.com/docs) and [articles](https://fleetdm.com/articles) be updated to reflect the changes included in this user story?
  - **REST API** If the Fleet API is changing, then the [REST API docs](https://fleetdm.com/docs/using-fleet/rest-api) will need to be updated.
  - **Configuration changes** If this user story includes any changes to the way Fleet is configured, then the server configuration reference will need to be updated.
  - **Telemetry schema** If osquery-compatible tables are changing as part of this user story, then the [telemetry data model reference](https://fleetdm.com/tables) will need to be updated.
  - **Other content** What keywords should we [search for](https://github.com/fleetdm/fleet/search?q=path%3A%2Fdocs%2F+path%3A%2Farticles%2F+path%3A%2Fschema+sso&type=) to locate doc pages and articles that need updates?  List these and any other aspects/gotchas the product group should make sure are covered by the documentation.
**Changes to paid features or tiers** Does this user story add or change any paid features, or modify features' tiers? If so, describe the changes that should be made to the [pricing page](https://fleetdm.com/pricing), and make sure that code for any non-free features lives in the `ee/` directory.
- **Semantic versioning** Does this change introduce breaking changes to Fleet's REST API or CLI usage?  If so, then we need to either figure out a crafty way to maintain backwards compatibility, or discuss a major version release with the CTO (`#help-engineering` and mention `@zwass`).
- **Scope transparency** Does this change the scope of access that Fleet has on end user devices?  If so, describe this user story so that it includes the edits necessary to the [transparency guide](https://fleetdm.com/transparency).
- **Measurement?** User stories are small changes that are best served by being released as quickly as possible in order to get real world feedback, whether quantitative or qualitative.  The norm is NOT to prioritize additional analytics or measurement work.  Is it especially important for the change described by this user story to come with extra investment in measuring usage, adoption, and success?  If so, describe what measurements we need to implement, along with the current state of any existing, related measurements.
- **QA** Changes are tested by hand prior to submitting pull requests. In addition, quality assurance will do an extra QA check prior to considering this story "done".  Any special QA notes?
- **Follow-through** Is there anything in particular that we should inform others (people who aren't in this product group) about after this user story is released?  For example: communication to specific customers, tips on how best to highlight this in a release post, gotchas, etc.


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


#### Design reviews

Design reviews are [conducted daily by the CEO](https://fleetdm.com/handbook/company/ceo#calendar-audit).

The product designer prepares proposed changes in the form of wireframes for this meeting, and presents them quickly.  Here are some tips for making this meeting effective:
- Bring 1 key engineer who has been helping out with the user story, when possible and helpful.
- Say the user story out loud to remind participants of what it is.
- Avoid explaining or showing multiple ways it could work.  Show the one way you think it should work and let your work speak for itself.
- For follow-ups, repeat the user story, but show only what has changed or been added since the last review.
- Zoom in.


### Implementing

#### Developing from wireframes
Please read carefully and [pay special attention](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) to UI wireframes.

Designs have usually gone through multiple rounds of revisions, but they could easily still be overlooking complexities or edge cases!  When you think you've discovered a blocker, communicate.  Leave a comment [mentioning the appropriate PM](https://fleetdm.com/handbook/company/development-groups) or ask for feedback at your next standup.  Then update this user story's estimation, wireframes, and "definition of done" to reflect your updated understanding. 


#### Sub-tasks
The simplest way to manage work is to use a single user story issue, then pass it around between contributors/asignees as seldom as possible.  But on a case-by-case basis, for particular user stories and teams, it can sometimes be worthwhile to invest additional overhead in creating separate **unestimated sub-task** issues ("sub-tasks").

A user story is estimated to fit within 1 sprint and drives business value when released, independent of other stories.  Sub-tasks are not.

Sub-tasks:
- are NOT estimated
- can be created by anyone
- add extra management overhead and should be used sparingly
- do NOT have nested sub-tasks
- will NOT necessarily, in isolation, deliver any business value
- are always attached to exactly ONE top-level "user story" (which does drive business value)
- are included as links in the parent user story's "definition of done" checklist
- are NOT the best place to post GitHub comments (instead, concentrate conversation in the top-level "user story" issue)
- will NOT be looked at or QA'd by quality assurance

#### API changes

> DRI: Rachael Shaw

To maintain consistency, ensure perspective, and provide a single pair of eyes in the design of Fleet's REST API and API documentation, there is a single Directly Responsible Individual (DRI). The API design DRI will review and approve any alterations at the pull request stage, instead of making it a prerequisite during drafting of the story. You may tag the DRI in a GitHub issue with draft API specs in place to receive a review and feedback prior to implementation. Receiving a pre-review from the DRI is encouraged if the API changes introduce new endpoints, or substantially change existing endpoints. 

No API changes are merged without accompanying API documentation and approval from the DRI. The DRI is responsible for ensuring that the API design remains consistent and adequately addresses both standard and edge-case scenarios. The DRI is also the code owner of the API documentation Markdown file. The DRI is committed to reviewing PRs within one business day. In instances where the DRI is unavailable, the Head of Product will act as the substitute code owner and reviewer.

#### Development best practices
- Remember the user.  What would you do if you saw that error message? [🔴](https://fleetdm.com/handbook/company#empathy)
- Communicate any blockers ASAP in your group Slack channel or standup. [🟠](https://fleetdm.com/handbook/company#ownership)
- Think fast and iterate.  [🟢](https://fleetdm.com/handbook/company#results)
- If it probably works, assume it's still broken.  Assume it's your fault.  [🔵](https://fleetdm.com/handbook/company#objectivity)
- Speak up and have short toes.  Write things down to make them complete. [🟣](https://fleetdm.com/handbook/company#openness)



<meta name="maintainedBy" value="mikermcneil">
<meta name="title" value="Product groups">
