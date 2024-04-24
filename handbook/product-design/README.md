# Product design
This handbook page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.

## Team
| Role                            | Contributor(s)           |
|:--------------------------------|:-----------------------------------------------------------------------------------------------------------|
| Head of Product Design           | [Noah Talerman](https://www.linkedin.com/in/noah-talerman/) _([@noahtalerman](https://github.com/noahtalerman))_
| Head of Design                   | [Mike Thomas](https://www.linkedin.com/in/mike-thomas-52277938) _([@mike-j-thomas](https://github.com/mike-j-thomas))_
| Product Designer                | [Rachael Shaw](https://www.linkedin.com/in/rachaelcshaw/) _([@rachaelshaw](https://github.com/rachaelshaw))_, [Marko Lisica](https://www.linkedin.com/in/markolisica/) _([@marko-lisica](https://github.com/marko-lisica))_
| Developer                        | [Eric Shaw](https://www.linkedin.com/in/eric-shaw-1423831a9/) _([@eashaw](https://github.com/eashaw))_


## Contact us
- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?labels=%3Aproduct&title=Product%20design%20request%C2%BB______________________&template=custom-request.md) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in `#help-design`.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.
  - Any Fleet team member can [view the kanban board](https://app.zenhub.com/workspaces/-g-digital-experience-6451748b4eb15200131d4bab/board?sprints=none) for this department, including pending tasks and the status of new requests.

## Responsibilities
The Product Design department is responsible for reviewing and collecting feedback from users, would-be users, and future users, prioritizing changes, designing the changes, and delivering these changes to the engineering team. Product Design prioritizes and shapes all changes involving functionality or usage, including the UI, REST API, command line, and webhooks. 

### Release relavent Figma files
- Once your designs are reviewed and approved, change the status on the cover page of the relevant Figma file and move the issue to the "Settled" column.
- After each release (every 3 weeks) make sure you change the status on the cover page of the relevant Figma files that you worked on during the sprint to "Released".

>**Questions and missing information:** Take a screenshot of the area in Figma and add a comment in the story's GitHub issue. Figma does have a commenting system, but it is not easy to search for outstanding concerns and is therefore not preferred.
>
>For external contributors: please consider opening an issue with reference screenshots if you have a Figma related question you need to resolve.

### Create a new Figma file
At Fleet, like [GitLab](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall) and [other organizations](https://speakerdeck.com/mikermcneil/i-love-apis), every change to the product's UI gets [wireframed first](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

- Take the top user story that is assigned to you in the "Prioritized" column of the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).
  
- Create a new file inside the [Fleet product](https://www.figma.com/files/project/17318630/%F0%9F%94%9C%F0%9F%93%A6-Fleet-EE%C2%AE-(product)?fuid=1234929285759903870) Figma project. See [Working with Figma](https://fleetdm.com/handbook/product#working-with-figma) below for more details.
  
- Use dev notes (component available in our library) to highlight important information to engineers and other teammates.

- Draft changes to the Fleet product that solve the problem specified in the story. Constantly place yourself in the shoes of a user while drafting changes. Place these drafts in the appropriate Figma file in Fleet product project.

- Be intentional about changes to design components (e.g. button border-radius or modal width) because these are expensive. They'll require code changes and QA in multiple parts of the product. Propose changes to a design component as part of an already-prioritized user story instead of [making a new request](#making-a-request) in 🎁🗣 Feature Fest.

- While drafting, reach out to sales, customer success, and demand for a business perspective.

- While drafting, engage engineering to gain insight into technical costs and feasibility.

When starting a new draft:
- Create a new file inside the [Fleet product](https://www.figma.com/files/project/17318630/%F0%9F%94%9C%F0%9F%93%A6-Fleet-EE%C2%AE-(product)?fuid=1234929285759903870) project by duplicating "\[TEMPLATE\] Starter file" (pinned to the top of the project).
- Right-click on the duplicated file, select "Share", and ensure **anyone with the link** can view the file.
- Rename each Figma file to include the number and name of the corresponding issue on the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board). (e.g. # 11766 Instructions for Autopilot enrollment).
- The starter file includes 3 predefined pages: Cover, Ready, and Scratchpad.
    -  **Cover.** This page has a component with issue number, issue name, and status fields. There are 3 statuses: Work in progress, Approved, and Released (the main source of truth is still the drafting board).
    -  **Ready.** Use this page to communicate designs reviews and development.
    -  **Scratchpad.** Use this page for work in progress and design that might be useful in the future.
- If the story requires API changes, open a draft PR with the proposed API design.

### Schedule a design review
- Prepare your draft in the user story issue.
- Prepare the agenda for your design review meeting, which should be an empty document other than the proposed changes you will present.
- Review the draft with the CEO at one of the daily design review meetings, or schedule an ad-hoc design review if you need to move faster.  (Efficient access to design reviews on-demand [is a priority for Fleet's CEO](https://fleetdm.com/handbook/company/ceo). Emphasizing design helps us live our [empathy](https://fleetdm.com/handbook/company#empathy) value.)
- When introducing a story, clarify which review "mode" the CEO should operate in:
  + **Final review** mode — you are 70% sure the design is 100% done.
  + **Feedback** mode — you know the design is not ready for final review, but would like to get early feedback. Before bringing something in feedback mode consider whether the CEO will be best for giving feedback or if it would be better suited for someone else (engineer or PM).
- During the review meeting, take detailed notes of any feedback on the draft.
- Address the feedback by modifying your draft.
- Rinse and repeat at subsequent sessions until there is no more feedback.

> As drafting occurs, inevitably, the requirements will change. The main description of the issue should be the single source of truth for the problem to be solved and the required outcome. The product manager is responsible for keeping the main description of the issue up-to-date. Comments and other items can and should be kept in the issue for historical record-keeping.

### Ensure story drafting is complete
<!--TODO update responsibility to reflect reality (e.g. line 75 == "Bugs board"?)-->
Once a story has gone through design and is considered "Settled", it moves to the "Settled" column on the drafting board and assign to the Engineering Manager (EM).

Before assigning an EM to [estimate](https://fleetdm.com/handbook/engineering#sprint-ceremonies) a user story, the product designer ensures the product section of the user story [checklist](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=story&projects=&template=story.md&title=) is complete. 

Once a bug has gone through design and is considered "Settled", the designer removes the `:product` label and moves the issue to the 'Sprint backlog' column on the "Bugs" board and assigns the group engineering manager. 

### Revise a draft currently in development
Expedited drafting is the revision of drafted changes currently being developed by
the engineering team. Expedited drafting aims to quickly adapt to unknown edge cases and
changing specifications while ensuring that Fleet meets our brand and quality guidelines. 

You'll know it's time for expedited drafting when:
- The team discovers that a drafted user story is missing crucial information that prevents contributors from continuing the development task.
- A user story is taking more effort than was originally estimated, and Product Manager wants to find ways to cut aspects of planned functionality in order to still ship the improvement in the currently scheduled release.
- A user story on the drafting board won't reach "Settled" by the last estimation session in the current sprint and cannot wait until the next sprint. This can also happen when we decide to bring a user story in mid-sprint.

What happens during expedited drafting?
1. If the user story wasn't "Settled" by the last estimation session, the product group's engineering manager (EM), [release DRI](https://fleetdm.com/handbook/company/communications#directly-responsible-individuals-dris), and Head of Product Design are notified in `#g-mdm` or `#g-endpoint-ops`. Decision to allow the user story to make it into the sprint is up to the release DRI.
2. If the user story is already in the sprint, the EM, release DRI, and Head of Product Design are notified in `#g-mdm` or `#g-endpoint-ops`. If there are significant changes to the requirements, then the user story might be pushed to the next sprint. Decision is up to the release DRI.
3. If the release DRI decides the user story will be worked on this sprint, drafts are updated or finished.
4. UI changes [are approved](https://fleetdm.com/handbook/company/development-groups#drafting-process), and the UI changes are brought back into the sprint or are estimated.

### Correctly prioritize a bug
Bugs are always prioritized. (Fleet takes quality and stability [very seriously](https://fleetdm.com/handbook/company/why-this-way#why-spend-so-much-energy-responding-to-every-potential-production-incident).) Bugs should be prioritized in the following order:
1. Quality: product does what it's supposed to (what is documented).
2. Common-sense user criticality: If no one can load any page, that's obviously important.
3. Age of bugs: Long-open bugs are open wounds bleeding quality out of the product.  They must be closed quickly.
4. Customer criticality: How important it is to a customer use case.


If a bug is unreleased or [critical](https://fleetdm.com/handbook/engineering#critical-bugs), it is addressed in the current sprint. Otherwise, it may be prioritized and estimated for the next sprint. If a bug [requires drafting](https://fleetdm.com/handbook/engineering#in-product-drafting-as-needed) to determine the expected functionality, the bug should undergo [expedited drafting](#expedited-drafting). 

If a bug is not addressed within six weeks, it is [sent to the product team for triage](https://fleetdm.com/handbook/engineering#in-engineering). Each sprint, the Head of Product Design reviews these bugs. Bugs are categorized as follows:
- **Schedule**: the bug should be prioritized in the next sprint if there's engineering capacity for it.
- **De-prioritized**: the issue will be closed and the necessary subsequent steps will be initiated. This might include updating documentation and informing the community.

The Head of Product Design meets with the Director of Product Development to discuss and finalize the outcomes for the churned bugs.

After aligning with the Director of Product Development on the outcomes, The Head of Product Design will clean up churned bugs. Below are the steps for each category:
- **Schedule**: Remove the `:product` label, move the bug ticket to the 'Sprint backlog' column on the bug board, and assign it to the appropriate group's Engineering Manager so that it can be prioritized into the sprint backlog.
- **De-prioritized**: The Head of Product Design should close the issue and, as the DRI, ensure all follow-up actions are finalized.

### Write a user story
Product Managers [write user stories](https://fleetdm.com/handbook/company/product-groups#writing-a-good-user-story) in the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board). The drafting board is shared by every [product group](https://fleetdm.com/handbook/company/development-groups).

### Draft a user story
Product Designers [draft user stories](https://fleetdm.com/handbook/company/product-groups#drafting) that have been prioritized by PMs. If the estimated user stories for a product group exceed [that group's capacity](https://fleetdm.com/handbook/company/product-groups#current-product-groups), all new design work for that group is paused, and the designer will contribute in other ways (documentation & handbook work, Figma maintenance, QA, etc.) until the PM deprioritizes estimated stories to make room, or until the next sprint begins. (If the designer has existing work-in-progress, they will continue to review and iterate on those designs and see the stories through to estimation.)

If an issue's title or user story summary (_"as a…I want to…so that"_) does not match the intended change being discussed, the designer will move the issue to the "Needs clarity" column of the drafting board and assign the group product manager.  The group product manager will revisit ASAP and edit the issue title and user story summary, then reassign the designer and move the issue back to the "Prioritized" column.

Engineering Managers estimate user stories.  They are responsible for delivering planned work in the current sprint (0-3 weeks) while quickly getting user stories estimated for the next sprint (3-6 weeks).  Only work that is slated to be released into the hands of users within ≤six weeks will be estimated. Estimation is run by each group's Engineering Manager and occurs on the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).

### Rank features before release
These measures exist to keep all contributors (including other departments besides engineering and product) up to date with improvements and changes to the Fleet product.  This helps folks plan and communicate with customers and users more effectively.

After the kickoff of a product sprint, the demand and product teams decide which improvements are most important to highlight in this release, whether that's through social media "drumbeat" tweets, collaboration with partners, or emphasized [content blocks](https://about.gitlab.com/handbook/marketing/blog/release-posts/#3rd-to-10th) within the release blog post.

When an improvement gets scheduled for release, the Head of Product sets its "echelon" to determine the emphasis the company will place on it. This leveling is based on the improvement's desirability and timeliness, and will affect demand effort for the feature.

- **Echelon 1: A major product feature announcement.** The most important release types, these require a specific and custom demand package. Usually including an individual blog post, a demo video and potentially a press release or official product demand launch. There is a maximum of one _echelon 1_ product announcement per release sprint.
- **Echelon 2: A highlighted feature in the release notes.** This product feature will be highlighted at the top of the Sprint Release blog post. Depending on the feature specifics this will include: a 1-2 paragraph write-up of the feature, a demo video (if applicable) and a link to the docs. Ideally there would be no more than three _echelon 2_ features in a release post, otherwise the top features will be crowded.
- **Echelon 3: A notable feature to mention in the [changelog](https://github.com/fleetdm/fleet/blob/main/CHANGELOG.md)**. Most product improvements fit into this echelon. This includes 1-2 sentences in the changelog and [release blog post](https://fleetdm.com/releases).

### Create release issue
Before each release, the Head of Product [creates a "Release" issue](https://github.com/fleetdm/confidential/issues/new/choose), which includes a list of all improvements included in the upcoming release.  Each improvement links to the relevant bug or user story issue on GitHub so it is easy to read the related discussion and history.

The product team is responsible for providing the demand team with the necessary information for writing the release blog post. Every three weeks after the sprint is kicked off, the product team meets with the relevant demand team members to go over the features for that sprint and recommend items to highlight as _echelon 2_ features and provide relevant context for other features to help demand decide which features to highlight.

### Consider a feature eligible to be flagged
At Fleet, features are placed behind feature flags if the changes could affect Fleet's availability of existing functionalities. The following highlights should be considered when deciding if we should leverage feature flags:

- The feature flag must be disabled by default.
- The feature flag will not be permanent. This means that the Directly Responsible Individual
 (DRI) who decides a feature flag should be introduced is also responsible for creating an issue to track the
  feature's progress towards removing the feature flag and including the feature in a stable
  release.
- The feature flag will not be advertised. For example, advertising in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

> Fleet's feature flag guidelines is borrowed from GitLab's ["When to use feature flags" section](https://about.gitlab.com/handbook/product-development-flow/feature-flag-lifecycle/#when-to-use-feature-flags) of their handbook. Check out [GitLab's "Feature flags only when needed" video](https://www.youtube.com/watch?v=DQaGqyolOd8) for an explanation of the costs of introducing feature flags.

### Consider promoting a feature as "beta"
At Fleet, features are advertised as "beta" if there are concerns that the feature may not work as intended in certain Fleet
deployments. For example, these concerns could be related to the feature's performance in Fleet
deployments with hundreds of thousands of hosts.

The following highlights should be considered when deciding if we promote a feature as "beta:"

- The feature will not be advertised as "beta" permanently. This means that the Directly
  Responsible Individual (DRI) who decides a feature is advertised as "beta" is also responsible for creating an issue that
  explains why the feature is advertised as "beta" and tracking the feature's progress towards advertising the feature as "stable."
- The feature will be advertised as "beta" in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

### View Fleet usage statistics
In order to understand the usage of the Fleet product, we [collect statistics](https://fleetdm.com/docs/using-fleet/usage-statistics) from installations where this functionality is enabled.

Fleeties can view these statistics in the Google spreadsheet [Fleet
usage](https://docs.google.com/spreadsheets/d/1Mh7Vf4kJL8b5TWlHxcX7mYwaakZMg_ZGNLY3kl1VI-c/edit#gid=0)
available in Google Drive.

Some of the data is forwarded to [Datadog](https://us5.datadoghq.com/dashboard/7pb-63g-xty/usage-statistics?from_ts=1682952132131&to_ts=1685630532131&live=true) and is available to Fleeties.


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

##### Beta features
Please see [handbook/product#consider-promoting-a-feature-as-beta](https://fleetdm.com/handbook/product#consider-promoting-a-feature-as-beta)

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

<meta name="maintainedBy" value="noahtalerman">
<meta name="title" value="🦢 Product design">
