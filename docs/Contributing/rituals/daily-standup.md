## Daily standup

Purpose: Provide a fast daily synchronization point for the product group and a quick triage of freshly reported bugs awaiting reproduction.

Cadence: Daily, 15 minutes, same time every working day during the sprint.

Participants: Full product group.

Ritual DRI: EM or assigned team member.

### Format
1. Share screen and open the team's GitHub Projects board.
2. Call on each participant and filter the project board to that assignee.
3. Participant answers the questions in the agenda below. If there are blockers, they are added to the parking lot and the standup continues.
4. Call on the next participant until everyone, including the ritual DRI, has provided an update.
5. Open [GitHub pull requests](https://github.com/pulls) filtered using the template below after being modified to include each product group member:
`is:open is:pr archived:false org:fleetdm org:osquery draft:false sort:created-asc author:sgress454 author:lucasmrod`
6. Call out verbally any PRs open for more than the time to merge KPI goal (24 hours).
7. End the Daily Standup for everyone except those with parking lot issues. 
8. Go through each parking lot item with the relevant participants and define and assign TODOs to resolve the blocker.

You can find our time to merge and other detailed engineering metrics on the [Grafana engineering metrics project](https://fleeteng.grafana.net/d/b97a629f-3626-4a28-9781-0fa3c8427897/engineering-metrics?orgId=1&from=now-90d&to=now&timezone=browser&var-user=$__all&var-user_group=$__all&var-issue_type=$__all). 

> To determine order of standup, some ideas are alphabetical order, [wheel of names](https://wheelofnames.com/), or random. Ritual DRI should call on participants and not wait for volunteers.

### Agenda
- What did you work on yesterday?
- What are you working on today?
- Do you have any blockers?

> Blockers are parked for later discussion to keep the standup short and focused. If a blocker is reported, the name of the team member and the issue are added to the parking lot agenda at the bottom of the ritual document.

### Notes
- See [daily standup definition](https://fleetdm.com/handbook/company/product-groups#sprint-ceremonies) in the handbook.

