# Engineering

## Meetings

### Goals

* Stay in alignment across the whole organization.
* Build teams, not groups of people.
* Provide substantial time for engineers to work on "focused work".

### Principles

* Support the [Maker Schedule](http://www.paulgraham.com/makersschedule.html) by keeping meetings to a minimum.
* Each individual must have a weekly sync 1:1 meeting with their manager. This is key to making sure each individual has a voice within the organization.
* Each team should have a fixed weekly sync check in. This helps reinforce team bonds and alignment.
* Favor async communication when possible. This is very important to make sure every stakeholder on a project can have a clear understanding of what’s happening, or what was decided, without needing to attend every meeting (i.e. if a person is sick or on vacation or just life happened.)
* If an async conversation is not proving to be effective, never hesitate to hop on or schedule a call. Always document the decisions made in a ticket, document, or whatever makes sense for the conversation.

The following is the subset of proposed engineering meetings. Each group is free to treat these as a subset of the expected meetings, and add any other meetings as they see fit.

### Eng together (Weekly ~ one hour)
Promote cohesion across groups in the engineering team. Disseminate engineering-wide announcements.

#### Participants
All of engineering

#### Sample agenda
- Announcements
- “Show and tell”
  - Each engineer gets two minutes to explain (showing, if desired) what they are working on, and why it’s important to the business and/or engineering team.
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

### Group Weeklies (Weekly ~ 30 minutes - one hour)
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
- Platform scale GOTCHAS doc
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

### Engineering Leadership Weekly (Weekly ~ one hour)
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

### How to reach the oncall

Oncall engineers do not need to actively monitor Slack channels, except when called in by the Community or Customer teams. Members of those teams are instructed to `@oncall` in `#help-engineering` to get the attention of the oncall engineer to continue discussing any issues that come up. In some cases, the Community or Customer representative will continue to communicate with the requestor. In others, the oncall engineer will communicate directly (team members should use their judgment and discuss on a case-by-case basis how to best communicate with community members and customers).

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

### The rotation

See [the internal Google Doc](https://docs.google.com/document/d/1FNQdu23wc1S9Yo6x5k04uxT2RwT77CIMzLLeEI2U7JA/edit#) for the engineers in the rotation.

## Project boards

[🚀 Release](https://github.com/orgs/fleetdm/projects/40) - The current release (daily go-to board) for engineers.

[⚗️ Roadmap](https://github.com/orgs/fleetdm/projects/41) - Planning for the next release (shared with product).

## Scaling GOTCHAS

### Overall

Nowadays, Fleet, as a Go server, scales horizontally very well. It’s not very CPU or memory intensive. In terms of load in infrastructure, from highest to lowest is: MySQL, Redis, and Fleet.

In general, we should burn a bit of CPU or memory on the Fleet side if it allows us to reduce the load on MySQL or Redis.

In many, caching helps, but given that we are not doing load balancing based on host id (i.e., make sure that the same host ends up in the same Fleet server). This goes only so far. Caching host-specific data is not done because round-robin LB means all Fleet instances end up circling the total list of hosts.

### How to prevent most of this

The best way we’ve got so far to prevent any scaling issues is to load test things. **Every new feature must have its corresponding osquery-perf implementation as part of the PR, and it should be tested at a scale that is reasonable for the feature**.

Besides that, you should consider the answer(s) to the following question: how can I know that the feature I’m working on is working and performing well enough? Add any logs, metrics, or anything that will help us debug and understand what’s happening when things unavoidably go wrong or take longer than anticipated.

**HOWEVER** (and forgive this Captain Obvious comment): do NOT optimize before you KNOW you have to. Don’t hesitate to take an extra day on your feature/bug work to load test things properly.

## What have we learned so far

This is a document that evolves and will likely always be incomplete. If you feel like something is missing, either add it or bring it up in any way you consider.

## Foreign keys and locking

Among the first things you learn in database data modeling is: that if one table references a row in another, that reference should be a foreign key. This provides a lot of assurances and makes coding basic things much simpler.

However, this database feature doesn’t come without a cost. The one to focus on here is locking, and here’s a great summary of the issue: https://www.percona.com/blog/2006/12/12/innodb-locking-and-foreign-keys/

The TLDR is: understand very well how a table will be used. If we do bulk inserts/updates, InnoDB might lock more than you anticipate and cause issues. This is not an argument to not do bulk inserts/updates, but to be very careful when you add a foreign key.

In particular, host_id is a foreign key we’ve been skipping in all the new additional host data tables, which is not something that comes for free, as with that [we have to keep the data consistent by hand with cleanups](https://github.com/fleetdm/fleet/blob/main/server/datastore/mysql/hosts.go#L309-L309).

### Insert on duplicate update

It’s very important to understand how a table will be used. If rows are inserted once and then updated many times, an easy reflex is to do an `INSERT … ON DUPLICATE KEY UPDATE`. While technically correct, it will be more performant to try to do an update, and if it fails because there are no rows, then do an insert for the row. This means that it’ll fail once, and then it’ll update without issues, while on the `INSERT … ON DUPLICATE KEY UPDATE`, it will try to insert, and 99% of the time, it will go into the `ON DUPLICATE KEY UPDATE`.

This approach has a caveat, it introduces a race condition between the `UPDATE` and the `INSERT` where another `INSERT` might happen in between the two, making the second `INSERT` fail. With the right constraints (and depending on the details of the problem), this is not a big problem. Alternatively, the `INSERT` could be one with an `ON DUPLICATE KEY UPDATE` at the end to recover from this scenario.

This is subtle, but an insert will update indexes, check constraints, etc. At the same time, an update might sometimes not do any of that, depending on what is being updated.

While not a performance GOTCHA, if you do use `INSERT … ON DUPLICATE KEY UPDATE`, beware that LastInsertId will return non-zero only if the INSERT portion happens. [If an update happens, the LastInsertId will be 0](https://github.com/fleetdm/fleet/blob/1aff4a4231ccff4d80889b46b57ed12c5ba1ae14/server/datastore/mysql/mysql.go#L925-L953).

### Host extra data and JOINs

Indexes are great. But like most good things, the trick is in the dosage. Too many indexes can be a performance killer on inserts/updates. Not enough, and it kills the performance of selects.

Data calculated on the fly cannot be indexed, unless it’s precalculated (see counts section below for more information.)

Host data is among the data that changes and grows the most in terms of what we store. It used to be that we used to add more columns in the host table for the extra data in some cases.

Nowadays, we don’t update the host table structure unless we really, really, REALLY need to. Instead, we create adjacent tables that reference a host by id (without an FK). These tables can then be JOINed with the host table whenever needed.

This approach works well for most cases, and for now, it should be the default when gathering more data from hosts in Fleet. However, it’s not a perfect approach as it has its limits.

JOINing too many tables, sorting based on the JOINed table, etc., can have a big performance impact on selects.

Sometimes one strategy that works is selecting and filtering the adjacent table with the right indexes; then, JOIN the host table to that. This works when only filtering/sorting by one adjacent table, and pagination can be tricky.

Solutions can become a curse too. Be mindful of when we might cross that threshold between good and bad performance.

### What db tables matter more when thinking about performance

While we need to be careful about handling everything in the database, not every table is the same. The host and host_* tables are the main cases where we have to be careful when using them in any way.

However, beware of tables that go through async aggregation processes (such as scheduled_query and scheduled_query_stats) or those that are read often as part of the osquery distributed/read and config endpoints.

### Expose more host data in the host listing

Particularly with extra host data (think MDM, Munki, Chrome profiles, etc.), another GOTCHA is that some users have built scripts that go through all hosts by using our list host endpoint. This means that any extra data we add might cause this process to be too slow or timeout (this has happened in the past).

Beware of this, and consider gating the extra data behind a query parameter to allow for a performant backward compatible API that can expose all the data needed in other use cases.

Calculated data is also tricky in the host listing API at scale, as those calculations have to happen for each host row. This can be extra problematic if the sort happens on the calculated data, as all data has to be calculated across all hosts before being able to sort and limit the results (more on this below).

### Understand main use-cases for queries

Be aware of the use cases for an API. For example, take the software listing endpoint. This endpoint lists software alongside the number of hosts with that item installed. It was designed to be performant in a limited use case: list the first eight software items, then count hosts for those software ids.

The problem came later when we learned that we missed an important detail: the UI wanted to sort by amount of host count so that the most popular software appeared on top of this.

This resulted in basically a full host_software table scan per each software row to calculate the count per software. Then, sort and limit. The API worked in the simple case, but it timed out for most customers in the real world.

### On constantly changing data

A very complex thing to do well shows presence data in a somewhat real time fashion. In the case of Fleet, that is host seen_time which is what we use to define if a host is online or offline.

Host seen_time is updated with basically every check-in from any kind of host. Hosts check in every 10 seconds by default. Given that it’s a timestamp reflecting the last time a host contacted Fleet for anything; it’s always different.

While we are doing a few things to make this better, this is still a big performance pain point we have. In particular, we are updating it in bulk. It used to be a column of the hosts' table, which caused a lot of locking. Now it’s an adjacent table without FK.

Luckily, we don’t have anything else (at least up to the moment of this writing) that changes as often as seen_time. However, other features such as software last used can cause similar issues.

### Counts and aggregated data

UX is key for any software. APIs that take longer than 500ms to respond can cause UX issues. Counting things in the database is a tricky thing to do.

In the ideal case, the count query will be covered by an index and be extremely fast. In the worst case, the query will be counting filtering on calculated data, which results in a full (multi) table scan on big tables.

One way to solve this is to pre-calculate data in an async fashion, so we would have a cron that once every hour or so would count whatever we want counted, store the counts, and then counting things is as fast as reading a row in a table.

This approach has a handful of issues:

- The accuracy of the data is worse. We will never get truly accurate counts (the “real-time” count the API returns could change 1ms after we get the value).
- Depending on how many ways we want to count things, we will have to calculate and store them.
- Communicating to the user the interval at which things update can sometimes be tricky.

All of this said, Fleet and osquery work in an “update at an interval” fashion, so we have exactly one pattern to communicate to the user, and then they can understand how most things work in the system.

### Caching data such as app config

Caching is a usual strategy to solve some performance issues in the case of Fleet level data, such as app config (of which we will only have one of), is easy, and we cache at the Fleet server instance level, refreshing the value every one second. App config gets queried with virtually every request, and with this, we reduce drastically how many times the database is hit with that query. The side effect is that a configuration would take one second to be updated in each Fleet instance, which is a price we are willing to pay.

Caching host-level data is a different matter, though. Given that Fleet is usually deployed in infrastructure where the load balancer distributes the load in a round-robin-like fashion (or maybe other algorithms, but nothing aware of anything within Fleet itself). Then virtually all hosts end up being seen by all Fleet instances, so caching host-level data (in the worst case) results in having a copy of all the hosts in each Fleet instance and refreshing that at an interval.

Caching at the Fleet instance level is a great strategy if it can reasonably handle big deployments, as Fleet utilizes minimal RAM.

Another place to cache things would be Redis. The improvement here is that all instances will see the same cache. However, Redis can also be a performance bottleneck depending on how it’s used.

### Redis SCAN

Redis has solved many scaling problems in general, but it’s not devoid of scaling problems of its own. In particular, we learned that the SCAN command scans the whole key space before it does the filtering. This can be very slow, depending on the state of the system. If Redis is slow, a lot suffers from it.


<meta name="maintainedBy" value="zwass">
<meta name="title" value="🚀 Engineering">

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
