## Engineering rituals

This folder contains guides for recurring engineering rituals ("ceremonies") practiced at Fleet.  The purpose of these docs is to:

- Clarify purpose (why we do the ritual)
- List roles and responsibilities (who does the ritual)
- Provide agenda and clear expectations for participants (how we do the ritual)
- Document decision flows that assist in the process

Definitions of process and policy live in the Fleet handbook (https://fleetdm.com/handbook). These ritual docs are tactical and should not duplicate the handbook. When a definition changes in the handbook, update links or brief summaries here (do not fork the process).

### Adding or updating a ritual doc

1. Create a new markdown file named after the ritual (e.g. `estimation.md`, `retrospective.md`).
2. Start with an `## Overview` section (Purpose, Cadence, Participants, Primary artifacts).
3. Provide an Agenda section with ordered or bulleted steps and explicit timeboxes where useful.
4. Include decision aids (Mermaid diagrams, tables) only when they accelerate the ritual (keep them small and actionable).
5. Link to the relevant handbook sections for deeper context instead of duplicating content.
6. Keep the tone instructional and brief.
7. Add the new file to the list below.

### Current ritual docs

- [Daily Standup](./daily-standup.md) – Daily ritual for status updates, blockers, and incoming bug triage.
- [Sprint Kickoff](./sprint-kickoff.md) – Forecast a body of work that the team is confident can be completed during the sprint.
- [Weekly Estimation](./weekly-estimation.md) – Review user stories and bugs that have completed the drafting and specification process and add point estimates.
- [Sprint Demo](./sprint-demo.md) – Showcase features, improvements, and bug fixes to all teams and stakeholders.
- [Sprint Retrospective](./sprint-retrospective.md) – Reflect on the sprint and identify areas to imrpove for future sprints.

### Maintenance

Ritual owners (usually the EM or designated facilitator) should review their doc at least quarterly, or immediately after a retrospective surfaces a change.  Keep PRs small and link to the retrospective or decision that prompted the update.
