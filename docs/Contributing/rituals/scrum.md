## [Daily Standup](https://fleetdm.com/handbook/company/product-groups#sprint-ceremonies) 

Purpose: Provide a fast daily synchronization point for the product group and a quick triage of freshly reported bugs awaiting reproduction.

Cadence: Daily on all working days of the sprint (3‑week sprint cadence aligned with Fleet release cycle).

Participants: Entire product group (engineers, EM, PD as needed). QA engineer (or whoever is wearing the QA hat) explicitly called out during the incoming bug review step.

Agenda:
* What changed since yesterday / what will change before tomorrow
* Review incoming bugs for your team.
* Blockers (surface, not solve in-room)

Out of scope: Deep design debates, estimation, and retro topics (park these and spin off after the ritual with only needed folks).

Handbook reference (authoritative process + item definitions): https://fleetdm.com/handbook/company/product-groups#scrum-at-fleet

## Incoming bug review

Here is a simple decision chart to make reviewing new incoming bugs with the team quick during scrum.

```mermaid
flowchart TD
    A[Is this a bug?] -->|Yes| B[Assign QA engineer timebox 30–60 min later today]
    A -->|No| C[Close with comment]

    B --> D[QA reproduces bug?]
    D -->|Yes| E[Move to Inbox Add :product, remove :reproduce]
    D -->|No| F[Comment asking for more info. If it's a customer reported bug, add :help-customers]

    F --> G[Wait 1 week for response]
    G -->|No response| H[Close with comment Can reopen if more info provided]
    G -->|Response| B
```