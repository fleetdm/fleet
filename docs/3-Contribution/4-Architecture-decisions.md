# Architecture decisions

Architecturally significant changes to this project are documented as a collection of records. Documents are numbered sequentially and monotonically. If a decision is reversed, we will keep the old one around, but mark it as superseded (it's still relevant to know that it was the decision, but is no longer the decision).

If you'd like to make a significant change to this program, copy the template below and submit a PR which explains your proposal. Your proposal may accompany a working prototype, but significant decisions should be documented before there is significant development.

# Template

## A Title Which Summarizes The Decision

### Authors

- Your Name ([@username](https://github.com/marpaia))

> List all people that have contributed to the decision and can speak authoritatively about some critical aspect it.

### Status

Accepted (March 15, 2018)

> This should be one of:
> - Proposal (the decision is being proposed and discussed (ie: in a PR))
> - Accepted (the decision has been agreed upon and is the current adopted decision)
> - Superseded (the decision has been superseded by a newer decision (include a link to the new decision))

### Context

The context section outlines the set of conditions which have brought you here. You should outline what it is that you're talking about and why this decision is being documented. Enumerate the context you have which has gone into this decision so that others understand not only the decision itself but why it was the best decision given the context of the situation.

If significant conversation on this topic happened in Slack or in a meeting, do your best to summarize the context. Include links to Slack logs, GitHub issues, Pull Request discussions, etc.

### Decision

Summarize the decision that has been made and it's immediate practical implications. This section should be brief. Save the story-telling for Context and Consequences.


### Consequences

The closing section of the document explains what the consequences of this decision will be.

Explain what's going to happen immediately. If this is a new approach to an existing problem, perhaps a set of refactors will need to take place. If a new programming pattern is being agreed upon, perhaps developers will need to be able to reference some code snippets.

You should also explain the long-term maintainability and scale properties of the decision. Often times, a decision will introduce some new advantage, but nothing in life is without compromise. Explain what we should watch out for as your decision is accepted. What are bottlenecks that you foresee with your solution. How will it fall over? We think about this during the decision making process so that our decisions have as few unexpected consequences as possible after they're adopted.
