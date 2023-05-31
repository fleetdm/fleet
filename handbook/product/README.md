# Product

Contributors in Fleet's product department [prioritize](#prioritizing-improvements) and [define](https://fleetdm.com/handbook/company/product-groups#drafting) the [changes we make to the product](https://fleetdm.com/handbook/company/product-groups#making-changes).

Changes begin as [ideas](#intake) or [code](#outside-contributions) that can be contributed by anyone.

> You can read what's coming in the next 3-6 weeks in Fleet's [⚗️ Drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).


## Product design

The product team is responsible for product design tasks like drafting [changes to the Fleet product](https://fleetdm.com/handbook/company/development-groups#making-changes), reviewing and collecting feedback from engineering, sales, customer success, and marketing counterparts, and delivering these changes to the engineering team. 

> Learn more about Fleet's philosophy and process for [making iterative changes to the product](https://fleetdm.com/handbook/company/development-groups#making-changes), or [why we use a wireframe-first approach](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

### Wireframing

At Fleet, like [GitLab](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall) and [other organizations](https://speakerdeck.com/mikermcneil/i-love-apis), every change to the product's UI gets [wireframed first](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

* Take the top issue that is assigned to you in the "Prioritized" column of the drafting board.

* Create a page in the [Fleet EE (scratchpad, dev-ready) Figma file](https://www.figma.com/file/hdALBDsrti77QuDNSzLdkx/%F0%9F%9A%A7-Fleet-EE-dev-ready%2C-scratchpad?node-id=3923%3A208793) and combine your issue's number and
  title to name the Figma page.

* Draft changes to the Fleet product that solve the problem specified in the issue. Constantly place
  yourself in the shoes of a user while drafting changes. Place these drafts in the appropriate
  Figma page in Fleet EE (scratchpad, dev-ready).

* While drafting, reach out to sales, customer success, and marketing for a business perspective.

* While drafting, engage engineering to gain insight into technical costs and feasibility.

### Scheduling design reviews

- Prepare your draft in the user story issue.
- Prepare the agenda for your design review meeting, which should be an empty document other than the proposed changes you will present.
- Review the draft with the CEO at one of the daily design review meetings, or schedule an ad-hoc design review if you need to move faster.  (Efficient access to design reviews on-demand [is a priority for Fleet's CEO](https://fleetdm.com/handbook/business-operations/ceo-handbook). Emphasizing design helps us live our [empathy](https://fleetdm.com/handbook/company#empathy) value.)
- During the review meeting, take detailed notes of any feedback on the draft.
- Address the feedback by modifying your draft.
- Rinse and repeat at subsequent sessions until there is no more feedback.

> As drafting occurs, inevitably, the requirements will change. The main description of the issue should be the single source of truth for the problem to be solved and the required outcome. The product manager is responsible for keeping the main description of the issue up-to-date. Comments and other items can and should be kept in the issue for historical record-keeping.

#### Estimating

Once the draft has been approved: 
* move it to the "Designed" column in the drafting board
* make sure that the issue is updated with the latest information on the work to be done, such as link to the correct page in the Fleet EE (scratchpad) Figma and most recent requirements.

Learn https://fleetdm.com/handbook/company/development-groups#making-changes

### Emergency drafting

> TODO: extrapolate this content to 

Emergency drafting is the revision of drafted changes currently being developed by
the engineering team. Emergency drafting aims to quickly adapt to unknown edge cases and
changing specifications while ensuring that Fleet meets our brand and quality guidelines. 

You'll know it's time for emergency drafting when:
- The team discovers that a drafted user story is missing crucial information that prevents contributors from continuing the development task.
- A user story is taking more effort than was originally estimated, and Product Manager wants to find ways to cut aspects of planned functionality in order to still ship the improvement in the currently scheduled release.

What happens during emergency drafting?
1. Everyone on the product and engineering teams know that a drafted change was brought back
   to drafting and prioritized.
2. Drafts are updated to cover edge cases or reduce functionality.
3. UI changes [are approved](https://fleetdm.com/handbook/company/development-groups#drafting-process), and the UI changes are brought back to the engineering team to continue the development task.

## Outside contributions

[Anyone can contribute](https://fleetdm.com/handbook/company#openness) at Fleet, from inside or outside the company.  Since contributors from the wider community don't receive a paycheck from Fleet, they work on whatever they want.

Many open source contributions that start as a small, seemingly innocuous pull request come with lots of additional [unplanned work](https://fleetdm.com/handbook/company/development-groups#planned-and-unplanned-changes) down the road: unforseen side effects, documentation, testing, potential breaking changes, database migrations, [and more](https://fleetdm.com/handbook/company/development-groups#defining-done).

Thus, it is still important to ensure consistency, completeness, and secure development practices, no matter where a contribution comes from:
- Prior to merging any change, small or large, that would change the expected behavior of the product, [prioritized](#prioritizing-improvements) by the [appropriate product group's](https://fleetdm.com/handbook/company/development-groups#current-product-groups) Product Manager and [drafted](https://fleetdm.com/handbook/company/development-groups#drafting) by the group's Product Designer prior to merging. 
- All changes to the user interface should be [wireframed first](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) by the appropriate Product Designer.

## Prioritizing improvements
Product Managers prioritize all potential product improvements worked on by contributors inside the company.

Bugs are always prioritized.  (Fleet takes quality and stability [very seriously](https://fleetdm.com/handbook/company/why-this-way#why-spend-so-much-energy-responding-to-every-potential-production-incident).)

If a bug is unreleased or [critical](https://fleetdm.com/handbook/engineering#critical-bugs), it is addressed in the current sprint. Otherwise, it may be prioritized in the sprint backlog for the next sprint. Bugs are never carried more than one sprint.

> Anyone can [suggest improvements](#intake).

## Writing user stories
Product Managers [write user stories](https://fleetdm.com/handbook/company/development-groups#writing-a-good-user-story) in the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).  The drafting board is shared by every [product group](https://fleetdm.com/handbook/company/development-groups).

## Drafting user stories
Product Designers [draft user stories](https://fleetdm.com/handbook/company/development-groups#drafting).

## Estimating user stories
Engineering Managers estimate user stories.  They are responsible for delivering planned work in the current sprint (0-3 weeks) while quickly getting user stories estimated for the next sprint (3-6 weeks).  Only work that is slated to be released into the hands of users within ≤six weeks will be estimated. Estimation is run by each group's Engineering Manager and occurs on the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).

## Sprints
Sprints (aka "iterations") align with Fleet's [3-week release cycle](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence).

On the first day of each release, all estimated issues are moved into the relevant section of the new "Release" board, which has a kanban view per group. 

Sprints are managed in [Zenhub](https://fleetdm.com/handbook/company/why-this-way#why-make-work-visible).  To plan capacity for a sprint, [create a "Sprint" issue](https://github.com/fleetdm/confidential/issues/new/choose), replace the fake constants with real numbers, and attach the appropriate labels for your product group.

### Sprint numbering
Sprint 1 began at the beginning of January 2023.  Sprint 4 began in late March 2023.  And so forth.

### Leftovers
Improvements are prioritized prior to estimation.  But sometimes, estimations will affect the calculus of what to include in an upcoming release.  Improvements that do not not "fit" into the capacity of the next scheduled release are left at the very top of the "Estimated" column of the drafting board.  The Product Manager always either includes these "leftovers" in the _next_ release (3-6 weeks) or deprioritizes and closes them.


<!-- 

----
TODO: Revisit.  I noticed on 2023-03-22 there are old references to no longer relevant product groups in here.  Rather than documenting incorrect things, I commented it out.  Some of this writing is not captured elsewhere though.  We should consider extrapolating a lot of this from eng and product handbooks into the "product groups" page, to avoid duplication and out of date content.  -mike
----

### Process

1. **Intake:** Product has a "time til estimated" timeframe, which measures the time from when an idea is first received until it is written up as an estimated issue and the requestor is notified exactly which aspects are scheduled for release. How intake works, and the estimation timeframe, vary per group, but every group has an estimation timeframe.

2. **Estimation:** The estimation process consists of drafting, API design, and either planning poker or a quick timebox decided by the group EM. When the Interface group relies on the Platform group for part of an issue, only the Interface group's work is estimated. It is up to the Interface PM to obtain estimated Platform issues for any needed work and thus make sure it is scheduled in the appropriate release. It is up to the Platform PM to get those specced (in consultation with Engineering), then up to the Engineering to estimate and communicate promptly if issues arise. We avoid having more estimated issues than capacity in the next release. If the team is fully allocated, no more issues will be estimated, or the PM will decide whether to swap anything out. Once estimated, an issue is scheduled for release. 

3. **Development:** Development starts on the first day of the new release. Only estimated issues are scheduled for release.

4. **Quality assurance (QA):** Everyone in each group is responsible for quality: engineers, PM, and the EM. The QA process varies per group and is set by the group's PM. For example, in the Interface group, every issue is QA'd (i.e. a per-change basis), as well as a holistic "smoke test" during the last few days of each release.

5. **Release:** Release dates are time-based and happen even if all features are not complete (± a day or two sometimes, if there's an emergency. Either way, the next release cycle starts on time). If anything is not finished, or can only be finished with changes, the PM finds out immediately and notifies the requestor right away.

### Timeframes

These are effectively internal SLAs. We moved away from the term "SLA" to avoid potential confusion with future, contractual Service Level Agreements Fleet might sign with its customers.

#### Prioritization

≤Five business days from when the initial request is weighed by PM, requestor has heard back from the group PM whether the request will be prioritized.

#### Release

≤Six weeks from when the initial request is weighed by PM, this is released into the hands of the Fleet community, generally available (no feature flags or limitations except as originally specced or as adjusted if necessary).

Work that is prioritized by the group PM should be released in the six week timeframe (two releases). Work that is too large for this timeframe should be split up.

#### Estimation

≤Five business days from the initial request, an issue is created with a summary of the purpose, the goal, and the plan to achieve it. The level of detail in that plan is up to the PM of the product group. The issue also has an estimation, expressed in story points, which is either determined through planning poker or a "timebox."

For the Interface group, "estimated" means UI wireframes and API design are completed, and the work to implement them has been estimated.

#### Adjustment

≤One business day from discovering some blocker or change necessary to already prioritized and estimated work. The group PM decides how the usage/UI will be changed and notifies the original requestor of changes to the spec.
 -->

### Product design conventions

We have certain design conventions that we include in Fleet. We will document more of these over time.

> TODO: Link to style guide here instead, and deduplicate all of this content (or as much as possible).

**Table empty states**

Use `---`, with color `$ui-fleet-black-50` as the default UI for empty columns.

**Form behavior**

Pressing the return or enter key with an open form will cause the form to be submitted.

**Internal links**

For text links that navigates the user to a different page within the Fleet UI, use the `$core-blue` color and `xs-bold` styling. You'll also want to make sure to use the underline style for when the user hovers over these links.

**External links**

For a link that navigates the user to an external site (e.g., fleetdm.com/docs), use the `$core-blue` color and `xs-bold` styling for the link's text. Also, place the link-out icon to the right of the link's text.

**Tooltips**

All tooltips change the cursor to a question mark on hover. All tooltips have a solid background color. 

There are two types of tooltips. The two types of tooltips have some unique styles:

1. Tooltips for text (column headers, input titles, inline text) appear when hovering over any text with a dashed underline. These tooltips use left-aligned text.

2. Tooltips for buttons, bubbles, table row elements, and other non-text elements appear when hovering over the element. These tooltips use center-aligned text. These tooltips include a centered arrow.

## Release 

This section outlines the communication between the product, marketing, and customer success teams prior to a release of Fleet.

These measures exist to keep all contributors (including other departments besides engineering and product) up to date with improvements and changes to the Fleet product.  This helps folks plan and communicate with customers and users more effectively.

### Ranking features
After the kickoff of a product sprint, the marketing and product teams decide which improvements are most important to highlight in this release, whether that's through social media "drumbeat" tweets, collaboration with partners, or emphasized [content blocks](https://about.gitlab.com/handbook/marketing/blog/release-posts/#3rd-to-10th) within the release blog post.

When an improvement gets scheduled for release, the Head of Product sets its "echelon" to determine the emphasis the company will place on it.  This leveling is based on the improvement's desirability and timeliness, and will affect marketing effort for the feature.

- **Echelon 1: A major product feature announcement.** The most important release types, these require a specific and custom marketing package. Usually including an individual blog post, a demo video and potentially a press release or official product marketing launch. There is a maximum of one _echelon 1_ product announcement per release sprint.
- **Echelon 2: A highlighted feature in the release notes.** This product feature will be highlighted at the top of the Sprint Release blog post. Depending on the feature specifics this will include: a 1-2 paragraph write-up of the feature, a demo video (if applicable) and a link to the docs. Ideally there would be no more than three _echelon 2_ features in a release post, otherwise the top features will be crowded.
- **Echelon 3: A notable feature to mention in the [changelog](https://github.com/fleetdm/fleet/blob/main/CHANGELOG.md)**. Most product improvements fit into this echelon. This includes 1-2 sentences in the changelog and [release blog post](https://fleetdm.com/releases).

### Blog post

Before each release, the Head of Product [creates a "Release" issue](https://github.com/fleetdm/confidential/issues/new/choose), which includes a list of all improvements included in the upcoming release.  Each improvement links to the relevant bug or user story issue on GitHub so it is easy to read the related discussion and history.

The product team is responsible for providing the marketing team with the necessary information for writing the release blog post. Every three weeks after the sprint is kicked off, the product team meets with the relevant marketing team members to go over the features for that sprint and recommend items to highlight as _echelon 2_ features and provide relevant context for other features to help marketing decide which features to highlight.

## Feature flags

At Fleet, features are placed behind feature flags if the changes could affect Fleet's availability of existing functionalities.

The following highlights should be considered when deciding if we should leverage feature flags:

- The feature flag must be disabled by default.
- The feature flag will not be permanent. This means that the Directly Responsible Individual
 (DRI) who decides a feature flag should be introduced is also responsible for creating an issue to track the
  feature's progress towards removing the feature flag and including the feature in a stable
  release.
- The feature flag will not be advertised. For example, advertising in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

> Fleet's feature flag guidelines is borrowed from GitLab's ["When to use feature flags" section](https://about.gitlab.com/handbook/product-development-flow/feature-flag-lifecycle/#when-to-use-feature-flags) of their handbook. Check out [GitLab's "Feature flags only when needed" video](https://www.youtube.com/watch?v=DQaGqyolOd8) for an explanation of the costs of introducing feature flags.

### Beta features

At Fleet, features are advertised as "beta" if there are concerns that the feature may not work as intended in certain Fleet
deployments. For example, these concerns could be related to the feature's performance in Fleet
deployments with hundreds of thousands of hosts.

The following highlights should be considered when deciding if we promote a feature as "beta:"

- The feature will not be advertised as "beta" permanently. This means that the Directly
  Responsible Individual (DRI) who decides a feature is advertised as "beta" is also responsible for creating an issue that
  explains why the feature is advertised as "beta" and tracking the feature's progress towards advertising the feature as "stable."
- The feature will be advertised as "beta" in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.


## Breaking changes

For product changes that cause breaking API or configuration changes or major impact for users (or even just the _impression_ of major impact!), the company plans migration thoughtfully.  That means the product department and E-group:

1. **Written:** Write a migration guide, even if that's just a Google Doc
2. **Tested:** Test out the migration ourselves, first-hand, as an engineer.
3. **Gamed out:** We pretend we are one or two key customers and try it out as a role play.
4. **Adapt:** If it becomes clear that the plan is insufficient, then fix it.
5. **Communicate:** Develop a plan for how to proactively communicate the change to customers.

That all happens prior to work getting prioritized for the change.

## Competition

We track competitors' capabilities and adjacent (or commonly integrated) products in Google doc [Competition](https://docs.google.com/document/d/1Bqdui6oQthdv5XtD5l7EZVB-duNRcqVRg7NVA4lCXeI/edit) (private).

## Intake

You can quickly suggest a product idea by adding a bullet to the bottom of ["Product feature requests"](https://docs.google.com/document/d/1mwu5WfdWBWwJ2C3zFDOMSUC9QCyYuKP4LssO_sIHDd0/edit).

Digestion of these new product ideas (requests) happens at the **🗣 Product Feature Requests** meeting.  This recurring meeting is located on the "Office hours" calendar.

At the **🗣 Product Feature Requests** meeting, the product team weighs all requests in the agenda. When the team weighs a request, it is immediately prioritized or put to the side.  The DRI for this decision is the Head of Product.

- A _request is prioritized_ when the business decides it is an immediate priority. When this happens, the team sets the request to be estimated within five business days.
- A _request is put to the side_ when the business perceives competing priorities as more pressing in the immediate moment.

### Why this way?

- At Fleet, we use quarterly metrics to align the organization with measurable goals.  These goals fill up a large portion, but not all, of planning (drafting, wireframing, spec'ing, etc.) and engineering capacity.   This means there is always some capacity to prioritize requests advocated for by customers, Fleet team members, and members of the wider Fleet community.
- The 🗣 Product Feature Requests meeting is a recurring ritual to make sure that the team weighs all requests.
- At Fleet, we tell the requestor whether their request is prioritized or put to the side within one business day from when the team weighs the request.
- Fleet always prioritizes bugs.

### Making a request

To make a request or advocate for a request from a customer or community member,  Fleet asks all members of the organization to add their name and a description of the request to the list in the [🗣 Product Feature Requests Google
doc](https://docs.google.com/document/d/1mwu5WfdWBWwJ2C3zFDOMSUC9QCyYuKP4LssO_sIHDd0/edit#heading=h.zahrflvvks7q).
Then attend the next scheduled 🗣 Product Feature Requests meeting.

All members of the Fleet organization are welcome to attend the 🗣 Product Feature Requests meeting. Requests will be
weighed from top to bottom while prioritizing attendee requests. 

This means that if the individual that added a feature request is not in attendance, the feature request will be discussed towards the end of the call if there's time.

All 🗣 Product Feature Requests meetings are recorded and uploaded to Gong.

### PFR cleanup 
Each week the DRI for the 🗣 Product Feature Requests meeting resets the document to blank by doing the following:
1. Create issues for accepted items
2. Notify absent requesters of decisions
3. Move that week's feature requests to the backup journal document

## Usage statistics

In order to understand the usage of the Fleet product, we [collect statistics](https://fleetdm.com/docs/using-fleet/usage-statistics) from installations where this functionality is enabled.

Fleet team members can view these statistics in the Google spreadsheet [Fleet
usage](https://docs.google.com/spreadsheets/d/1Mh7Vf4kJL8b5TWlHxcX7mYwaakZMg_ZGNLY3kl1VI-c/edit#gid=0)
available in Google Drive.

## Rituals

Directly Responsible Individuals (DRI) engage in the ritual(s) below at the frequency specified.

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-------------------------|:----------------------------------------------------|-------------------|
| 🗣 Product feature requests  | Weekly (Tuesdays) | We make a decision regarding which customer and community feature requests can be committed to in the next six weeks. We create issues for any requests that don't already have one. | Mo Zhu |
| 🗣️ Product feature requests prep and cleanup | Weekly (Tuesdays) | Every week a backup doc is created to accompany the 🗣️ Product Feature Requests event | Mo Zhu |
| 🗣 Product office hours  | Weekly (Thursdays) | Ask questions to the product team | Mo Zhu |
| Sprint release notes kick-off meeting | Triweekly (Wednesday) | Communicate high-value features from the current sprint to prepare release blog post and drumbeat social posts, etc in the leadup to release at the end of each sprint.  Marketing is responsible for getting what they need to publish and promote the release, including a great release post.  Product is responsible for helping marketing understand what is coming early enough that there is time to prepare. | Mo Zhu |
| ⚗️✨🗣 Design review (MDM)  | Daily | Review designs from the MDM team | Marko Lisica |
| ⚗️✨🗣 Design review (CX)   | Daily | Review designs from the CX team | Rachael Shaw |
| ⚗️✅🎉Product confirm and celebrate | Weekly | Product teams gets together to review work completed | Mo Zhu |
| ⚗️ Sprint release notes kickoff | Tri-weekly | Product provides recommended features to highlight for the current sprint to enable the Marketing team to start writing release notes | Mo Zhu |

## Slack channels

This group maintains the following [Slack channels](https://fleetdm.com/handbook/company#why-group-slack-channels):

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#why-group-slack-channels)|
|:------------------------------------|:--------------------------------------------------------------------|
| `#help-product`                     | Mo Zhu                                                              |
| `#g-mdm`                            | Noah Talerman                                                       |
| `#g-cx`                             | Zay Hanlon                                                          |

<meta name="maintainedBy" value="zhumo">
<meta name="title" value="⚗️ Product">
