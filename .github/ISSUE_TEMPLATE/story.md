---
name: 🎟  Story
about: Specify an iterative change to the Fleet product.  (e.g. "As a user, I want to sign in with SSO.")
title: ''
labels: 'story'
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

### Product
- [ ] UI changes: TODO <!-- Insert the link to the relevant Figma file describing all relevant changes. Remove this checkbox if there are no changes to the user interface. -->
- [ ] CLI usage changes: TODO <!-- Specify what changes to the CLI usage are required. Remove this checkbox if there are no changes to the CLI. -->
- [ ] Permissions changes: TODO <!-- Specify what changes to the permissions are required.  Remove this checkbox if there are no changes necessary. -->
- [ ] Outdated documentation changes: TODO <!-- Specify what changes to the documentation are required. Remove this checkbox if there are no changes necessary. -->
- [ ] Changes to paid features or tiers TODO  <!-- List changes to paid features or tiers required.  Implementation of paid features should live in the `ee/` directory.  Remove this checkbox if there are no changes necessary. -->

### Engineering
- [ ] REST API changes: TODO <!-- Specify what changes to the API are required.  Remove this checkbox if there are no changes necessary. Product may move this item to the list above if they choose to design the API changes. -->
- [ ] Database schema migrations: TODO <!-- Specify what changes to the database schema are required. (This willl be used to change migration scripts accordingly.) Remove this checkbox if there are no changes necessary. -->
- [ ] Documentation changes complete

> ℹ️  Please read this issue carefully and understand it.  Pay [special attention](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) to UI wireframes, especially "dev notes".

## Context
- Requestor(s): _________________________ <!-- Who are the non-customer requestor(s) for this story, if any? Put their github usernames here. They should be notified if the story gets de-prioritized. For customer requestors, use the `customer-xyz` label instead. -->
<!--
What else should contributors [keep in mind](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) when working on this change?  (Optional.)
1. 
2. 
-->

## QA

### Risk assessment

- Requires load testing: TODO <!-- User story has performance implications that require load testing. Otherwise, remove this item. -->
- Risk level: Low / High TODO <!-- Choose one. -->
- Risk description: TODO <!-- If risk level is high, explain why. If low, remove. -->

### Manual testing steps
<!-- 
Add detailed manual testing steps for all affected user roles. 
-->

1. Step 1
2. Step 2
3. Step 3

<!-- Consider: Do the steps above apply to all global access roles, including admin, maintainer, observer, observer+, and GitOps?  Do the steps above apply to all team-level access roles?  If not, write the steps used to test each variation.
-->

### Testing notes
<!-- Any additional testing notes relevant to this story or tools required for testing. -->

### Confirmation
<!-- The engineer responsible for implementing this user story completes the test plan before moving to the "Ready for QA" column. -->

1. [ ] Engineer (@____): Added comment to user story confirming succesful completion of QA.
2. [ ] QA (@____): Added comment to user story confirming succesful completion of QA.
