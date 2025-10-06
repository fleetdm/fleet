## Sprint Kickoff

Purpose: Forecast a body of work that the team is at least 70% confident can be completed during the sprint.

Cadence: 1 hour or more as needed.

Participants: Full product group.

Ritual DRI: EM or assigned team member.

### Agenda
1. Review the [pre-sprint prioritization sheet](https://docs.google.com/spreadsheets/d/1DlSiRv0HVT2ANuBb08knEg_GCCAMEzE1RlKevGCFA20/edit?usp=sharing) with the team and bring selected items into the sprint.
2. Filter the [Drafting board](https://github.com/orgs/fleetdm/projects/67) to bugs assigned to the product group, and fill remaining sprint capacity based on our bug prioritization criteria documented in the [pre-sprint prioritization sheet](https://docs.google.com/spreadsheets/d/1DlSiRv0HVT2ANuBb08knEg_GCCAMEzE1RlKevGCFA20/edit?usp=sharing).
3. For each issue, complete the sprint inclusion process below.

### Sprint inclusion 
- Read the issue title, and ask if there are any questions or concerns.
- After any discussions of the issue, the issue is updated to include any additional context. 
- Add the issue to the sprint project, add the `:release` label, copy over the estimate story points, remove the `:product` label, and remove from the drafting board
  - Note: you can use our [GitHub Management Tool](https://github.com/fleetdm/fleet/tree/main/tools/github-manage) to do this in bulk at the end with `gm estimated <product group>` -> select all relevant issues and run 'bulk sprint kickoff' workflow.
- [Search](https://github.com/fleetdm/fleet/issues) for any remaining TODO's with a query `is:open is:issue label:<team label eg: #g-mdm> todo in:body label::release ` and address them immediately.

### Notes
- See [sprint kickoff definition](https://fleetdm.com/handbook/company/product-groups#sprint-ceremonies) in the handbook.
- Sprint Kickoff is intended to [forecast the sprint](https://www.scrum.org/resources/commitment-vs-forecast) and surface any remaining questions or concerns. 
- The goal is to select a realistic body of work that the team is at least 70% confident they can deliver with available capacity.
