---
name: 🎟  Story
about: Specify an iterative change to the Fleet product.  (e.g. "As a user, I want to sign in with SSO.")
title: ''
labels: 'story,:product'
assignees: ''

---

> **A user story is estimated to fit within 1 sprint, is QA'd, and drives independent business value.**

## Goal

<!--
| User story |
|:------------------------------------|
| As a _____________________,                             |
| I want to ___________________                           |
| so that I can ___________________.                      |

-->


As a _________, I want to ________________ so that I can ________________.

> Read more about crafting [great user stories](https://fleetdm.com/handbook/company/development-groups#drafting) and [defining "done"](https://fleetdm.com/handbook/development-groups#defining-done).

<!--  TODO: instead of these commented out goodies throughout, pull into the handbook and leave behind a link -->

<!--
### Drafting

"Drafting" is the art of defining a change and preparing it for implementation.

In the context of a product group, the DRI for defining and drafting issues for a group is the product manager, with close involvement from the designer and engineering manager.  But keep in mind that any changes we make to Fleet are a team effort, and everyone in the product group is encouraged to contribute.

> Fleet's drafting process is focused first and foremost on cire product development, but drafting can be a useful methodology for any change that benefits from planning.

#### Getting ready for development
A user story is considered "ready for development" when:
- [x] Issue created
- [ ] [Product group](https://fleetdm.com/handbook/company/product-groups) label added (e.g. `#cx`, `#mdm`)
- [ ] [Designed](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach)
- [ ] [Designs reviewed](https://fleetdm.com/handbook/business-operations/ceo-handbook#calendar-audit)
- [ ] [Estimated](https://fleetdm.com/handbook/company/why-this-way#why-scrum)
- [ ] Scheduled for [development](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence)

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



#### Defining "done"
The "definition of done" in a user story is written by the product manager, but the designer, engineering manager, developers, and quality assurance lead are all invited to contribute, and the accuracy and "release-ability" of an estimated user story is a team effort for everyone in the product group.

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


## Defining "done"

This user story is estimated to include the following changes, and will be considered "done" when:

- [ ] UI changes: TODO <!-- Insert the link to the relevant Figma file describing all relevant changes. Remove this checkbox if there are no changes to the user interface. -->
- [ ] CLI usage changes: TODO <!-- Specify what changes to the CLI usage are required. Remove this checkbox if there are no changes to the CLI. -->
- [ ] REST API changes: TODO <!-- Specify what changes to the API are required.  Remove this checkbox if there are no changes necessary. -->
- [ ] Database schema migrations: TODO <!-- Specify what changes to the database schema are required. (This willl be used to change migration scripts accordingly.) Remove this checkbox if there are no changes necessary. -->
- [ ] Outdated documentation changes: TODO <!-- Specify what changes to the documentation are required. Remove this checkbox if there are no changes necessary. -->
- [ ] Transparency promise changes? TODO <!-- Remove this checkbox if there are no changes necessary. -->
- [ ] Breaking changes requiring major version bump? TODO  <!-- Breaking changes to the CLI or REST API require a major version bump, which is rarely a good idea.  Remove this checkbox if there are no changes necessary. -->
- [ ] Changes to paid features or tiers? TODO  <!-- List changes to paid features or tiers required.  Implementation of paid features should live in the `ee/` directory.  Remove this checkbox if there are no changes necessary. -->
- [ ] QA complete?

> Please read carefully, [implement thoughtfully](https://fleetdm.com/handbook/company/why-this-way/development-groups#implementation), and pay special attention to UI wireframes.

<!--
## Context
What else should contributors keep in mind when working on this change?  (Optional.)
1. 
2. 
-->
