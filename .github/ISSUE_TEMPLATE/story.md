---
name: 🎟  Story
about: Specify an iterative change to the Fleet product.  (e.g. "As a user, I want to sign in with SSO.")
title: ''
labels: 'story'
assignees: ''

---

<!-- **This issue's remaining effort can be completed in ≤1 sprint.  It will be valuable even if nothing else ships.**
It is [planned and ready](https://fleetdm.com/handbook/company/development-groups#making-changes) to implement.  It is on the proper kanban board. -->


## Goal

| User story  |
|:---------------------------------------------------------------------------|
| As a _________________________________________,
| I want to _________________________________________
| so that I can _________________________________________.

## Context
- Requestor(s): _________________________ <!-- Who are the non-customer requestor(s) for this story, if any? Put their GitHub usernames here. They should be notified if the story gets de-prioritized. For customer requestors, use the `customer-xyz` label instead. -->
- Product designer: _________________________ <!-- Who is the product designer to contact if folks have questions about the UI, CLI, or API changes? -->
  
<!--
What else should contributors [keep in mind](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) when working on this change?  (Optional.)
1. 
2. 
-->

## Changes

### Product
- [ ] UI changes: TODO <!-- Insert the link to the relevant Figma cover page. Remove this checkbox if there are no changes to the user interface. -->
- [ ] CLI usage changes: TODO <!-- Insert the link to the relevant Figma cover page. Remove this checkbox if there are no changes to the CLI. -->
- [ ] REST API changes: TODO <!-- Specify what changes to the API are required.  Remove this checkbox if there are no changes necessary. The product manager may move this item to the engineering list below if they decide that engineering will design the API changes. -->
- [ ] Permissions changes: TODO <!-- Specify what changes to the permissions are required.  Remove this checkbox if there are no changes necessary. -->
- [ ] Outdated documentation changes: TODO <!-- Specify required documentation changes (public-facing fleetdm.com/docs or contributors) & redirects to add to /website/config/routes.js. -->
- [ ] Changes to paid features or tiers: TODO  <!-- Specify "Fleet Free" or "Fleet Premium".  If only certain parts of the user story involve paid features, specify which parts.  Implementation of paid features should live in the `ee/` directory. -->

### Engineering
- [ ] Database schema migrations: TODO <!-- Specify what changes to the database schema are required. (This will be used to change migration scripts accordingly.) Remove this checkbox if there are no changes necessary. -->
- [ ] Load testing: TODO  <!-- List any required scalability testing to be conducted.  Remove this checkbox if there is no scalability testing required. -->

> ℹ️  Please read this issue carefully and understand it.  Pay [special attention](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) to UI wireframes, especially "dev notes".

## QA

### Risk assessment

- Requires load testing: TODO <!-- User story has performance implications that require load testing. Otherwise, remove this item. -->
- Risk level: Low / High TODO <!-- Choose one. Consider: Does this change come with performance risks?  Any risk of accidental log spew? Any particular regressions to watch out for?  Any potential compatibility issues, even if it's not technically a breaking change? -->
- Risk description: TODO <!-- If the risk level is high, explain why. If low, remove. -->

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

1. [ ] Engineer (@____): Added comment to user story confirming successful completion of QA.
2. [ ] QA (@____): Added comment to user story confirming successful completion of QA.
