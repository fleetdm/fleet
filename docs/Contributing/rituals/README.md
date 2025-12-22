## Engineering rituals
> <em>Ritual</em>: the prescribed order of performing a ceremony.

This folder contains guides for recurring engineering rituals ("ceremonies") practiced at Fleet.  The purpose of these docs is to:

- Clarify purpose (why we do the ritual).
- List roles and responsibilities (who does the ritual).
- Provide agenda and clear expectations for participants (how we do the ritual).
- Document decision flows that assist in the process.

Definitions of process and policy live in the Fleet handbook (https://fleetdm.com/handbook). These ritual docs are tactical and should not duplicate the handbook. When a definition changes in the handbook, update links or brief summaries here; don't fork the process.

### Adding or updating a ritual doc
1. Copy the [daily standup](./daily-standup.md) doc and rename it to the name of the ritual.
2. Keep the overview, format, and agenda sections. Extend with new sections as necessary.
3. Include decision aids (Mermaid diagrams, tables) only when they accelerate the ritual (keep them small and actionable).
4. Link to the relevant handbook sections for deeper context instead of duplicating content.
5. Keep the tone instructional and brief.
6. Add the new file to the list below.

### Current ritual docs
- [Daily standup](./daily-standup.md) – Daily ritual for status updates, blockers, and incoming bug triage.
- [Sprint kickoff](./sprint-kickoff.md) – Forecast a body of work for the next sprint.
- [Weekly estimation](./weekly-estimation.md) – Review user stories and bugs that have completed spec process.
- [Sprint demo](./sprint-demo.md) – Showcase features, improvements, and bug fixes to all teams.
- [Sprint retrospective](./sprint-retrospective.md) – Reflect on the sprint and identify areas to improve.

### Maintenance
Ritual owners (usually the EM or designated facilitator) should review rituals at least quarterly, or immediately after a retrospective surfaces a change.  Keep PRs small and link to the retrospective or decision that prompted the update.
