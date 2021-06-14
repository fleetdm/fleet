# Support process

This living document outlines the customer and community support process at Fleet.

The support process is accomplished via an on-call rotation and the weekly Roundup meeting.

The individual on-call is responsible for responding to Slack comments, Slack threads, and GitHub issues raised by customers and the community.

The Roundup meeting at Fleet provides time to discuss action items from that are collected during the support process. The individual on-call is responsible for preparing for and leading the Roundup meeting. The Roundup meeting occurs at a weekly cadence and usually falls on a Tuesday (EST time).

## Goals

- Get familiar with and stay abreast of what our community wants and the problems they're having.

- Make people feel heard and understood.  

- Celebrate contributions. 

- Identify actionable bugs, feature requests, pull requests and questions.

## How?

- No matter what, folks who post a new comment in Slack or issue in GitHub get a **response** from the individual on-call within 1 business day. The response doesn't need to include an immediate answer.

- The individual on-call is responsible to either schedule a 10 minute call or join the 🧩 Product meeting to ask questions they were unable to answer. If a response is needed quicker, you can always DM Fleet team members in Slack. This way, people get answers within 1 business day.

- If you do not understand the question or comment raised, request more details to best understand the next steps.

- If an appropriate response is outside your scope, please post to #oncall-chatter, an internal slack channel designed to filter community support questions to the Fleet team.

## Sources

There are three sources that the individual on-call should monitor for activity:

1. Customer Slack channels - Found under the "Connections" section in Slack. These channels are usually titled "at-insert-customer-name-here"

2. Community chatroom - https://osquery.slack.com, #fleet channel

3. GitHub issues and pull requests - [Github Triage: Community contributions with no milestones or assignees](https://github.com/issues?q=is%3Aopen+archived%3Afalse+org%3Afleetdm+no%3Amilestone+no%3Aassignee+sort%3Aupdated-desc+)

## Roundup preparation

The Roundup meeting occurs at Fleet one every week. One to two days prior to the meeting, the individual on-call will revisit old threads and determine which items are actionable.

A list of all social channels to visit during Roundup preparation can be found in the [Community support spin Google doc](https://docs.google.com/document/d/1dPxB88SQeDdZkZjg7RMwzdq0umMSHCZ2B2UdiZ4ko5s/edit#).

All pull requests, bugs, feature requests, and questions are candidates for discussion at the Roundup meeting. 

The steps taken to determine if an item should be brought to the Roundup meeting are as follows:

#### Pull requests

- Would this pull request result in any current documentation becoming inaccurate or out of date?  If so, then make sure that the PR also covers those documentation changes.

- Does this pull request seem low risk, e.g. a typo fix for the docs?  Could it possibly be merged on the spot during the roundup? If no, try to QA the change and verify it works.  If you aren't sure, work with the person who submitted it and other people who might be reading the PR to get answers.

- If yes, then add to the [🐄 Roundup Google doc](https://docs.google.com/document/d/16n0xT9RVqnlNSGaTLXmPJp-KJT9JN3cEyXSbudqBiZQ/edit#heading=h.le0crozigvb) in the following format:

```
PULL REQUEST: (Who is the individual submitting the PR? Where do they work?)

1. Include the title of the pull request here.

2. Include a description of the changes here.

3. Include reasoning on why you think it makes sense or does not make sense to merge these changes here.
```

#### Bugs

- Wait... is this actually the intentional, documented behavior of the product? If so, gently, empathetically let the reporter know and link them to the docs.

- Prove the bug exists. Record a Loom video proving the bug (shorter the better), or work with the reporter to gather up concise steps to reproduce, then verify the bug yourself if possible.

- If you're able to reproduce the bug, let the reporter know-- share your attempted proof.

- When you have a proof of the bug, add to the [🐄 Roundup Google doc](https://docs.google.com/document/d/16n0xT9RVqnlNSGaTLXmPJp-KJT9JN3cEyXSbudqBiZQ/edit#heading=h.le0crozigvb) in the following format:

```
BUG: (Who is the individual reporting the bug? Where do they work?)

1. Expected behavior: Provide a short description of the expected behavior here.

2. Actual behavior Provide a short description of the actual behavior here. Include a link to the Loom video that includes proof of the bug.
```

#### Feature requests

- Wait... does this feature already exist in Fleet? If so, gently, empathetically let the reporter know and link them to the release notes if the feature was introduced in a recent release of Fleet.

- Is there already an open PR and/or issue seeking to address this? If so, link the person to the PR and triage it as "Ready for roundup".

- Otherwise reply to let the person know you'll discuss with the rest of the team and add to the [🐄 Roundup Google doc](https://docs.google.com/document/d/16n0xT9RVqnlNSGaTLXmPJp-KJT9JN3cEyXSbudqBiZQ/edit#heading=h.le0crozigvb) in the following format:

```
FEATURE REQUEST: (Who is the individual submitting the feature request? Where do they work?)

1. What does the user want to be able to do in Fleet? Is the requested feature for fleetctl, REST API or the Fleet UI?

2. Motivation: What is the use case or motivation behind the request? You may have to ask the reporter additional questions to uncover this information. For example, "why would it be helpful to have this ability in Fleet?"
```

#### Questions

- Is this question already answered in our docs / website? If so, link to the specific section of the docs/website ± summarize for them.

- Do you think you know the answer? If so, make a PR to the docs/website. Link the person to your PR.

- Otherwise if you don't know the answer, reply to let the person know you're working on it add to the [🐄 Roundup Google doc](https://docs.google.com/document/d/16n0xT9RVqnlNSGaTLXmPJp-KJT9JN3cEyXSbudqBiZQ/edit#heading=h.le0crozigvb) in the following format:

```
QUESTION: (Who is the individual asking the question? Where do they work?)

1. Include the question here.

2. Include your best guess answer here.

3. Include the location in the Fleet documentation where inserting the future answer makes the most sense to you.
```
