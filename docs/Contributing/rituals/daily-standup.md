## [Daily Standup](https://fleetdm.com/handbook/company/product-groups#sprint-ceremonies) 

Purpose: Provide a fast daily synchronization point for the product group and a quick triage of freshly reported bugs awaiting reproduction.

Cadence: Daily, 15 minutes, same time every working day during the sprint.

Participants: Full product group.

Facilitator: EM or assigned team member.

### Agenda
1. Shares screen and open the team's GitHub Projects board. 
2. Call on each participant and filters the project board to that assignee. 
3. Participant answers the questions in the standup agenda below. 
4. Call on the next participant until everyone, including the facilitator, has provided an update. 
5. Complete incoming bug review.
6. End the Daily Standup for everyone except those with parking lot issues. 
7. Go through each parking lot item with the relevant participants and define and assign TODOs to resolve the blocker.

### Standup agenda
- What did you work on yesterday?
- What are you working on today? 
- Do you have any blockers? 

### Notes
Topics outside the agenda are parked for later discussion to keep the standup short and focused. If a blocker is reported, or there is need for further discussion, the name of the team member and the issue are added to the parking lot agenda at the bottom of the ritual document.

Handbook reference (process and definitions): https://fleetdm.com/handbook/company/product-groups#scrum-at-fleet

## Incoming bug review
Each bug review should take less than 1 minute. The intent is not to debug, but to acknowledge, assign, and share any immediate context. 

1. Open the [Fleet GitHub issue list filtered to the labels](https://github.com/fleetdm/fleet/issues?q=is%3Aissue%20state%3Aopen%20label%3Abug%20label%3A%3Areproduce) `bug` and `:reprodce`.
2. Make sure there are no new bugs without a product group label. If there are, add the appropriate product group label.
3. Filter the list by `bug`, `:reproduce`, and your product group label.
4. For each item, quickly move through the decision flow chart below.

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
