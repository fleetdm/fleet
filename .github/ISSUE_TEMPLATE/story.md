---
name: 🎟  Story
about: Specify an iterative change to the Fleet product.  (e.g. "As a user, I want to sign in with SSO.")
title: '🎟 _______________'
labels: 'story,:product'
assignees: ''

---

> A user story is always estimated to fit within 1 sprint, is QA'd, and drives independent business value.

## User story

As a _________, I want to ________________ so that I can ________________.

<!--
Things to consider:
- What screen are they looking at?  (`As an observer on the host details page…`)
- What do they want to do? (`As an observer on the host details page, I want to run a permitted query.`) 
- Don't get hung up on the "so that I can ________" clause.  It is helpful, but optional.
- Example: "As an admin I would like to be asked for confirmation before deleting a user so that I do not accidentally delete a user."
-->


### Definition of done

This user story is estimated to include the following changes:

- [ ] TODO
- [ ]  
- [ ]  
- [ ] **QA** Any special QA notes?

> Please read carefully and pay special attention to UI wireframes.

<!--
TODO: extrapolate into handbook and include a link
Designs have usually gone through multiple rounds of revisions, but they could easily still be overlooking complexities or edge cases!  When you think you've discovered a blocker, communicate.  Leave a comment [mentioning the appropriate PM](https://fleetdm.com/handbook/company/development-groups) or ask for feedback at your next standup.  Then update this user story's estimation, wireframes, and "definition of done" to reflect your updated understanding.
-->

<!--
Specifying work:
- **Design changes** Does this story include changes to the user interface, or to how the CLI is used?  If so, those designs [will need to reviewed and revised](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) prior to estimation and before code is written.

- **Database schema migrations** Does this story require changes to the database schema and need schema migrations?  If so, those migrations will need to be written as part of the changes, and additional quality assurance will be required.

- **Technical sub-tasks** The simplest way to manage work is to use a single user story issue, then pass it around between contributors/asignees as seldom as possible.  For some teams, for particular user stories on a case-by-case basis, it may be worthwhile to invest additional overhead in creating separate technical sub-task issues.  If such issues are created, then they can be included as links in the checklist above.

- **QA** Changes are tested by hand prior to submitting pull requests. In addition, quality assurance will do an additional QA check prior to considering this story "done".  Any special QA notes?


- **Out-of-date docs** How will [Fleet's documentation](https://fleetdm.com/docs) and [articles](https://fleetdm.com/articles) need to change to reflect the changes included in this user story?
Any pages we should be sure to review?  Any keywords we should be sure to [search for](https://github.com/fleetdm/fleet/search?q=path%3A%2Fdocs%2F+path%3A%2Farticles%2F+path%3A%2Fschema+sso&type=)?  List these and any other aspects/gotchas the product group should make sure are covered by the documentation.
  - **REST API** If the API is changing, then the [REST API docs](https://fleetdm.com/docs/using-fleet/rest-api) will need to be updated.
  - **Telemetry schema** If osquery-compatible tables are changing as part of this user story, then the [telemetry data model reference](https://fleetdm.com/tables) will need to be updated.
  - **Configuration changes** If this user story includes any changes to the way Fleet is configured, then the server configuration reference will need to be updated.


Rarer things we tend to forget about:


- **Breaking changes (semver)** Does this change introduce breaking changes changes to Fleet's REST API or CLI usage?  If so, then we need to either discuss a major version release with the CTO, or figure out a way to maintain backwards compatibility.

**Changes to paid features** Does this user story add or change any paid features? If so, describe the changes that should be made to the pricing page, and make sure that code for any non-free features lives in the `ee/` directory.
- **Measurement** User stories are small changes that are best served by being released as quickly as possible in order to get real world feedback, whether quantitative or qualitative.  The norm is NOT to prioritize additional analytics or measurement work.  Is it especially important for the change described by this user story to come with extra investment in measuring usage, adoption, and success?  If so, describe what measurements we need to implement, along with the current state of any existing, related measurements.
- **Scope transparency** Does this change the scope of access that Fleet has on end user devices?  If so, describe this user story so that it includes the edits necessary to the [transparency guide](https://fleetdm.com/transparency).

- **Follow-through** Is there anything in particular that we should inform others (people who aren't in this product group) about after this user story is released?  For example: communication to specific customers, tips on how best to highlight this in a release post, gotchas, etc.
-->

This user story is considered "done" when:
- [ ] QA'd
- [ ] **Reference documentation** If the API is changing, then the [REST API docs](https://fleetdm.com/docs/using-fleet/rest-api) will need to be updated.
- [ ] **Compatibility** Does this story require changes to the database schema and need schema migrations?  Does it introduce breaking changes or non-reversible changes to Fleet's REST API or CLI usage?
- [ ] Released
- [ ] Estimation
- [ ] TODO 



## How?


### Design

#### UI

TODO?
<!-- Insert the link to the relevant Figma file. Remove this section if there are no changes to the user interface. -->

#### CLI usage

TODO?
<!-- Specify what changes to the CLI usage are required. Remove this section if there are no changes to the CLI. -->


### Compatibility
#### REST API changes

TODO?
<!-- Specify what changes to the API are required.  Remove this section if there are no changes necessary. -->

#### Database schema migrations

TODO?
<!-- Specify what changes to the database schema are required. Remove this section if there are no changes necessary. -->



## Context

Anything else that contributors should keep in mind when working on this change?

- [Remember the user](https://fleetdm.com/handbook/company#empathy)
- [Iterate](https://fleetdm.com/handbook/company#results)

<!--  TODO: instead of these commented out goodies, pull into the handbook and leave behind a link:

This section is optional and can be included or deleted, as time allows.

As Fleet grows as an all-remote company with more asynchronous processes across timezones, we will rely on this section more and more.

Here are some examples of questions that might be helpful to answer:

- What else should a contributor keep in mind when working on this change?
- Why create this user story?  Why should Fleet work on it?
- Why now?  Why prioritize this user story today?
- What is the business case?  How does this contribute to reaching Fleet's strategic goals?
- What's the problem?
- What is the current situation? Why does the current situation hurt?
- Who are the affected users?
- What are they doing right now to resolve this issue? Why is this so bad?

These questions are helpful for the product team when considering what to prioritize.  (The act of writing the answers is a lot of the value!)  But these answers can also be helpful when users or contributors (including our future selves) have questions about how best to estimate, iterate, or refine.
-->

This user story is considered "ready for development" when:
- [x] Issue created
- [ ] [Product group](https://fleetdm.com/handbook/company/product-groups) label added (e.g. `#cx`, `#mdm`)
- [ ] [Designed](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach)
- [ ] [Designs reviewed](https://fleetdm.com/handbook/business-operations/ceo-handbook#calendar-audit)
- [ ] [Estimated](https://fleetdm.com/handbook/company/why-this-way#why-scrum)
- [ ] Scheduled for [development](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence)
