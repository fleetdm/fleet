## Engineering rituals

This folder contains concise, tactical guides for the recurring engineering rituals ("ceremonies") practiced at Fleet.  These docs exist to make the rituals easy to run and improve by:

* Clarifying purpose (why we do the ritual)
* Listing roles, inputs, and expected outputs
* Supplying lightweight agendas / checklists and any timeboxes
* Capturing tips, gotchas, and decision charts that help them run smoothly

Authoritative definitions of process and policy live in the public Fleet handbook (https://fleetdm.com/handbook).  Ritual docs here are intentionally practical and should avoid duplicating full policy text.  When a definition changes in the handbook, update links or brief summaries here (do not fork the process).

### Adding or updating a ritual doc

1. Create a new markdown file named after the ritual (e.g. `sprint-demo.md`, `oncall-handoff.md`).
2. Start with an `## Overview` section (Purpose, Cadence, Participants, Primary artifacts).
3. Provide an Agenda / Flow section with ordered or bulleted steps and explicit timeboxes where useful.
4. Include decision aids (Mermaid diagrams, tables) only when they accelerate the ritual (keep them small and actionable).
5. Link to the relevant handbook sections for deeper context instead of duplicating content.
6. Keep the tone instructional and brief. Remove information that drifts into historical narrative.
7. Add the new file to the list below.

### Current ritual docs

* [Scrum](./scrum.md) – daily standup focus and incoming bug review flowchart (overview + decision chart). Canonical process: https://fleetdm.com/handbook/company/product-groups#scrum-at-fleet

### Future candidates (create when there is persistent friction)

These are examples; only add if a stable, repeatable ritual with clear ownership emerges:

* Sprint demo (live run checklist, recording tips)
* Sprint retrospective variations
* Estimation session facilitation
* Incident review (if / when standardized beyond existing handbook guidance)

### Maintenance

Ritual owners (usually the EM or designated facilitator) should review their doc at least quarterly, or immediately after a retrospective surfaces a change.  Keep PRs small and link to the retrospective or decision that prompted the update.
