# Engineering

## Release process

This section outlines the release process at Fleet.

The current release cadence is once every 3 weeks and concentrated around Wednesdays.

### Release freeze period

In order to ensure quality releases, Fleet has a freeze period for testing prior to each release. Effective at the start of the freeze period, new feature work will not be merged. 

Release blocking bugs are exempt from the freeze period and are defined by the same rules as patch releases, which include:
1. Regressions
2. Security concerns
3. Issues with features targeted for current release

Non-release blocking bugs may include known issues that were not targeted for the current release, or newly documented behaviors that reproduce in older stable versions. These may be addressed during a release period by mutual agreement between [Product](./product.md) and Engineering teams. 

### Release day

Documentation on completing the release process can be found
[here](../docs/03-Contributing/05-Releasing-Fleet.md).

## On-call rotation

This section outlines the on-call rotation at Fleet.

The on-call engineer is responsible for responding to technical Slack comments, Slack threads, and GitHub issues raised by customers and the community which cannot handled by the [Customer Success team](./customers.md).

### Goals
At Fleet, our primary quality objectives are *customer service* and *defect reduction*. This entails Key Performance Indicators such as customer response time and number of bugs resolved per cycle and. 

- Get familiar with and stay abreast of what our community wants and the problems they're having.

- Make people feel heard and understood.  

- Celebrate contributions. 

- Triage bugs, identify community feature requests, community pull requests and community questions.

### How?

- No matter what, folks who post a new comment in Slack or issue in GitHub get a **response** from the on-call engineer within 1 business day. The response doesn't need to include an immediate answer.

- The on-call engineer can discuss any items that require assistance at the end of the daily standup. They are also requested to attend the "Customer experience standup" where they can bring questions and stay abreast of what's happening with our customers.

- If you do not understand the question or comment raised, [request more details](#requesting-more-details) to best understand the next steps.

- If an appropriate response is outside your scope, please post to `#help-oncall`, a confidential Slack channel in the Fleet Slack workspace.

- If things get heated, remember to stay [positive and helpful](https://canny.io/blog/moderating-user-comments-diplomatically/).  If you aren't sure how best to respond in a positive way, or if you see behavior that violates the Fleet code of conduct, get help.

### Requesting more details

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

### Feature requests

If the feature is requested by a customer, the on-call engineer is requested to create a feature request issue and follow up with the customer by linking them to this issue. This way, the customer can add additional comments or feedback to the newly filed feature request issue.

If the feature is requested by anyone other than a customer (ex. user in #fleet Slack), the on-call engineer is requested to point to the user to the [feature request GitHub issue template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title=) and kindly ask the user to create a feature request.

### Closing issues

It is often a good idea to let the original poster (OP) close their issue themselves, since they are usually well equipped to decide whether the issue is resolved.   In some cases, circling back with the OP can be impractical, and for the sake of speed issues might get closed.

Keep in mind that this can feel jarring to the OP.  The effect is worse if issues are closed automatically by a bot (See [balderashy/sails#3423](https://github.com/balderdashy/sails/issues/3423#issuecomment-169751072) and [balderdashy/sails#4057](https://github.com/balderdashy/sails/issues/4057) for examples of this.)

To provide another way of tracking status without closing issues altogether, consider using the
green labels that begin with "+".  To explore them, type `+` from GitHub's label picker.

### Version support

In order to provide the most accurate and efficient support, Fleet will only target fixes based on the latest released version. Fixes in current versions will not be backported to older releases.

Community version supported for bug fixes: **Latest version only**
 
Community support for support/troubleshooting: **Current major version**

Premium version supported for bug fixes: **Latest version only**

Premium support for support/troubleshooting: **All versions**

### Sources

There are four sources that the on-call engineer should monitor for activity:

1. Customer Slack channels - Found under the "Connections" section in Slack. These channels are usually titled "at-insert-customer-name-here"

2. Community chatroom - https://osquery.slack.com, #fleet channel

3. Reported bugs - [GitHub issues with the "bug" and ":reproduce" label](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3Abug+label%3A%3Areproduce). Please remove the ":reproduce" label after you've followed up in the issue.

4. Pull requests opened by the community - [GitHub open pull requests](https://github.com/fleetdm/fleet/pulls?q=is%3Aopen+is%3Apr)

### Resources

There are several locations in Fleet's public and internal documentation that can be helpful when answering questions raised by the community:

1. The frequently asked question (FAQ) documents in each section found in the `/docs` folder. These documents are the [Using Fleet FAQ](https://fleetdm.com/docs/using-fleet/faq), [Deploying FAQ](https://fleetdm.com/docs/deploying/faq), and [Contributing FAQ](https://fleetdm.com/docs/contributing/faq).

2. The [Internal FAQ](https://docs.google.com/document/d/1I6pJ3vz0EE-qE13VmpE2G3gd5zA1m3bb_u8Q2G3Gmp0/edit#heading=h.ltavvjy511qv) document.

### Handoff

Every week, the oncall engineer changes. Here are some tips for making this handoff go smoothly:

1. The new oncall engineer should change the `@oncall` alias in Slack to point to them. In the
   search box, type "people" and select "People & user groups". Switch to the "User groups" tab.
   Click `@oncall`. In the right sidebar, click "Edit Members". Remove the former oncall, and add
   yourself.

2. Hand off newer conversations. For newer threads, the former oncall can unsubscribe from the
   thread, and the new oncall should subscribe. The former oncall should explicitly share each of
   these threads, and the new oncall can select "Get notified about new replies" in the "..." menu.
   The former oncall can select "Turn off notifications for replies" in that same menu. It can be
   helpful for the former oncall to remain available for any conversations they were deeply involved
   in, so use your judgement on which threads to hand off.

<meta name="maintainedBy" value="zwass">
