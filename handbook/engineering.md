## Release process

This section outlines the release process at Fleet.

The current release cadence is once every 3 weeks and concentrated around Wednesdays. 

- [Blog post](#release-day)

### Blog post

Fleet posts a release blogpost, to the [Fleet blog](https://blog.fleetdm.com/ ), on the same day a new minor or major release goes out.

Patch releases do not have a release blogpost.

Check out the [Fleet 4.1.0 blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1) for an example release blogpost. The suggested format of a release blogpost is the following:

**Title** - "Fleet `<insert Fleet version here>`

**Description** - "Fleet `<insert Fleet version here>` released with `<insert list of primary features here>`

**Main image** - This is the image that Medium will use as the thumbnail and link preview for the blogpost.

**Summary** - This section includes 3-4 sentences that answers the 'what?' and 'why should the user care?' questions for the primary features.

**Link to release notes** - One sentence that includes a link to the GitHub release page.

**Primary features** - Includes the each primary feature's name, availability (Free v. Premium), image/gif, and 3-4 sentences that answer the 'why should the user care?' and 'how do I find this feature?' questions.

**More improvements** - Includes each additional feature's name, availability (Free v. Premium), and 1-2 sentences that answer the 'why should the user care?' questions.

**Upgrade plan** - Once sentence that links to user to the upgrading Fleet documentation here: https://github.com/fleetdm/fleet/blob/main/docs/01-Using-Fleet/08-Updating-Fleet.md

#### Manual QA

After all changes required for release have been merged into the `main` branch, the individual tasked with managing the release should perform smoke tests. Manual smoke tests should be generated for a release via the [Release QA ticket template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=&template=smoke-tests.md&title=) and assigned to the person responsible. 

Further ocumentation on conducting the manual QA pass can be found [here](#manual-qa). 

#### Release freeze period

In order to ensure quality releases, Fleet has a freeze period for testing prior to each release. Effective at the start of the freeze period, new feature work will not be merged. 

Release blocking bugs are exempt from the freeze period and are defined by the same rules as patch releases, which include:
1. Regressions
2. Security concerns
3. Issues with features targeted for current release

Non-release blocking bugs may include known issues that were not targeted for the current release, or newly documented behaviors that reproduce in older stable versions. These may be addressed during a release period by mutual agreement between Product and Engineering teams. 


### Release day

Documentation on completing the release process can be found [here](../docs/03-Contributing/05-Releasing-Fleet.md).  

### Goals
At Fleet, our primary quality objectives are *customer service* and *defect reduction*. This entails Key Performance Indicators such as customer response time and number of bugs resolved per cycle and. 

- Get familiar with and stay abreast of what our community wants and the problems they're having.

- Make people feel heard and understood.  

- Celebrate contributions. 

- Triage bugs, identify community feature requests, community pull requests and community questions.

### Version support

In order to provide the most accurate and efficient support, Fleet will only target fixes based on the latest released version. Fixes in current versions will not be backported to older releases.

Community version supported for bug fixes: **Latest version only**
 
Community support for support/troubleshooting: **Current major version**

Premium version supported for bug fixes: **Latest version only**

Premium support for support/troubleshooting: **All versions**

### How?

- No matter what, folks who post a new comment in Slack or issue in GitHub get a **response** from the on-call engineer within 1 business day. The response doesn't need to include an immediate answer.

- The on-call engineer can discuss any items that require assistance at the end of the daily standup. They are also requested to attend the "Customer experience standup" where they can bring questions and stay abreast of what's happening with our customers.

- If you do not understand the question or comment raised, [request more details](#requesting-more-details) to best understand the next steps.

- If an appropriate response is outside your scope, please post to `#help-oncall`, a confidential Slack channel in the Fleet Slack workspace.

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

#### Feature requests

If the feature is requested by a customer, the on-call engineer is requested to create a feature request issue and follow up with the customer by linking them to this issue. This way, the customer can add additional comments or feedback to the newly filed feature request issue.

If the feature is requested by anyone other than a customer (ex. user in #fleet Slack), the on-call engineer is requested to point to the user to the [feature request GitHub issue template](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title=) and kindly ask the user to create a feature request.

#### Closing issues

It is often a good idea to let the original poster (OP) close their issue themselves, since they are usually well equipped to decide whether the issue is resolved.   In some cases, circling back with the OP can be impractical, and for the sake of speed issues might get closed.

Keep in mind that this can feel jarring to the OP.  The effect is worse if issues are closed automatically by a bot (See [balderashy/sails#3423](https://github.com/balderdashy/sails/issues/3423#issuecomment-169751072) and [balderdashy/sails#4057](https://github.com/balderdashy/sails/issues/4057) for examples of this.)

To provide another way of tracking status without closing issues altogether, consider using the green labels that begin with "+".  To explore them, type `+` from GitHub's label picker.


### Sources

There are four sources that the on-call engineer should monitor for activity:

1. Customer Slack channels - Found under the "Connections" section in Slack. These channels are usually titled "at-insert-customer-name-here"

2. Community chatroom - https://osquery.slack.com, #fleet channel

3. Reported bugs - [GitHub issues with the "bug" and ":reproduce" label](https://github.com/fleetdm/fleet/issues?q=is%3Aopen+is%3Aissue+label%3Abug+label%3A%3Areproduce). Please remove the ":reproduce" label after you've followed up in the issue.

4. Pull requests opened by the community - [GitHub open pull requests](https://github.com/fleetdm/fleet/pulls?q=is%3Aopen+is%3Apr)

### Resources

There are several locations in Fleet's public and internal documentation that can be helpful when answering questions raised by the community:

1. The frequently asked question (FAQ) documents in each section found in the `/docs` folder. These documents are the [Using Fleet FAQ](../docs/01-Using-Fleet/FAQ.md), [Deploying FAQ](../docs/02-Deploying/FAQ.md), and [Contributing FAQ](../docs/03-Contributing/FAQ.md).

2. The [Internal FAQ](https://docs.google.com/document/d/1I6pJ3vz0EE-qE13VmpE2G3gd5zA1m3bb_u8Q2G3Gmp0/edit#heading=h.ltavvjy511qv) document.

<meta name="maintainedBy" value="zwass">

