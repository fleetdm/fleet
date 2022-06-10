# Product

⚗️ #g-product: https://github.com/orgs/fleetdm/projects/17

🧱📡 Fleet core roadmap: https://github.com/orgs/fleetdm/projects/8

## Job to be done

Every product should have a single job that it strives to do. We use the [Jobs to be Done
(JTBD) framework](https://about.gitlab.com/handbook/engineering/ux/jobs-to-be-done/). Fleet's
overarching job to be done is the following:

"I need a way to see what laptops and servers I have and what I need to do to keep them secure and
compliant."

## Objectives and key results

Fleet uses objectives and key results (OKRs) to align the organization with measurable
goals.

The product team is responsible for sub-OKRs that contribute to organization-wide OKRs.

### Q2 OKRs

The following Q2 OKRs Google doc lists the "Product" sub-OKRs under each organization-wide OKR:

[Q2 OKRs](https://docs.google.com/document/d/1SfzdeY0mLXSg1Ew0N4yhJppakCgGnDW7Bf8xpKkBczo/edit#heading=h.krtfhfsshh3u)

## Q1 2022 product objectives

In Q1 2022, Fleet set company-wide objectives. The product team was responsible for
the product objectives that contributed to the company-wide objectives.

The following list includes the company-wide objectives, product objectives ("How?" sections), and whether or not the
product team hit or missed each objective:

#### Ultimate source of truth

Fleet + osquery gives organizations the ability to see an almost [endless amount of
data](https://osquery.io/schema/5.1.0/) for all their devices. We want to build on this reputation
by consistently providing the freshest, most accurate, and most understandable data possible
in the UI and API.

##### How?

- Solve the "Undetermined" performance impact limitation for new scheduled queries as well as
reflect unfinished policy runs in the UI/API (Miss).
- Only advertise working osquery tables inside the product by looking at your `wifi_networks` table
  (Miss).

#### Programmable

Fleet differentiates itself from other security tools by providing a simple and easy-to-use API and
CLI tool (fleetctl). This allows users and customers to leverage Fleet's superb ability to gather
device data in unique ways to their organization.

##### How?

- Add integrations for policy and vulnerability automations (Miss).
- Take steps toward feature parity with other vulnerability management solutions (Miss).
- Roll up software and vulnerabilities across the entire organization and teams (Hit).

#### Who's watching the watchers

Many current Fleet users and customers hire Fleet to increase their confidence that other security
tools are functioning properly. We will continue to expose valuable information about these tools to meet customer requirements.

##### How?

- Detect the health of other installed agents and verify device enrollment in Jamf,
Kandji, and SimpleMDM. (Hit)
- Roll up mobile device management (MDM) and Munki data across the entire organization and teams. (Hit)

#### Self-service, 2-way IT

Fleet is poised to enable an organization's employees to resolve issues with their devices on their own. This saves for IT administrators and security practitioners, but it also builds
trust so that an organization can focus on achieving its business outcomes together.

##### How?

- Enable end-users to self-serve issues with their devices using Fleet Desktop(Miss).
- Enable end-users to see what information is collected about their device by maintaining the scope transparency(Hit).

#### Easy to use

We'd like to make maintaining secure laptops and servers as easy as possible.

##### How?

- Improve the standard query library to include 80% of the most common
policies that any organization needs(Miss).

## Product design process

The product team is responsible for product design tasks. These include drafting
changes to the Fleet product, reviewing and collecting feedback from engineering, sales, customer success, and marketing counterparts, and delivering
these changes to the engineering team.

### Drafting

* Move an issue that is assigned to you from the "Ready" column of the [🛸 Product team (weekly) board](https://github.com/orgs/fleetdm/projects/17) to the "In progress" column.

* Create a page in the [Fleet EE (scratchpad, dev-ready) Figma file](https://www.figma.com/file/hdALBDsrti77QuDNSzLdkx/%F0%9F%9A%A7-Fleet-EE-dev-ready%2C-scratchpad?node-id=3923%3A208793) and combine your issue's number and
  title to name the Figma page.

* Draft changes to the Fleet product that solve the problem specified in the issue. Constantly place
  yourself in the shoes of a user while drafting changes. Place these drafts in the appropriate
  Figma page in Fleet EE (scratchpad, dev-ready).

* While drafting, reach out to sales, customer success, and marketing for a new perspective.

* While drafting, engage engineering to gain insight into technical costs and feasibility.

### Review

* Move the issue into the "Ready for review" column. The drafted changes that correspond to each
  issue in this column will be reviewed during the recurring product huddle meeting.

* During the product huddle meeting, record any feedback on the drafted changes.

### Deliver

* Once your work is complete and all feedback is addressed, make sure that the issue is updated with
  a link to the correct page in the Fleet EE (scratchpad) Figma. This page is where the design
  specifications live.

* Add the issue to the 🏛 Architect column in [the 🛸 Product project](https://github.com/orgs/fleetdm/projects/27). This way, an architect on the engineering team knows that the issue is ready for engineering specifications and, later,
  engineering estimation.

#### Priority drafting

Priority drafting is the revision of drafted changes currently being developed by
the engineering team. Priority drafting aims to quickly adapt to unknown edge cases and
changing specifications while ensuring
that Fleet meets our brand and quality guidelines. 

Priority drafting occurs in the following scenarios:

* A drafted UI change is missing crucial information that prevents the engineering team from
  continuing the development task.

* Functionality included in a drafted UI change must be cut down in order to ship the improvement in
  the currently scheduled release.

What happens during priority drafting?

1. Everyone on the product and engineering teams know that a drafted change was brought back
   to drafting and prioritized. 

2. Drafts are updated to cover edge cases or reduce functionality.

3. UI changes are reviewed, and the UI changes are brought back to the engineering team to continue
  the development task.

## Planning

- The intake process for a given group (how new issues are received from a given requestor and estimated within the group's timeframe) is up to each group's PM. For example, the Interface group's intake process consists of attending Interface PM's office hours and making a case, at which time a decision about whether to draft an estimate will be made on the spot.

- New unestimated issues are created in the Planning board, which is shared by each group.

- The estimation process to use is up to the EM of each group (with buy-in from the PM), with the goal of delivering estimated issues within the group's timeframe, which is set for each group by the Head of Product. No matter the group, only work that is slated to be released into the hands of users within ≤six weeks will be estimated. Estimation is run by each group's EM and occurs on the Planning board. Some groups may choose to use "timeboxes" rather than estimates.

- Prioritization will now occur at the point of intake by the PM of the group. Besides the 20% "engineering initiatives," only issues prioritized by the group PM or worked on or estimated. On the first day of each release, all estimated issues are moved into the relevant section of the new "Release" board, which has a kanban view per group. 

- Work that does not "fit" into the scheduled release (due to lack of capacity or otherwise) remains in the "Estimated" column of the product board and is removed from that board if it is not prioritized in the following release.

### Process

1. **Intake:** Each group has a "time til estimated" timeframe, which measures the time from when an idea is first received until it is written up as an estimated issue and the requestor is notified exactly which aspects are scheduled for release. How intake works, and the estimation timeframe, vary per group, but every group has an estimation timeframe.

2. **Estimation:** The estimation process varies per group. In the Interface group, it consists of drafting, API design, and either planning poker or a quick timebox decided by the group EM. When the Interface group relies on the Platform group for part of an issue, only the Interface group's work is estimated. It is up to the Interface PM to obtain estimated Platform issues for any needed work and thus make sure it is scheduled in the appropriate release. It is up to the Platform PM to get those specced (in consultation with Engineering), then up to the Engineering to estimate and communicate promptly if issues arise. We avoid having more estimated issues than capacity in the next release. If the team is fully allocated, no more issues will be estimated, or the PM will decide whether to swap anything out. Once estimated, an issue is scheduled for release. 

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

## Product quality

Fleet uses a human-oriented quality assurance (QA) process to make sure the product meets the standards of users and organizations.

To try Fleet locally for QA purposes, run `fleetctl preview`, which defaults to running the latest stable release.

To target a different version of Fleet, use the `--tag` flag to target any tag in [Docker Hub](https://hub.docker.com/r/fleetdm/fleet/tags?page=1&ordering=last_updated), including any git commit hash or branch name. For example, to QA the latest code on the `main` branch of fleetdm/fleet, you can run: `fleetctl preview --tag=main`

To start preview without starting the simulated hosts, use the `--no-hosts` flag (e.g., `fleetctl preview --no-hosts`).

### Why human-oriented QA?

Automated tests are important, but they can't catch everything. Many issues are hard to notice until a human looks empathetically at the user experience, whether in the user interface, the REST API, or the command line.

The goal of quality assurance is to catch unexpected behavior before release:
- Bugs
- Edge cases
- Error message UX
- Developer experience using the API/CLI
- Operator experience looking at logs
- API response time latency
- UI comprehensibility
- Simplicity
- Data accuracy
- Perceived data freshness
- Product’s ability to save users from themselves


### Collecting bugs

All QA steps should be possible using `fleetctl preview`. Please refer to [docs/Contributing/Testing.md](https://fleetdm.com/docs/contributing/testing) for flows that cannot be completed using `fleetctl preview`.

Please start the manual QA process by creating a blank GitHub issue. As you complete each
flow, record a list of the bugs you encounter in this new issue. Each item in this list should
contain one sentence describing the bug and a screenshot of the item if it is a frontend bug.

### Fleet UI

For all following flows, please refer to the [permissions documentation](https://fleetdm.com/docs/using-fleet/permissions) to make sure that actions are limited to the appropriate user type. Any users with access beyond what this document lists as available should be considered a bug and reported for either documentation updates or investigation.

#### Set up flow

Successfully set up `fleetctl preview` using the preview steps outlined [here](https://fleetdm.com/get-started)

#### Log in and log out flow

Successfully log out and then log in to your local Fleet.

#### Host details page

Select a host from the "Hosts" table as a global user with the Maintainer role. You may create a user with a fake email for this purpose.

You should be able to see and select the "Delete" button on this host's **Host details** page.

You should be able to see and select the "Query" button on this host's **Host details** page.

#### Label flow

`Flow is covered by e2e testing`

Create a new label by selecting "Add a new label" on the host's page. Make sure it correctly filters the host on the host's page.

Edit this label. Confirm users can only edit the "Name" and "Description" fields for a label. Users cannot edit the "Query" field because label queries are immutable.

Delete this label.

#### Query flow

`Flow is covered by e2e testing`

Create a new saved query.

Run this query as a live query against your local machine.

Edit this query and then delete this query.

#### Pack flow

`Flow is covered by e2e testing`

Create a new pack (under Schedule/advanced).

Add a query as a saved query to the pack. Remove this query. Delete the pack.


#### My account flow

Head to the My Account page by selecting the dropdown icon next to your avatar in the top navigation. Select "My account" and successfully update your password. Please do this with an extra user created for this purpose to maintain the accessibility of `fleetctl preview` admin user.


### fleetctl CLI

#### Set up flow

Successfully set up Fleet by running the `fleetctl setup` command.

You may have to wipe your local MySQL database in order to set up Fleet successfully. Check out the [Clear your local MySQL database](#clear-your-local-mysql-database) section of this document for instructions.

#### Log in and log out flow

Successfully log in by running the `fleetctl login` command.

Successfully log out by running the `fleetctl logout` command. Then, log in again.

#### Hosts

Run the `fleetctl get hosts` command.

You should see your local machine returned. If your host isn't showing up, you may have to re-enroll your local machine. Check out the [Orbit for osquery documentation](https://github.com/fleetdm/fleet/blob/main/orbit/README.md) for instructions on generating and installing an Orbit package.

#### Query flow

Apply the standard query library by running the following command:

`fleetctl apply -f docs/01-Using-Fleet/standard-query-library/standard-query-library.yml`

Make sure all queries were successfully added by running the following command:

`fleetctl get queries`

Run the "Get the version of the resident operating system" query against your local machine by running the following command:

`fleetctl query --hosts <your-local-machine-here> --query-name "Get the version of the resident operating system"`

#### Pack flow

Apply a pack by running the following commands:

`fleetctl apply -f docs/Using-Fleet/configuration-files/multi-file-configuration/queries.yml`

`fleetctl apply -f docs/Using-Fleet/configuration-files/multi-file-configuration/pack.yml`

Make sure the pack was successfully added by running the following command:

`fleetctl get packs`

#### Organization settings flow

Apply organization settings by running the following command:

`fleetctl apply -f docs/Using-Fleet/configuration-files/multi-file-configuration/organization-settings.yml`

#### Manage users flow

Create a new user by running the `fleetctl user create` command.

Logout of your current user and log in with the newly created user.


## UI design

### Communicating design changes to the engineering team.
NEW feature that have been added to [Figma Fleet EE (current, dev-ready)](https://www.figma.com/file/qpdty1e2n22uZntKUZKEJl/?node-id=0%3A1):
1. Create a new [GitHub issue](https://github.com/fleetdm/fleet/issues/new)
2. Detail the required changes (including page links to the relevant layouts), then assign the issue to the __"Initiatives"__ project.

<img src="https://user-images.githubusercontent.com/78363703/129840932-67d55b5b-8e0e-4fb9-9300-5d458e1b91e4.png" alt="Assign to Initiatives project"/>

> ___NOTE:___ Artwork and layouts in Figma Fleet EE (current) are final assets, ready for implementation. Therefore, it’s important NOT to use the "idea" label, as designs in this document are more than ideas - they are something that WILL be implemented.

3. Navigate to the [Initiatives project](https://github.com/orgs/fleetdm/projects/8), hit "+ Add cards," pick the new issue, and drag it into the "🤩Inspire me" column. 

<img src="https://user-images.githubusercontent.com/78363703/129840496-54ea4301-be20-46c2-9138-b70bff7198d0.png" alt="Add cards"/>

<img src="https://user-images.githubusercontent.com/78363703/129840735-3b270429-a92a-476d-87b4-86b93057b2dd.png" alt="Inspire me"/>

### Communicating unplanned design changes

For issues related to something that was ALREADY in Figma Fleet EE (current, dev-ready), but __implemented differently__, e.g., padding/spacing inconsistency, etc. Create a [bug issue](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=) and detail the required changes.

### Design conventions

We have certain design conventions that we include in Fleet. We will document more of these over time.

**Table empty states**

Use `---`, with color `$ui-fleet-black-50` as the default UI for empty columns.

**Form behavior**

Pressing the return or enter key with an open form will cause the form to be submitted.

**External links**

For a link that navigates the user to an external site (e.g., fleetdm.com/docs), use the `$core-blue` color and `xs-bold` styling for the link's text. Also, place the link-out icon to the right of the link's text.

## Release 

This section outlines the communication between the product team, growth team, product team,
and customer success team prior to a release of Fleet.

### Goal

Keep the business up to date with improvements and changes to the Fleet product so that all stakeholders can communicate
with customers and users.

### Blog post

The product team is responsible for providing the [growth team](./brand.md) with the necessary information for writing
the release blog post. This is accomplished by filing a release blog post issue and adding
the issue to the growth board on GitHub.

The release blog post issue includes a list of the primary features included in the upcoming
release. This list of features should point the reader to the GitHub issue that explains each
feature in more detail.

Find an example release blog post issue [here](https://github.com/fleetdm/fleet/issues/3465).

### Customer announcement

The product team is responsible for providing the [customer success team](./customers.md) with the necessary information
for writing a release customer announcement. This is accomplished by filing a release customer announcement issue and adding
the issue to the customer success board on GitHub. 


The release blog post issue is filed in the private fleetdm/confidential repository because the
comment section may contain private information about Fleet's customers.

Find an example release customer announcement blog post issue [here](https://github.com/fleetdm/confidential/issues/747).

## Beta features

At Fleet, features are advertised as "beta" if there are concerns that the feature may not work as intended in certain Fleet
deployments. For example, these concerns could be related to the feature's performance in Fleet
deployments with hundreds of thousands of hosts.

The following highlights should be considered when deciding if we promote a feature as "beta:"

- The feature will not be advertised as "beta" permanently. This means that the Directly
  Responsible Individual (DRI) who decides a feature is advertised as "beta" is also responsible for creating an issue that
  explains why the feature is advertised as "beta" and tracking the feature's progress towards advertising the feature as "stable."
- The feature will be advertised as "beta" in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

## Feature flags

At Fleet, features are placed behind feature flags if the changes could affect Fleet's availability of existing functionalities.

The following highlights should be considered when deciding if we should leverage feature flags:

- The feature flag must be disabled by default.
- The feature flag will not be permanent. This means that the Directly Responsible Individual
 (DRI) who decides a feature flag should be introduced is also responsible for creating an issue to track the
  feature's progress towards removing the feature flag and including the feature in a stable
  release.
- The feature flag will not be advertised. For example, advertising in the documentation on fleetdm.com/docs, release notes, release blog posts, and Twitter.

Fleet's feature flag guidelines is borrowed from GitLab's ["When to use feature flags" section](https://about.gitlab.com/handbook/product-development-flow/feature-flag-lifecycle/#when-to-use-feature-flags) of their handbook. Check out [GitLab's "Feature flags only when needed" video](https://www.youtube.com/watch?v=DQaGqyolOd8) for an explanation of the costs of introducing feature flags.

## Competition

We track competitors' capabilities and adjacent (or commonly integrated) products in this [Google Doc](https://docs.google.com/document/d/1Bqdui6oQthdv5XtD5l7EZVB-duNRcqVRg7NVA4lCXeI/edit) (private).

### Intake process

Intake for new product ideas (requests) happens at the 🗣 Product office hours meeting.

At the 🗣 Product office hours meeting, the product team weighs all requests. When the team weighs a request, it is prioritized or put to the side.

The team prioritizes a request when the business perceives it as an immediate priority. When this happens, the team sets the request to be estimated or deferred within five business days.

The team puts a request to the side when the business perceives competing priorities as more pressing in the immediate moment.

#### Why this way?

At Fleet, we use objectives and key results (OKRs) to align the organization with measurable goals.
These OKRs fill up a large portion, but not all, of planning (drafting, wireframing, spec'ing, etc.)
and engineering capacity. 

This means there is always some capacity to prioritize requests advocated for by customers, Fleet team members, and members of the
greater Fleet community.

> Note Fleet always prioritizes bugs.

At Fleet, we tell the requestor whether their
request is prioritized or put to the side within one business day from when the team weighs the request.

The 🗣 Product office hours meeting is a recurring ritual to make sure that the team weighs all requests.

#### Making a request

To make a request or advocate for a request from a customer or community member,  Fleet asks all members of the organization to add their name and a description of the request to the list in the [🗣 Product office hours Google
doc](https://docs.google.com/document/d/1mwu5WfdWBWwJ2C3zFDOMSUC9QCyYuKP4LssO_sIHDd0/edit#heading=h.zahrflvvks7q).
Then attend the next scheduled 🗣 Product office hours meeting.

All members of the Fleet organization are welcome to attend the 🗣 Product office hours meeting. Requests will be
weighed from top to bottom while prioritizing attendee requests. 

This means that if the individual that added a feature request is not in attendance, the feature request will be discussed towards the end of the call if there's time.

All 🗣 Product office hours meetings are recorded and uploaded to the [🗣 Product office hours
folder](https://drive.google.com/drive/folders/1nsjqDyX5WDQ0HJhg_2yOaqBu4J-hqRIW) in the shared
Google drive.

## Rituals

Directly Responsible Individuals (DRI) engage in the ritual(s) below at the frequency specified.

| Ritual                       | Frequency                | Description                                         | DRI               |
|:-----------------------------|:-----------------------------|:----------------------------------------------------|-------------------|
| 🎙 Product huddle | Daily | We discuss "In progress" issues and place any issues that are "ready for review" on the list for the product design review call. On Mondays, issues are broken down into a week's work and added into "ready." We move issues out of "delivered" every Friday. | Noah Talerman |
| 🗣 Product office hours  | Weekly (Tuesdays) | We make a decision regarding which customer and community feature requests can be committed to in the next six weeks. We create issues for any requests that don't already have one. | Noah Talerman |
| 🎨 UI/UX huddle      | Weekly (Wednesdays) | We discuss "In progress" issues and place any issues that are "ready for review" on the list for the product design review call. We hold  separate times for 🎙 Product huddle so Mike Thomas can make it.    | Noah Talerman |
| ✨ Product design review  | Weekly (Thursdays) | The Product team discusses "ready for review" items and makes the decision on whether the UI changes are ready for engineering specification and later implementation. | Noah Talerman |
| 👀 Product review      | Every three weeks | Fleeties present features and improvements in the upcoming release. A discussion is held about bugs, fixes, and changes to be made prior to release.  | Noah Talerman |


## Slack channels

This group maintains the following [Slack channels](https://fleetdm.com/handbook/company#group-slack-channels):

| Slack channel                       | [DRI](https://fleetdm.com/handbook/company#group-slack-channels)    |
|:------------------------------------|:--------------------------------------------------------------------|
| `#help-product`                     | Noah Talerman                                                       |
| `#help-qa`                          | Reed Haynes                                                         |
| `#g-platform`                       | Zach Wasserman                                                      |
| `#g-interface`                      | Noah Talerman                                                       |
| `#g-agent`                          | Mo Zhu                                                              |




<meta name="maintainedBy" value="noahtalerman">
<meta name="title" value="⚗️ Product">
