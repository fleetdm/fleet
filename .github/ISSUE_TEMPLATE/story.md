---
name: 🎟  Story
about: Specify an iterative change to the Fleet product.  (e.g. "As a user, I want to sign in with SSO.")
title: ''
labels: 'story,:product,#cx'
assignees: ''

---

> **This issue's remaining effort can be completed in ≤1 sprint.  It will be valuable even if nothing else ships.**
>
> It will be prioritized, [drafted](https://fleetdm.com/handbook/company/development-groups#drafting), estimated, and scheduled prior to starting implementation.

## Goal

| User story  |
|:---------------------------------------------------------------------------|
| As a _________________________________________,
| I want to _________________________________________
| so that I can _________________________________________.


## Bibliography
This user story is ready for implementation if the following are true:
- [x] Issue created
- [ ] [Product group](https://fleetdm.com/handbook/company/product-groups) label added (e.g. `#cx`, `#mdm`)
- [ ] Changes [specified](https://fleetdm.com/handbook/company/development-groups#drafting) and [designed](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach)
- [ ] [Designs revised and approved](https://fleetdm.com/handbook/business-operations/ceo-handbook#calendar-audit)
- [ ] [Estimated](https://fleetdm.com/handbook/company/why-this-way#why-scrum)
- [ ] [Scheduled](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence) for development

> ℹ️  Great user stories have [clearly-defined changes]().

## Changes

This user story's estimation [includes](https://fleetdm.com/handbook/company/development-groups#drafting) completing:
- [ ] UI changes: https://fleetdm.com/handbook/company/development-groups#defining-done <!-- Insert the link to the relevant Figma file describing all relevant changes. Remove this checkbox if there are no changes to the user interface. -->
- [ ] CLI usage changes: TODO <!-- Specify what changes to the CLI usage are required. Remove this checkbox if there are no changes to the CLI. -->
- [ ] ... <!-- Include any other notable requirements to draw extra attention to. -->
- [ ] REST API changes: TODO <!-- Specify what changes to the API are required.  Remove this checkbox if there are no changes necessary. -->
- [ ] Database schema migrations: TODO <!-- Specify what changes to the database schema are required. (This willl be used to change migration scripts accordingly.) Remove this checkbox if there are no changes necessary. -->
- [ ] Outdated documentation changes: TODO <!-- Specify what changes to the documentation are required. Remove this checkbox if there are no changes necessary. -->
- [ ] Scope transparency changes? TODO <!-- Remove this checkbox if there are no changes necessary. -->
- [ ] Breaking changes requiring major version bump? TODO  <!-- Breaking changes to the CLI or REST API require a major version bump, which is rarely a good idea.  Remove this checkbox if there are no changes necessary. -->
- [ ] Changes to paid features or tiers? TODO  <!-- List changes to paid features or tiers required.  Implementation of paid features should live in the `ee/` directory.  Remove this checkbox if there are no changes necessary. -->
- [ ] QA complete?

> ℹ️  Please read this issue carefully and pay special [attention to UI wireframes](https://fleetdm.com/handbook/company/why-this-way/development-groups#implementation).

<!--
## Context
What else should contributors keep in mind when working on this change?  (Optional.)
1. 
2. 
-->



<!--  TODO: instead of these commented-out goodies, pull into the handbook and leave behind a link :: -->

<!--

### Making changes

Fleet's product goal is to create experiences that users want.

To deliver on this mission, we need a clear, repeatable process for turning an idea into concrete changes to the product that work every time. We also need to allow [open source contributions](https://fleetdm.com/handbook/company#open-source) at any point in the process from the wider Fleet community - these won't necessarily follow this process.

#### Planned and unplanned changes
Most changes to Fleet are planned changes. They are [prioritized](https://fleetdm.com/handbook/product), defined, designed, revised, estimated, and scheduled into a release sprint _prior to starting implementation_.  The process of going from a prioritized goal to an estimated, scheduled, committed user story with a target release is called "drafting", or "the drafting phase".

Occasionally, changes are unplanned.  Like a patch for an unexpected bug, or a hotfix for a security issue.  Or if an open source contributor suggests an unplanned change in the form of a pull request.  These unplanned changes are sometimes OK to merge as-is.  But if they change the user interface, the CLI usage, or the REST API, then they need to go through drafting and reconsideration before merging.

> But wait, [isn't this "waterfall"?](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall)  Thankfully no.  And it isn't just a Fleet concept.  In fact, between 2015-2023, GitLab and The Sails Company independently developed and coevolved almost the exact same delivery processes from first principles.  (Albeit with slightly different names for the same things.  What we call "drafting" and "implementation" at Fleet, is called "the validation phase" and "the build phase" at GitLab.)

### Drafting
"Drafting" is the art of defining a change, designing and shepherding it through the drafting process until it is ready for implementation.

The goal of drafting is to deliver software that works every time with less total effort and investment, without making contribution any less fun.  By researching and iterating [prior to development](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach), we design better product features, crystallize fewer bad, preemptive naming decisions, and achieve better throughput: getting more done in less time. 

> Fleet's drafting process is focused first and foremost on product development, but it can be used for any kind of change that benefits from planning or a "dry run".  For example, imagine you work for a business who has decided to swap out one of your payroll or device management vendors.  You will probably need to plan and execute changes to a number of complicated onboarding/offboarding processes.

#### Drafting process

The DRI for defining and drafting issues for a product group is the product manager, with close involvement from the designer and engineering manager.  But drafting is a team effort, and all contributors participate.

A user story is considered ready for implementation once:
- [ ] User story [issue created](https://github.com/fleetdm/fleet/issues/new/choose)
- [ ] [Product group](https://fleetdm.com/handbook/company/product-groups) label added (e.g. `#cx`, `#mdm`)
- [ ] Changes [specified](https://fleetdm.com/handbook/company/development-groups#drafting) and [designed](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach)
- [ ] [Designs revised and approved](https://fleetdm.com/handbook/business-operations/ceo-handbook#calendar-audit)
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
- Is it valuable enough? Will this task drive business value when released, indepenent of other tasks?


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



### Implementation

#### Developing from wireframes
Please read carefully and [pay special attention](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) to UI wireframes.

Designs have usually gone through multiple rounds of revisions, but they could easily still be overlooking complexities or edge cases!  When you think you've discovered a blocker, communicate.  Leave a comment [mentioning the appropriate PM](https://fleetdm.com/handbook/company/development-groups) or ask for feedback at your next standup.  Then update this user story's estimation, wireframes, and "definition of done" to reflect your updated understanding. 


#### Technical sub-tasks
The simplest way to manage work is to use a single user story issue, then pass it around between contributors/asignees as seldom as possible.  For some teams, for particular user stories on a case-by-case basis, it may be worthwhile to invest additional overhead in creating separate **technical sub-task** issues.

A user story is estimated to fit within 1 sprint and drives business value when released, independent of other stories.  **Technical sub-tasks** are not.  If sub-task issues are created for a given user story, then they:
- are NOT estimated
- will NOT be looked at or QA'd by quality assurance
- will NOT, in isolation, necessarily deliver any direct, independent business value
- can be included as links in this user story's "definition of done" checklist
- are NOT the right place to post GitHub comments (instead, concentrate conversation in the top-level "user story" issue)


#### Development best practices
- Remember the user.  What would you do if you saw that error message? [🔴](https://fleetdm.com/handbook/company#empathy)
- Communicate any blockers ASAP in your group Slack channel or standup. [🟠](https://fleetdm.com/handbook/company#ownership)
- Think fast and iterate.  [🟢](https://fleetdm.com/handbook/company#results)
- If it probably works, assume it's still broken.  Assume it's your fault.  [🔵](https://fleetdm.com/handbook/company#objectivity)
- Speak up and have short toes.  Assume positive intent. [🟣](https://fleetdm.com/handbook/company#openness)

-->
