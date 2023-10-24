# Product design

Contributors in Fleet's product department [prioritize](#prioritizing-improvements) and [define](https://fleetdm.com/handbook/company/product-groups#drafting) the [changes we make to the product](https://fleetdm.com/handbook/company/product-groups#making-changes).

Changes begin as [ideas](#intake) or [code](#outside-contributions) that can be contributed by anyone.

> You can read what's coming in the next 3-6 weeks in Fleet's [⚗️ Drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).

## Making changes to the product

Fleet's product designers are responsible for [prioritizing and shaping changes to the Fleet product](https://fleetdm.com/handbook/company/development-groups#making-changes), from the outside-in,  reviewing and collecting feedback from users, would-be users, and future users, prioritizing changes, designing the changes, and delivering these changes to the engineering team.

The scope of product design at Fleet is any change that involves changes to functionality or usage, including the UI, REST API, command line, and webhooks.

> Learn more about Fleet's philosophy and process for [making interface changes to the product](https://fleetdm.com/handbook/company/development-groups#making-changes), or [why we use a wireframe-first approach](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

### Wireframing

At Fleet, like [GitLab](https://about.gitlab.com/handbook/product-development-flow/#but-wait-isnt-this-waterfall) and [other organizations](https://speakerdeck.com/mikermcneil/i-love-apis), every change to the product's UI gets [wireframed first](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach).

- Take the top user story that is assigned to you in the "Prioritized" column of the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).
  
- Create a new file inside the [Fleet product](https://www.figma.com/files/project/17318630/%F0%9F%94%9C%F0%9F%93%A6-Fleet-EE%C2%AE-(product)?fuid=1234929285759903870) Figma project. See [Working with Figma](https://fleetdm.com/handbook/product#working-with-figma) below for more details.
  
- Use dev notes (component available in our library) to highlight important information to engineers and other teammates.

- Draft changes to the Fleet product that solve the problem specified in the story. Constantly place yourself in the shoes of a user while drafting changes. Place these drafts in the appropriate Figma file in Fleet product project.

- Be intentional about changes to design components (e.g. button border-radius or modal width) because these are expensive. They'll require code changes and QA in multiple parts of the product. Propose changes to a design component as part of an already-prioritized user story instead of [making a new request](#making-a-request) in 🎁🗣 Feature Fest.

- While drafting, reach out to sales, customer success, and marketing for a business perspective.

- While drafting, engage engineering to gain insight into technical costs and feasibility.

### Working with Figma

#### Create a new file

When starting a new draft:

- Create a new file inside the [Fleet product](https://www.figma.com/files/project/17318630/%F0%9F%94%9C%F0%9F%93%A6-Fleet-EE%C2%AE-(product)?fuid=1234929285759903870) project by duplicating "\[TEMPLATE\] Starter file" (pinned to the top of the project).
- Right-click on the duplicated file, select "Share", and ensure **anyone with the link** can view the file.
- Rename each Figma file to include the number and name of the corresponding issue on the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board). (e.g. # 11766 Instructions for Autopilot enrollment).
-  The starter file includes 3 predefined pages: Cover, Ready, and Scratchpad.
    -  **Cover.** This page has a component with issue number, issue name, and status fields. There are 3 statuses: Work in progress, Approved, and Released (the main source of truth is still the drafting board).
    -  **Ready.** Use this page to communicate designs reviews and development.
    -  **Scratchpad.** Use this page for work in progress and design that might be useful in the future.


#### Keep projects/files clean and up-to-date

- Once your designs are reviewed and approved, change the status on the cover page of the relevant Figma file and move the issue to the "Settled" column.
- After each release (every 3 weeks) make sure you change the status on the cover page of the relevant Figma files that you worked on during the sprint to "Released".

#### Questions and missing information

1. Take a screenshot of the area in Figma
2. Start a thread in the #help-product-design Slack channel and paste in the screenshot

Note: Figma does have a commenting system, but it is not easy to search for outstanding concerns and is therefore not preferred.

For external contributors: please consider opening an issue with reference screenshots if you have a Figma related question you need to resolve.

### Scheduling design reviews

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

### Settled 

Once the draft has been approved, it moves to the "Settled" column on the drafting board. 

Before assigning an engineering manager to [estimate](https://fleetdm.com/handbook/engineering#sprint-ceremonies) a user story, the product designer ensures the product section of the user story [checklist](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=story&projects=&template=story.md&title=) is complete. 

Once a bug has gone through design and is considered "Settled", the designer removes the `:product` label and moves the issue to the 'Sprint backlog' column on the "Bugs" board and assigns the group engineering manager. 

Learn https://fleetdm.com/handbook/company/development-groups#making-changes

### Expedited drafting

Expedited drafting is the revision of drafted changes currently being developed by
the engineering team. Expedited drafting aims to quickly adapt to unknown edge cases and
changing specifications while ensuring that Fleet meets our brand and quality guidelines. 

You'll know it's time for expedited drafting when:
- The team discovers that a drafted user story is missing crucial information that prevents contributors from continuing the development task.
- A user story is taking more effort than was originally estimated, and Product Manager wants to find ways to cut aspects of planned functionality in order to still ship the improvement in the currently scheduled release.

What happens during expedited drafting?
1. Everyone on the product and engineering teams know that a drafted change was brought back
   to drafting and prioritized.
2. Drafts are updated to cover edge cases or reduce functionality.
3. UI changes [are approved](https://fleetdm.com/handbook/company/development-groups#drafting-process), and the UI changes are brought back to the engineering team to continue the development task.

## Outside contributions

[Anyone can contribute](https://fleetdm.com/handbook/company#openness) at Fleet, from inside or outside the company.  Since contributors from the wider community don't receive a paycheck from Fleet, they work on whatever they want.

Many open source contributions that start as a small, seemingly innocuous pull request come with lots of additional [unplanned work](https://fleetdm.com/handbook/company/development-groups#planned-and-unplanned-changes) down the road: unforseen side effects, documentation, testing, potential breaking changes, database migrations, [and more](https://fleetdm.com/handbook/company/development-groups#defining-done).

Thus, to ensure consistency, completeness, and secure development practices, no matter where a contribution comes from, Fleet will still follow the standard process for [prioritizing](#prioritizing-improvements) and [drafting](https://fleetdm.com/handbook/company/development-groups#drafting) a feature when it comes from the community.

## Prioritizing improvements
Product Managers prioritize all potential product improvements worked on by Fleeties. Anyone (Fleeties, customers, and community members) are invited to suggest improvements. See [the intake section](#intake) for more information on how Fleet's product team intakes new feature requests.

## Prioritizing bugs
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

## Writing user stories
Product Managers [write user stories](https://fleetdm.com/handbook/company/development-groups#writing-a-good-user-story) in the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board). The drafting board is shared by every [product group](https://fleetdm.com/handbook/company/development-groups).

## Drafting user stories
Product Designers [draft user stories](https://fleetdm.com/handbook/company/development-groups#drafting) that have been prioritized by PMs. If the estimated user stories for a product group exceed [that group's capacity](https://fleetdm.com/handbook/company/product-groups#current-product-groups), all new design work for that group is paused, and the designer will contribute in other ways (documentation & handbook work, Figma maintenance, QA, etc.) until the PM deprioritizes estimated stories to make room, or until the next sprint begins. (If the designer has existing work-in-progress, they will continue to review and iterate on those designs and see the stories through to estimation.)

If an issue's title or user story summary (_"as a…I want to…so that"_) does not match the intended change being discussed, the designer will move the issue to the "Needs clarity" column of the drafting board and assign the group product manager.  The group product manager will revisit ASAP and edit the issue title and user story summary, then reassign the designer and move the issue back to the "Prioritized" column.

## Estimating user stories
Engineering Managers estimate user stories.  They are responsible for delivering planned work in the current sprint (0-3 weeks) while quickly getting user stories estimated for the next sprint (3-6 weeks).  Only work that is slated to be released into the hands of users within ≤six weeks will be estimated. Estimation is run by each group's Engineering Manager and occurs on the [drafting board](https://app.zenhub.com/workspaces/-product-backlog-coming-soon-6192dd66ea2562000faea25c/board).

## Sprints
Sprints align with Fleet's [3-week release cycle](https://fleetdm.com/handbook/company/why-this-way#why-a-three-week-cadence).

On the first day of each release, all estimated issues are moved into the relevant section of the new "Release" board, which has a kanban view per group. 

Sprints are managed in [Zenhub](https://fleetdm.com/handbook/company/why-this-way#why-make-work-visible). To plan capacity for a sprint, [create a "Sprint" issue](https://github.com/fleetdm/confidential/issues/new/choose), replace the fake constants with real numbers, and attach the appropriate labels for your product group.

### Sprint numbering
Sprints are numbered according to the release version. For example, for the sprint ending on June 30th, 2023, on which date we expect to release Fleet v4.34, the sprint is called the 4.34 sprint. 

### Product design conventions

#### MDM behind-the-frame

Behind every MDM [wireframe at Fleet](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach), there are 3 foundational design principles:  

- **Use-case first.** Taking advantage of top-level features vs. per-platform options allows us to take advantage of similarities and avoid having two different ways to configure the same thing.
Start off cross-platform for every option, setting, and feature. If we **prove** it's impossible, _then_ work backward making it platform-specific.

- **Bridge the Mac and Windows gap.** Implement enough help text, links, guides, gifs, etc that a reasonably persistent human being can figure it out just by trying to use the UI.
   Even if that means we have fewer features or slightly lower granularity (we can iterate and add more granularity later), Make it easy enough to understand. Whether they're experienced Mac admins people or career Windows folks (even if someone has never used a Windows tool) they should _"get it"_. 

- **Control the noise.** Bring the needs surface level, tuck away things you don't need by default (when possible, given time). For example, hide Windows controls if there are no Windows devices (based on number of Windows hosts).

##### Wireframes 

- Showing these principles and ideas, to help remember the pros and cons and conceptualize the above visually.

   - Figma: [⚗️ Fleet product project](https://www.figma.com/files/project/17318630/%E2%9A%97%EF%B8%8F-Fleet-product?fuid=1234929285759903870)

We have certain design conventions that we include in Fleet. We will document more of these over time.

**Figma component library**

Use the 🧩 ["Design System (current)"](https://www.figma.com/file/8oXlYXpgCV1Sn4ek7OworP/%F0%9F%A7%A9-Design-System-(current)?type=design&mode=design&t=BytcobQwypszkxf5-1) Figma library as a source of truth for components. Components in the product ([Storybook](https://fleetdm.com/storybook/)) should match the style of components defined in the Figma library. If the frontend component is inconsistent with one in the Figma library, treat that as a [bug](https://fleetdm.com/handbook/engineering#finding-bugs).

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

**Bold text**

For copy in the Fleet UI and Fleet documentation, use bold text when referencing UI elements such as buttons, links, column names, form field names, page names, and section names. For an example, check out the bold text in the docs [here](https://fleetdm.com/docs/using-fleet/mdm-disk-encryption#step-1-enforce-disk-encryption).

This way, if a user is scanning instructions, the bold text tells them what UI element they should look for.

In the docs, if a UI element is part of a section title (already bold) use double quotes. For an example, see this section title [here](https://fleetdm.com/docs/get-started/faq#what-happened-to-the-schedule-page).

**Copy in parentheses (additional information)**

When writing copy, consider whether additional information is necessary before adding it as a new sentence or in parentheses. If the information is needed, use parentheses with an incomplete sentence to keep the copy shorter.

**Writing the time**

When writing the time in the UI using "am" and "pm" abbreviations, write them **without space** between time and abbreviation, with **no punctuation**, and use **lowercase** letters (e.g. Working hours are 8am to 5pm).

**Writing error messages**

When writing error messages in the UI or CLI, follow these rules:
- If the solution to the error isn't obvious, write a message with the **error** followed by the **solution**. For example, "No hosts targeted. Make sure you provide a valid hostname, UUID, osquery host ID, or node key."
- If the solution is obvious when additional info is provided, write a message with the **error** followed by **additional info**. For example, "You don’t have permission to run the script. Only users with the maintainer role and above can run scripts."

**Fleetctl commands with `--hosts` or `--host` flag**

When designing CLI experience for commands that target hosts (e.g. `fleetctl query` or `fleetctl mdm run-command` when including the `--hosts` or `--host` flag), if a non-existing host is specified, use a single error message such as: `Error: No hosts targeted. Make sure you provide a valid hostname, UUID, osquery host ID, or node key.`

When writing copy for CLI help pages use the following descriptions:
```
$ fleetctl <command with --hosts/--host flag> -h

OPTIONS
--hosts     Hosts specified by hostname, uuid, osquery_host_id or node_key that you want to target.     
--host      Host specified by hostname, uuid, osquery_host_id or node_key that you want to target.     
```

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

We track competitors' capabilities and adjacent (or commonly integrated) products in Google doc [Competition](https://docs.google.com/document/d/1Bqdui6oQthdv5XtD5l7EZVB-duNRcqVRg7NVA4lCXeI/edit) (private Google doc).

## Intake

- [Making a request](#making-a-request)
- [How feature requests are evaluated](#how-feature-requests-are-evaluated)
- [After the feature is accepted](#after-the-feature-is-accepted)
- [Why this way?](#why-this-way)

To stay in-sync with our customers' needs, Fleet accepts feature requests from customers and community members on a sprint-by-sprint basis in the [regular 🎁🗣 Feature Fest meeting](#rituals). Anyone in the company is invited to submit requests or simply listen in on the 🎁🗣 Feature Fest meeting. Folks from the wider community can also [request an invite](https://fleetdm.com/contact). 

### Making a request

To make a feature request or advocate for a feature request from a customer or community member, [create an issue](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=~customer+request&projects=&template=feature-request.md&title=) and attend the next scheduled 🎁🗣 Feature Fest meeting. 

Requests are weighed from top to bottom while prioritizing attendee requests. This means that if the individual that added a feature request is not in attendance, the feature request will be discussed towards the end of the call if there's time.

To be acceptable for consideration, a request must:
- Have a clear proposed change
- Have a well-articulated underlying user need
- Specify the requestor (either internal stakeholder or customer or community user)

To help the product team, other pieces of information can be optionally included:
- How would they solve the problem without any changes if pressed?
- How does this change fit into the requester's overall usage of Fleet?
- What other potential changes to the product have you considered?

To ensure your request appears on the ["Feature Fest" board](https://app.zenhub.com/workspaces/-feature-fest-651b2962605ba29209324c57/board):
- Add the `~feature fest` label to your issue
- Add the relevant customer label (if applicable) 

To maximize your chances of having a feature accepted, requesters can visit the [🗣 Product office hours](#rituals) meeting to get feedback on requests prior to being accepted. 

### How feature requests are evaluated
Digestion of these new product ideas (requests) happens at the **🎁🗣 Feature Fest** meeting.

At the **🎁🗣 Feature Fest** meeting, the DRI (Head of Product) weighs all requests on the board. When the team weighs a request, it is immediately prioritized or put to the side.
- A _request is prioritized_ when the DRI decides it is a priority. When this happens, the team sets the request to be estimated within five business days.
- A _request is put to the side_ when the business perceives competing priorities as more pressing in the immediate moment.

If a feature is not prioritized during a 🎁🗣 Feature Fest meeting, it only means the feature has been rejected _at that time_. Requestors will be notified by the Head of Product, and they can resubmit their request at a future meeting.

Requests are weighed by:
- The completeness of the request (see [making a request](#making-a-request))
- How urgent the need is for the customer
- How much impact the request will have. This may be a wide impact across many customers and/or high impact on one
- How well the request fits within Fleet's product vision and roadmap
- Whether the feature seems like it can be designed, estimated, and developed in 6 weeks, given its individual complexity and when combined with other work already accepted

### Customer feature requests 
The product team's goal is to prioritize 16 customer feature requests at Feature Fest, then take them from settled to shipped. The customer success team is responsible for providing the Head of Product a live count during the Feature Fest meeting. Product Operations is responsible for monitoring this KPI and raising alarms throughout the design and engineering sprints. 
> Customer stories should be estimated at 1-3 points each to count as 1 request. If a feature request spans across multiple customers, it will be counted as the number of customers involved. 

### After the feature is accepted
After the 🎁🗣 Feature Fest meeting, Product Operations will clear the Feature Fest board as follows:
**Prioritized features:** Remove `feature fest` label, add `:product` label, and assign the group Product Manager. 
**Put to the side features:** Remove `feature fest` label and close the issue.

Group Product Managers will then develop user stories for the prioritized features. 

> The product team's commitment to the requester is that a prioritized feature will be delivered within 6 weeks or the requester will be notified within 1 business day of the decision to de-prioritize the feature. 

Potential reasons for why a feature may be de-prioritized include:
- The work was not designed in time. Since Fleet's engineering sprints are 3 weeks each, this means that a prioritized feature has 3 weeks to be designed, approved, and estimated in order to make it to the engineering sprint. At the prioritization meeting, the perceived design complexity of proposed features will inevitably be different from the actual complexity. 
  - This may be because other higher-priority design work took longer than expected or the work itself was more complex than expected
- The was designed but was not selected for the sprint. When a new sprint starts, it is populated with bugs, features, and technical tasks. Depending on the size and quantity of non-feature work, certain features may not be selected for the sprint.

Just as when a feature request is not accepted in the 🎁🗣 Feature Fest meeting, whenever a feature is de-prioritized after it has been accepted, it only means that the feature has been _de-prioritized at this time_. It is up to the requester to bring the request back again at another 🎁🗣 Feature Fest meeting.

### Why this way?

Most requests are not prioritized.  The goal is to narrow our focus and avoid creating an overflowing, aspirational backlog where good ideas inevitably go to die.  Instead, at Fleet we manage a small "frontlog" of changes we intend to ship. Responsibility for keeping backlogs then belong to the stakeholder who is closest to the customer. 

### Misc.
- All 🎁🗣 Feature Fest meetings are recorded and uploaded to Gong.
- 🎁🗣 Feature Fest is located on the "Office hours" calendar.

## Usage statistics

In order to understand the usage of the Fleet product, we [collect statistics](https://fleetdm.com/docs/using-fleet/usage-statistics) from installations where this functionality is enabled.

Fleeties can view these statistics in the Google spreadsheet [Fleet
usage](https://docs.google.com/spreadsheets/d/1Mh7Vf4kJL8b5TWlHxcX7mYwaakZMg_ZGNLY3kl1VI-c/edit#gid=0)
available in Google Drive.

Some of the data is forwarded to [Datadog](https://us5.datadoghq.com/dashboard/7pb-63g-xty/usage-statistics?from_ts=1682952132131&to_ts=1685630532131&live=true) and is available to Fleeties.

## Maintenance
Fleet's product offerings depend on the capabilities of other platforms. This requires the ongoing attention of the product and engineering teams to ensure that we are up-to-date with new capabilities and that our existing capabilities continue to function. The first step to staying up-to-date with Fleet's partners is to know when the partner platform changes. 

Every week, a member of the product team (as determined in the [rituals](#rituals) section) looks up whether there is:
1. a new major or minor version of [macOS](https://support.apple.com/en-us/HT201260)
2. a new major or minor version of [CIS Benchmarks Windows 10 Enterprise](https://workbench.cisecurity.org/community/2/benchmarks?q=windows+10+enterprise&status=&sortBy=version&type=desc)
3. a new major or minor version of [CIS Benchmarks macOS 13 Ventura](https://workbench.cisecurity.org/community/20/benchmarks?q=macos+13.0+Ventura&status=&sortBy=version&type=desc)
4. a release of CIS Benchmarks for [macOS 14 Sonoma](https://workbench.cisecurity.org/community/20/benchmarks?q=sonoma&status=&sortBy=version&type=desc)
5. a new major or minor version of [ChromeOS](https://chromereleases.googleblog.com/search/label/Chrome%20OS)

The DRI should record the latest versions in the [maintenance tracker](https://docs.google.com/spreadsheets/d/1IWfQtSkOQgm_JIQZ0i2y3A8aaK5vQW1ayWRk6-4FOp0/edit#gid=0) and then notify the [#help-product-design Slack channel](https://fleetdm.slack.com/archives/C02A8BRABB5) with an update, noting the current versions and highlighting any changes. 


### Restart Algolia manually
At least once every hour, an Algolia crawler reindexes the Fleet website's content. If an error occurs while the website is being indexed, Algolia will block our crawler and respond to requests with this message: `"This action cannot be executed on a blocked crawler"`.

When this happens, you'll need to manually start the crawler in the [Algolia crawler dashboard](https://crawler.algolia.com/admin/) to unblock it. 
You can do this by logging into the crawler dashboard using the login saved in 1password and clicking the "Restart crawling" button on our crawler's "overview" page](https://crawler.algolia.com/admin/crawlers/497dd4fd-f8dd-4ffb-85c9-2a56b7fafe98/overview).

No further action is needed if the crawler successfully reindexes the Fleet website. If another error occurs while the crawler is running, take a screenshot of the error and add it to the GitHub issue created for the alert and @mention `eashaw` for help.



### New CIS benchmarks
When we create new CIS benchmarks, also submit the new CIS benchmark set to CIS for [certification](https://www.cisecurity.org/cis-securesuite/pricing-and-categories/product-vendor/cis-benchmark-assessment#:~:text=In%20order%20to%20incorporate%20and,recommendations%20in%20the%20associated%20CIS). 

## Rituals

Directly Responsible Individuals (DRI) engage in the ritual(s) below at the frequency specified.

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-------------------------|:----------------------------------------------------|-------------------|
| Design sprint review (#g-endpoint-ops) | Sprintly (Wednesday) | After the last estimation session, the Head of Product reviews the board with each group PM and designer and de-prioritizes all design issues that were not estimated. The Head of Product also collects all items that are product-driven and puts them in the 🎁🗣 Feature fest meeting agenda to consider for continuing work. The number of de-prioritized issues should be recorded in the KPI spreadsheet. | Noah Talerman |
| Design sprint review (#g-mdm) | Sprintly (Thursday) | After the last estimation session, the Head of Product Design reviews the board with each group PM and designer and de-prioritizes all design issues that were not estimated. The group PM collects all items that were de-prioritized and, if desired, puts them in the 🎁🗣 Feature fest meeting agenda to consider for continuing work. The number of de-prioritized issues should be recorded in the KPI spreadsheet. | Noah Talerman |
| 🎁 Feature fest prep | Sprintly (Thursday) | The Head of Product Design reviews the agenda and pre-comments on items in order to be well-prepared for the discussion. | Noah Talerman |
| 🎁🗣 Feature fest  | Sprintly (Thursday) | We make a decision regarding which customer and community feature requests can be committed to in the next six weeks. We create issues for any requests that don't already have one. | Noah Talerman |
| 🎁 Feature fest cleanup | Sprintly (Thursday) | Clean up the agenda in anticipation of the next meeting | Noah Talerman |
| Design sprint kickoff (#g-endpoint-ops) | Sprintly (Thursday) | the Head of Product Design introduces and determines the order of the newly prioritized list of work with each group PM | Noah Talerman |
| Design sprint kickoff (#g-mdm) | Sprintly (Thursday) | the Head of Product Design introduces and determines the order of the newly prioritized list of work with each group PM | Noah Talerman |
| 🗣 Product office hours  | Weekly (Tuesday) | Ask questions to the product team | Noah Talerman |
| Sprint kickoff review | Sprintly (Monday) | After each sprint kickoff, the Head of Product Design reviews the Estimated column with each group EM and de-prioritizes the features that were not included in the sprint and prepares recommended highlights for the release notes. The number of de-prioritized issues should be recorded in the KPI spreadsheet. | Noah Talerman |
| 🦢🗣 Design review (#g-mdm)  | Daily | Review designs from the MDM team | Marko Lisica |
| 🦢🗣 Design review (#g-endpoint-ops)   | Daily | Review designs from the Endpoint ops team | Rachael Shaw |
| Product development process review | Sprintly | CEO, Director of Product Development, and Head of Product get together to review boards and process to make sure everything still makes sense | Noah Talerman |
| Maintenance | Weekly (Friday) | Head of Product Design checks the latest versions of relevant platforms, updates the maintenance tracker, and notifies the #help-product-design Slack channel. | Noah Talerman |
| Quality check  | Daily         | Every day, Head of Product design will review the "Settled" column on the drafting board to ensure all product action items are complete.                                | Noah Talerman |
| Product confirm and celebrate                 | Weekly (Wednesday)       | The Head of Product meets with the designers and product managers to discuss completed user stories. They also verify all updates to documentation, communications, guides, and the pricing and transparency pages, ensuring everything is set for the next steps.                                      | Noah Talerman            |
| Pre-sprint prioritization          | Sprintly (Monday)        | The Head of Product Design and each group's EM meet before each sprint to align on priorities and note what wasn't completed in the previous sprint.                                                                                                                                                              | Noah Talerman            |
| Bug round-up   | Mid-sprint | Head of Product Design will compile a list of churned bugs, including issue numbers, specifics, and age. They will also notify the Customer Success team of any churned bugs that have customer tags  | Noah Talerman |
| Churned bug review                            | Mid-sprint     | The Head of Product Design meets with the Head of Product Development to examine churned bugs and categorize them as either schedule, needs prioritization, or de-prioritize.                                                                    | Churned bug clean-up | Mid-sprint  | Following the churned bug review, Head of Product Design completes the churned bug clean-up, ensuring all necessary follow-up tasks are actioned to classify bugs as schedule, needs prioritization, or de-prioritized. This may include relocating bug tickets, adjusting labels, communicating with stakeholders, writing documentation, and closing issues. | Noah Talerman |                                                                 | Noah Talerman            |
| Stand-up (#g-website)           | Daily (Monday - Thursday) | The website product team meets to discuss completed tasks, upcoming work, and address any questions. | Mike Thomas |
| Prioritization session (#g-website) | Sprintly                | The website product team meets to prioritize tasks for the upcoming sprint. | Mike Thomas |
| Design review (#g-website)      | Daily (Monday - Thursday) | Review designs from the website team.                                       | Mike Thomas |
| PMMs R Us                     | Weekly (Sunday)         | The CEO meets with the Head of Design to discuss product marketing strategies. | Mike Thomas |



## Slack channels

This group maintains the following [Slack channels](https://fleetdm.com/handbook/company#why-group-slack-channels):

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#why-group-slack-channels)|
|:------------------------------------|:--------------------------------------------------------------------|
| `#help-product-design`              | Noah Talerman                                                              |
| `#g-mdm`                            | Noah Talerman                                                       |
| `#g-endpoint-ops`                             | Noah Talerman                                                              |

<meta name="maintainedBy" value="noahtalerman">
<meta name="title" value="🦢 Product design">
