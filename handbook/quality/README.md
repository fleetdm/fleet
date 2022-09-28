# Quality

## Human-oriented QA

Fleet uses a human-oriented quality assurance (QA) process to make sure the product meets the standards of users and organizations.

Automated tests are important, but they can't catch everything. Many issues are hard to notice until a human looks empathetically at the user experience, whether in the user interface, the REST API, or the command line.

The goal of quality assurance is to catch unexpected behavior before release:
- Bugs
- Edge cases
- Error message UX
- Developer experience using the API/CLI
- Operator experience looking at logs
- API response time latency
- UI comprehensibility
- Simplicity
- Data accuracy
- Perceived data freshness
- Product’s ability to save users from themselves

## Collecting bugs

To try Fleet locally for QA purposes, run `fleetctl preview`, which defaults to running the latest stable release.

To target a different version of Fleet, use the `--tag` flag to target any tag in [Docker Hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated), including any git commit hash or branch name. For example, to QA the latest code on the `main` branch of fleetdm/fleet, you can run: `fleetctl preview --tag=main`

To start preview without starting the simulated hosts, use the `--no-hosts` flag (e.g., `fleetctl preview --no-hosts`).


For each bug found, please use the [bug report template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=) to create a new bug.

## Bug process

### Bug States
The lifecycle stages of a bug at Fleet are: 
1. Inbox 
2. Acknowledged 
3. Reproduced 
4. In Engineering Process
5. Awaiting QA

