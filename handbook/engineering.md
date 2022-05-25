# Engineering

## Meetings

### Goals

* Stay in alignment across the whole organization.
* Build teams, not groups of people.
* Provide substantial time for engineers to work on "focused work".

### Principles

* Keep meetings to a minimum. Sometimes that will be very very few meetings, and sometimes the minimum will be quite a few of them. But always try to reduce meetings, just like we do with process.
* Each individual must have a weekly sync 1:1 meeting with their manager. This is key to making sure each individual has a voice within the organization.
* Each team should have a fixed weekly sync check in. This helps reinforce team bonds and alignment.
* Favor async communication when possible. This is very important to make sure every stakeholder on a project can have a clear understanding of what’s happening, or what was decided, without needing to attend every meeting (i.e. if a person is sick or on vacation or just life happened.)
* If an async conversation is not proving to be effective, never hesitate to hop on a call. Always document the decisions made in a ticket, document, or whatever makes sense for the conversation.

The following is the subset of proposed engineering meetings. Each group is free to treat these as a subset of the expected meetings, and add any other meetings as they see fit.

### Eng Together (Weekly ~ 1 hour)
Promote cohesion across groups in the engineering team. Disseminate engineering-wide announcements.

#### Participants
All of engineering

#### Sample agenda
- Announcements
- “Show and tell”
  - Each engineer gets 2 minutes to explain (showing, if desired) what they are working on, and why it’s important to the business and/or engineering team.
- Deeper dive
  - One or a few engineers go deeper on a topic relevant to all of engineering.
- Social
  - Structured and/or unstructured social activities

### Release Retro (Each release ~ 30 minutes)
Gather feedback from all participants in each release. Used to improve communication and processes.

This meeting will likely need to be split in the future as the number of attendees increases.

#### Participants
Members of each group (+ quality)

#### Sample agenda
For each attendee:
- What went well this release cycle?
- What could have gone better this release cycle?
- What should we remember next time?

### Group Weeklies (Weekly ~ 30 minutes - 1 hour)
A chance for deeper, synchronous discussion on topics relevant to that group.

eg. “Interface Weekly” - “Platform Weekly” - “Agent Weekly”

In some groups, this may be split into smaller discussions related to the differing focuses of members within the group.

#### Participants
Members of each group

#### Sample agenda (Platform)
- Announcements
- Anything at risk for the release?
- Bug assignment
- Retries in the datastore
- Platform scale gotchas doc
- MarshalJSON to hide passwords and API tokens. Thoughts?

#### Sample Agenda (Interface)
- What’s good?
- Anything at risk for the release?
- Bug assignment
- Confirm response payload matches spec
- Discuss completion of Redux removal

### Standup (Optional, varies by group)

Provide status reports, discover blockers, and keep the group in sync.

Each group can implement daily (or some other cadence) standups if desired. Ultimately, it’s up to the Engineering Manager to ensure that the team is communicating appropriately to deliver results.

#### Participants
Members of the group

### Engineering Leadership Weekly (Weekly ~ 1 hour)
Engineering leaders discuss topics of importance that week.

#### Participants
CTO + Engineering managers

#### Sample agenda
- Fullstack engineer hiring
- Engineering process discussion
- Review Q2 OKRs

### Product/Eng Weekly (Weekly - 30 minutes)
Engineering and Product sync on priorities for the upcoming release, surface and address any inter-group dependencies.

#### Participants
CTO + Engineering managers + PMs

#### Sample agenda
- Plan for what's going into next release
- Identify inter-group dependencies
- Ensure items are moving through architect/estimation


## Release process

This section outlines the release process at Fleet.

The current release cadence is once every 3 weeks and concentrated around Wednesdays.

### Release freeze period

In order to ensure quality releases, Fleet has a freeze period for testing prior to each release. Effective at the start of the freeze period, new feature work will not be merged.

Release blocking bugs are exempt from the freeze period and are defined by the same rules as patch releases, which include:
1. Regressions
2. Security concerns
3. Issues with features targeted for current release

Non-release blocking bugs may include known issues that were not targeted for the current release, or newly documented behaviors that reproduce in older stable versions. These may be addressed during a release period by mutual agreement between the [Product](./product.md) and Engineering teams.

### Release day

Documentation on completing the release process can be found
[here](../docs/Contributing/Releasing-Fleet.md).

## Oncall rotation

The oncall engineer is a second-line responder to questions raised by customers and community members. The Community team is responsible for the first response to GitHub issues, pull requests, and Slack messages in the osquery and other public Slacks. The Customer team is responsible for the first response to messages in private customer Slack channels.

Oncall engineers do not need to actively monitor Slack channels, except when called in by the Community or Customer teams. Members of those teams are instructed to `@oncall` in `#help-engineering` to get the attention of the oncall engineer to continues discussing any issues that come up. In some cases, the Community or Customer representative will continue to communicate with the requestor, and in others, the oncall engineer will communicate directly.

### Handoff

Every week, the oncall engineer changes. Here are some tips for making this handoff go smoothly:

1. The new oncall engineer should change the `@oncall` alias in Slack to point to them. In the
   search box, type "people" and select "People & user groups." Switch to the "User groups" tab.
   Click `@oncall`. In the right sidebar, click "Edit Members." Remove the former oncall, and add
   yourself.

2. Hand off newer conversations. For more recent threads, the former -call can unsubscribe from the
   thread, and the new oncall should subscribe. The former oncall should explicitly share each of
   these threads, and the new on-call can select "Get notified about new replies" in the "..." menu.
   The former oncall can select "Turn off notifications for replies" in that same menu. It can be
   helpful for the former oncall to remain available for any conversations they were deeply involved
   in, so use your judgment on which threads to hand off.

## Project boards

[🚀 Release](https://github.com/orgs/fleetdm/projects/40) - The current release (daily go-to board) for engineers.

[⚗️ Roadmap](https://github.com/orgs/fleetdm/projects/41) - Planning for the next release (shared with product).

## Rituals

The following rituals are engaged in by the  directly responsible individual (DRI) and at the frequency specified for the ritual.

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Pull request review | Daily | Engineers go through pull requests for which their review has been requested. | Zach Wasserman |
| Engineering group discussions | Weekly | See "Group Weeklies".  | Zach Wasserman |
| On-call handoff | Weekly | Hand off the on-call engineering responsibilities to the next on-call engineer. | Zach Wasserman |
| Release ritual | Every three weeks | Go through the process of releasing the next iteration of Fleet. | Zach Wasserman |

## Slack channels

The following [Slack channels are maintained](https://fleetdm.com/handbook/company#group-slack-channels) by this group:

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:------------------------------------|:--------------------------------------------------------------------|
| `#help-engineering`                 | Zach Wasserman
| `#g-platform`                       | Tomás Touceda
| `#g-interface`                      | Luke Heath
| `#g-agent`                          | Zach Wasserman
| `#_pov-environments`                | Ben Edwards



<meta name="maintainedBy" value="zwass">
