## Community

As an open-core company, Fleet endeavors to build a community of engaged users, customers, and
contributors.

### Posting on social media as Fleet

Self-promotional tweets are non-ideal tweets.  (Same goes for, to varying degrees, Reddit, HN, Quora, StackOverflow, LinkedIn, Slack, and almost anywhere else.)  See also https://www.audible.com/pd/The-Impact-Equation-Audiobook/B00AR1VFBU

Great brands are [magnanimous](https://en.wikipedia.org/wiki/Magnanimity).

### Press releases

If we are doing a press release, we are probably pitching it to one or more reporters as an exclusive story, if they choose to take it.  Consider not sharing or publicizing any information related to the upcoming press release before the announcement.  See also https://www.quora.com/What-is-a-press-exclusive-and-how-does-it-work


### Community contributions (pull requests)

The top priority when community members contribute PRs is to help the person feel engaged with
Fleet. This means acknowledging the contribution quickly (within 1 business day), and driving to a
resolution (close/merge) as soon as possible (may take longer than 1 business day).

#### Process

1. Decide whether the change is acceptable (see below). If this will take time, acknowledge the
   contribution and let the user know that the team will respond. For changes that are not
   acceptable, thank the contributor for their interest and encourage them to open an Issue, or
   discuss proposed changes in the `#fleet` channel of osquery Slack before working on any more
   code.
2. Help the contributor get the quality appropriate for merging. Ensure that the appropriate manual
   and automated testing has been performed, changes files and documentation are updated, etc.
   Usually this is best done by code review and coaching the user. Sometimes (typically for
   customers) a Fleet team member may take a PR to completion by adding the appropriate testing and
   code review improvements.
3. Any Fleet team member may merge a community PR after reviewing it and addressing any necessary
   changes. Before merging, double-check that the CI is passing, documentation is updated, and
   changes file is created. Please use your best judgement.
4. Thank and congratulate the contributor! Consider sharing with the team in the `#g-growth` channel
   of Fleet Slack so that it can be publicized by social media. Folks who contribute to Fleet and
   are recognized for their contributions often become great champions for the project.

#### What is acceptable?

Generally, any small documentation update or bugfix is acceptable, and can be merged by any member
of the Fleet team. Additions or fixes to the Standard Query Library are acceptable as long as the
SQL works properly and they attributed correctly. Please use your best judgement.

Larger changes and new features should be approved by the appropriate [Product
DRI](./product.md#product-dris). Ask in the `#g-product` channel in Fleet Slack.

<meta name="maintainedBy" value="zwass">
