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

## Changes

### Product
- [ ] UI changes: TODO <!-- Insert the link to the relevant Figma cover page. If there are substantial UI changes at one of Fleet's breakpoints (480, 768, 1024, 1280, and 1440px), make sure wireframes show the UI at the relevant breakpoint(s). Put "No changes" if there are no changes to the user interface. -->
- [ ] CLI (fleetctl) usage changes: TODO <!-- Insert the link to the relevant Figma cover page. Put "No changes" if there are no changes to the CLI. -->
- [ ] YAML changes: TODO <!-- Specify changes in the YAML files doc page as a PR to the reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. -->
- [ ] REST API changes: TODO <!-- Specify changes in the REST API doc page as a PR to reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. Move this item to the engineering list below if engineering will design the API changes. -->
- [ ] Fleet's agent (fleetd) changes: TODO <!-- Specify changes to fleetd. If the change requires a new Fleet (server) version, consider specifying to only enable this change in new Fleet versions. If there are new tables, specify changes in the schema/ folder as a PR to the reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. -->
- [ ] Fleet server configuration changes: TODO <!-- Specify changes in the Fleet server configuration doc page as a PR to reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting File a :help-customers request and assign the SVP of Customer Success. Up to Customer Success to device if any changes to cloud environments is needed. Put "No changes" if there are no changes necessary. -->
- [ ] Exposed, public API endpoint changes: TODO <!-- Specify changes in the "Which API endpoints to expose to the public internet?" guide as a PR to reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting File a :help-customers request and assign the SVP of Customer Success. Up to Customer Success to device if any changes to cloud environments is needed.  Put "No changes" if there are no changes necessary. -->
- [ ] fleetdm.com changes: TODO <!-- Does this story include changes to fleetdm.com? (e.g. new API endpoints) If yes, create a blank subtask with the #g-website label, assign @eashaw, and add @eashaw and @lukeheath to the next design review meeting. fleetdm.com changes are up to @eashaw -->
- [ ] GitOps mode UI changes: TODO <!-- Specify UI changes for read-only GitOps mode. Put "No changes" if there are no changes necessary. -->
- [ ] GitOps generation changes: TODO <!-- Specify changes to results from the fleetctl generate-gitops command. Put "No changes" if there are no changes necessary. -->
- [ ] Activity changes: TODO <!-- Specify the display name that will appear on the dashboard ("type" filter) and changes to the Audit log page in the contributor docs as a PR to reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. -->
- [ ] Permissions changes: TODO <!-- Specify changes in the permissions doc page here: https://fleetdm.com/docs/using-fleet/manage-access as a PR to the reference docs release branch. If doc changes aren't necessary, explicitly mention no changes to the doc page. Put "No changes" if there are no permissions changes. -->
- [ ] Changes to paid features or tiers: TODO  <!-- Specify changes in pricing-features-table.yml as a PR to reference docs release branch. Specify "Fleet Free" and/or "Fleet Premium" if there are no changes to the pricing page necessary. -->
- [ ] My device and fleetdm.com/better changes: TODO <!-- If there are changes to the personal information Fleet can see on end user workstations, make sure wireframes include changes to the My device page. Also, specify changes as a PR to the fleetdm.com/better (aka Transparency page). Put "No changes" if there are no changes necessary. -->
- [ ] Usage statistics: TODO <!-- Specify changes in the Fleet usage statistics guide as a PR to reference docs release branch. Put "No changes" if there are no changes necessary. -->
- [ ] Other reference documentation changes: TODO <!-- Any other reference doc changes? Specify changes as a PR to reference docs release branch. Put "No changes" if there are no changes necessary. -->
- [ ] First draft of test plan added
- [ ] Once shipped, requester has been notified
- [ ] Once shipped, dogfooding issue has been filed

