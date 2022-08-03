# Community

As an open-core company, Fleet endeavors to build a community of engaged users, customers, and
contributors.

## Communities

Fleet's users and broader audience are spread across many online platforms.  Here are the most active communities where Fleet's developer relations and social media team members participate at least once every weekday:

- [Osquery Slack](https://join.slack.com/t/osquery/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw) (`#fleet` channel)
- [MacAdmins Slack](https://www.macadmins.org/) (`#fleet` channel)
- [osquery discussions on LinkedIn](https://www.linkedin.com/search/results/all/?keywords=osquery)
- [osquery discussions on Twitter](https://twitter.com/search?q=osquery&src=typed_query)
- [reddit.com/r/sysadmins](https://www.reddit.com/r/sysadmin/)
- [reddit.com/r/SysAdminBlogs](https://www.reddit.com/r/SysAdminBlogs/)
- [r/sysadmin Discord](https://discord.gg/sysadmin)

### Goals
Our primary quality objectives are *customer service* and *defect reduction*. We try to optimize the following:

- Customer response time
- The number of bugs resolved per release cycle
- Staying abreast of what our community wants and the problems they're having
- Making people feel heard and understood
- Celebrating contributions
- Triaging bugs, identifying community feature requests, community pull requests, and community questions

### How?

- Folks who post a new comment in Slack or issue on GitHub should receive a response **within one business day**. The response doesn't need to include an immediate answer.
- If you feel confused by a question or comment raised, [request more details](#requesting-more-details) to better your understanding of the next steps.
- If an appropriate response is outside of your scope, please post to `#help-engineering` (in the Fleet Slack)), tagging `@oncall`.
- If things get heated, remember to stay [positive and helpful](https://canny.io/blog/moderating-user-comments-diplomatically/). If you aren't sure how best to respond positively, or if you see behavior that violates the Fleet code of conduct, get help.

### Requesting more details

Typically, the *questions*, *bug reports*, and *feature requests* raised by community members will be missing helpful context, recreation steps, or motivations.

❓ For questions that you don't immediately know the answer to, it's helpful to ask follow-up questions to receive additional context.

- Let's say a community member asks, "How do I do X in Fleet?" A follow-up question could be, "What are you attempting to accomplish by doing X?"
- This way, you have additional details when you bring this to the Roundup meeting. In addition, the community member receives a response and feels heard.

🦟 For bug reports, it's helpful to ask for re-creation steps so you're later able to verify the bug exists.

- Let's say a community member submits a bug report. An example follow-up question could be, "Can you please walk me through how you encountered this issue so that I can attempt to recreate it?"
- This way, you now have steps that verify whether the bug exists in Fleet or if the issue is specific to the community member's environment. If the latter, you now have additional information for further investigation and question-asking.

💡 For feature requests, it's helpful to ask follow-up questions to understand better the "Why?" or underlying motivation behind the request.

- Let's say a community member submits the feature request "I want the ability to do X in Fleet." A follow-up question could be "If you were able to do X in Fleet, what's the next action you would take?" or "Why do you want to do X in Fleet?."
- Both of these questions provide helpful context on the underlying motivation behind the feature request when brought to the Roundup meeting. In addition, the community member receives a response and feels heard.

### Closing issues

It is often good to let the original poster (OP) close their issue themselves since they are usually well equipped to decide to mark the issue as resolved. In some cases, circling back with the OP can be impractical, and for the sake of speed, issues might get closed.

Keep in mind that this can feel jarring to the OP. The effect is worse if issues are closed automatically by a bot (See [balderashy/sails#3423](https://github.com/balderdashy/sails/issues/3423#issuecomment-169751072) and [balderdashy/sails#4057](https://github.com/balderdashy/sails/issues/4057) for examples of this).

### Version support

To provide the most accurate and efficient support, Fleet will only target fixes based on the latest released version. In the current version fixes, Fleet will not backport to older releases.

Community version supported for bug fixes: **Latest version only**

Community support for support/troubleshooting: **Current major version**

Premium version supported for bug fixes: **Latest version only**

Premium support for support/troubleshooting: **All versions**

### Tools

Find the script in `scripts/oncall` for use during oncall rotation (only been tested on macOS and Linux).
Its use is optional but contains several useful commands for checking issues and PRs that may require attention.
You will need to install the following tools to use it:

- [GitHub CLI](https://cli.github.com/manual/installation)
- [JQ](https://stedolan.github.io/jq/download/)

### Resources

There are several locations in Fleet's public and internal documentation that can be helpful when answering questions raised by the community:

1. Find the frequently asked question (FAQ) documents in each section in the `/docs` folder. These documents are the [Using Fleet FAQ](./../../docs/Using-Fleet/FAQ.md), [Deploying FAQ](./../../docs/Deploying/FAQ.md), and [Contributing FAQ](./../../docs/Contributing/FAQ.md).

2. Use the [internal FAQ](https://docs.google.com/document/d/1I6pJ3vz0EE-qE13VmpE2G3gd5zA1m3bb_u8Q2G3Gmp0/edit#heading=h.ltavvjy511qv) document.

### Assistance from engineering

Community team members can reach the engineering oncall for assistance by writing a message with `@oncall` in the `#help-engineering` channel of the Fleet Slack.

## Pull requests

The most important thing when community members contribute to Fleet is to show them we value their time and effort. We need to get eyes on community pull requests quickly (within one business day) and get them merged or give feedback as soon as we can.

### Process for managing community contributions

The Community Engagement DRI is responsible for keeping an eye out for new community contributions, getting them merged if possible, and getting the right eyes on them if they require a review. 

Each business day, the Community Engagement DRI will check open pull requests to

1. check for new pull requests (PRs) from the Fleet community. 
2. approve and merge any community PRs that are ready to go.
3. make sure there aren't any existing community PRs waiting for a follow-up from Fleet. 

#### Identify community contributions

When a new pull request is submitted by a community contributor (someone not a member of the Fleet organization):

- Add the `:community` label.
- Self-assign for review.
- Check whether the PR can be merged or needs to be reviewed by the Product team.
    - Things that generally don't need additional review:
        - Minor changes to the docs.
        - Small bug fixes.

        - Additions or fixes to the Standard Query Library (as long as the SQL works properly and is attributed correctly).
    - If a review is needed:
        - Request a review from the [Product DRI](../people/README.md#directly-responsible-individuals). They should approve extensive changes and new features. Ask in the #g-product channel in Fleet's Slack for more information.
        - Tag the DRI and the contributor in a comment on the PR, letting everyone know why an additional review is needed. Make sure to say thanks!
        - Find any related open issues and make a note in the comments.

> Please refer to our [PRs from the community](https://docs.google.com/document/d/13r0vEhs9LOBdxWQWdZ8n5Ff9cyB3hQkTjI5OhcrHjVo/edit?usp=sharing) guide and use your best judgment. 

#### Communicate with contributors

Community contributions are fantastic, and it's important that the contributor knows how much they are appreciated. The best way to do that is to keep in touch while we're working on getting their PR approved.

While each team member is responsible for monitoring their active issues and pull requests, the Community Engagement DRI will check in on pull requests with the `:community ` label daily to make sure everything is moving along. If there's a comment or question from the contributor that hasn't been addressed, reach out on Slack to get more information and update the contributor. 

#### Merge Community PRs

When merging a pull request from a community contributor:

- Ensure that the checklist for the submitter is complete.
- Verify that all necessary reviews have been approved.
- Merge the PR.
- Thank and congratulate the contributor.
- Share the merged PR with the team in the #help-promote channel of Fleet Slack to be publicized on social media. Those who contribute to Fleet and are recognized for their contributions often become great champions for the project.

### Reviewing PRs from the community

If you're assigned a community pull request for review, it is important to keep things moving for the contributor. The goal is to not go more than one business day without following up with the contributor.

A PR should be merged if:

- It's a change that is needed and useful. 
- The CI is passing.
- Tests are in place.
- Documentation is updated.
- Changes file is created.

For PRs that aren't ready to merge:

- Thank the contributor for their hard work and explain why we can't merge the changes yet.
- Encourage the contributor to reach out in the #fleet channel of osquery Slack to get help from the rest of the community.
- Offer code review and coaching to help get the PR ready to go (see note below).
- Keep an eye out for any updates or responses.

> Sometimes (typically for Fleet customers), a Fleet team member may add tests and make any necessary changes to merge the PR. 

If everything is good to go, approve the review.

For PRs that will not be merged:

- Thank the contributor for their effort and explain why the changes won't be merged.
- Close the PR.

## Updating Docs and FAQ

When someone asks a question in a public channel, it's pretty safe to assume that they aren't the only person looking for an answer to the same question. To make our docs as helpful as possible, the Community team gathers these questions and uses them to make a weekly documentation update. 

Our goal is to answer every question with a link to the docs and/or result in a documentation update.

> **Remember**, when submitting any pull request that changes Markdown files in the docs, request an editor review from Desmi Dizney.

### Tracking

When responding to a question or issue in the [#fleet](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/) channel of the osquery Slack workspace, push the thread to Zapier using the `TODO: Update docs` Zap. This will add information about the thread to the [Slack Questions Spreadsheet](https://docs.google.com/spreadsheets/d/15AgmjlnV4oRW5m94N5q7DjeBBix8MANV9XLWRktMDGE/edit#gid=336721544). In the `Notes` field, you can include any information that you think will be helpful when making weekly doc updates. That may be something like

- proposed change to the documentation.
- documentation link that was sent as a response.
- link to associated thread in [#help-oncall](https://fleetdm.slack.com/archives/C024DGVCABZ).

### Making the updates

Every week, the Community Engagement DRI will:

- Create a new `Weekly Doc Update` issue on Monday and add it to the [Community board](https://github.com/orgs/fleetdm/projects/36).
- Review the Slack Questions Spreadsheet and make sure that any necessary updates to the documentation are made. 
    - Update the spreadsheet to indicate what action was taken (Doc change, FAQ added, or None) and add notes if need be. 
- Set up a single PR to update the Docs. 
    - In the notes, include a list of changes made as well as a link to the related thread. 
- Bring any questions to DevRel Office Hours (time TBD).
- Submit the PR by the end of the day on Thursday. 
- Once the PR is approved, share in the [#fleet channel](https://osquery.slack.com/archives/C01DXJL16D8) of Osquery Slack Workspace and thank the community for being involved and asking questions. 

## Fleet swag

We want to recognize and congratulate community members for their contributions to Fleet. Nominating a contributor for Fleet swag is a great way to show our appreciation.

### How to order swag

1. Reach out to the contributor to thank them for their contribution and ask if they would like any swag.

2. Fill out our [swag request sheet](https://docs.google.com/spreadsheets/d/1bySsYVYHY8EjxWhhAKMLVAPLNjg3IYVNpyg50clfB6I/edit?usp=sharing).

3. Once approved, place the order through our Printful account (credentials in 1Password).

4. If available through the ordering process, add a thank you note for their contribution and "Feel free to tag us on Twitter."

## Rituals

The following table lists the Community group's rituals, frequency, and Directly Responsible Individual (DRI).

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| Community Slack  | Daily   | Check Fleet and osquery Slack channels for community questions, make sure questions are responded to and logged. | Kathy Satterlee |
| Social media check-in |  Daily | Check social media for community questions and make sure to respond to them. Generate dev advocacy-related content. | Kelvin Omereshone   |
| Issue check-in | Daily | Check GitHub for new issues submitted by the community, check the status of existing requests, and follow up when needed. | Kathy Satterlee |
| Outside contributor follow-up | Weekly | Bring pull requests from outside contributors to engineering and make sure they are merged promptly and promoted. | Kathy Satterlee |
| Documentation update | Weekly | Turn questions answered from Fleet and osquery Slack into FAQs in Fleet’s docs. | Kathy Satterlee |
| StackOverflow  | Weekly | Search StackOverflow for “osquery,” answer questions with Grammarly, and find a way to feature Fleet in your StackOverflow profile prominently. | Rotation: Community team |

## Slack channels

This group maintains the following [Slack channels](https://fleetdm.com/handbook/company#group-slack-channels):

| Slack channel               | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:----------------------------|:--------------------------------------------------------------------|
| `#g-community`              | Kathy Satterlee

<meta name="maintainedBy" value="ksatter">
<meta name="title" value="🪂 Community">