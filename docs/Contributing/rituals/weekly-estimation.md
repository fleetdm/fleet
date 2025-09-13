## [Weekly Estimation](https://fleetdm.com/handbook/company/product-groups#sprint-ceremonies) 

Purpose: Review user stories and bugs that have completed the drafting and specification process and add point estimates.

Cadence: Weekly, 1 hour or more as needed.

Participants: Full product group.

Facilitator: EM or assigned team member.

Requirements: All user stories and bugs must meet the minimum specification threshold. It is the product group's responsibility to enforce this requirement before estimation.

> All participants are expected to review every user story for their product group in the "Ready to estimate" column before Weekly Estimation. It is the facilitator's responsibility to make sure the team is notified and has adequate time to review each issue before the ritual.

### Format
1. Share screen and navigate to the [Drafting board](https://github.com/orgs/fleetdm/projects/67) filtered by product group label.
2. Filter by the `story` label. 
3. For each user story, complete the user story estimation process below. 
4. Filter by the `bug` label. 
5. For each bug, the facilitator completes the bug estimation process below.

### User story estimation
- Read the user story title and description aloud. 
- Open any Figma designs and review together. 
- Read all sub-task titles aloud. 
- Confirm that the user story meets the minimum specification threshold.
- Ask the team if there are any questions or concerns. 
- If no, go through each sub-task and complete estimation (sync or async).
- If yes, discuss questions or concerns and atempt to resolve on the call to complete estimation. If more time is needed, the user story is pushed to the next Weekly Estimation, or an ad-hoc estimation session if needed.

### Bug estimation
- Read the user story title and description aloud.
- Read the reproduction steps aloud. 
- Confirm that the bug meets the minimum specification threshold.
- Ask the team if there are any questions or concerns.
- If no, estimate the bug (sync or async). 
- If yes, discuss questions or concerns and atempt to resolve on the call to complete estimation. If more time is needed, the bug is pushed to the next Weekly Estimation, or an ad-hoc estimation session if needed.

### Minimum specification threshold

***User Story**
- Title
- Goal (user story format)
- Sub-issues (if required) for all components with applicable labels added (`~frontend`, `~backend`, etc.) 
- Context
- Changes (complete, no TODOs)
- QA: Risk assessment
- QA: Test plan

> If the user story requires sub-issues, all components of the user story must be separated into clear and defined sub-issues. Do not create placeholder sub-issues that will get filled in later. They must each contain a summary of the changes required. Full implementation details are not required.

Examples of well specified user stories ready for estimation: [#25549](https://github.com/fleetdm/fleet/issues/25549), [#28631](https://github.com/fleetdm/fleet/issues/28631).

**Bug**
- Title
- Fleet version
- Actual behavior 
- Steps to reproduce
- Applicable labels added (`~released bug`, `~frontend`, etc.)

Examples of well specified bugs ready for estimation: 

### Notes
The EM is responsible for final point values and ensuring estimates are realistic. These sessions focus on understanding scope, effort, and complexity. Estimation sessions help align the roadmap with business needs by providing realistic timelines for work completion.

Handbook reference (process and definitions): https://fleetdm.com/handbook/company/product-groups#sprint-ceremonies