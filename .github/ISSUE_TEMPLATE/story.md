---
name: 🎟  Story
about: Specify a Scrum user story.  (e.g. "As a user, I want to sign in with SSO.")
title: ''
labels: 'story'
assignees: ''

---

## User story 

<!-- Always has an estimation. Always drives business value.  Always gets QA'd.  Always fits within 1 sprint. -->

TODO

<!-- 
Describe in a way of a user story what needs to be done, who wants it and for what purpose.
Use this format:
"As a _________, I want ________________."
e.g. "As an admin I would like to be asked for confirmation before deleting a user so that I do not accidentally delete a user."

Things to consider:
- Who is the human? (`As an observer…`)
- What screen are they looking at?  (`As an observer on the host details page…`)
- What do they want to do? (`As an observer on the host details page, I want to run a permitted query.`) 
- What is the current situation? Why does the current situation hurt? 
-->

## Requirements

<!-- Things we tend to forget about -->
- **Documentation** If the API is changing, then the [REST API docs](https://fleetdm.com/docs/using-fleet/rest-api) will need to be updated.
- **Design changes** Does this story include changes to the user interface, or to how the CLI is used?
- **Compatibility** Does this story require changes to the database schema and need schema migrations?  Does it introduce breaking changes or non-reversible changes to Fleet's REST API or CLI usage?
- **Premium feature** Should this be a premium-only feature? If so, make sure to update the pricing page, and that relevant code lives in the `ee/` directory.
- **Transparency** Do we need to update the [transparency guide](https://fleetdm.com/transparency) to reflect new functionality for end users?
- **QA** Any special QA notes?


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
<!-- Specify what changes to the API are required.Remove this section if there are no changes necessary. -->

#### Database schema migrations

TODO?
<!-- Specify what changes to the database schema are required. Remove this section if there are no changes necessary. -->

## Technical sub-tasks (if any)
N/A
<!--
It is simplest to use only a single user story issue.  If additional issues for technical sub-tasks are necessary, they're listed here: 
- TODO
- TODO
-->
