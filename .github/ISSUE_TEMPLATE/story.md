---
name: 🎟  Story
about: Specify an iterative change to the Fleet product.  (e.g. "As a user, I want to sign in with SSO.")
title: ''
labels: 'story,:product'
assignees: ''

---

> **This issue's remaining effort can be completed in ≤1 sprint.  It will be valuable even if nothing else ships.**
> 
> It is [planned and ready](https://fleetdm.com/handbook/company/development-groups#making-changes) to implement.  It is on the proper kanban board.

## Goal

| User story  |
|:---------------------------------------------------------------------------|
| As a _________________________________________,
| I want to _________________________________________
| so that I can _________________________________________.

## Changes

This issue's estimation includes completing:
- [ ] UI changes: TODO <!-- Insert the link to the relevant Figma file describing all relevant changes. Remove this checkbox if there are no changes to the user interface. -->
- [ ] CLI usage changes: TODO <!-- Specify what changes to the CLI usage are required. Remove this checkbox if there are no changes to the CLI. -->
- [ ] REST API changes: TODO <!-- Specify what changes to the API are required.  Remove this checkbox if there are no changes necessary. -->
- [ ] Permissions changes: TODO <!-- Specify what changes to the permissions are required.  Remove this checkbox if there are no changes necessary. -->
- [ ] Database schema migrations: TODO <!-- Specify what changes to the database schema are required. (This willl be used to change migration scripts accordingly.) Remove this checkbox if there are no changes necessary. -->
- [ ] Outdated documentation changes: TODO <!-- Specify what changes to the documentation are required. Remove this checkbox if there are no changes necessary. -->
- [ ] Scope transparency changes? TODO <!-- Remove this checkbox if there are no changes necessary. -->
- [ ] Breaking changes requiring major version bump? TODO  <!-- Breaking changes to the CLI or REST API require a major version bump, which is rarely a good idea.  Remove this checkbox if there are no changes necessary. -->
- [ ] Changes to paid features or tiers? TODO  <!-- List changes to paid features or tiers required.  Implementation of paid features should live in the `ee/` directory.  Remove this checkbox if there are no changes necessary. -->
- [ ] QA complete?
- [ ] ... <!-- If there are any other notable requirements to draw extra attention to, add them as checkboxes here.  Otherwise, remove this checkbox. -->

> ℹ️  Please read this issue carefully and understand it.  Pay [special attention](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) to UI wireframes, especially "dev notes".


## Context
- Requestor(s): _________________________ <!-- Who are the non-customer requestor(s) for this story, if any? Put their github usernames here. They should be notified if the story gets de-prioritized. For customer requestors, use the `customer-xyz` label instead. -->
<!--
What else should contributors [keep in mind](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) when working on this change?  (Optional.)
1. 
2. 
-->


## Test plan

- [ ] Requires load testing TODO <!-- User story has performance implications that require load testing. Otherwise, remove this checkbox. -->
## Risk assessment

Risk level: Low / Medium / High TODO <!-- Choose one. -->

Risk description: TODO <!-- If risk level is medium or high, explain why. If low, remove. -->

#### Automated:

- Fleet: Covered / Will not cover / Scoped <!-- Choose one. -->
- QAWolf: Covered / Will not cover / Scoped <!-- Choose one. -->

## Manual testing steps
<!-- Add detailed manual testing steps for all affected user roles. -->

Admin: TODO
1. Step 1
2. Step 2
3. Step 3

Maintainer: TODO
1. Step 1
2. Step 2
3. Step 3

Observer: TODO
1. Step 1
2. Step 2
3. Step 3

## Testing notes
<!-- Any additional testing notes relevant to this story. -->

## Tools required
<!-- Any additional tools required (local TUF service, API testing, specific software or VM) -->

## Confirmation
<!-- The engineer responsible for implementing this user story completes the test plan before moving to the "Ready for QA" column. -->

1. [ ] Engineer (@____): Added comment to user story confirming succesful completion of testing plan.
2. [ ] QA (@____): Added comment to user story confirming succesful completion of testing plan.