The above are all the possible states for a bug as envisioned in this process. These states each correspond to a set of Github labels, assignees, and board memberships. 
See [Appendix A](#appendix-a) at the end of this document for a description of these states and a convenience link to each GitHub filter.

### Inbox
When a new bug is created using the [bug report form](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), it is in the "inbox" state. 
At this state, the [bug review DRI](#rituals) (QA) is responsible for going through the inbox and asking for more reproduction details from the reporter, asking the product team for more guidance, or acknowledging the bugs.

> Some bugs may also be the domain of the digital-experience team. If QA believes this is the case, then QA should put the bug onto the g-digital-experience board and assign it to ... TODO. The digital experience team has their own bug process which is not governed by this process.

### Weekly bug review
QA has weekly check-in with product to go over the inbox items. QA is responsible for proposing “not a bug”, closing due to lack of response (with a nice message), or raising other relevant questions. All requires product agreement

Requesters have six weeks to provide follow-up information for each info request. We will ping them again as a reminder at three weeks. After six weeks, we will close the bug to remove it from our visibility, but requesters are welcome to re-open and provide context.

QA may also propose that a reported bug is not actually a bug. A bug is defined as “behavior which is not according to spec or implied by spec.” Thereafter, it is assigned to the relevant product manager for them to decide on its priority.

### Acknowledging bugs
Otherwise, QA should apply the acknowledged state to the bug. QA has one week to reproduce the bug.

Once reproduced, QA should document the reproduction steps and move it to the reproduced state.

### Reproduced
When reproduced, the assigned engineering manager is responsible for investigating root cause of the bug and proposing solutions to their product counterpart if it requires discussion. Otherwise, the EM includes it in this release if there is space or the next release.

### After reproduced
After it is in a release formally, the bug should be treated like any other piece of work per the standard engineering process.

### Fast track for Fleeties
Fleeties do not have to wait for QA to reproduce the bug. If you are confident it is reproducible, is a bug, and the reproduction steps are well-documented, it can be moved directly to the reproduced state.

### During release testing
When release is in testing, QA should use the the slack channel #help-release to keep everyone aware of issues found. All bugs related to a release should be reported in the channel after creating the bug first.

In the release channel, product may decide whether the bug is a release blocker. Release blockers must be fixed before a release can be cut.

### Critical bugs
A critical bug is defined as: “a bug that causes users to be unable to use a workflow, upgrade Fleet, or causes irreversible damage such as loss of data.”

The key thing about a critical bug is that we need to immediately inform customers and the community about it so they don’t trigger it themselves. When bug meeting the definition of critical is found, the bug finder is responsible for raising an alarm immediately.
Raising an alarm means: pinging @here in the #help-product channel with the filed bug.

If the “bug finder” is not a Fleetie (such as community-reported), then whoever sees the critical bug should raise the alarm. (We would expect this to be CX in the community slack or QA in the bug inbox, though it could be anyone.)
Note that the “bug finder” here is NOT necessarily the **first** person who sees the bug. If you come across a bug you think is critical but it has not been escalated, raise the alarm!

Once raised, product confirms whether it is critical or not, and defines expected behavior.
When outside of working hours for the product team or if no one from product responds within 1 hour, then fall back to the #help-p1.

Once the critical bug is confirmed, customer experience needs to ping both customers and the community to warn them. If CX is not available, the oncall engineer is responsible for doing this.
If a quick fix workaround exists, that should be communicated as well for those who are already upgraded.

### Measurement
We will track the success of this process by observing the throughput of issues through the system and identifying where buildups (and therefore bottlenecks) are occurring. 
The metrics are: Total # bugs open. Bugs in each state (inbox, acknowledged, reproduced). Each week these are tracked and shared in the weekly update by Charlie Chance.

### Orphans
Occasionally, bugs may get lost if, for example, a label is misapplied. Miscategorized issues may slip through the filters and languish in a grey zone. The “orphans” and “reproduced orphans” states exist to catch these issues. 
Every week, the head of product is responsible for reviewing these two states to identify any that are not properly categorized in the process.

## Appendix A: Bug states and filters

### Inbox
The bug has just come in. 

If using the standard bug report, the bug is labeled “bug” and “reproduce” and not assigned to anyone and not on a board. [See on Github](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+label%3Abug+label%3A%3Areproduce+-project%3Afleetdm%2F37+-project%3Afleetdm%2F40+sort%3Aupdated-asc).

### Acknowledged 
QA has gone through the inbox and has accepted it as a bug to be reproduced. 

QA assigns themselves and adds it to the Release board under “awaiting QA”. [See on Github](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+label%3Abug+label%3A%3Areproduce+-project%3Afleetdm%2F37+sort%3Aupdated-asc).

### Reproduced
QA has reproduced the issue successfully. It should now be transferred to engineering to work on. 

Remove the “reproduce” label and add the label of the relevant team (#agent, #platform, #interface) and assign it to the relevant engineering manager (make your best guess as to which team – the EM will re-assign if they think it belongs to another team). Move it to “Ready” in the Release board. [See on Github](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+label%3Abug+-label%3A%3Areproduce+-project%3Afleetdm%2F37+project%3Afleetdm%2F40+-assignee%3Axpkoala+sort%3Aupdated-asc).

### Orphans 
Bugs which do not have the reproduce label and do not exist on the release board. This filter serves as a sanity check. There should be no bugs in this state, because it means this bug is likely to be forgotten by our process. [See on Github](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+sort%3Aupdated-asc+label%3Abug+-label%3A%3Areproduce+-project%3Afleetdm%2F37+-project%3Afleetdm%2F40+).

### Reproduced orphans 
Bugs which do not have the reproduce label and do exist on the release board, but do not have one of the three teams tagged. There should be no bugs in this state. This will risk being forgotten by the process because it does not appear in any of the standard team-based filters, which means it risks never being seen by engineering. [See on Github](https://github.com/fleetdm/fleet/issues?q=archived%3Afalse+org%3Afleetdm+is%3Aissue+is%3Aopen+sort%3Aupdated-asc+label%3Abug+-label%3A%3Areproduce+-project%3Afleetdm%2F37+project%3Afleetdm%2F40+-assignee%3Axpkoala+-label%3A%23interface+-label%3A%23platform+-label%3A%23agent+).

### All bugs
[See on Github](https://github.com/fleetdm/fleet/issues?q=is%3Aissue+is%3Aopen+label%3Abug).

## Rituals

Directly Responsible Individuals (DRI) engage in the ritual(s) below at the frequency specified.

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-------------------------|:----------------------------------------------------|-------------------|
| Bug Review                   | Weekly                   | Review bugs that are in QA's inbox                  | Reed Haynes       |

## Slack channels

This group maintains the following [Slack channels](https://fleetdm.com/handbook/company#why-group-slack-channels):

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#why-group-slack-channels)|
|:------------------------------------|:--------------------------------------------------------------------|
| `#help-qa`                          | Reed Haynes                                                         |
| `#help-release`                     | Reed Haynes                                                         |

<meta name="maintainedBy" value="zhumo">
<meta name="title" value="🪢Quality">
