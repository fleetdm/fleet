# Product Design

This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                             | Contributor(s)           |
|:---------------------------------|:-----------------------------------------------------------------------------------------------------------|
| Head of Product Design           | [Noah Talerman](https://www.linkedin.com/in/noah-talerman/) _([@noahtalerman](https://github.com/noahtalerman))_
| Product Designer                 | <sup><sub>_See [🛩️ Product groups](https://fleetdm.com/handbook/company/product-groups#current-product-groups)_ </sup></sub>


## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?labels=%3Aproduct&title=Product%20design%20request%C2%BB______________________&template=custom-request.md) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in `#help-design`.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/-g-digital-experience-6451748b4eb15200131d4bab/board) for this department, including pending tasks and the status of new requests.


## Responsibilities

The Product Design department is responsible for reviewing and collecting feedback from users, would-be users, and future users, prioritizing changes, designing the changes, and delivering these changes to the engineering team. Product Design prioritizes and shapes all changes involving functionality or usage, including the UI, REST API, command line, and webhooks. 


### Unpacking the why

The Head of Product Design and a former IT admin review the new customer/prospect/community requests in the "Inbox" column the [drafting board](https://github.com/fleetdm/fleet/issues#workspaces/drafting-6192dd66ea2562000faea25c/board) to synthesize why users are making the request (i.e. what problem are they trying to solve).

If a customer/prospect request is missing a Gong snippet or requires additional information to understand the "why", the Head of Product Design will @ mention the relevant Customer Success Manager (CSM), assign them, and move the request to the [🏹 #g-customer-success](https://github.com/fleetdm/fleet/issues#workspaces/g-customer-success-642c83a53e96760014c978bd/board) board.


### Unpacking the how

3 weeks before end of each quarter, The Head of Product Design starts a daily 1h meeting with the CEO. The Head of Product Design brings an objective for the next quarter and the appropriate subject matter expert to understand what users will expect. This helps Product Designers at Fleet understand how Fleet will design a particular feature. 

As soon as we've addressed the next quarter's objectives, the Head of Product Design cancels the daily meeting. 


### Product design check in

The Head of Product Design summarizes the current week's design reviews to discuss with the CEO.


### Drafting

At Fleet, like [GitLab](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall) and [other organizations](https://speakerdeck.com/mikermcneil/i-love-apis), every change to the product's UI gets [wireframed first](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

1. Take the top user story that is assigned to you in the "Ready" column of the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board) and move it to "In progress."
  
2. Create a new file inside the [Fleet product](https://www.figma.com/files/project/17318630/%F0%9F%94%9C%F0%9F%93%A6-Fleet-EE%C2%AE-(product)?fuid=1234929285759903870) Figma project by duplicating "\[TEMPLATE\] Starter file" (pinned to the top of the project). The starter file includes three predefined pages: "Cover," "Ready," and "Scratchpad."
   -  **Cover**: This page has a component with issue number, issue name, and status fields. There are three statuses: "Work In Progress (WIP)," "Approved," and "Released" (the drafting board is still the source of truth).
   -  **Ready**: Use this page to communicate design reviews and development.
   -  **Scratchpad**: Use this page to keep "work in progress" designs that might be useful in the future.

3. Add page names (e.g. "Host details" page) to the user story's title and/or description to help contributors find Figma wireframes for the area of the UI you're making changes to.

4. If the story requires API or YAML file changes, open a pull request (PR) to the reference docs with the proposed design. Pay attention to existing conventions (URL structure, parameter names, response format) and aim to be consistent. Your PR should follow these guidelines:
  - Make a PR against the docs release branch for the version you expect this feature to be in. Docs release branches are named using the format `docs-vX.X.X`, so if you're designing for Fleet 4.61.0, you would make a PR to `docs-v4.61.0`.
  - Add a link to the issue in the PR description.
  - Attach the `~api-or-yaml-design` label.
  - Mark the PR ready for review. (Draft PRs do not auto-request reviews.)
  - After your changes are approved by the API design DRI, they will merge your changes into the docs release branch.

5. Add links to the user story as specified in the [issue template](https://github.com/fleetdm/fleet/issues/new?template=story.md).

6. Draft changes to the Fleet product that solve the problem specified in the story.
- Constantly place yourself in the shoes of a user while drafting changes.
- Use dev notes (component available in our library) to highlight important information to engineers and other teammates. - Reach out to sales, customer success, and demand for a business perspective.
- Engage engineering to gain insight into technical costs and feasibility.

Additionally:

- To make changes to the design system or a component (e.g. button border-radius or modal width), [make a new request](#making-a-request) and attach the `:improve design system` label.

- If the story has a requester and the title and/or description change during drafting (scope change), notify the requester. The customer DRI should confirm that the updated scope still meets the requester's needs.

- Each [product group](https://fleetdm.com/handbook/company/product-groups#current-product-groups) stops drafting once they reach engineering capacity for the upcoming engineering sprint. This way, we avoid creating a backlog which causes us to spend time updating soon-to-be stale designs. It's up to the product group's Product Designer to stop drafting and shift their focus to the following tasks:
  - Run back through the unestimated user stories and do extra iterations to make sure they're as good as we think they are
  - Go through the Fleet UI and look for bad/inconsistent text
  - Go through bugs to see if there’s Product Design input needed
  - File stories and draft changes for making form fields in the Fleet UI consistent (fixing conventions, moving out the tooltips, etc.)
  - File stories and draft changes for bringing the screen width down to 375px. (one screen at a time, in which we can squeeze it into engineering sprints as front-end only work, small stories, doesn't compete with other stuff)

>**Questions, missing information, and notes:** Take a screenshot of the area in Figma and add a comment in the story's GitHub issue. Figma does have a commenting system, but it is not easy to search for outstanding concerns and is therefore not preferred. Also, commenting in Figma, sends all contributors email notifications.
>
>For external contributors: please consider opening an issue with reference screenshots if you have a Figma related question you need to resolve.

### Prepare for design review

1. Link to your draft in the user story issue.
2. Add the user story to the agenda for the [design review](https://fleetdm.com/handbook/company/product-groups#design-reviews) meeting.
3. Attend design review or schedule an ad-hoc design review if you need to move faster.

> As drafting occurs, inevitably, the requirements will change. The main description of the issue should be the single source of truth for the problem to be solved and the required outcome. The product manager is responsible for keeping the main description of the issue up-to-date. Comments and other items can and should be kept in the issue for historical record-keeping.


### Ensure story drafting is complete

Once a story is approved in [design review](https://fleetdm.com/handbook/company/product-groups#design-reviews), the Product Designer is responsible for moving the user story to the "Ready to spec" column, assigning the appropriate Engineering Manager (EM), adding a product group label, and changing the status on the cover page of the relevant Figma file to "Approved".

The EM is responsible for moving the user story to the "Specified" and "Estimated" columns.

Before assigning an EM, double-check that the "Product" section of the user story [checklist](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=story&projects=&template=story.md&title=) is complete (no TODOs). 

If the story is tied to a customer feature request, the Head of Product Design (HPD) is responsible for adding the feature request issue to the [🏹 #g-customer-success board](https://github.com/fleetdm/fleet/issues#workspaces/g-customer-success-642c83a53e96760014c978bd/board). This way the Customer Success Manager (CSM) can review the wireframes and provide feedback on whether the proposed changes solve the customer's problem. If the changes don't, it's up to the HPD to decide whether to bring the user story back for more drafting or file a follow up user story (iteration).

Once a bug is approved in design review, The Product Designer is responsible for moving the bug to the appropriate release board.


### Revise a draft currently in development

Expedited drafting is the revision of drafted changes currently being developed by
the engineering team. Expedited drafting aims to quickly adapt to unknown edge cases and
changing specifications while ensuring that Fleet meets our brand and quality guidelines. 

You'll know it's time for expedited drafting when:
- The team discovers that a drafted user story is missing crucial information that prevents contributors from continuing the development task.
- A user story is taking more effort than was originally estimated, and Product Manager wants to find ways to cut aspects of planned functionality in order to still ship the improvement in the currently scheduled release.
- A user story on the drafting board won't reach "Ready for spec" by the last estimation session in the current sprint and cannot wait until the next sprint. This can also happen when we decide to bring a user story in mid-sprint.

What happens during expedited drafting?
1. If the story has a requester, notify the requester. The customer DRI should confirm that the updated scope still meets the requester's need.
2. If the user story wasn't "Ready for spec" by the last estimation session, the product group's engineering manager (EM), [release DRI](https://fleetdm.com/handbook/company/communications#directly-responsible-individuals-dris), and Head of Product Design are notified in the `#g-mdm` or `#g-endpoint-ops` Slack channel. Decision to allow the user story to make it into the sprint is up to the release DRI.
3. If the user story is already in the sprint, the EM, release DRI, and Head of Product Design are notified in the `#g-mdm` or `#g-endpoint-ops` channel. If there are significant changes to the requirements, then the user story might be pushed to the next sprint. Decision is up to the release DRI.
4. If the release DRI decides the user story will be worked on this sprint, drafts are updated or finished.
5. UI changes [are approved](https://fleetdm.com/handbook/company/development-groups#drafting-process), and the UI changes are brought back into the sprint or are estimated.


### Write a user story

Product Managers [write user stories](https://fleetdm.com/handbook/company/product-groups#writing-a-good-user-story) in the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board). The drafting board is shared by every [product group](https://fleetdm.com/handbook/company/development-groups).


### Consider a feature eligible to be flagged

At Fleet, features are placed behind feature flags if the changes could affect Fleet's availability of existing functionalities. The following highlights should be considered when deciding if we should leverage feature flags:

- The feature flag must be disabled by default.
- The feature flag will not be permanent. This means that the Directly Responsible Individual
 (DRI) who decides a feature flag should be introduced is also responsible for creating an issue to track the
  feature's progress towards removing the feature flag and including the feature in a stable
  release.
- The feature flag will not be advertised. For example, advertising in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

> Fleet's feature flag guidelines is borrowed from GitLab's ["When to use feature flags" section](https://about.gitlab.com/handbook/product-development-flow/feature-flag-lifecycle/#when-to-use-feature-flags) of their handbook. Check out [GitLab's "Feature flags only when needed" video](https://www.youtube.com/watch?v=DQaGqyolOd8) for an explanation of the costs of introducing feature flags.


### View Fleet usage statistics

In order to understand the usage of the Fleet product, we [collect statistics](https://fleetdm.com/docs/using-fleet/usage-statistics) from installations where this functionality is enabled.

Fleeties can view these statistics in the Google spreadsheet [Fleet
usage](https://docs.google.com/spreadsheets/d/1Mh7Vf4kJL8b5TWlHxcX7mYwaakZMg_ZGNLY3kl1VI-c/edit#gid=0)
available in Google Drive.

Some of the data is forwarded to [Datadog](https://us5.datadoghq.com/dashboard/7pb-63g-xty/usage-statistics?from_ts=1682952132131&to_ts=1685630532131&live=true) and is available to Fleeties.


### Prepare reference docs for release

Every change to how Fleet is used is reflected live on the website in reference documentation **at release day** (REST API, config surface, tables, and other already-existing docs under /docs/using-fleet).

To make sure this happens, first, the [DRI for what goes in a release](https://fleetdm.com/handbook/company/communications#directly-responsible-individuals-dris) @ mentions the [API design DRI](https://fleetdm.com/handbook/company/communications#directly-responsible-individuals-dris) in a message in [#help-engineering Slack channel](https://fleetdm.slack.com/archives/C019WG4GH0A) when we cut the release candidate (RC). 

Next, the API design DRI reviews all user stories and bugs with the release milestone to check that all reference doc PRs are merged into the reference docs release branch. To see which stories were pushed to the next release, and thus which reference doc changes need to be removed from the branch, the API design DRI filters issues by the `~pushed` label and the next release's milestone.

To signal that the reference docs branch is ready for release, the API design DRI opens a PR to `main`, adds the DRI for what goes in a release as the reviewer, and adds the release milestone.

> Anytime there is a missing or incorrect configuration option or REST API endpoint in the docs, it is treated as a released bug to be filed and fixed ASAP.

### Interview a Product Designer candidate

Ensure the interview process follows these steps in order. This process must follow [creating a new position](https://fleetdm.com/handbook/company/leadership#creating-a-new-position) through [receiving job applications](https://fleetdm.com/handbook/company/leadership#receiving-job-applications).

1. **Reach out**: Send an email or LinkedIn message introducing yourself. Include the URL for the position, your Calendly URL, and invite the candidate to schedule a 30 minute introduction call.
2. **Conduct screening call**: Discuss the requirements of the position with the candidate, and answer any questions they have about Fleet. Look for alignment with [Fleet's values](https://fleetdm.com/handbook/company#values) and technical expertise necessary to meet the requirements of the role.
2. **Deliver design challenge**: Share the [design challenge](https://docs.google.com/document/d/1S4fD5fPUU9YUjlKy2YAbRZPb_IK4EPkmmO7j09iPWR8/edit) and ask them to complete and send their project back within 5 business days.
5. **Schedule design challenge interview**: Send the candidate a Calendly link for 1 hour call to review the candidate's project. The goal is to understand the design capabilities of the candidate. An additional Product Designer can optionally join if available.
6. **Schedule EM interview**: Send the candidate a calendly link for 30m talk with the Engineering Manager (EM) of the [product group](https://fleetdm.com/handbook/company/product-groups#current-product-groups) the candidate will be working with.
7. **Schedule CTO interview**: Send the candidate a calendly link for 30m talk with our CTO @lukeheath.

If the candidate passes all of these steps then continue with [hiring a new team member](https://fleetdm.com/handbook/company/leadership#hiring-a-new-team-member).


## Rituals
<rituals :rituals="rituals['handbook/product-design/product-design.rituals.yml']"></rituals>


#### Stubs
The following stubs are included only to make links backward compatible.

##### Maintenance
Please see [handbook/product-design#rituals](https://fleetdm.com/handbook/product-design#rituals)

##### New CIS benchmarks
Please see [handbook/product#submit-a-new-cis-benchmark-set-for-certification](https://fleetdm.com/handbook/product#submit-a-new-cis-benchmark-set-for-certification)

##### Usage statistics
Please see [handbook/product#view-fleet-usage-statistics](https://fleetdm.com/handbook/product#view-fleet-usage-statistics)

Please see [handbook/product#create-a-new-figma-file](https://fleetdm.com/handbook/product#create-a-new-figma-file) for **below**
##### Create a new file
##### Wireframing
Please see [handbook/product#create-a-new-figma-file](https://fleetdm.com/handbook/product#create-a-new-figma-file) for **above**

##### Competition
Please see [handbook/company/communications#competition](https://fleetdm.com/handbook/company/communications#competition)

##### Breaking changes
Please see [handbook/company/product-groups#breaking-changes](https://fleetdm.com/handbook/company/product-groups#breaking-changes)

##### Making changes to the product
Please see [handbook/product#responsibilities](https://fleetdm.com/handbook/product#responsibilities)

Please see [handbook/product#release-relevant-figma-files](https://fleetdm.com/handbook/product#release-relevant-figma-files) for **below**
##### Working with Figma
##### Keep projects/files clean and up-to-date
##### Questions and missing information
Please see [handbook/product#release-relevant-figma-files](https://fleetdm.com/handbook/product#release-relevant-figma-files) for **above**


##### Scheduling design reviews
Please see [handbook/product#schedule-a-design-review](https://fleetdm.com/handbook/product#schedule-a-design-review)

##### Settled 
Please see [handbook/product#ensure-product-user-story-is-complete](https://fleetdm.com/handbook/product#ensure-product-user-story-is-complete)

##### Expedited drafting
Please see [handbook/product#revise-a-draft-currently-in-development](https://fleetdm.com/handbook/product#revise-a-draft-currently-in-development)

##### Outside contributions
Please see [handbook/product#outside-contributions](https://fleetdm.com/handbook/product#outside-contributions)

##### Prioritizing bugs
Please see [handbook/product#correctly-prioritize-a-bug](https://fleetdm.com/handbook/product#correctly-prioritize-a-bug)

##### Writing user stories
Please see [handbook/product#write-a-user-story](https://fleetdm.com/handbook/product#write-a-user-story)

##### Drafting user stories
Please see [handbook/product#draft-a-user-story](https://fleetdm.com/handbook/product#draft-a-user-story)

##### Estimating user stories
Please see [handbook/product#estimate-a-user-story](https://fleetdm.com/handbook/product#estimate-a-user-story)

##### Sprints
Please see [handbook/company/product-groups#sprints](https://fleetdm.com/handbook/company/product-groups#sprints)

##### Sprint numbering
Please see [handbook/company/product-groups#sprint-numbering](https://fleetdm.com/handbook/company/product-groups#sprint-numbering)

##### Product design conventions
Please see [handbook/company/product-groups#product-design-conventions](https://fleetdm.com/handbook/company/product-groups#product-design-conventions)

##### Wireframes 
Please see [handbook/company/product-groups#wireframes](https://fleetdm.com/handbook/company/product-groups#wireframes)

Please see [handbook/product#rank-features-before-release](https://fleetdm.com/handbook/product#rank-features-before-release) for **below**
##### Release 
##### Ranking features
Please see [handbook/product#rank-features-before-release](https://fleetdm.com/handbook/product#rank-features-before-release) for **above**

##### Blog post
Please see [handbook/product#create-release-issue](https://fleetdm.com/handbook/product#create-release-issue)

##### Feature flags
Please see [handbook/product#consider-a-feature-eligible-to-be-flagged](https://fleetdm.com/handbook/product#consider-a-feature-eligible-to-be-flagged)

##### Feature fest
Please see [handbook/product-groups#feature-fest](https://fleetdm.com/handbook/product-groups#feature-fest)

##### Making a request
Please see [handbook/product-groups#making-a-request](https://fleetdm.com/handbook/product-groups#making-a-request)

Please see [handbook/product-groups#how-feature-requests-are-evaluated](https://fleetdm.com/handbook/product-groups#how-feature-requests-are-evaluated)
##### How feature requests are evaluated
##### Prioritizing improvements
Please see [handbook/product-groups#how-feature-requests-are-evaluated](https://fleetdm.com/handbook/product-groups#how-feature-requests-are-evaluated)

##### Customer feature requests 
Please see [handbook/product-groups#customer-feature-requests](https://fleetdm.com/handbook/product-groups#customer-feature-requests)

##### After the feature is accepted
Please see [handbook/product-groups#after-the-feature-is-accepted](https://fleetdm.com/handbook/product-groups#after-the-feature-is-accepted)

##### Restart Algolia manually
Please see [handbook/digital-experience#restart-algolia-manually](https://fleetdm.com/handbook/digital-experience#restart-algolia-manually)

##### Schedule a design review
Please see [handbook/product#prepare-for-design-review](https://fleetdm.com/handbook/product#prepare-for-design-review)

##### Create a new Figma file
Please see [handbook/product#drafting](https://fleetdm.com/handbook/product#drafting)

<meta name="maintainedBy" value="noahtalerman">
<meta name="title" value="🦢 Product design">