### Engineering
- [ ] Test plan is finalized
- [ ] Contributor API changes: TODO <!-- Specify changes in the the Contributor API doc page as a PR to reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. -->
- [ ] Feature guide changes: TODO <!-- Specify if a new feature guide is required at fleetdm.com/guides, or if a previous guide should be updated to reflect feature changes. -->
- [ ] Database schema migrations: TODO <!-- Specify what changes to the database schema are required. (This will be used to change migration scripts accordingly.) Remove this checkbox if there are no changes necessary. -->
- [ ] Load testing: TODO  <!-- List any required scalability testing to be conducted.  Remove this checkbox if there is no scalability testing required. -->
- [ ] Pre-QA load test: TODO <!-- If this story has high risk of changing load profile, engineers must load-test prior to QA, with a subtask dedicated to that effort. Remove this checkbox if the change won't measurably modify Fleet's load profile, such that either load testing isn't needed at all or load testing is expected to be only performed during QA. -->
- [ ] Load testing/osquery-perf improvements: TODO <!-- List, or link a subtask for, any osquery-perf or load test environment changes required to comprehensively load test this story if load testing is needed. -->
- [ ] This is a premium only feature: Yes / No  <!-- If yes, make sure the test plan includes confirmation that both the frontend and backend are protected. -->

> ℹ️  Please read this issue carefully and understand it.  Pay [special attention](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) to UI wireframes, especially "dev notes".

### Risk assessment

- Requires testing in a hosted environment: TODO <!-- User story has features that require testing in a hosted environment. Otherwise, remove this item. -->
- Requires load testing: TODO <!-- User story has performance implications that require load testing. Otherwise, remove this item. -->
- Risk level: Low / High TODO <!-- Choose one. Consider: Does this change come with performance risks?  Any risk of accidental log spew? Any particular regressions to watch out for?  Any potential compatibility issues, even if it's not technically a breaking change? -->
- Risk description: TODO <!-- If the risk level is high, explain why. If low, remove. -->

### Test plan
<!-- Add detailed manual testing steps for all affected user roles. -->
> Make sure to go through [the list](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/ui/design-qa-considerations.md) and consider all events that might be related to this story, so we catch edge cases earlier.
> 
<!-- The following sections can be removed if they are inapplicable for this User Story -->

#### Core flow
<!-- Product TODO -->
- TODO
- TODO
- TODO

#### UI
- [ ] Verify that all UI changes specified in the Figma wireframes are correctly implemented
- [ ] Verify expected UI states (loading, empty, error states if applicable)

#### API
- [ ] Test all API endpoints added or modified in the **API changes** section of this issue
- [ ] Verify any new API endpoints appear in the list when adding an API-only user. The API endpoints display name, method, and path is the same as listed in the API reference docs
- [ ] Verify error handling for invalid inputs where applicable

#### GitOps (generate + run)
- [ ] Configure the feature through the UI and run `fleetctl generate-gitops`
- [ ] Confirm the generated `.yml` includes the expected fields (compare with YAML changes in the Product section)
- [ ] Modify the generated `.yml` and run `fleetctl gitops`
- [ ] Confirm the configuration updates correctly in Fleet
- [ ] Enable GitOps mode and verify the feature behaves correctly

#### Permissions
<!-- Consider: Do the steps above apply to all global access roles, including admin, maintainer, observer, observer+, and GitOps?  Do the steps above apply to all fleet-level access roles?  If not, write the steps used to test each variation.
-->
- [ ] Verify role restrictions are applied correctly for **global roles**
- [ ] Verify role restrictions are applied correctly for **fleet-level roles**

#### Edge cases

<!-- QA TODO: Replace the TODO below with relevant edge cases or remove this section if not applicable -->

<!-- Edge case examples:
1. Invalid or unexpected input values
2. Boundary conditions
3. Behavior when required configuration is missing
4. Behavior when related objects are deleted or modified
-->

- TODO
- TODO
- TODO

#### Supplemental testing

<!-- Mid-cycle testing checks. Added by QA after the issue was moved to Awaiting QA -->

### Testing notes
<!-- Any additional testing notes relevant to this story or tools required for testing. -->

### Confirmation
<!-- The engineer responsible for implementing this user story completes the test plan before moving to the "Awaiting QA" column. -->

1. [ ] Engineer: Added comment to user story confirming successful completion of test plan (include any special setup, test data, or configuration used during development/testing if applicable).
2. [ ] QA: Added comment to user story confirming successful completion of test plan.
