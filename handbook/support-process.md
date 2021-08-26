# Support process

This living document outlines the customer and community support process at Fleet.

The support process is accomplished via an on-call rotation and the weekly on-call retro meeting.

The individual on-call is responsible for responding to Slack comments, Slack threads, and GitHub issues raised by customers and the community.

The daily standup meeting at Fleet provides time to discuss highlights and answer the following questions about the previous week's on-call:

1. What went well?

2. What could have gone better?

3. What should we remember next time?

This way, the Fleet team can constantly improve the effectiveness and experience during future on-call rotations.

## Goals

- Get familiar with and stay abreast of what our community wants and the problems they're having.

- Make people feel heard and understood.  

- Celebrate contributions. 

- Identify actionable bugs, feature requests, pull requests and questions.

## How?

- No matter what, folks who post a new comment in Slack or issue in GitHub get a **response** from the individual on-call within 1 business day. The response doesn't need to include an immediate answer.

- The individual on-call is responsible to either schedule a 10 minute call or join the 🧩 Product meeting to ask questions they were unable to answer. If a response is needed quicker, you can always DM Fleet team members in Slack. This way, people get answers within 1 business day.

- If you do not understand the question or comment raised, [request more details](#requesting-more-details) to best understand the next steps.

- If an appropriate response is outside your scope, please post to `#oncall-chatter`, a confidential Slack channel in the Fleet Slack workspace.

- If the comment appears to be a feature request in a customer channel, please post a link to the customer's comment in `#oncall-chatter`. This way, an individual on the Product team can collect relevant information and file a GitHub issue.

- If things get heated, remember to stay [positive and helpful](https://canny.io/blog/moderating-user-comments-diplomatically/).  If you aren't sure how best to respond in a positive way, or if you see behavior that violates the Fleet code of conduct, get help.

#### Requesting more details

Typically, the *questions*, *bug reports*, and *feature requests* raised by members of the community will be missing helpful context, recreation steps, or motivations respectively.

❓ For questions that you don't immediately know the answer to, it's helpful to ask follow up questions to receive additional context. 

- Let's say a community member asks the question "How do I do X in Fleet?" A follow question could be "what are you attempting to accomplish by doing X?" 
- This way, you now has additional details when the primary question is brought to the Roundup meeting. In addition, the community member receives a response and feels heard.

🦟 For bug reports, it's helpful to ask for recreation steps so you're later able to verify the bug exists.

- Let's say a community member submits a bug report. An example follow up question could be "Can you please walk me through how you encountered this issue so that I can attempt to recreate it?"
- This way, you now have steps the verify whether the bug exists in Fleet or if the issue is specific to the community member's environment. If the latter, you now have additional information for further investigation and question asking.

💡 For feature requests, it's helpful to ask follow up questions in an attempt to understand the "why?" or underlying motivation behind the request.

- Let's say a community member submits the feature request "I want the ability to do X in Fleet." A follow up question could be "If you were able to do X in Fleet, what's the next action you would take?" or "Why do you want to do X in Fleet?." 
- Both of these questions provide helpful context on the underlying motivation behind the feature request when it is brought to the Roundup meeting. In addition, the community member receives a response and feels heard.

#### New feature request issues

After [requesting more details](#requesting-more-details), please add the milestone associated with the current time we are along the roadmap timeline. For example, if the current date is June 25, 2021, we would add the H1 2021 milestone to the issue.

Feature request issues automatically include the "idea" label. The "idea" label provides the signal that this issue is an item the Fleet team would like to discuss at a later date. The time of discussion is indicated by the issue's milestones.

#### Closing issues

It is often a good idea to let the original poster (OP) close their issue themselves, since they are usually well equipped to decide whether the issue is resolved.   In some cases, circling back with the OP can be impractical, and for the sake of speed issues might get closed.

Keep in mind that this can feel jarring to the OP.  The effect is worse if issues are closed automatically by a bot (See [balderashy/sails#3423](https://github.com/balderdashy/sails/issues/3423#issuecomment-169751072) and [balderdashy/sails#4057](https://github.com/balderdashy/sails/issues/4057) for examples of this.)

To provide another way of tracking status without closing issues altogether, consider using the green labels that begin with "+".  To explore them, type `+` from GitHub's label picker.


## Sources

There are three sources that the individual on-call should monitor for activity:

1. Customer Slack channels - Found under the "Connections" section in Slack. These channels are usually titled "at-insert-customer-name-here"

2. Community chatroom - https://osquery.slack.com, #fleet channel

3. GitHub issues and pull requests - [Github Triage: Community contributions with no milestones or assignees](https://github.com/issues?q=is%3Aopen+archived%3Afalse+org%3Afleetdm+no%3Amilestone+no%3Aassignee+sort%3Aupdated-desc+)

## Resources

There are several locations in Fleet's public and internal documentation that can be helpful when answering questions raised by the community:

1. The frequently asked question (FAQ) documents in each section found in the `/docs` folder. These documents are the [Using Fleet FAQ](../docs/1-Using-Fleet/FAQ.md), [Deploying FAQ](../docs/2-Deploying/FAQ.md), and [Contributing FAQ](../docs/3-Contributing/FAQ.md).

2. The [Internal FAQ](https://docs.google.com/document/d/1I6pJ3vz0EE-qE13VmpE2G3gd5zA1m3bb_u8Q2G3Gmp0/edit#heading=h.ltavvjy511qv) document.
